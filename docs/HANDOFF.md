# 交接说明

面向接手这个仓库的人。这里只写「拿到代码后怎么跑起来、当前是什么状态、
哪些地方容易踩坑」，不再重复设计理由（那在 DECISIONS.md）。

**最后更新**：2026-10-21

---

## 一、这是什么

一个本地运行的单二进制 API 工作台：用 SQLite 保存请求定义，在浏览器里编辑
并执行，并把任意一条已保存的请求「再发布」成一个可供他人调用的 mock 端点。

不需要账号、不需要云端、不需要额外的服务进程。

技术栈：Go（后端 + 内嵌前端）+ React + Tailwind + shadcn/ui（Vite 构建）。

---

## 二、仓库与分支

| 项 | 值 |
| -- | -- |
| 仓库 | <https://github.com/iamartinlong/Reqbox> |
| `main` | `e2c67e9` — 当前开发主线，Go 1.24.1 |
| `legacy/win2008r2` | `20118de` — Windows Server 2012 R2，Go 1.20 |
| 许可证 | MIT |

**仓库是重新初始化的**，只有这两个分支、共 2 个提交。旧仓库（Gitea /
`SuperGet`）的历史已废弃，不迁移。

### 两个分支的关系

`legacy/win2008r2` 不是 main 的旧版本，而是**针对旧操作系统的独立分支**：

- main 用 `log/slog`（Go 1.21+）；legacy 用自写的 `internal/logging`
  （Go 1.20 没有 slog）
- legacy 的依赖版本被钉在能编译 Go 1.20 的版本上
- 两者**不能直接合并**。改动若同时影响两边，需要手工各改一次

只有需要在 Windows Server 2016 之前的系统上运行时，才用 legacy 分支。

---

## 三、环境要求

| 工具 | 版本 | 说明 |
| ---- | ---- | ---- |
| Go | 1.24.1 | main 分支 |
| Go | 1.20.14 | **仅** legacy 分支，见下文 |
| Node | 22.x | |
| pnpm | 9.x | 前端包管理器 |

---

## 四、从零跑起来

```bash
# 1. 前端（必须先做，否则 go:embed 没有内容）
cd web
pnpm install
pnpm build          # 产物写入 web/dist

# 2. 后端
cd ..
go build -o reqbox ./cmd/reqbox
./reqbox
```

启动后会打印 API token 和一条可直接打开的登录链接：

```
msg="API token generated; open this URL once to sign in"
  token=8fK2…  url="http://127.0.0.1:5178/?token=8fK2…"
```

打开那个 URL 即完成登录。token 只以 SHA-256 哈希存库，明文只打印这一次。

**忘记 token 时**：用 `-token` 指定一个自己记得的（至少 8 字符），
或用 `-no-auth` 关闭校验（会打印警告）。

### 忘了先 `pnpm build`

`go build` 仍然成功，但页面会显示一个占位提示，说明前端未内嵌。这是
`web/dist/.gitkeep` 占位机制导致的，不是构建失败。

---

## 五、常用命令

```bash
# 后端
go test ./...                    # 78 个测试
go vet ./...
gofmt -l internal/ cmd/

# 前端（在 web/ 目录）
pnpm test                        # 112 个测试
pnpm exec tsc -b                 # 类型检查
pnpm lint                        # oxlint
pnpm build

# 一键构建全部平台产物（可选）
./scripts/build.ps1 -All         # Windows
./scripts/build.sh               # 类 Unix
```

CI（`.github/workflows/ci.yml`）会把这些各跑一遍，额外两项本机做不到：
Windows 上的 Go 测试，以及 Linux 上的 `go test -race`。推送前想本地复现，
见 [DEVELOPMENT.md](./DEVELOPMENT.md) 的「CI 会跑什么」。

---

## 六、容易踩的坑

### 1. 默认只监听 127.0.0.1

不是 `:5178`。要从别的机器访问，必须显式指定：

```bash
./reqbox -addr :5178          # 或 -addr 0.0.0.0:5178
```

启动时会打印警告，说明这会把 API 暴露给整个网段。

### 2. legacy 分支不能用新版 Go 构建

必须用真实的 Go 1.20 工具链。用 Go 1.24 编译**能通过**，但产物在
Windows Server 2012 R2 上会立即退出——这是最难排查的一类失败。

```bash
# 让 Go 自动下载并使用 1.20
GOTOOLCHAIN=go1.20.14 go build ./cmd/reqbox

# 或用脚本（会自动找 1.20 工具链）。此脚本只存在于 legacy 分支，
# 在 main 上是找不到的，必须先切换分支。
git checkout legacy/win2008r2
./scripts/build-legacy.ps1
```

### 3. 不要在这个分支上跑 `go mod tidy`

用新版 Go 跑 `tidy` 会把 `go 1.20` 抬到 1.21、并写入 `toolchain` 行，
直接破坏该分支存在的意义。需要改依赖时用 `go mod edit`。

改完务必验证：

```bash
# 此脚本只存在于 legacy 分支（在 main 上不存在）
./scripts/check-legacy-pins.ps1     # 5 项 pin 必须全部 OK
```

### 4. 凭据以明文存于数据库

`requests` 与 `history.request_snapshot` 里都是明文 Bearer / Basic / API Key。
数据库文件权限已收紧到当前账户，但**能读到文件的人就能读出全部凭据**。

这是有意的范围决定，理由与将来如何加密记录在
[TODO.md](./TODO.md) 的「历史快照中的凭证为明文」一节。

### 5. 前端测试需要 `window.matchMedia` 桩

jsdom 没有实现它。`web/src/test/setup.ts` 里有桩，不要删，否则所有渲染
`CodeEditor` 的组件测试都会失败。

### 6. Tailwind 是手工接入的

shadcn/ui 的 `data-active` / `data-horizontal` 变体**不是内置的**。
`web/src/index.css` 里用 `@custom-variant` 声明了它们；命名 group 变体
（`group-data-horizontal/tabs`）无法声明，改用 `data-slot` 的普通 CSS 规则。
删掉这些会导致标签选中态失效（曾经发生过，表现为下划线宽度 0px）。

---

## 七、当前状态

| 项 | 状态 |
| -- | ---- |
| Go 测试 | 78 通过 |
| 前端测试 | 112 通过 |
| CI | `.github/workflows/ci.yml`，Linux + Windows 上的 Go 测试（含 `-race`）、前端测试与二进制冒烟（**待首次运行**） |
| `tsc` / `lint` / `vet` | 干净 |
| 上线审计 | 已完成，见 [AUDIT.md](./AUDIT.md) |
| 已知阻塞项 | 无 |

### 已完成的主要能力

- 请求的保存、执行、历史（SQLite）
- 环境变量与 `{{placeholder}}` 替换
- 把任意请求发布为 mock 端点（实时执行，非缓存回放）
- 响应区：Headers（含请求头、Cookies、耗时明细）/ Request Body / Preview / Response
- 分阶段耗时：DNS / 连接 / TLS / 首字节 / 下载
- 中英双语界面，跟随浏览器语言，可手动切换
- API token 鉴权 + 数据库文件权限收紧

### 下一步可以做的

见 [TODO.md](./TODO.md)。按优先级排，每条都有工时估算与验收方式。

其中最值得先看的两项：

- **加密落盘**（1.5 天）：目前凭据明文存盘。触发条件是「数据库文件可能
  离开这台机器」。
- **SSE 支持**（1～1.5 天）：需要调试流式接口时。

---

## 八、内部信息说明

这个仓库曾服务于一个具体客户项目，公开前已清理：

- 删除了 `docs/HUSHU.md`、`scripts/hushu-import.json`、`scripts/import-hushu.ps1`
- 文档中相关的接口说明、内部项目名、包名一并移除

因此**仓库里没有任何客户业务信息**。如果将来需要恢复那批接口定义，旧资料
不在此仓库内，需要另找来源。

---

## 九、删除本地目录前要知道的事

`E:\Project\5-义乌有线\projects\ApiMock` 目录删除后，以下内容**不会被
重新拉取**（它们本来就不在 git 里）：

| 内容 | 是否需要保留 |
| ---- | ------------ |
| `data/*.db` | 本地测试数据，可丢 |
| `dist/*` | 构建产物，可重新生成 |
| `web/node_modules` | 重新 `pnpm install` 即可 |
| API token | 存在数据库里；新库会生成新的 |

**如果你的库里有还想保留的请求定义**，删目录前先导出：

```bash
# 需要服务在运行
curl -H "Authorization: Bearer <你的token>" http://127.0.0.1:5178/api/export -o backup.json
```

重新拉取后，启动服务，在侧栏的导入按钮里选这个文件恢复。

---

## 十、重新拉取

```bash
git clone git@github.com:iamartinlong/Reqbox.git
cd Reqbox

cd web && pnpm install && pnpm build && cd ..
go build -o reqbox ./cmd/reqbox
./reqbox
```

克隆后**必须先 `pnpm build`**，否则前端不会内嵌（见上文「忘了先 pnpm build」）。

如果需要 legacy 分支：

```bash
git checkout legacy/win2008r2
GOTOOLCHAIN=go1.20.14 go build ./cmd/reqbox
```

---

## 十一、需要联系原维护者确认的事

- 生产环境（Windows Server 2012 R2）上现存的数据库是否要迁移
- 是否需要保留旧 Gitea 仓库作为归档
- 客户接口的原始来源（不在本仓库内）
