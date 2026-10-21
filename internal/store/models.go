package store

import (
	"strings"
	"time"
)

// Timestamp is a time.Time that survives round-tripping through the web client.
// Drafts arrive with empty or null timestamps, which time.Time would reject, so
// unparseable values degrade to the zero time instead of failing the request.
type Timestamp struct {
	time.Time
}

// UnmarshalJSON accepts RFC 3339 strings plus "" and null.
func (t *Timestamp) UnmarshalJSON(data []byte) error {
	raw := strings.Trim(strings.TrimSpace(string(data)), `"`)
	if raw == "" || raw == "null" {
		t.Time = time.Time{}
		return nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		t.Time = time.Time{}
		return nil
	}
	t.Time = parsed
	return nil
}

// MarshalJSON always emits an RFC 3339 string, never null.
func (t Timestamp) MarshalJSON() ([]byte, error) {
	return []byte(`"` + t.UTC().Format(time.RFC3339Nano) + `"`), nil
}

// stamp wraps a time.Time, treating zero values as zero.
func stamp(t time.Time) Timestamp {
	if t.IsZero() {
		return Timestamp{}
	}
	return Timestamp{Time: t.UTC()}
}

// KV is a key/value row used for query params, headers and form fields.
type KV struct {
	ID          string `json:"id"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	Enabled     bool   `json:"enabled"`
	Description string `json:"description"`
}

// Auth holds credentials for the supported auth helpers.
type Auth struct {
	Token    string `json:"token,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Key      string `json:"key,omitempty"`
	Value    string `json:"value,omitempty"`
	In       string `json:"in,omitempty"`
}

// Request is a saved request definition.
type Request struct {
	ID           string  `json:"id"`
	CollectionID *string `json:"collectionId"`
	Name         string  `json:"name"`
	Method       string  `json:"method"`
	URL          string  `json:"url"`
	Params       []KV    `json:"params"`
	Headers      []KV    `json:"headers"`
	BodyType     string  `json:"bodyType"`
	Body         string  `json:"body"`
	Form         []KV    `json:"form"`
	AuthType     string  `json:"authType"`
	Auth         Auth    `json:"auth"`
	SortIndex    int     `json:"sortIndex"`
	// ExposeSlug and Exposed control the public mock endpoint for this request.
	ExposeSlug string    `json:"exposeSlug"`
	Exposed    bool      `json:"exposed"`
	CreatedAt  Timestamp `json:"createdAt"`
	UpdatedAt  Timestamp `json:"updatedAt"`
}

// Collection groups saved requests.
type Collection struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	SortIndex int       `json:"sortIndex"`
	CreatedAt Timestamp `json:"createdAt"`
	UpdatedAt Timestamp `json:"updatedAt"`
}

// EnvVar is a single environment variable.
type EnvVar struct {
	ID      string `json:"id"`
	Key     string `json:"key"`
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
	Secret  bool   `json:"secret"`
}

// Environment is a named set of variables.
type Environment struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	SortIndex int       `json:"sortIndex"`
	Variables []EnvVar  `json:"variables"`
	CreatedAt Timestamp `json:"createdAt"`
	UpdatedAt Timestamp `json:"updatedAt"`
}

// HistoryEntry is one executed request.
type HistoryEntry struct {
	ID              string `json:"id"`
	RequestID       string `json:"requestId"`
	Name            string `json:"name"`
	Method          string `json:"method"`
	URL             string `json:"url"`
	Status          int    `json:"status"`
	DurationMs      int64  `json:"durationMs"`
	SizeBytes       int64  `json:"sizeBytes"`
	OK              bool   `json:"ok"`
	Error           string `json:"error"`
	RequestSnapshot string `json:"requestSnapshot,omitempty"`
	ResponseHeaders string `json:"responseHeaders,omitempty"`
	ResponseBody    string `json:"responseBody,omitempty"`
	BodyEncoding    string `json:"bodyEncoding,omitempty"`
	// Timing is the per-phase breakdown as JSON. Empty for rows recorded before
	// the column existed, which the UI renders as "not available".
	Timing    string    `json:"timing,omitempty"`
	CreatedAt Timestamp `json:"createdAt"`
}

// Export is the portable backup format.
type Export struct {
	Version             string        `json:"version"`
	ExportedAt          Timestamp     `json:"exportedAt"`
	Collections         []Collection  `json:"collections"`
	Requests            []Request     `json:"requests"`
	Environments        []Environment `json:"environments"`
	ActiveEnvironmentID string        `json:"activeEnvironmentId"`
}
