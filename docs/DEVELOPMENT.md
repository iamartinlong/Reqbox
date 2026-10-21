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
go test ./...       # 41 个测试
```

前端：

```bash
cd web
pnpm exec tsc -b    # 类型检查
pnpm lint           # oxlint
```

### 手工验证（自动化覆盖不到的部分）

前端交互没有单元测试，涉及 UI 的改动需要手工确认：

1. 新建请求 → 填 URL → SEND，能拿到响应
2. 切换请求，响应区**不应**残留上一个结果
3. 保存 → 刷新页面，数据仍在
4. 发布 mock → 用 `curl` 访问 `/mock/{slug}` 能通
5. 删除操作弹出的是**应用内**对话框，不是系统对话框
6. 深/浅色主题切换正常

### 关于 `go test -race`

当前环境没有 gcc，`-race` 需要 cgo，无法运行。并发相关改动请在带 gcc 的
机器或 Linux CI 上补跑：

```bash
CGO_ENABLED=1 go test -race ./...
```

这条已记入 [TODO.md](./TODO.md)。

## 目录结构

```
cmd/reqbox/        入口：命令行参数、优雅关闭
embed.go            //go:embed web/dist
internal/store/     SQLite schema、迁移、查询
internal/runner/    模板解析、HTTP 执行、curl 渲染
internal/server/    chi 路由、JSON API、mock 网关、SPA 托管
web/src/
  components/       界面，每个文件一个区域
  components/ui/    shadcn/ui 基础组件（生成，通常不改）
  lib/              API 客户端、类型、格式化、主题、zustand store
docs/               本目录
```

## 几个容易踩的点

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
