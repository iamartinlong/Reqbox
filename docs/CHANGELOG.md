# 变更记录

按时间倒序。重点记录**行为变化**与**修复的缺陷**，不逐条罗列代码改动。
格式参考 [Keep a Changelog](https://keepachangelog.com/)，版本号遵循语义化版本。

---

## [0.1.0] — 2026-10-20

首个可用版本。Go + React 单二进制，请求保存在 SQLite，支持发布 mock 端点。

### 新增

- 请求管理：方法、URL、query 参数、headers、body（JSON / 文本 / XML / 表单）、
  鉴权（Bearer / Basic / API Key，可放 header 或 query）。
- 环境变量：`{{name}}` 占位符；内置 `{{$timestamp}}`、`{{$timestampMs}}`、
  `{{$isoTimestamp}}`、`{{$uuid}}`、`{{$randomInt}}`。
- 服务端执行：请求由 Go 发出，规避浏览器 CORS；连接池按
  「跟随重定向 / 校验 TLS」复用。
- 历史记录：每次执行的耗时、体积、状态、响应头、响应体，以及请求快照；
  支持重放。
- Mock 发布：任意已保存请求可发布到 `/mock/{slug}`。
- cURL 导出、工作区导入导出、深色/浅色主题。
- 构建脚本：`scripts/build.ps1`、`scripts/build.sh`，支持跨平台矩阵。

### 修复（开发与自测阶段）

按发现顺序，均已加入回归测试或手工验证清单：

| 问题 | 影响 | 根因 |
| ---- | ---- | ---- |
| 新建草稿发送 400 | 无法发出任何新请求 | 空时间戳无法被 `time.Time` 解析 |
| 发布未填 slug 产生死链 | 端点无法访问 | 未从名称推导 slug |
| 切换请求后响应残留 | 误读成另一个请求的结果 | 响应未与请求 id 绑定 |
| 地址栏比 SEND 矮 13px | 视觉不对齐 | 行高被 `<input>` 内容盒撑起，而非固定轨道 |
| 对话框按钮贴边/溢出 | 布局破损 | `DialogFooter` 的负边距依赖父级 `p-4`，被覆盖后失效 |
| `New collection` 点击无反应 | 功能不可用 | `window.prompt` 在部分环境直接抛异常 |
| 删除仍用原生对话框 | 与整体风格不一致 | 未替换 `window.confirm` |

### 安全修复（审计阶段）

详见 [DECISIONS.md](./DECISIONS.md) 的 DR-003、DR-004。

| 问题 | 影响 |
| ---- | ---- |
| TLS 校验可被静默关闭 | 只传 `timeoutSeconds` 时 `verifyTls` 落为零值 `false`，自签名站点被放行 |
| Mock 泄露凭证 | 调用方可通过 query 覆盖主机名，使保存的 `Authorization` 被发往他处 |
| 跨站写入（CSRF） | 无 Origin 校验，任意网页可 `text/plain` 简单请求改写工作区 |

### 已知限制

- 仅 HTTP，不支持 WebSocket / SSE。
- 无鉴权机制，凭证以明文存储（见 TODO）。
- Mock 实时执行，不转发调用方 body。
- 未做 race detector 验证（环境缺 cgo/gcc）。

---

## 未发布

下一版本的内容见 [TODO.md](./TODO.md)。当前唯一待决策项是项目改名。
