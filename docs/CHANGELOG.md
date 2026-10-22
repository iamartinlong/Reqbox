# 变更记录

按时间倒序。重点记录**行为变化**与**修复的缺陷**，不逐条罗列代码改动。
格式参考 [Keep a Changelog](https://keepachangelog.com/)，版本号遵循语义化版本。

---

## [0.2.0] — 2026-10-21

### 新增

- **API 鉴权：启动生成 token，浏览器用 cookie 交换。** 此前 `GET /api/requests`
  无需任何凭据就返回明文 Bearer token，默认又监听 `:5178`（所有网卡），
  同网段任何人都能直接读走全部凭据。

  现在的做法：
  - 首次启动生成 32 字节随机 token，在终端打印一次，附一个可直接打开的
    登录链接。
  - 数据库只存 **SHA-256 哈希**，比对用恒定时间比较。
  - 打开 `/?token=…` 后换成 `HttpOnly; SameSite=Strict` cookie，并 302 到 `/`，
    token 不留在地址栏或历史记录里。
  - 同时接受 `Authorization: Bearer …`，curl 与脚本可直接用。
  - 重启后沿用已存的哈希，不会把人踢下线；`-token` 可指定固定值以主动轮换。
  - `-no-auth` 可关闭校验，启动时打印警告。

  **两个路由刻意保持公开**：`/api/health`（顶栏的在线指示，否则未登录时
  会显示「离线」，正好误导用户）和 `/mock/*`（发布出去就是给别人调的）。

  没有 token 时会显示解锁页，而不是弹一个用户无法处理的 toast。

- **数据库文件权限按平台真正收紧。** 原以为设成 `0600` 就够了，实测
  `os.Chmod` 在 Windows 上只切换只读位、不设 ACL（`-rw-rw-rw-`），而实际
  部署目标是 Windows Server 2012 R2。改为用 `SetNamedSecurityInfo` 设置
  DACL：仅当前用户 + SYSTEM + Administrators。目录使用可继承 ACE，
  这样 SQLite 自己创建的 `-wal` / `-shm` 侧文件也会继承受限权限——它们
  含有同样的数据，漏掉就等于没做。

  收紧失败不会阻止启动，但会打印警告：读不到数据库比权限宽松更糟。

- **中英双语界面。** 右上角语言菜单可在中文与 English 之间切换，即时生效
  且刷新后保持。默认跟随浏览器语言（`zh*` → 中文，其余 → 英文）。

  字典在 `web/src/lib/i18n/{en,zh}.ts`，扁平 key（`surface.thing`），共 268 条。
  组件内用 `useT()`；`toast`、确认框这类异步回调用 `tNow()`，因为那里不能调
  hook，而把 `t` 放进 `useEffect` 依赖会因切换语言而重跑请求。

  覆盖范围包括正文、`placeholder`、`aria-label`、tooltip、toast、对话框标题
  与按钮。`aria-label` 是最容易漏的一类——它不显示在屏幕上，但屏幕阅读器会
  读，是「切了语言仍有英文」的典型残留点。

  语言状态独立于工作区 store：语言是设备偏好，混进工作区会让切语言把请求
  标记为「有未保存的修改」。

- **多语言正确性测试（19 个用例）。** 这项本来打算写成 grep 脚本，改为测试
  后可进 CI：

  | 测试 | 防的问题 |
  | ---- | ---- |
  | 键对齐 | 英文有、中文缺 → 中文界面冒出英文句子 |
  | 空值检查 | 翻译成空串 |
  | 插值一致 | 译文漏掉 `{count}` → 句子里缺数字 |
  | 组件用键存在 | 改了 key 名，界面显示裸 `sidebar.foo` |
  | 硬编码英文 | 新组件又用回英文字面量 |

  两处做过变异验证：把 `t('resp.sentAfterSubstitution')` 改回英文、写一个
  不存在的 key，测试都如实报错，还原后恢复全绿。

- **分阶段耗时（DNS / 连接 / TLS / 首字节 / 下载）。** 响应区 `Headers` 下新增
  `Timing` 分组，用横向条展示各阶段占时，一眼能看出时间花在哪。

  实现基于 `net/http/httptrace`。几个设计点：
  - `TTFB` 从请求发起算到收到首字节，因此**包含** DNS/连接/TLS，界面上标注了
    这一点，避免被误读成「等待服务器」的净耗时。
  - `Download` 由总耗时减去 TTFB 得出，与状态栏显示的总耗时自洽。
  - 连接复用时 connect/tls 为 0，界面额外标注 `connection reused`。
  - 未发生的阶段显示 `—` 而非 `0 ms`。
  - 失败请求同样记录，因为超时本身就是耗时问题。

  历史表新增 `timing` 列（追加式迁移）。旧记录该字段为空，界面按「无数据」
  处理，不报错。

- **前端自动化测试（101 个用例）。** 引入 Vitest + Testing Library，覆盖：

  | 范围 | 用例 |
  | ---- | ---- |
  | `lib/format` 格式化与配色 | 20 |
  | `lib/api` 请求封装与错误映射 | 8 |
  | `lib/store` 状态转换 | 12 |
  | `response-panel` 四种渲染分支、标签与耗时展示 | 17 |
  | `kv-editor` 增删改与禁用 | 6 |
  | `confirm-dialog` 确认/取消与默认焦点 | 6 |
| `unlock-screen` 解锁页与 401 处理 | 13 |
  | `lib/i18n` 字典对齐、切换与持久化 | 19 |

  `pnpm test` 已纳入提交前检查。做过破坏性验证：改掉 `Cookies Set` 标题、
  把 `Preview` 标签改回 `Body`，对应用例均立即失败，还原后恢复全绿。

- **长 URL 可完整查看。** URL 输入框是单行 `<input>` 且 `overflow-x: clip`，
  实测 556px 宽时内容需 1089px，**533px 被裁掉且无法读取**。现在悬停有
  `title` 提示，下方另有可展开的 `full URL` 行，完整地址换行显示。

- **请求/响应详情按浏览器开发工具的结构展示。** 响应区标签重组为四个：

  | 标签 | 内容 |
  | ---- | ---- |
  | `Headers` | General（URL、远程地址、HTTP 版本、状态、耗时、体积）+ 响应头 + 请求头 |
  | `Request Body` | **实际发出的**请求体（变量替换后） |
  | `Preview` | 格式化后的响应体（JSON 缩进、图片预览、二进制提示） |
  | `Response` | 原始响应内容，不做任何格式化 |

  此前 `Body` 只显示格式化后的响应体，`Raw` 里放的却是 URL/Cookie 等元信息，
  名不副实；请求体则完全没有展示。现在四个标签各自只做一件事。

- 请求头捕获：`Content-Type`、`Accept`、`User-Agent`、`Authorization` 这些
  由程序自动添加的头，此前在界面上完全看不到。

- **GitHub Actions 持续集成（`.github/workflows/ci.yml`）。** 之前所有验证都
  只在本机跑，没有 CI；而「本机没有 gcc」正是 `-race` 一直没做的原因。
  三个并行任务：

  | 任务 | 内容 |
  | ---- | ---- |
  | `Go` | Linux + Windows 矩阵：`gofmt` 检查、`go vet`、`go test`；Linux 另跑 `-race` |
  | `Web` | `pnpm install --frozen-lockfile`、`tsc -b`、`oxlint`、`vitest` |
  | `Build` | 先 `pnpm build` 再 `go build`（验证 `go:embed` 真有内容），启动二进制打 `/api/health` 与 `/`，上传产物 |

  几处刻意的选择：

  - **Windows 也在矩阵里。** `internal/store/acl_windows_test.go` 带
    `//go:build windows`，DACL 收紧的代码在 Linux 上**一条测试都不会跑**，
    而那正是保护明文凭据的部分。
  - **`-race` 放在 Linux。** 它需要 cgo 与 C 编译器，本机两者都没有。
    Windows 上跑需要 mingw 且不稳定，故只在 Linux 跑。
  - **`gofmt` 只在 Linux 检查。** 提交内容（blob）是 LF，但 Windows 检出后
    可能是 CRLF，`gofmt` 会误报整棵树。
  - **`Build` 里的冒烟测试**会真正启动二进制并请求 `/api/health` 与 `/`。
    只编译不启动，抓不到 `go:embed` 落到占位页、端口默认值坏了、数据库建不
    出来这类问题。
  - 前端流水线里没有 `pnpm build`：`tsc -b` 已能暴露类型问题，真正的构建
    放在 `Build` 任务里做一次，避免重复。
  - `Build` 任务还会跑 `goreleaser check`。发版配置写坏时，应该在改它的
    那个 PR 上失败，而不是等到打 tag 才发现。

- **发版自动化（`.github/workflows/release.yml` + `.goreleaser.yaml`）。**
  推送 `v*` tag 即发布：构建前端 → 重跑测试 → GoReleaser 生成六平台
  压缩包、`checksums.txt` 与 GitHub Release。

  | 项 | 取值 |
  | ---- | ---- |
  | 平台 | linux / darwin / windows × amd64 / arm64 |
  | 产物 | `.tar.gz`（Windows 为 `.zip`），内含二进制、`LICENSE`、`README.md` |
  | 版本号 | tag 去掉 `v`，注入 `server.Version` |
  | 预发布 | tag 形如 `v0.2.0-rc1` 时自动标 prerelease |

  几个要点：

  - **必须先构建前端**。`go:embed` 读的是 `web/dist` 的现状，不构建就会
    把「前端未内嵌」占位页发出去。工作流把这一步放在最前面。
  - **发版前重跑测试**。tag 可以打在 CI 没跑过的提交上，所以发布流程自己
    再跑一遍 `go test`，失败则不出包。
  - **`workflow_dispatch` 手动触发只做快照**（`--snapshot`）：构建但不发布，
    用来验证配置；产物仍作为 workflow artifact 上传。
  - **`fetch-depth: 0`**：GoReleaser 用「上一个 tag 之后的提交」生成变更说明，
    浅克隆会导致 changelog 为空。
  - `mod_timestamp` 设为提交时间，同一 tag 重复构建得到相同字节，
    校验和不会无故变化。

  实测（本地 `--snapshot`）：6 个包全部生成，Windows 包解开后
  `reqbox.exe -version` 输出快照版本，启动后 `/api/health` 与内嵌前端均正常。

  首次真实发版为 `v0.2.0`（2026-10-21）：7 个资产（6 个包 + `checksums.txt`），
  下载后 SHA-256 与 `checksums.txt` 一致，Linux 包内二进制的 `-version`
  输出 `0.2.0`。

  **自动 changelog 的教训**：最初把 `docs:` 与 `ci:` 前缀从 changelog 里
  过滤掉了，而本仓库从首个提交以来的提交几乎全带这两个前缀，于是 v0.2.0
  的自动说明只剩一条。现在只过滤 `chore:` / `test:`，并且优先使用手写
  说明文件（`docs/release-notes/<tag>.md`）。

- **兼容分支也有 CI 与发版（2026-10-22，`v0.1.1-win2008r2`）。**
  `legacy/win2008r2` 分支此前只有构建脚本，靠人工验证，也没有发布产物。
  现在它有自己的 CI（Go 1.20 工具链校验、依赖 pin、测试、构建、冒烟）与
  发版流程（`v*-win2008r2` tag → `windows/amd64` 兼容包）。

  同一次改动修掉了一个**实际失效的校验**：构建脚本只比较 PE 头的最低系统
  版本，并断言 Go 1.21+ 会写成 10.0。实测 Go 1.24 的产物 PE 头**仍是 6.1**，
  而 `ProcessPrng` 是运行时解析的、导入表里看不到——两项检查都通过，程序在
  2008 R2 上照样秒退。新增的 `scripts/legacyguard` 改为三项检查（PE 头、
  全文禁用符号、导入 DLL 白名单）。

  细节见 `docs/LEGACY-WINDOWS.md` 与 [WORKLOG](./WORKLOG.md)。兼容分支的
  完整文档在那条分支上，主分支只保留一份简版说明。

### 决定

- **加密落盘（方案 B）暂不实现。** 凭据仍以明文存于数据库，靠文件权限与
  API token 保护。理由：当前定位是本机调度工具，数据库文件不离开这台机器，
  加密的主要收益（防「文件被拷走」）不成立；而密钥管理会引入新故障模式
  （密钥丢失、重装后无法解密、启动索要口令）。

  触发重新考虑的条件：数据库放入同步目录或会被外发、多人共用、部署到
  不受信网络。届时首选 DPAPI（`golang.org/x/sys/windows` 已提供
  `CryptProtectData`，无需新依赖），但需注意它挡不住「以该账户身份运行的
  程序」。完整方案、信任边界与实现要点见 `docs/TODO.md`。

### 修复

- **查询参数冲突改为 Params 覆盖 URL。** 此前 URL 中手写 `?q=1`、Params 面板
  又配 `q=2` 时，结果变成 `?q=1&q=2`。现在 Params 视为权威配置，URL 字段是
  草稿，同名参数以 Params 为准；但 Params 内部重复的 key 仍会保留为多个值，
  因为那是在面板里的明确选择。

- **标签选中态没有视觉标识。** 实测选中项的 `::after` 下划线高度 0px、
  宽度 0px、透明度 0，选中与未选中只差文字深浅。根因是组件库把
  `data-active` / `data-horizontal` 当作 Tailwind 内置变体使用，而本项目
  手工接入 Tailwind，这些变体从未定义，导致整条 `variant="line"` 样式链失效。
  已在 `index.css` 中补齐变体映射，并用显式 CSS 规则处理无法用变体表达的
  命名 group 场景。
- 请求失败时显示真实 URL，此前回填的是未解析的模板（如 `{{baseUrl}}/nope`）。
  历史记录也改为存解析后的地址。
- 失败时同样返回请求头与发起地址。

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

下一版本的内容见 [TODO.md](./TODO.md)。0.2.0 已发布；SSE、Mock 转发请求体、
链式调用与 WebSocket 仍在待办中。
