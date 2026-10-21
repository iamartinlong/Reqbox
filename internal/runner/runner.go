package runner

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"reqbox/internal/store"
)

// Opts tunes a single call. Zero values fall back to defaults.
type Opts struct {
	Timeout         time.Duration
	FollowRedirects bool
	VerifyTLS       bool
	MaxBodyBytes    int64
}

// DefaultOpts is what the UI offers before the user changes anything.
func DefaultOpts() Opts {
	return Opts{
		Timeout:         30 * time.Second,
		FollowRedirects: true,
		VerifyTLS:       true,
		MaxBodyBytes:    2 << 20, // capture at most 2 MiB of a response
	}
}

func (o Opts) normalized() Opts {
	d := DefaultOpts()
	if o.Timeout <= 0 {
		o.Timeout = d.Timeout
	}
	if o.MaxBodyBytes <= 0 {
		o.MaxBodyBytes = d.MaxBodyBytes
	}
	return o
}

// Response is the normalized outcome of one call. Body carries the raw bytes;
// BodyText is either the decoded text or base64, matching Encoding.
type Response struct {
	Status      int                 `json:"status"`
	StatusText  string              `json:"statusText"`
	Headers     map[string][]string `json:"headers"`
	Body        []byte              `json:"-"`
	BodyText    string              `json:"body"`
	Encoding    string              `json:"encoding"`
	ContentType string              `json:"contentType"`
	DurationMs  int64               `json:"durationMs"`
	SizeBytes   int64               `json:"sizeBytes"`
	Truncated   bool                `json:"truncated"`
	FinalURL    string              `json:"finalUrl"`
	Cookies     []string            `json:"cookies"`
	ReceivedAt  time.Time           `json:"receivedAt"`
}

// Executor runs requests through a small pool of reusable clients. Clients are
// keyed by the two transport-level choices (redirects, TLS verification) so a
// hot mock endpoint keeps its connections warm.
type Executor struct {
	mu      sync.Mutex
	clients map[string]*http.Client
}

// New builds an executor with no clients yet; they are created on demand.
func New() *Executor {
	return &Executor{clients: map[string]*http.Client{}}
}

// Close releases idle connections for every client.
func (e *Executor) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, c := range e.clients {
		c.CloseIdleConnections()
	}
}

func (e *Executor) client(o Opts) *http.Client {
	key := fmt.Sprintf("%t/%t", o.FollowRedirects, o.VerifyTLS)
	e.mu.Lock()
	defer e.mu.Unlock()
	if c, ok := e.clients[key]; ok {
		return c
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   16,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: !o.VerifyTLS},
	}

	client := &http.Client{Transport: transport}
	if !o.FollowRedirects {
		client.CheckRedirect = func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}
	} else {
		client.CheckRedirect = func(_ *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("stopped after 10 redirects")
			}
			return nil
		}
	}
	e.clients[key] = client
	return client
}

var builtinVars = map[string]func() string{
	"$timestamp": func() string { return strconv.FormatInt(time.Now().Unix(), 10) },
	"$timestampMs": func() string {
		return strconv.FormatInt(time.Now().UnixMilli(), 10)
	},
	"$isoTimestamp": func() string { return time.Now().UTC().Format(time.RFC3339) },
	"$uuid":         newUUID,
	"$randomInt":    func() string { return strconv.Itoa(int(time.Now().UnixNano()%1000) + 1) },
}

// Resolve replaces {{name}} tokens. Unknown tokens are left untouched so a typo
// stays visible instead of silently becoming an empty string.
func Resolve(input string, vars map[string]string) string {
	if !strings.Contains(input, "{{") {
		return input
	}
	var b strings.Builder
	b.Grow(len(input))
	rest := input
	for {
		start := strings.Index(rest, "{{")
		if start < 0 {
			b.WriteString(rest)
			break
		}
		end := strings.Index(rest[start+2:], "}}")
		if end < 0 {
			b.WriteString(rest)
			break
		}
		end += start + 2
		b.WriteString(rest[:start])
		name := strings.TrimSpace(rest[start+2 : end])
		if fn, ok := builtinVars[name]; ok {
			b.WriteString(fn())
		} else if v, ok := vars[name]; ok {
			b.WriteString(v)
		} else {
			b.WriteString(rest[start : end+2])
		}
		rest = rest[end+2:]
	}
	return b.String()
}

// BuildURL joins the base URL with the enabled query params.
func BuildURL(base string, params []store.KV, vars map[string]string) (string, error) {
	base = strings.TrimSpace(Resolve(base, vars))
	if base == "" {
		return "", errors.New("request URL is empty")
	}
	if !strings.Contains(base, "://") {
		base = "http://" + base
	}
	u, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	query := u.Query()
	for _, p := range params {
		if !p.Enabled {
			continue
		}
		key := strings.TrimSpace(Resolve(p.Key, vars))
		if key == "" {
			continue
		}
		query.Add(key, Resolve(p.Value, vars))
	}
	u.RawQuery = query.Encode()
	return u.String(), nil
}

// BuildBody returns the request body reader and the content type it implies.
func BuildBody(r store.Request, vars map[string]string) (io.Reader, string, error) {
	switch strings.ToLower(r.BodyType) {
	case "", "none":
		return nil, "", nil
	case "json":
		body := Resolve(r.Body, vars)
		if strings.TrimSpace(body) == "" {
			return nil, "", nil
		}
		return strings.NewReader(body), "application/json", nil
	case "text", "raw", "xml":
		body := Resolve(r.Body, vars)
		if strings.TrimSpace(body) == "" {
			return nil, "", nil
		}
		ct := "text/plain; charset=utf-8"
		if strings.EqualFold(r.BodyType, "xml") {
			ct = "application/xml"
		}
		return strings.NewReader(body), ct, nil
	case "form":
		values := url.Values{}
		for _, f := range r.Form {
			if !f.Enabled {
				continue
			}
			key := strings.TrimSpace(Resolve(f.Key, vars))
			if key == "" {
				continue
			}
			values.Add(key, Resolve(f.Value, vars))
		}
		return strings.NewReader(values.Encode()), "application/x-www-form-urlencoded", nil
	default:
		return strings.NewReader(Resolve(r.Body, vars)), "text/plain; charset=utf-8", nil
	}
}

// ApplyAuth adds the configured credentials to the request.
func ApplyAuth(req *http.Request, r store.Request, vars map[string]string) {
	if r.AuthType == "" || strings.EqualFold(r.AuthType, "none") {
		return
	}
	switch strings.ToLower(r.AuthType) {
	case "bearer":
		token := strings.TrimSpace(Resolve(r.Auth.Token, vars))
		if token != "" {
			value := token
			if !strings.HasPrefix(strings.ToLower(value), "bearer ") {
				value = "Bearer " + value
			}
			req.Header.Set("Authorization", value)
		}
	case "basic":
		user := Resolve(r.Auth.Username, vars)
		pass := Resolve(r.Auth.Password, vars)
		req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(user+":"+pass)))
	case "apikey":
		key := Resolve(r.Auth.Key, vars)
		value := Resolve(r.Auth.Value, vars)
		if key == "" {
			return
		}
		if strings.EqualFold(r.Auth.In, "query") {
			q := req.URL.Query()
			q.Set(key, value)
			req.URL.RawQuery = q.Encode()
			return
		}
		req.Header.Set(key, value)
	}
}

// Run executes a saved request definition.
func (e *Executor) Run(ctx context.Context, r store.Request, vars map[string]string, o Opts) (Response, error) {
	o = o.normalized()

	target, err := BuildURL(r.URL, r.Params, vars)
	if err != nil {
		return Response{}, err
	}

	method := strings.ToUpper(strings.TrimSpace(r.Method))
	if method == "" {
		method = http.MethodGet
	}

	body, contentType, err := BuildBody(r, vars)
	if err != nil {
		return Response{}, err
	}

	// The deadline lives on the context so a single client can serve many
	// different timeouts without being rebuilt.
	ctx, cancel := context.WithTimeout(ctx, o.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return Response{}, err
	}

	for _, h := range r.Headers {
		if !h.Enabled {
			continue
		}
		key := strings.TrimSpace(Resolve(h.Key, vars))
		if key == "" {
			continue
		}
		req.Header.Set(key, Resolve(h.Value, vars))
	}
	if contentType != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", contentType)
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "*/*")
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "Reqbox/1.0")
	}
	ApplyAuth(req, r, vars)

	start := time.Now()
	res, err := e.client(o).Do(req)
	if err != nil {
		return Response{ReceivedAt: time.Now()}, friendlyError(err)
	}
	defer res.Body.Close()

	raw, readErr := io.ReadAll(io.LimitReader(res.Body, o.MaxBodyBytes+1))
	if readErr != nil && len(raw) == 0 {
		return Response{}, friendlyError(readErr)
	}

	elapsed := time.Since(start)
	truncated := int64(len(raw)) > o.MaxBodyBytes
	if truncated {
		raw = raw[:o.MaxBodyBytes]
	}

	out := Response{
		Status:      res.StatusCode,
		StatusText:  http.StatusText(res.StatusCode),
		Headers:     res.Header,
		Body:        raw,
		ContentType: res.Header.Get("Content-Type"),
		DurationMs:  elapsed.Milliseconds(),
		SizeBytes:   int64(len(raw)),
		Truncated:   truncated,
		FinalURL:    res.Request.URL.String(),
		Cookies:     []string{},
		ReceivedAt:  time.Now(),
	}
	for _, c := range res.Cookies() {
		out.Cookies = append(out.Cookies, c.Name+"="+c.Value)
	}

	mediaType, _, _ := mime.ParseMediaType(out.ContentType)
	if isTextual(mediaType) {
		out.BodyText = string(raw)
		out.Encoding = "text"
	} else {
		out.BodyText = base64.StdEncoding.EncodeToString(raw)
		out.Encoding = "base64"
	}
	return out, nil
}

func isTextual(mediaType string) bool {
	if mediaType == "" {
		return true
	}
	if strings.HasPrefix(mediaType, "text/") {
		return true
	}
	switch mediaType {
	case "application/json", "application/xml", "application/javascript",
		"application/x-www-form-urlencoded", "application/problem+json",
		"application/ld+json", "image/svg+xml", "application/xhtml+xml",
		"application/graphql", "application/yaml", "application/x-yaml":
		return true
	}
	return strings.HasSuffix(mediaType, "+json") || strings.HasSuffix(mediaType, "+xml")
}

// friendlyError turns transport failures into messages a user can act on.
func friendlyError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return errors.New("request timed out")
	}
	if errors.Is(err, context.Canceled) {
		return errors.New("request canceled")
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return errors.New("request timed out")
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return fmt.Errorf("cannot resolve host %q", dnsErr.Name)
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.Err != nil {
		return fmt.Errorf("%s", urlErr.Err.Error())
	}
	return err
}

// EncodeCurl renders the request as a copy-pasteable curl command.
func EncodeCurl(r store.Request, vars map[string]string) string {
	target, err := BuildURL(r.URL, r.Params, vars)
	if err != nil {
		target = strings.TrimSpace(r.URL)
	}
	var b bytes.Buffer
	b.WriteString("curl -X ")
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	if method == "" {
		method = http.MethodGet
	}
	b.WriteString(method)
	b.WriteString(" " + shellQuote(target))

	for _, h := range r.Headers {
		if !h.Enabled || strings.TrimSpace(h.Key) == "" {
			continue
		}
		b.WriteString(" \\\n  -H " + shellQuote(Resolve(h.Key, vars)+": "+Resolve(h.Value, vars)))
	}

	body, contentType, _ := BuildBody(r, vars)
	if body != nil {
		if contentType != "" && !hasContentType(r.Headers) {
			b.WriteString(" \\\n  -H " + shellQuote("Content-Type: "+contentType))
		}
		if content := readerString(body); content != "" {
			b.WriteString(" \\\n  --data-raw " + shellQuote(content))
		}
	}
	return b.String()
}

func hasContentType(headers []store.KV) bool {
	for _, h := range headers {
		if h.Enabled && strings.EqualFold(strings.TrimSpace(h.Key), "content-type") {
			return true
		}
	}
	return false
}

func readerString(r io.Reader) string {
	if r == nil {
		return ""
	}
	b, err := io.ReadAll(r)
	if err != nil {
		return ""
	}
	return string(b)
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
