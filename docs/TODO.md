# 待办

按优先级排列。每条都写明**为什么**和**验收方式**，避免只留一句「优化一下」。

图例：`[ ]` 未开始 · `[~]` 进行中 · `[x]` 已完成

---

<a id="p1"></a>

## P1 · 待决策

### [ ] 项目改名（阻塞其他改名工作）

**现状**：项目、Go module、二进制、环境变量、数据库路径全部叫 `reqbox`；
远端仓库叫 `SuperGet`（随手取的）。三者不一致。

**为什么现在不做**：改名是破坏性改动，会同时影响 module 路径、二进制名、
已部署的脚本、用户本地的数据目录。一旦有真实使用者，成本会显著上升。
所以它是一个需要先决策、再动手的事项。

**候选名及占用核查结果**（2026-10 用 pkg.go.dev / 网页搜索确认）：

| 候选 | 核查结果 |
| ---- | -------- |
| `patchbay` | 已占用：Go 生态有 5+ 个同名模块 |
| `shunt` | 已占用：多个模块 |
| `spigot` | 已占用：18 个模块，且 Minecraft 联想过强 |
| `apibench` | 已占用：同名 App |
| `apiforge` | 已占用：同名公司与多个项目 |
| `airlock` | 已占用：已有同名 HTTP 客户端，定位几乎重复 |
| `apibay` | 已占用：撞 The Pirate Bay 的 API |
| **`reqbox`** | 未发现占用（推荐） |
| **`hatchbox`** | 未发现占用 |

**推荐 `reqbox`**：直指 request，与 UI 的 Requests 一致，短、好拼、终端里好敲。

**三个方案，按代价从低到高**：

1. **仅改仓库名**（零风险，推荐先做）
   代码已统一为 `reqbox`；仓库迁移到 GitHub 并以全新状态发布。
   不产生任何破坏。

2. **只改展示名**
   改 README 标题、UI 标题、`<title>`、User-Agent。
   代码标识符、module、二进制、数据路径保持不变。
   代价：代码与展示名不一致，后来者会困惑。

3. **彻底改**
   见下方完整清单。适合在有真实用户前一次到位。

**彻底改的完整清单**（92 处 / 24 个文件，2026-10 核实）：

```bash
# 1. Go module 与全部内部 import
go mod edit -module <name>
# 然后替换所有 internal import 前缀：
#   "reqbox/internal/..."  ->  "<name>/internal/..."

# 2. 目录改名
#   cmd/reqbox/  ->  cmd/<name>/

# 3. 二进制与构建产物名
#   scripts/build.ps1、scripts/build.sh 里的 -o 文件名
#   .gitignore 里的 /reqbox、/reqbox.exe

# 4. 运行时标识
#   环境变量 REQBOX_ADDR / REQBOX_DB / REQBOX_STATIC / REQBOX_DEV
#     -> <NAME>_*
#   User-Agent "Reqbox/1.0"  (internal/runner/runner.go)
#   导出文件名 "reqbox-export.json"（服务端与前端各一处）

# 5. 前端存储键
#   localStorage "reqbox-theme"、"reqbox-session"
#   （改了会让老用户丢主题与发送设置，见下方迁移方案）

# 6. 默认数据路径
#   <exe>/data/reqbox.db
#   ~/.config/reqbox/reqbox.db

# 7. 文档与 UI 文案
#   README.md、web/README.md、web/index.html <title>、
#   web/src/App.tsx（标题与图标 aria）、embed.go 包注释
```

**必须一并处理：数据迁移。**
默认库路径随名字变化后，老用户升级会「看不见」自己的数据。需要：

- 启动时若新路径不存在、而旧路径 `reqbox.db` 存在，自动改用它并打印提示，
  或直接复制过去；
- localStorage 键名变更时，读取不到旧键就回退读旧键一次。

**验收**：

- [ ] 全新克隆能 `go build` 并 `go test ./...` 全绿
- [ ] 老数据库仍能被打开（造一个旧名库，改名后启动验证）
- [ ] 老 localStorage 的主题与发送设置仍在
- [ ] `scripts/build.ps1 -All` 产出的文件名与新名一致

---

## P2 · 已知技术债

### [ ] 并发安全缺少 race detector 验证

`go test -race` 需要 cgo，当前环境无 gcc。
现有的并发结论来自 20 并发执行 / 15 并发写入的手工实测，不是检测器确认的。

**验收**：在有 gcc 的机器（或 Linux CI）跑 `CGO_ENABLED=1 go test -race ./...` 并全绿。

### [ ] 历史快照中的凭证为明文

`requests` 表与 `history.request_snapshot` 都以明文保存 Bearer / Basic /
API Key。这是「能重放」的前提，因此不是简单的脱敏就能解决。

**若要处理，需要先定方向**：

- 方案 A：保留明文，但把数据库文件权限收紧（当前 `0644`），并在 UI 明示风险；
- 方案 B：用主密钥加密 `auth` 字段，重放时解密。需要引入密钥管理，复杂度显著上升。

当前仅在 README 的 Known limits 中说明。

### [ ] 前端无自动化测试

只有 `tsc` 与 `oxlint` 把关，交互靠手工验证。
`web/` 下没有测试框架。

**验收**：至少覆盖「新建 → 保存 → 发送 → 发布」主流程。

---

## P3 · 功能缺口

### [ ] 不支持 WebSocket / SSE

当前只有 HTTP。SSE 相对容易（读取流式响应）；WebSocket 需要独立的连接管理界面。

### [ ] Mock 不转发调用方请求体

已发布端点固定使用保存的 body，调用方的 payload 被忽略。
若要做「透传代理」语义，需要区分 Mock（固定响应）与 Proxy（透传）两种模式。

### [ ] 无请求链式调用 / 前置换算

无法「先登录拿 token，再带着 token 请求」。目前只能靠环境变量手工粘贴。

---

## 已修复（供检索，不必重做）

- TLS 校验可被静默关闭（`verifyTls` 缺省为 false）
- Mock 会把保存的凭证发给调用方指定的主机
- 跨站写入（CSRF）：无 Origin 校验
- 请求无法移回「未分类」（`collectionId: null` 被忽略）
- 导入会静默清空整个工作区，且无确认
- 原生 `prompt` / `confirm` 在部分环境不可用或样式不一致
- 新建草稿因空时间戳导致 400
- 发布 mock 时未填 slug 会产生不可访问的端点
- 切换请求时响应区残留上一个请求的结果
- 请求地址栏比 SEND 按钮矮 13px
- 对话框底部按钮贴边 / 溢出卡片
