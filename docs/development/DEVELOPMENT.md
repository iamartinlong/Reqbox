# 开发与验证

给第一次在这个仓库里工作的人。目标：**从零到能跑起来并确认改动没坏**。

---

## 环境要求

| 工具 | 版本 | 用途 |
| ---- | ---- | ---- |
| Go | 1.24+ | 后端与二进制构建 |
| Node | 22+ | 前端构建 |
| pnpm | 9+ | 前端包管理 |

SQLite 驱动是纯 Go 实现（`modernc.org/sqlite`），**不需要 cgo**，因此可以
直接交叉编译。

## 最短路径：直接跑

```bash
go build -o reqbox ./cmd/reqbox
./reqbox
```

打开 <http://localhost:5178>。

前提是 `web/dist` 里已有前端产物。全新克隆时它只有 `.gitkeep`，
二进制仍能启动，但只显示一个提示页 —— 需要先构建前端。

## 完整构建

```bash
# 1. 前端
cd web
pnpm install
pnpm build          # 输出到 web/dist，会被 go:embed 打进二进制

# 2. 后端
cd ..
go build -ldflags "-s -w -X reqbox/internal/server.Version=0.1.0" -o reqbox ./cmd/reqbox
```

或用脚本一次做完（含版本号与跨平台矩阵）：

```bash
./scripts/build.sh 0.1.0          # 当前平台
./scripts/build.sh 0.1.0 --all    # linux/darwin/windows
```

```powershell
.\scripts\build.ps1 -Version 0.1.0
.\scripts\build.ps1 -Version 0.1.0 -All
```

产物在 `dist/`。

## 发版（真正的 CD）

推送一个 `v*` 形式的 tag 即可：CI 会构建前端、重跑测试，然后由 GoReleaser
生成六平台压缩包与校验和并创建 GitHub Release。

```bash
git tag v0.3.0
git push origin v0.3.0
```

| 事项 | 说明 |
| ---- | ---- |
| 产物 | `reqbox_<版本>_<os>_<arch>.tar.gz`，Windows 为 `.zip`，附 `checksums.txt` |
| 平台 | linux / darwin / windows × amd64 / arm64；`CGO_ENABLED=0`（SQLite 是纯 Go 实现） |
| 内容 | 二进制 + `LICENSE` + `README.md` |
| 版本号 | tag 名去掉 `v` 注入 `server.Version`，`reqbox -version` 可查 |
| 预发布 | tag 名带 `-rc1` 之类时自动标为 prerelease |
| 配置 | `.goreleaser.yaml`；工作流在 `.github/workflows/release.yml` |
| 发布说明 | 若存在 `docs/history/release-notes/<tag>.md` 就用它覆盖自动生成的提交列表，否则用 changelog |

**发布说明优先手写。** 自动生成的列表只有 commit 标题，说明不了行为变化。
按 tag 建文件（如 `docs/history/release-notes/v0.3.0.md`），写清亮点、安装方式和
已知限制；没有对应文件的版本自动退回 changelog，不会因此失败。

**先演练再发。** `Release` 工作流也支持手动触发（Actions 页面 → Run
workflow），那种模式下 GoReleaser 加 `--snapshot`：只构建、不发布，产物
仍会作为 workflow artifact 上传，用来确认配置没写坏。

**本地只想看产物：**

```bash
cd web && pnpm build && cd ..
goreleaser release --snapshot --clean
```

`go:embed` 读的是 `web/dist` 的当前内容，所以**必须先构建前端**，否则包里
是「前端未内嵌」的占位页。工作流里已包含这一步。

发版前建议先确认 CI 在要打 tag 的那个提交上是绿的。

## 前端开发（热更新）

两个终端。Vite 会把 `/api` 代理到 Go 服务，因此前端改动实时生效，同时
调用的是真实后端：

```bash
go run ./cmd/reqbox        # 后端 :5178
cd web && pnpm dev          # 前端 :5173（代理到 :5178）
```

## 验证清单

改完代码后至少跑这些：

```bash
go vet ./...        # 静态检查
go test ./...       # 78 个测试
```

前端：

```bash
cd web
pnpm exec tsc -b    # 类型检查
pnpm lint           # oxlint
pnpm test           # 112 个测试
```

### CI 会跑什么

`.github/workflows/ci.yml` 在推送到 `main` 和所有 PR 上运行，三个并行任务：

| 任务 | 内容 |
| ---- | ---- |
| `Go` | `gofmt` 检查、`go vet`、`go test`，Linux + Windows；Linux 额外 `go test -race` |
| `Web` | `pnpm install --frozen-lockfile`、`tsc -b`、`oxlint`、`vitest` |
| `Build` | `pnpm build` → `go build` → 启动二进制打 `/api/health` 和 `/` → 上传产物 |

想在推送前复现 CI 的结果，按上表逐条执行即可；`-race` 那一步需要 C 编译器：

```bash
CGO_ENABLED=1 go test -race ./...
```

CI 的覆盖范围有意比本机验证宽两项：**Windows 上的 Go 测试**（DACL 收紧的
测试带 `//go:build windows`，Linux 上不会执行）和 **`-race`**（本机没有
gcc）。本机验证不能替代 CI，反过来也一样。

### 手工验证（自动化覆盖不到的部分）

前端交互有单元测试覆盖主流程，但真实浏览器里的观感仍需手工确认：

1. 新建请求 → 填 URL → SEND，能拿到响应
2. 切换请求，响应区**不应**残留上一个结果
3. 保存 → 刷新页面，数据仍在
4. 发布 mock → 用 `curl` 访问 `/mock/{slug}` 能通
5. 删除操作弹出的是**应用内**对话框，不是系统对话框
6. 深/浅色主题切换正常

## 目录结构

```
cmd/reqbox/        入口：命令行参数、配置加载、监听与优雅关闭
  listen.go        端口绑定与占用时顺延
  browser.go       启动时打开浏览器
  console_*.go     Windows 控制台模式调整（其他平台为空实现）
internal/config/   reqbox.yaml 解析
internal/store/     SQLite schema、迁移、查询
internal/runner/   模板解析、HTTP 执行、curl 渲染
internal/server/    chi 路由、JSON API、mock 网关、SPA 托管
web/src/
  components/       界面，每个文件一个区域
  components/ui/    shadcn/ui 基础组件（生成，通常不改）
  lib/              API 客户端、类型、格式化、主题、zustand store
docs/
  README.md         索引与维护规则
  onboarding/       交接说明
  planning/         待办与实施计划
  design/           决策记录与上线审计
  development/      开发、构建、发版、旧系统兼容（本文件在这里）
  history/          变更记录、工作日志、发布说明
.github/workflows/  ci.yml（每次推送/PR）与 release.yml（tag）
.goreleaser.yaml    发版产物配置
reqbox.example.yaml 配置文件示例，可直接改名使用
```

## 几个容易踩的点

**配置文件只从二进制旁边读。** `reqbox.yaml` 的位置由 `os.Executable()`
决定，不是当前工作目录。用 `go run` 时二进制在临时目录里，所以那边放的配置
**不会生效**——启动日志会打印实际查找的路径，先看那一行。

**配置优先级是「命令行 > 环境变量 > 文件 > 默认」。** 改文件后没变化，先确认
是不是命令行或环境变量覆盖了它。日志里的 `using config file` 只说明读到了，
不说明它的每个值都被采用。

**端口冲突会自动顺延。** 默认最多试后 20 个端口，并打印
`requested=... using=...`。依赖固定端口的东西（反向代理、书签）应该把
`server.autoPort` 设为 `false`，让冲突变成明确失败而不是悄悄换端口。

**数据库位置。** 默认在二进制旁边的 `data/reqbox.db`；若二进制位于临时目录
（如 `go run` 的构建目录），会退回到系统配置目录。用 `-db` 显式指定可避免
意外。

**改界面后没变化。** 二进制里嵌的是 `web/dist` 的快照。改了前端必须
`pnpm build` 再 `go build`，否则看到的是旧界面。用 `pnpm dev` 开发时不受
此影响。

**新增数据库字段。** 迁移靠 `PRAGMA user_version`，在 `internal/store/store.go`
的 `migrations` 切片**末尾追加**一个语句块。不要修改已发布的迁移 —— 已有
数据库不会重跑它们。

**测试用的时间戳。** `store.Timestamp` 接受 RFC 3339、空串和 null。
浏览器传来的草稿没有时间戳，早先因此导致 400，已有回归测试保护。
