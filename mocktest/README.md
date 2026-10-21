# mocktest — 手工验证用的靶子服务

一个独立的小型 HTTP 服务，用来验证 Reqbox 在各种响应形态下的表现。

```bash
go run ./mocktest          # 监听 127.0.0.1:5250
```

浏览器打开 <http://127.0.0.1:5250> 有带索引的说明页。

## 覆盖的场景

| 类别 | 路由 |
| ---- | ---- |
| 状态码 | `/status/{100-599}` |
| 响应类型 | `/json`、`/json/array`、`/json/nested`、`/text`、`/html`、`/xml` |
| 空响应 | `/empty`（204） |
| 二进制 | `/binary`（2KB）、`/image`（PNG） |
| 异常响应体 | `/invalid-json`（声称 JSON 但语法错误） |
| 大响应 | `/big?mb=3`（测 2MB 截断） |
| 编码 | `/chinese`（中文与 emoji） |
| 错误形态 | `/error/plain`、`/error/json`、`/error/unauthorized` |
| 重定向 | `/redirect/301`、`/302`、`/307`、`/redirect/chain`、`/redirect/loop` |
| 延迟 | `/slow?ms=5000`、`/slow/headers`（头快体慢） |
| 回显 | `/echo`、`/echo/headers`、`/echo/cookies` |
| 其他 | `/query`、`/paged`、`/method/{verb}`、`/auth/required` |

## 为什么单独放一个目录

它不依赖项目其余代码，也不参与构建产物。放在同一仓库是为了让「怎么验证」
和「验证对象」一起版本化——改动了请求执行逻辑时，可以立刻用一致的方式复现。

不属于生产代码，`go build ./cmd/reqbox` 不会包含它。
