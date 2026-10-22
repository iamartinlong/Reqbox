# 工作日志

按时间倒序，记录**这一轮做了什么、为什么、留下什么**。与 [CHANGELOG](./CHANGELOG.md)
的分工：CHANGELOG 面向使用者，写「行为变了什么」；这里面向维护者，写
「当时怎么想的、踩了什么、下次别重复什么」。

一条记录可以只有几行，但必须能回答：改了什么、验证方式、有没有遗留。

---

## 2026-10-21 ～ 10-22 · CI、CD 与首个正式发布

**起点**：`main` @ `ead2f5f`，只有本机验证；没有 CI，没有 `.github/`，
没有正式 tag 或 Release。

**终点**：`main` @ `a2c52fb`，`v0.2.0` 已发布，CI 与发版全自动。

### 1. 接入 CI（`8d79a84`）

之前所有验证都在开发机跑，而「本机没有 gcc」正是 `-race` 一直没做的原因；
同时 `internal/store/acl_windows_test.go` 带 `//go:build windows`，在
Windows 之外的机器上一条都不会执行——而那正是保护明文凭据的代码。

新建 `.github/workflows/ci.yml`，三个并行任务：

| 任务 | 内容 |
| ---- | ---- |
| `Go` | Linux + Windows：`gofmt`、`vet`、`test`；Linux 另跑 `-race` |
| `Web` | `pnpm install --frozen-lockfile`、`tsc -b`、`oxlint`、`vitest` |
| `Build` | 构建前端 → `go build` → 启动二进制打 `/api/health` 与 `/` → 上传产物 |

三个刻意的取舍：

1. **`gofmt` 只在 Linux 检查。** 提交的 blob 是 LF，但 Windows 检出可能是
   CRLF；在开发机上 `gofmt -l` 会列出整棵树，纯属误报。实测把仓库按 LF
   检出后 `gofmt` 干净，所以问题在检出方式，不在代码。
2. **`-race` 只在 Linux。** 需要 cgo + C 编译器。Windows 上要 mingw，
   不稳定且 CI 里价值不大。
3. **冒烟测试下载到文件再 `grep`。** `curl … | grep -q` 会让 curl 在
   grep 提前退出时死于 SIGPIPE，而 runner 默认 `pipefail`，步骤会莫名变红。

`ci.yml` 的首次运行（run #39611849740）全绿，`-race` 未报竞态。

### 2. 真正的 CD（`24c4b18`、`4f5718e`）

新增 `.goreleaser.yaml` 与 `.github/workflows/release.yml`：推 `v*` tag
即构建六平台产物并创建 Release。

要点：

- **必须先 `pnpm build`。** `go:embed` 读的是 `web/dist` 的现状，跳过这步
  会把「前端未内嵌」占位页发出去。工作流把它放在最前面，`Build` 任务的
  冒烟测试也会验证 `/` 里真有 `<div id="root">`。
- **发版流程自己再跑一次 `go test`。** tag 可以打在 CI 没跑过的提交上。
- **`fetch-depth: 0`。** GoReleaser 用「上一个 tag 之后的提交」生成
  changelog，浅克隆会得到空列表。
- **`mod_timestamp` 用提交时间。** 同一 tag 重复构建字节一致，校验和稳定。
- **`workflow_dispatch` 只做 `--snapshot`。** 手动触发用于验证配置，绝不
  意外发布。
- `ci.yml` 的 `Build` 任务加了 `goreleaser check`：配置写坏应在改它的 PR
  上失败，而不是等到打 tag。

### 3. 发布 v0.2.0

发布前先在本地和 CI 双重验证，再打 tag：

| 步骤 | 结果 |
| ---- | ---- |
| CI #3（tag 所在提交） | 4 任务全绿 |
| 本地 `goreleaser release --snapshot --clean` | 6 包 + `checksums.txt` |
| 解开 Windows 包启动 | `/api/health` 正常、内嵌前端存在 |
| 打 tag 后 Release #1 | 成功，2 分 33 秒，7 个资产 |
| 下载 `linux_amd64` 重算 SHA-256 | 与 `checksums.txt` 一致 |
| 该包内 `reqbox -version` | `reqbox 0.2.0` |

**踩到的问题：自动 changelog 几乎为空。**

`.goreleaser.yaml` 最初把 `docs:` 与 `ci:` 前缀从 changelog 里过滤掉了。
本仓库自首个提交以来，除两条以外全部带这两个前缀，于是 v0.2.0 的自动
说明只剩一条 “Initial public release”。已发布页面上线后才发现。

处理：

1. 只过滤 `chore:` / `test:`，保留 `docs:` / `ci:`。
2. 增加**手写发布说明**机制：存在 `docs/release-notes/<tag>.md` 时用它，
   否则退回自动 changelog。工作流里用一步 shell 判断文件是否存在，再把
   `--release-notes` 传给 GoReleaser。

**别踩的坑**：GoReleaser 的 `release` 段**没有** `notes` 字段。先按直觉写了
`release.notes:`，`goreleaser check` 直接报 `field notes not found`——这
说明在 CI 里跑 `check` 是对的，否则这个错会在打 tag 时才暴露。

已手动更新线上 v0.2.0 的说明（2457 字符）。下次发版前先写好
`docs/release-notes/v0.3.0.md` 即可，不再依赖自动生成。

### 4. 收尾

- `docs/TODO.md`：`-race` 标为完成（由 CI 覆盖）；新增「CI 与发版自动化」
  条目；总览改为「无阻塞项」；P3 补上触发条件与验收方式。
- `docs/HANDOFF.md`：状态表、仓库快照、发版命令、下载即用的入口。
- `docs/PLAN.md`：1.1 仓库托管完成；新增 1.4 CI/CD。
- `docs/CHANGELOG.md`：0.2.0 定版，记录 CI、CD 与 changelog 的教训。
- `README.md`：下载与发版说明。
- 清理开发机上遗留的 `dist/`、临时校验目录，以及一个由工具调用把路径分隔符
  写坏而产生的异常目录（名字形如 `C` + 一个私有区字符 + `UsersEDY`，内容只有
  一个空的 `.local`，已删除；不是项目文件，也不在 git 里）。

### 遗留

| 项 | 说明 |
| ---- | ---- |
| legacy 分支没有 CI | `.github/` 只存在于 `main`。改动同时影响两个分支时，legacy 侧仍需手工用 Go 1.20.14 验证 |
| 发布说明存在性无提示 | 缺 `docs/release-notes/<tag>.md` 时静默退回自动 changelog，日志里没有提示 |
| 发版前的 CI 是人工确认的 | 打 tag 不必等 CI 绿；发版流程自己重跑 `go test`，但不跑前端测试与 `-race` |
| SSE / WebSocket / 链式调用 | 见 TODO，均未开始 |

---

## 2026-10-22 · 兼容分支的 CI/CD 与 2008 R2 发布

**起点**：`legacy/win2008r2` @ `20118de`，只有构建脚本，没有 CI、没有发布
产物，兼容性校验只看 PE 头。
**终点**：该分支 `8c21014`，`v0.1.1-win2008r2` 已发布；`main` @ `d51ba89`
之后又更新了指向 legacy 的文档。

### 1. 发现校验是空的

原 `build-legacy.ps1` 检查 PE 头最低系统版本是不是 6.x，注释断言
「Go 1.27+ 会写成 10.0」。**实测：Go 1.24 构建的二进制 PE 头仍然是 6.1。**

原因是 `ProcessPrng` 通过 `LoadLibrary` + `GetProcAddress` 在运行时解析，
导入表里没有这个函数名。于是 PE 头与导入表都「看起来合法」，进程在
2008 R2 上照样秒退且没有任何输出——正是这个脚本声称要拦的情况。

实测对比（`debug/pe` 读取 + 全文搜索）：

| 二进制 | PE minOS | 含 `ProcessPrng` |
| ------ | -------- | ---------------- |
| main（Go 1.24） | 6.1 | 是 |
| legacy（Go 1.20） | 6.1 | 否 |

两者 PE 头相同，区别只在字符串与导入表。

### 2. `scripts/legacyguard`

三项独立检查：PE 头为 6.x；全文不含 `ProcessPrng` / `bcryptprimitives`；
静态导入的 DLL 在 NT 6.1 白名单内。第三条用白名单而非黑名单——出现新导入
时必须先确认目标系统有该 DLL。

3 个单元测试覆盖两个方向：拒绝含 `ProcessPrng` 的输入、放行干净的导入表。
只测放行等于没测。

两个构建脚本改为调用它，CI 与发版流程各跑一次；发版流程还会**先解包**
再对解出的 exe 跑一次，确认打包没错。

### 3. legacy 分支的 CI 与发版

| 文件 | 触发 | 结果 |
| ---- | ---- | ---- |
| `legacy-ci.yml` | 推送到该分支、PR | [#1](https://github.com/iamartinlong/Reqbox/actions/runs/39713506157)、[#2](https://github.com/iamartinlong/Reqbox/actions/runs/39714627207) 全绿 |
| `legacy-release.yml` | `v*-win2008r2` | [#1](https://github.com/iamartinlong/Reqbox/actions/runs/39714240061) 全绿，2 分 40 秒 |

几处选择：

- **工具链写死 `1.20.14`**，不读 `go.mod`。这个分支的兼容性是工具链的
  属性，不是偏好；顺手加一步 `go version` 断言，防止 runner 镜像变化。
- **跑在 `windows-latest`**。目标平台就是 Windows，在 Windows 上冒烟比在
  Linux 上交叉编译更能说明问题。
- **`-race` 不在此分支**：需要 cgo，且与 6.1 兼容性无关；主分支已覆盖。
- **tag 带 `-win2008r2` 后缀**：与主分支 `v0.2.0` 区分，也把约束写进版本号。
  发版工作流只认这个后缀，不会误触发主分支发布。

### 4. 发布 v0.1.1-win2008r2

| 检查 | 结果 |
| ---- | ---- |
| 发布前 Legacy CI #1 | 全绿 |
| Release #1 | 全绿 |
| 下载 zip 重算 SHA-256 | `38aa1968…f09cdf`，与 `checksums.txt` 一致 |
| `legacyguard` 查解出的 exe | OK（PE 6.1、无 `ProcessPrng`、仅 `kernel32.dll`） |
| `-version` | `reqbox 0.1.1` |
| 带 token 启动 | 打印登录链接；`/api/health` 返回 `0.1.1`；`/` 含 `<div id="root">` |
| 数据库 | `reqbox.db` / `-wal` / `-shm` 正常创建 |

### 5. 踩到的坑

- **legacy 分支缺 `LICENSE`**：`Copy-Item LICENSE` 直接失败，打包中断。
  补上了文件，README 也加了 License 小节。
- **`$args` 是 PowerShell 自动变量**：发版工作流里 `$args = @(...)` 再
  `gh @args` 会静默出错。改名为 `$ghArgs`。
- **`workflow_dispatch` 时 `ref_name` 是分支名**：`legacy/win2008r2` 带斜杠，
  塞进 zip 文件名会出错。手动触发时改用 `0.0.0-snapshot-<run_number>`。
- **主分支文档一度落后**：上一轮只更新了 legacy 分支的文档，`main` 的
  `HANDOFF.md` / `TODO.md` / `LEGACY-WINDOWS.md` 里还写着「legacy 没有 CI」
  「最新版本只有 v0.2.0」。本轮补齐。

### 遗留

| 项 | 说明 |
| ---- | ---- |
| **2008 R2 SP1 实机验证** | 唯一没打勾的关键项。步骤在 legacy 分支 `docs/TODO.md` 的 P1 |
| 两个分支各改一次 | 工作流与 `legacyguard` 只存在于 legacy，主分支没有对应物；这是刻意的（主分支不需要） |
| legacy 分支无前端测试 | 只构建不测试；主分支的 Vitest 不会带过去（会把新依赖带进 Go 1.20 约束） |

## 2026-10-22 · 首次在 2008 R2 上运行后的三个反馈

真机验证结果：`v0.1.1-win2008r2` 在 Windows Server 2008 R2 上**能正常打开**
（本分支的一号目标达成）。同时收到三个问题，其中第二个是**实现缺陷**。

### 1. 命令行窗口无法复制 token

表现：双击 exe，窗口里滚过登录链接，但复制不了。

根因不在控制台本身，而在于流程设计：token 明文**只打印一次**，第二次启动
就只剩 `using the previously generated API token`。一旦错过那一屏，配合下面
的第 2 个缺陷，用户就**永久无法登录**，只能删库重来。

因此修法不是「让控制台更好复制」，而是**根本不需要复制**：启动后自动打开
浏览器，直接落到登录链接。`-no-browser` 可关。

顺带清掉 Windows 控制台的 `ENABLE_QUICK_EDIT_MODE`：默认开启时，一次拖选会
暂停程序输出，而 Ctrl+C 从「中断」变成「复制」，两个手势互相打架。

### 2. 能打开界面，但保存时提示无效 token（**缺陷**）

表现：访问 `127.0.0.1:5178` 界面能出来，没有要求输入 token；保存请求时
报 `missing or invalid API token`。

根因：**legacy 分支的前端从未移植解锁页。**

后端鉴权在 `51c7ac0` 完整移植了（`auth.go`、`handlers_auth.go`），但那次提交
只改了 Go，前端 `web/src/` 下没有 `unlock-screen.tsx`，也没有 `session` API
封装。直接查构建产物可以确认：

| 检查 | legacy（修复前） | main |
| ---- | ---------------- | ---- |
| 产物里有 `unlock.title` | ❌ | ✅ |
| 产物里有 `session/status` | ❌ | ✅ |

于是 `/` 能打开（它本来就不需要鉴权，否则解锁页加载不出来），但所有
`/api/*` 都 401；`workspace.tsx` 只在加载失败时弹一个 toast，然后渲染空界面。

修法：把 `unlock-screen`、`session` API、401 广播事件移植到 legacy 分支。
**这打破了「本分支只接受兼容性改动」的约定**，但缺一个登录入口等于功能不可
用，属于该约定的例外。分支内没有 i18n，所以字符串是内联英文——移植整套
翻译系统才是真正越界。

验证：用浏览器实测四步——解锁页出现、错误 token 显示提示、正确 token 进入
工作区、刷新后会话保持。

### 3. 端口写死，冲突就无法启动

之前只能 `-addr` / `REQBOX_ADDR`，对双击启动的程序没有可用途径。

新增 `reqbox.yaml`（示例 `reqbox.example.yaml`），并让端口冲突时顺延到下一个
空闲端口。两条都刻意**写进日志**：配置文件路径、以及
`requested=... using=...`。这类功能的失败模式几乎都是「悄悄用了别的东西」，
日志是唯一的外部依据。

顺带发现并修掉一个相关 bug：`-addr 127.0.0.1:0` 时登录链接指向**端口 0**，
因为 `LogToken` 在 `net.Listen` 之前执行。改为先绑定、再用
`listener.Addr()` 拼 URL——自动顺延也依赖这个顺序。

### 踩到的坑

- **`Default()` 预填 `server.addr` 会导致只写 `port` 的配置文件报错**，因为
  校验看到「两者都设置」。写 `TestParseFullFile` 时暴露，改成默认留空、由
  调用方传入 fallback。
- **测试里改 `gofmt` 会因 CRLF 误报整棵树**。与主分支同样的老问题：提交的
  blob 是 LF，工作区是 CRLF。校验要在 LF 副本上做。
- **`$args` 是 PowerShell 自动变量**（上一轮已记录，这次在探测脚本里又踩到）。

### 遗留

| 项 | 说明 |
| ---- | ---- |
| 2008 R2 实机复验 | 三项改动需要重新打包后在该机器上再跑一次 |
| legacy 无 i18n | 解锁页只有英文；移植整套翻译越界，见上文 |
| `-race` | 本分支 CI 仍不跑，与 2008 R2 兼容性无关 |

---

## 模板

新增记录时复制这段：

```markdown
## YYYY-MM-DD · 一句话主题

**起点**：分支与提交、当时的已知状态。
**终点**：分支与提交、达到的状态。

### 做了什么

- 改了什么，为什么这样改

### 验证

| 检查 | 结果 |

### 踩到的坑

现象 → 根因 → 如何避免

### 遗留

| 项 | 说明 |
```

写作原则（与 [docs/README](./README.md) 的三条约定一致）：

- 写**表现和根因**，不写「调了很久」。
- 每条结论都要有验证方式，否则读者无法判断它现在是否还成立。
- 遗留要写清触发条件，避免下一个人重新调研。
