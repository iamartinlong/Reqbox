package main

// 测试用 mock 服务。
//
// 覆盖各种 HTTP 状态码与异常场景，供 Reqbox 手工验证请求/响应展示。
// 端口 5250，路由见 GET / 返回的索引页。
//
// 启动：go run ./mocktest
// 或：  go build -o mocktest.exe ./mocktest && ./mocktest.exe

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

func main() {
	mux := http.NewServeMux()

	// ---- 索引 -------------------------------------------------------------
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			notFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, indexHTML)
	})

	// ---- 状态码 -----------------------------------------------------------
	// /status/200 ... /status/599，返回对应状态码与简短 JSON
	mux.HandleFunc("/status/", func(w http.ResponseWriter, r *http.Request) {
		code := parseCode(r.URL.Path, "/status/")
		if code < 100 || code > 599 {
			http.Error(w, "status code out of range", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(code)
		fmt.Fprintf(w, `{"requestedStatus":%d,"message":%q}`, code, http.StatusText(code))
	})

	// ---- 业务形态 ---------------------------------------------------------
	mux.HandleFunc("/json", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":   true,
			"data": sampleUser(),
		})
	})

	mux.HandleFunc("/json/array", func(w http.ResponseWriter, r *http.Request) {
		items := make([]map[string]any, 0, 5)
		for i := 1; i <= 5; i++ {
			items = append(items, map[string]any{
				"id": i, "name": fmt.Sprintf("user-%d", i),
			})
		}
		writeJSON(w, http.StatusOK, items)
	})

	mux.HandleFunc("/json/nested", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"level1": map[string]any{
				"level2": map[string]any{
					"level3": map[string]any{
						"items": []any{1, 2, 3},
						"meta":  map[string]any{"deep": true},
					},
				},
			},
		})
	})

	mux.HandleFunc("/empty", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent) // 204，无响应体
	})

	mux.HandleFunc("/text", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, "这是一段纯文本响应。\n第二行。\n")
	})

	mux.HandleFunc("/html", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, "<h1>HTML 响应</h1><p>用于测试非 JSON 的文本类型。</p>")
	})

	mux.HandleFunc("/xml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		fmt.Fprint(w, `<?xml version="1.0"?><root><item id="1">hello</item></root>`)
	})

	mux.HandleFunc("/binary", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", `attachment; filename="blob.bin"`)
		// 2KB 伪随机二进制（含 0x00，确保不是文本）
		buf := make([]byte, 2048)
		rand.New(rand.NewSource(42)).Read(buf)
		_, _ = w.Write(buf)
	})

	mux.HandleFunc("/image", func(w http.ResponseWriter, r *http.Request) {
		// 1x1 透明 PNG
		png := []byte{
			0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
			0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
			0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
			0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
			0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41,
			0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
			0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00,
			0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
			0x42, 0x60, 0x82,
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(png)
	})

	mux.HandleFunc("/invalid-json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"broken": tru, "missing": `) // 故意不合法
	})

	mux.HandleFunc("/big", func(w http.ResponseWriter, r *http.Request) {
		// 默认 3MB，可用 ?mb=5 调整；用于测试响应体截断
		mb := 3
		if v := r.URL.Query().Get("mb"); v != "" {
			fmt.Sscanf(v, "%d", &mb)
		}
		if mb <= 0 || mb > 20 {
			mb = 3
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"payload":"`))
		chunk := strings.Repeat("x", 1024)
		for i := 0; i < mb*1024; i++ {
			w.Write([]byte(chunk))
		}
		w.Write([]byte(`"}`))
	})

	mux.HandleFunc("/chinese", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"message": "中文响应，用于检查字符编码",
			"emoji":   "✅ 🎯 🚀",
			"mixed":   "abc中文123",
		})
	})

	// ---- 错误形态 ---------------------------------------------------------
	mux.HandleFunc("/error/plain", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "出错了：这是纯文本错误", http.StatusInternalServerError)
	})

	mux.HandleFunc("/error/json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"code":"INTERNAL_ERROR","message":"服务端内部错误","traceId":"abc-123"}`)
	})

	mux.HandleFunc("/error/unauthorized", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", `Bearer realm="mock"`)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":"invalid_token"}`)
	})

	// ---- 重定向 -----------------------------------------------------------
	mux.HandleFunc("/redirect/301", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/json", http.StatusMovedPermanently)
	})
	mux.HandleFunc("/redirect/302", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/json", http.StatusFound)
	})
	mux.HandleFunc("/redirect/307", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/json", http.StatusTemporaryRedirect)
	})
	// 重定向链：3 跳
	mux.HandleFunc("/redirect/chain", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/redirect/chain/2", http.StatusFound)
	})
	mux.HandleFunc("/redirect/chain/2", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/redirect/chain/3", http.StatusFound)
	})
	mux.HandleFunc("/redirect/chain/3", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/json", http.StatusFound)
	})
	// 无限重定向，用于测 10 次上限
	mux.HandleFunc("/redirect/loop", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/redirect/loop", http.StatusFound)
	})

	// ---- 超时与延迟 -------------------------------------------------------
	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		// 默认 5 秒，可用 ?ms=10000 调整
		ms := 5000
		if v := r.URL.Query().Get("ms"); v != "" {
			fmt.Sscanf(v, "%d", &ms)
		}
		select {
		case <-time.After(time.Duration(ms) * time.Millisecond):
		case <-r.Context().Done():
			return // 客户端已断开
		}
		writeJSON(w, http.StatusOK, map[string]any{"sleptMs": ms})
	})

	mux.HandleFunc("/slow/headers", func(w http.ResponseWriter, r *http.Request) {
		// 头快、体慢：立即发头，之后逐字节缓慢输出
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)
		for i := 0; i < 20; i++ {
			select {
			case <-time.After(300 * time.Millisecond):
			case <-r.Context().Done():
				return
			}
			fmt.Fprintf(w, "chunk-%d\n", i)
			if flusher != nil {
				flusher.Flush()
			}
		}
	})

	// ---- 回显 -------------------------------------------------------------
	mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(io.LimitReader(r.Body, 2<<20))
		headers := map[string][]string(r.Header)
		writeJSON(w, http.StatusOK, map[string]any{
			"method":  r.Method,
			"path":    r.URL.Path,
			"query":   r.URL.RawQuery,
			"headers": headers,
			"body":    string(body),
			"bodyLen": len(body),
		})
	})

	mux.HandleFunc("/echo/headers", func(w http.ResponseWriter, r *http.Request) {
		// 回传自定义响应头，便于检查 headers 展示
		w.Header().Set("X-Custom-One", "value-1")
		w.Header().Set("X-Custom-Multi", "a")
		w.Header().Add("X-Custom-Multi", "b")
		w.Header().Set("X-Request-Id", fmt.Sprintf("req-%d", time.Now().UnixNano()))
		writeJSON(w, http.StatusOK, map[string]any{"headers": "see response headers"})
	})

	mux.HandleFunc("/echo/cookies", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc123", Path: "/", HttpOnly: true})
		http.SetCookie(w, &http.Cookie{Name: "theme", Value: "dark", Path: "/"})
		writeJSON(w, http.StatusOK, map[string]any{"cookies": 2})
	})

	// ---- 方法 -------------------------------------------------------------
	for _, m := range []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"} {
		method := m
		mux.HandleFunc("/method/"+strings.ToLower(method), func(w http.ResponseWriter, r *http.Request) {
			if r.Method != method {
				w.Header().Set("Allow", method)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusMethodNotAllowed)
				fmt.Fprintf(w, `{"expected":%q,"got":%q}`, method, r.Method)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"method": r.Method})
		})
	}

	// ---- 查询参数 ---------------------------------------------------------
	mux.HandleFunc("/query", func(w http.ResponseWriter, r *http.Request) {
		values := map[string][]string(r.URL.Query())
		writeJSON(w, http.StatusOK, map[string]any{
			"raw":    r.URL.RawQuery,
			"parsed": values,
		})
	})

	// ---- 分页模拟 ---------------------------------------------------------
	mux.HandleFunc("/paged", func(w http.ResponseWriter, r *http.Request) {
		page, size := 1, 10
		fmt.Sscanf(r.URL.Query().Get("page"), "%d", &page)
		fmt.Sscanf(r.URL.Query().Get("size"), "%d", &size)
		if page < 1 {
			page = 1
		}
		if size < 1 || size > 100 {
			size = 10
		}
		items := make([]map[string]any, 0, size)
		for i := 0; i < size; i++ {
			id := (page-1)*size + i + 1
			items = append(items, map[string]any{"id": id, "name": fmt.Sprintf("item-%d", id)})
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"page": page, "size": size, "total": 95, "items": items,
		})
	})

	// ---- 认证检查 ---------------------------------------------------------
	mux.HandleFunc("/auth/required", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			w.Header().Set("WWW-Authenticate", `Bearer realm="mock"`)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, `{"error":"missing Authorization header"}`)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"authorization": auth})
	})

	log.Println("mock service listening on http://127.0.0.1:5250")
	log.Fatal(http.ListenAndServe("127.0.0.1:5250", logRequests(mux)))
}

// ---- helpers --------------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func parseCode(path, prefix string) int {
	s := strings.TrimPrefix(path, prefix)
	code := 0
	fmt.Sscanf(s, "%d", &code)
	return code
}

func notFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, `{"error":"no such route","path":%q}`, r.URL.Path)
}

func sampleUser() map[string]any {
	return map[string]any{
		"id":        1001,
		"name":      "张三",
		"email":     "zhangsan@example.com",
		"active":    true,
		"age":       32,
		"tags":      []string{"vip", "beta"},
		"createdAt": "2026-01-15T08:30:00Z",
		"profile": map[string]any{
			"city":    "Shanghai",
			"balance": 128.50,
			"avatar":  nil,
		},
	}
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%-7s %-40s %s", r.Method, r.URL.RequestURI(), time.Since(start).Round(time.Millisecond))
	})
}

const indexHTML = `<!doctype html>
<html lang="zh"><head><meta charset="utf-8"><title>Mock Service</title>
<style>
body{font:14px/1.6 ui-monospace,Menlo,Consolas,monospace;max-width:900px;margin:40px auto;padding:0 20px;color:#222}
h1{font-size:20px;margin-bottom:4px} .sub{color:#777;margin-bottom:28px}
h2{font-size:14px;margin:26px 0 8px;padding-bottom:4px;border-bottom:1px solid #ddd;text-transform:uppercase;letter-spacing:.08em;color:#555}
table{width:100%;border-collapse:collapse} td{padding:3px 8px 3px 0;vertical-align:top}
td:first-child{white-space:nowrap;color:#0b5;width:300px}
td:last-child{color:#666}
code{background:#f4f4f4;padding:1px 4px;border-radius:3px}
</style></head><body>
<h1>Mock Service</h1>
<div class="sub">127.0.0.1:5250 &mdash; 供 Reqbox 验证各种请求/响应形态</div>

<h2>状态码</h2>
<table>
<tr><td>/status/{code}</td><td>任意 100-599，如 <code>/status/404</code></td></tr>
</table>

<h2>业务形态</h2>
<table>
<tr><td>/json</td><td>标准 JSON 对象</td></tr>
<tr><td>/json/array</td><td>JSON 数组</td></tr>
<tr><td>/json/nested</td><td>多层嵌套</td></tr>
<tr><td>/empty</td><td>204 无内容</td></tr>
<tr><td>/text</td><td>纯文本</td></tr>
<tr><td>/html</td><td>HTML</td></tr>
<tr><td>/xml</td><td>XML</td></tr>
<tr><td>/binary</td><td>2KB 二进制</td></tr>
<tr><td>/image</td><td>PNG 图片</td></tr>
<tr><td>/invalid-json</td><td>声称 JSON 但不合法</td></tr>
<tr><td>/big?mb=3</td><td>大响应，测截断</td></tr>
<tr><td>/chinese</td><td>中文与 emoji</td></tr>
</table>

<h2>错误形态</h2>
<table>
<tr><td>/error/plain</td><td>500 纯文本</td></tr>
<tr><td>/error/json</td><td>500 JSON 错误体</td></tr>
<tr><td>/error/unauthorized</td><td>401 + WWW-Authenticate</td></tr>
</table>

<h2>重定向</h2>
<table>
<tr><td>/redirect/301 | 302 | 307</td><td>单跳</td></tr>
<tr><td>/redirect/chain</td><td>3 跳链</td></tr>
<tr><td>/redirect/loop</td><td>无限重定向（测上限）</td></tr>
</table>

<h2>超时与延迟</h2>
<table>
<tr><td>/slow?ms=5000</td><td>延迟响应</td></tr>
<tr><td>/slow/headers</td><td>头快体慢（逐块输出）</td></tr>
</table>

<h2>回显</h2>
<table>
<tr><td>/echo</td><td>回显方法/头/体</td></tr>
<tr><td>/echo/headers</td><td>返回多个自定义头</td></tr>
<tr><td>/echo/cookies</td><td>返回 2 个 Cookie</td></tr>
<tr><td>/query?x=1&y=2</td><td>回显查询参数</td></tr>
<tr><td>/paged?page=2&size=5</td><td>分页数据</td></tr>
</table>

<h2>方法与认证</h2>
<table>
<tr><td>/method/{get|post|put|patch|delete|head|options}</td><td>校验方法</td></tr>
<tr><td>/auth/required</td><td>无 Authorization 则 401</td></tr>
</table>

<h2>提示</h2>
<p style="color:#666">
<code>?ms=</code>、<code>?mb=</code> 等参数可用于调整强度。所有响应都会在控制台打日志。
</p>
</body></html>`
