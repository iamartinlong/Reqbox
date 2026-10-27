# 交接说明

面向接手这个仓库的人。这里只写「拿到代码后怎么跑起来、当前是什么状态、
哪些地方容易踩坑」，不再重复设计理由（那在 DECISIONS.md）。

**最后更新**：2026-10-27（交接收尾）

---

## 零、三十秒速览

如果你只有一分钟：

1. **能用的最新版本**：`v0.3.0`（下载即用），包含配置文件、端口冲突顺延、
   启动自动开浏览器。
2. **构建**：`cd web && pnpm install && pnpm build`，然后 `go build -o reqbox ./cmd/reqbox`。
3. **测试**：`go test ./...`（96 个）、`cd web && pnpm test`（112 个）。
4. **没有阻塞项、没有未完成的半成品**。剩余工作是按需功能。
5. **旧系统的兼容包**在另一个分支 `legacy/win2008r2`，Go 1.20 是硬要求。

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
| `main` | `a83df90` — 当前开发主线，Go 1.24.1 |
| `legacy/win2008r2` | `dba69f1` — Windows Server 2008 R2 及更早，Go 1.20.14 |
| 最新发布 | `v0.3.0`（主分支）、`v0.1.2-win2008r2`（兼容包） |
| 许可证 | MIT |

**仓库是重新初始化的**，只有这两个分支。旧仓库（Gitea / `SuperGet`）的历史
已废弃，不迁移。

### 两个分支的关系

`legacy/win2008r2` 不是 main 的旧版本，而是**针对旧操作系统的独立分支**：

- main 用 `log/slog`（Go 1.21+）；legacy 用自写的 `internal/logging`
  （Go 1.20 没有 slog）
- legacy 的依赖版本被钉在能编译 Go 1.20 的版本上
- 两者**不能直接合并**。改动若同时影响两边，需要手工各改一次
- 两边都有完整的 CI 与发版流程，但工作流文件各自维护

两个分支的基本情况对比：

| 项 | `main` | `legacy/win2008r2` |
| -- | ------ | ------------------ |
| Go | 1.24.1 | **1.20.14**（硬要求） |
| 目标系统 | Windows 10 / Server 2016+ | **Server 2008 R2 SP1 (NT 6.1)** 及更早 |
| 已发布版本 | `v0.3.0` | `v0.1.2-win2008r2` |
| 产物 | 六平台压缩包 | 仅 `windows/amd64` 兼容包 |
| Go 测试 | 96 个 | 80 个 |
| 兼容性守卫 | 无（不需要） | `scripts/legacyguard` |
| 前端测试 | 112 个（Vitest） | 无（只构建） |
| 交接文档 | 本文件 | 该分支的 `docs/onboarding/HANDOFF.md` |

只有需要在 Windows Server 2016 之前的系统上运行时，才用 legacy 分支。

**两个分支的发布节奏不同步**：`main` 的 `v0.3.0` 与 legacy 的 `v0.1.2` 都包含
配置文件、端口顺延、自动开浏览器，但 `v0.1.2` 先发了两天，因为那台 2008 R2
机器正被这三个问题卡着。现在两边功能一致。

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

**只想用，不想构建**：从 [Releases](https://github.com/iamartinlong/Reqbox/releases)
下载对应平台的压缩包，解开即用（前端已内嵌）。

从源码构建：

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
go test ./...                    # 96 个测试
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

**在 Windows 上 `gofmt -l` 会列出整棵树**，那是 CRLF 换行导致的误报，不是
格式问题：仓库里的 blob 是 LF，检出成 CRLF 后 gofmt 就会抱怨。校验时用
`git archive` 导出到临时目录再跑，或者直接看 CI 的 Linux 任务。

CI（`.github/workflows/ci.yml`）会把这些各跑一遍，额外两项本机做不到：
Windows 上的 Go 测试，以及 Linux 上的 `go test -race`。推送前想本地复现，
见 [DEVELOPMENT.md](../development/DEVELOPMENT.md) 的「CI 会跑什么」。

发版：

```bash
# 1. 写发布说明（可选，缺省退回自动 changelog）
$EDITOR docs/history/release-notes/v0.3.0.md

# 2. 打 tag 并推送，剩下的交给 CI
git tag v0.3.0 && git push origin v0.3.0
```

详细约定见 [DEVELOPMENT.md](../development/DEVELOPMENT.md) 的「发版」。

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
Windows Server 2008 R2 与 2012 R2 上会立即退出——这是最难排查的一类失败。

```bash
# 让 Go 自动下载并使用 1.20
GOTOOLCHAIN=go1.20.14 go build ./cmd/reqbox

# 或用脚本（它会拒绝非 1.20 的工具链，并在构建后跑 legacyguard）。
# 此脚本只存在于 legacy 分支，必须先切换分支。
git checkout legacy/win2008r2
./scripts/build-legacy.ps1
```

**注意**：只检查 PE 头不够。Go 1.24 的二进制 PE 头仍写着 6.1，`ProcessPrng`
是运行时解析的，导入表里也看不到。所以 legacy 分支用 `scripts/legacyguard`
做三项检查（PE 头、全文禁用符号、导入 DLL 白名单），详见该分支的
`docs/development/LEGACY-WINDOWS.md`。

### 3. 不要在这个分支上跑 `go mod tidy`

用新版 Go 跑 `tidy` 会把 `go 1.20` 抬到 1.21、并写入 `toolchain` 行，
直接破坏该分支存在的意义。需要改依赖时用 `go mod edit`。

改完务必验证：

```bash
# 此脚本只存在于 legacy 分支（在 main 上不存在）
./scripts/check-legacy-pins.ps1     # 5 项 pin 必须全部 OK
```

这条在 legacy 分支的 CI 里已经自动执行，手工只在本地改依赖后需要。

### 4. 凭据以明文存于数据库

`requests` 与 `history.request_snapshot` 里都是明文 Bearer / Basic / API Key。
数据库文件权限已收紧到当前账户，但**能读到文件的人就能读出全部凭据**。

这是有意的范围决定，理由与将来如何加密记录在
[TODO.md](../planning/TODO.md) 的「历史快照中的凭证为明文」一节。

### 5. 前端测试需要 `window.matchMedia` 桩

jsdom 没有实现它。`web/src/test/setup.ts` 里有桩，不要删，否则所有渲染
`CodeEditor` 的组件测试都会失败。

### 6. Tailwind 是手工接入的

shadcn/ui 的 `data-active` / `data-horizontal` 变体**不是内置的**。
`web/src/index.css` 里用 `@custom-variant` 声明了它们；命名 group 变体
（`group-data-horizontal/tabs`）无法声明，改用 `data-slot` 的普通 CSS 规则。
删掉这些会导致标签选中态失效（曾经发生过，表现为下划线宽度 0px）。

### 7. 配置文件只从二进制旁边读

`reqbox.yaml` 的位置由 `os.Executable()` 决定，**不是当前工作目录**。
用 `go run` 时二进制在临时目录里，那边放的配置不会生效。启动日志会打印
实际查找的路径（`using config file` 或 `no config file; looked in=...`），
先看那一行。

另外：**exe 放在系统临时目录时，程序会故意忽略同级目录**，改用
`%APPDATA%\reqbox\reqbox.db`。所以测试「旧库能否被自动找到」时，别把 exe
放在 `%TEMP%` 里，会得到错误结论。

### 8. 发布说明路径改动会静默失效

发版工作流按 `docs/history/release-notes/<tag>.md` 查找手写说明，
**找不到就退回自动生成的提交列表，且不报错**。所以移动或改名发布说明时，
必须同时改 `.github/workflows/release.yml`，否则会以为说明生效了。

v0.2.0 就踩过这个坑：当时自动 changelog 把 `docs:` / `ci:` 前缀过滤掉了，
而本仓库的提交几乎全带这两个前缀，结果发布说明只剩一条「Initial release」。

---

## 七、当前状态

| 项 | 状态 |
| -- | ---- |
| Go 测试 | 96 通过（主分支） |
| 前端测试 | 112 通过 |
| CI | `.github/workflows/ci.yml`：Linux + Windows 的 Go 测试（Linux 带 `-race`）、前端测试、二进制冒烟。最近一次 `main` 全绿 |
| CD | `.github/workflows/release.yml` + `.goreleaser.yaml`：推 `v*` tag 自动发布六平台产物 |
| `tsc` / `lint` / `vet` | 干净 |
| 上线审计 | 已完成，见 [AUDIT.md](../design/AUDIT.md) |
| 已知阻塞项 | 无 |
| 未提交的改动 | 无；两个分支都与远端一致 |

### 版本与代码现在是一致的

`v0.3.0`（2026-10-27）已把 `main` 上全部功能发出：

| 项 | 在 `main` | 在 `v0.3.0` |
| -- | ---------- | ----------- |
| 请求、环境、历史、mock、双语、分阶段耗时 | ✅ | ✅ |
| token 鉴权、数据库权限收紧 | ✅ | ✅ |
| CI / CD | ✅ | ✅ |
| `reqbox.yaml` 配置文件 | ✅ | ✅ |
| 端口冲突自动顺延 | ✅ | ✅ |
| 启动自动打开浏览器 | ✅ | ✅ |

这一节以前是「代码比发布新」的警示。**下一次改动后请重新核对它**：
判断方法是 `git diff --stat v0.3.0..HEAD`——如果改的是功能而不是文档，
这张表就又该更新了，或者该发新版本了。

### 已完成的主要能力

- 请求的保存、执行、历史（SQLite）
- 环境变量与 `{{placeholder}}` 替换
- 把任意请求发布为 mock 端点（实时执行，非缓存回放）
- 响应区：Headers（含请求头、Cookies、耗时明细）/ Request Body / Preview / Response
- 分阶段耗时：DNS / 连接 / TLS / 首字节 / 下载
- 中英双语界面，跟随浏览器语言，可手动切换
- API token 鉴权 + 数据库文件权限收紧
- 配置文件、端口冲突顺延、启动自动开浏览器
- 旧库（apimock 时代）自动迁移，有回归测试
- CI（每次推送）+ CD（推 tag 即发版）

### 下一步可以做的

见 [TODO.md](../planning/TODO.md)。当前**没有阻塞项，也没有进行中的条目**，
剩余全是按需功能。

优先顺序（判断依据是「投入产出比」而非「想不想要」）：

1. **SSE 支持**（1～1.5 天）—— 需要在 `internal/runner` 加不缓冲的读取路径，
   响应区做增量渲染。
2. **Mock 转发请求体**（0.5～1 天）—— 让 mock 可以当透传代理用。

加密落盘（1.5 天）仍暂不实现，触发条件是「数据库文件可能离开这台机器」。

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

或者直接下载兼容包，不必构建：

<https://github.com/iamartinlong/Reqbox/releases/tag/v0.1.2-win2008r2>

---

## 十一、需要联系原维护者确认的事

按「不确认就会做错」排序：

| # | 事项 | 不确认的后果 |
| - | ---- | ------------ |
| 1 | 生产环境（2012 R2 / 2008 R2）的数据库是否需要迁移 | 迁移会自动发生，不需人工，但需要知道有这件事 |
| 2 | 是否需要保留旧 Gitea 仓库作为归档 | 可能删掉唯一的历史副本 |
| 3 | 客户接口的原始来源 | 不在本仓库内，找不到就只能重写 |

原来的第 1 项「2008 R2 三项修复是否复验」已经关闭，见下。

### 2008 R2 的验证已经闭环

`v0.1.1-win2008r2` 确认能在 Windows Server 2008 R2 启动；那次实测暴露的三个
问题在 `v0.1.2-win2008r2` 修复：

| 问题 | 修复 |
| ---- | ---- |
| 控制台复制不出 token 链接 | 启动自动开浏览器；清掉快速编辑模式 |
| 界面能开但保存报「无效 token」 | **移植缺失的解锁页**（前端从未移植过） |
| 端口写死，冲突就起不来 | `reqbox.yaml` + 冲突时顺延 |

**2026-10-27 三项均在真机复验通过。** 构建、启动、功能三层都有实机证据。

这仍不是一次性结论：再改浏览器启动、解锁页或端口/配置，必须重新在 2008 R2
上跑。步骤保留在 legacy 分支 `docs/planning/TODO.md` 的 P1。

---

## 十二、交接清单

移交时已确认：

- [x] 两个分支工作区干净，与远端一致
- [x] `main` @ `a83df90`（`v0.3.0` 的提交），`legacy/win2008r2` @ `62a01db`
- [x] CI 全绿（`main` CI #13、Legacy CI #7）
- [x] 四个 tag 已推送：`v0.3.0`、`v0.2.0`、`v0.1.1-win2008r2`、`v0.1.2-win2008r2`
- [x] `v0.3.0` 产物已下载核验：校验和一致、`-version` 正确、三项新功能实测生效
- [x] `go test ./...` 96 个通过（main）/ 80 个（legacy）
- [x] `pnpm test` 112 个通过，`tsc -b` 与 `oxlint` 干净
- [x] 文档已按读者目的分组，相对链接与锚点全部有效
- [x] 工作日志记录了每一轮的取舍与踩坑
- [x] 无未提交改动、无临时测试文件残留

**本地遗留物**（都不在 git 里，可安全删除）：

| 路径 | 说明 |
| ---- | ---- |
| `data/*.db` | 本地测试数据 |
| `dist/`、`reqbox.exe` | 构建产物，可重新生成 |
| `web/node_modules` | `pnpm install` 可恢复 |
| `C:\Users\EDY\AppData\Local\Temp\reqbox-legacy` | legacy 分支的 git worktree |

那个 worktree 是 `git worktree add` 建的，方便在 legacy 分支上改文档。
不需要时用 `git worktree remove` 移除即可，**不影响任何提交**。
