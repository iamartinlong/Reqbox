# 实施计划

本文件是 [TODO.md](./TODO.md) 的执行版：TODO 说明**要做什么**，这里说明
**按什么顺序做、每一步怎么落地、如何验收**。

工时单位为人日（8 小时），按「已熟悉本代码库」估算。
单项的动机与取舍见 [TODO.md](./TODO.md) 与 [DECISIONS.md](../design/DECISIONS.md)。

---

## 起点快照

计划基于以下状态（2026-10-21，2026-10-22 更新）：

| 项 | 值 |
| -- | -- |
| 主分支 | `main`，工作区干净 |
| 兼容分支 | `legacy/win2008r2`（Go 1.20.14，用于 2012 R2 及更早） |
| Go 代码 | 20 文件 / 约 4050 行 |
| 前端代码 | 35 文件 / 约 4547 行 |
| Go 测试 | 78 个（主分支）+ 61 个（兼容分支），全绿 |
| 前端测试 | 112 个，全绿 |
| CI / CD | 两个分支各自一套；`v0.3.0` 与 `v0.1.2-win2008r2` 均已发布 |
| 已知未决 | 2008 R2 SP1 实机验证（兼容分支） |

不想构建的话，直接从 [Releases](https://github.com/iamartinlong/Reqbox/releases)
下载对应平台的包，解开即用（前端已内嵌）。

---

## 阶段一 · 决策与收尾　约 0.5 天

目标：清掉阻塞项，让后续工作没有「名字还没定」这种悬而未决的前提。

### 1.1 改名　✅ 已完成（代码 + 仓库托管）

**代码改名已全部执行**：`apimock` → `reqbox`，44 个文件、142 处引用，
含 module 路径、二进制、环境变量、数据路径、UI 与前端存储键。老数据通过
回退逻辑自动沿用。详见 [TODO 的改名记录](./TODO.md#p1)。

**仓库托管已完成**：已迁移到 <https://github.com/iamartinlong/Reqbox>，
以全新状态发布（不带旧历史），`main` 与 `legacy/win2008r2` 均已推送。

**验收**：

- [x] `git remote -v` 指向新地址
- [x] `git push` 在 `main` 与 `legacy/win2008r2` 均成功
- [x] 克隆新地址能 `go build`

### 1.2 长 URL 可见性　0.3 天　✅ 已完成

**做什么**：实测当前 URL 输入框在 638px 宽时需 1082px，444px 内容被裁且无法查看。

**实际实现**（两项都做了，不只择一）：

- `<input>` 加 `title={draft.url}`，悬停即看全文（一行改动，零风险）
- 下方新增可展开的 `full URL` 行，`break-all` 换行显示完整地址，默认收起

同时移除了原先那行只显示 `pathOf` 的小字——它被 `truncate` 截断，对长 URL
没有帮助，且会与新的完整 URL 行重复。

**验收结果**：142 字符的 URL 完整可见；输入框仍裁 533px，但 `title` 与
展开行补齐。

### 1.3 查询参数去重　0.1 天　✅ 已完成

**做什么**：决定 `?q=1&q=1` 的处理方式。

**已决定的语义**：Params 覆盖 URL 中的同名参数。URL 字段视为草稿，Params
面板是权威配置（与 Postman 一致）。

**实现要点**：先按 key 聚合 params 的值，再对每个 key 执行一次 `Del` 后
重新 `Add`。这样 params 内部同名的多个值（用户刻意配置的）仍会保留，
而 URL 里手写的同名值被整体替换。

**验收结果**：`?q=1` + params `q=2` → `?q=2`；params 内两个 `tag` →
`?tag=a&tag=b`；禁用的 param 不影响 URL 中的同名值。三个用例均已加入
`runner_test.go`。

### 1.4 CI 与发版自动化　0.5 天　✅ 已完成

代码不再只在本机验证：`.github/workflows/ci.yml` 覆盖 Linux + Windows 的
Go 测试（Linux 带 `-race`）、前端测试与二进制冒烟；`.github/workflows/release.yml`
配合 `.goreleaser.yaml`，推送 `v*` tag 即产出六平台压缩包并创建 Release。
`v0.3.0` 已按此流程发布。细节见 [DEVELOPMENT 的「发版」](../development/DEVELOPMENT.md)、
CHANGELOG 与 [WORKLOG](../history/WORKLOG.md)。

---

## 阶段二 · 前端测试　1.5 ～ 2 天

**为什么排在功能之前**：这个项目踩过的坑几乎都是「旧功能被改坏」，而不是
「功能没做」。四次回退中 `tsc` 与 lint 全部通过。详见 DR-005。

### 2.1 搭骨架　0.5 天　✅ 已完成

```bash
pnpm add -D vitest @vitest/ui jsdom \
  @testing-library/react @testing-library/user-event @testing-library/jest-dom
```

- `vitest.config.ts`：独立于 `vite.config.ts`，避免测试配置影响构建
- `src/test/setup.ts`：引入 jest-dom、清理 localStorage、**补 `window.matchMedia`**

  `matchMedia` 是踩到的第一个坑：jsdom 不提供它，而 `useTheme` 会读取，
  于是每个渲染 `CodeEditor` 的组件都会在挂载时抛错。补齐后 11 个失败用例
  一次通过。

`package.json` 新增 `test` 与 `test:watch`。

**验收结果**：`pnpm test` 可运行。

### 2.2 API 层与纯函数　0.5 天　✅ 已完成

| 目标 | 用例数 |
| ---- | ------ |
| `lib/format.ts` | 20 |
| `lib/api.ts` | 8 |
| `lib/store.ts` | 12 |

共 40 个。覆盖点包括：

- `formatBytes` / `formatDuration` 的进位边界（1023 → 1024）
- `statusTint` 对 0 / 2xx / 3xx / 4xx 的分档
- `pathOf` 对无 scheme、根路径、未解析变量的降级行为
- `slugify` 丢弃路径不安全字符、截断 80 字符
- `api` 的 204 返回 undefined、错误体非 JSON 时回退 `statusText`、query 编码
- `store` 的核心约束：**草稿是副本**（编辑不影响已保存列表）、
  切换请求清空 dirty、`setDraft` 不置脏

### 2.3 关键组件　0.5 ～ 1 天　✅ 已完成

| 组件 | 用例数 |
| ---- | ------ |
| `response-panel` | 17 |
| `kv-editor` | 6 |
| `confirm-dialog` | 6 |

共 29 个。

`response-panel` 覆盖的是已经出过问题的分支：
空响应 / 等待中 / 失败 / 成功、四个标签的顺序与切换、
请求头可见（含 Authorization）、Cookies 分组、Received 时间、
重定向后的 Final URL、空请求体的提示。

### 2.4 多语言与鉴权的界面测试　0.3 天　✅ 已完成

| 目标 | 用例数 |
| ---- | ------ |
| `lib/i18n/i18n.test.ts`（字典与用法） | 12 |
| `lib/i18n/runtime.test.ts`（切换与持久化） | 7 |
| `components/unlock-screen`（解锁页） | 7 |
| `lib/api.auth`（401 事件与状态） | 6 |

共 32 个。合计 **101 个测试**。

**验收结果**：

- `pnpm test` 101 个全绿，`tsc`、`lint`、`build` 均通过
- **破坏性验证**：把 `Cookies Set` 标题改掉，对应用例立即失败；
  把 `Preview` 标签改回 `Body`，「stable order」用例失败。
  两次均还原后恢复全绿 —— 证明测试确实在保护这些行为。

---

## 阶段三 · 多语言　1 ～ 1.5 天　✅ 已完成

### 3.1 抽取文案　0.5 天　✅

实际处理 **13 个文件**，字典共 **268 个键**：

| 文件 | 说明 |
| ---- | ---- |
| `sidebar.tsx` | 请求列表、集合菜单、历史、底部工具栏 |
| `response-panel.tsx` | 四个标签、Header 分组、耗时明细、复制菜单 |
| `request-panel.tsx` | 五个标签、Body 类型、Auth 表单、发布面板 |
| `workspace.tsx` / `request-bar.tsx` | 顶栏、空状态、toast、删除确认 |
| `environment-dialog.tsx` | 环境管理与变量编辑 |
| `kv-editor.tsx` / `prompt-dialog.tsx` / `confirm-dialog.tsx` | 通用组件的默认文案 |
| `curl-dialog.tsx` / `run-settings-dialog.tsx` / `App.tsx` | 对话框与外壳 |

**结构**：`src/lib/i18n/{en,zh}.ts` 扁平 key（`surface.thing`），
`index.ts` 提供 `useT()`（组件内）与 `tNow()`（异步回调、toast 内）。

分开两个入口的原因：`toast` 与 `confirm` 在事件回调里触发，那里不能调
hook；而 `tNow()` 直接读 store 当前值，避免把 `t` 塞进每个 `useEffect`
的依赖数组（否则切语言会重跑请求或重复加载）。

**已覆盖**：正文、placeholder、`aria-label`、tooltip、toast、确认框
标题与按钮。`aria-label` 容易漏——它不渲染成文字，但屏幕阅读器会读，
是「切换后仍有英文」的常见残留点。

### 3.2 语言切换与持久化　0.3 天　✅

- 右上角语言菜单（`LanguagesIcon` + 单选列表），与主题切换并列
- `localStorage` 键 `reqbox-language`，由 zustand `persist` 写入
- 默认取 `navigator.language` 前缀：`zh` → 中文，其余 → 英文
- 语言状态**独立于**工作区 store：语言是设备偏好，放进工作区会让切
  语言把请求标记为「有未保存的修改」

### 3.3 校验　0.3 天　✅

脚本扫描的替代方案——写成测试，因为它能进 CI：

| 测试 | 防的问题 |
| ---- | ---- |
| 键对齐 | 英文有、中文缺 → 中文界面出现英文句子 |
| 空值检查 | 占位符翻译成空串 |
| 插值一致 | 翻译漏掉 `{count}` → 句子里缺数字 |
| 组件用键存在 | 改了 key 名，界面显示裸 `sidebar.foo` |
| 硬编码英文 | 新写的组件又用回英文字面量 |

**均已做变异验证**：故意把 `t('resp.sentAfterSubstitution')` 改回英文，
以及写一个不存在的 key，两条测试都如实报错。

**验收结果**：

- 切换语言不刷新即生效 ✅
- 刷新后保持 ✅
- 中文界面无英文残留；英文界面同样完整 ✅
- 四种响应分支（JSON / 图片 / 二进制 / 错误）在新语言下正常 ✅
- 耗时明细、Header 分组、两个对话框均正确 ✅

**遗留**：`{{variable}}` 这类占位符与 `my-endpoint` 示例保持原样，
不翻译——它们是语法的一部分，翻译反而会误导。

---

## 阶段四 · 阶段耗时　0.5 ～ 1 天　✅ 已完成

### 4.1 采集　0.5 天　✅

用 `net/http/httptrace` 记录：

| 阶段 | 钩子 |
| ---- | ---- |
| DNS | `DNSStart` / `DNSDone` |
| 建连 | `ConnectStart` / `ConnectDone` |
| TLS | `TLSHandshakeStart` / `TLSHandshakeDone` |
| 首字节 | `GotFirstResponseByte` |

累积到 `Response.Timing`（各阶段毫秒）。

**实现要点**：trace 回调在 transport 的 goroutine 中触发，累积器加锁；
成功/失败/读取失败三条返回路径都带上 timing。

**实测验证**：

| 场景 | 预期 | 实测 |
| ---- | ---- | ---- |
| 延迟 3s | TTFB ≈ 3000 | 3003ms |
| 头快体慢 | TTFB 小、Download 大 | 327 / 5712ms |
| HTTPS 外网 | 出现 DNS 与 TLS | 46 / 1167ms |
| 内网复用 | `reused=true`，connect/tls 为 0 | ✓ |

### 4.2 存储与展示　0.5 天　✅

- 历史表追加 `timing TEXT` 列（**新增迁移块，未改动已有块**）
- 响应区 `Headers` 下新增 `Timing` 分组，横向条展示占比

**验收结果**：

- 已有数据库启动后自动迁移，42 条旧记录完好，`timing` 为空且不报错
- 新记录写入并可回溯
- 旧数据在界面上不显示 Timing 分组（而非显示 0）

**设计取舍**：`TTFB` 从请求发起算到首字节，因此包含 DNS/连接/TLS，
界面上明确标注，避免被误读为「等待服务器」的净耗时。`Download` 由总耗时
减去 TTFB 得出，与状态栏数字自洽。

---

## 阶段五 · 凭证保护与鉴权　1 天　✅ 已完成

原计划只有「方案 A：收紧文件权限 0.3 天」，实施时发现两个前提不成立，
最终范围扩大到 1 天。

### 5.1 API 鉴权　0.7 天　✅

**问题**：`GET /api/requests` 无需任何凭据就返回明文 Bearer token，
默认监听 `:5178`（所有网卡）。

**做法**：

| 环节 | 实现 |
| ---- | ---- |
| 生成 | 32 字节 `crypto/rand`，base64url |
| 存储 | 只存 SHA-256，比对用 `subtle.ConstantTimeCompare` |
| 浏览器 | `/?token=…` → `HttpOnly; SameSite=Strict` cookie → 302 到 `/` |
| 脚本 | `Authorization: Bearer …`，也接受裸值 |
| 重启 | 复用已存哈希，不踢下线 |
| 轮换 | `-token` 指定固定值 |

**公开路径**（两条，都有理由）：`/api/health` 是顶栏在线指示；`/mock/*`
是发布给别人调的接口。

**前端**：401 时整屏显示解锁页。不用弹窗——没 token 时没有工作区可显示，
弹窗后的空壳会让人以为程序坏了。

### 5.2 文件权限　0.3 天　✅

**发现**：`os.Chmod(path, 0600)` 在 Windows 上只切换只读位，实测
`mode after chmod(0600) = -rw-rw-rw-`。而实际部署目标是
Windows Server 2012 R2——原计划在这一平台上等于没做。

**做法**：`golang.org/x/sys/windows` + `SetNamedSecurityInfo`，DACL 只留
当前用户 + SYSTEM + Administrators。目录用**可继承 ACE**，使 SQLite 自己
创建的 `-wal` / `-shm` 也受限。Unix 分支保留 `0600`/`0700`。

**验收**：

| 项 | 结果 |
| -- | ---- |
| 原 ACL | `BUILTIN\Users` 可读、`Authenticated Users` 可改 |
| 收紧后 | 仅当前用户 / SYSTEM / Administrators |
| `-wal`、`-shm` | 通过继承同样受限 |
| 真实数据库 | 43 个请求完好，服务正常 |

### 5.3 验证　✅

| 层 | 用例 | 覆盖 |
| -- | ---- | ---- |
| Go | 19 | 拒绝未授权、接受 Bearer/裸值、健康检查与 mock 公开、**每条 API 路由都被覆盖**、cookie 交换、重启复用、`-token` 轮换、`-no-auth` |
| Go | 1 | 数据库及侧文件不可被 `Users` 读取 |
| 前端 | 13 | 解锁页、401 事件、失败提示 |

**变异验证**：

- 移除 `r.Use(cfg.Auth.guard)` → 3 个测试立即失败
- 把 `/api/export` 误加进公开名单 → 精确报出 `GET /api/export = 200, want 401`
- 跳过目录收紧 → 报出 `api.db-wal: DACL grants access to BUILTIN\Users`
- 移除 401 事件派发 → 前端用例失败

其中 ACL 测试第一版是**假阳性**：用 `t.TempDir()` 时父目录本就受限，变异后
仍然通过。改为先显式把目录放宽到 `BUILTIN\Users` 可读，才真正验证到。

### 未做

加密落盘（方案 B）：拿到数据库文件的人仍能读出凭据。已写入 README 的
Known limits。

**2026-10-21 决定暂不实现**：当前定位是本机调度工具，数据库文件不离开
这台机器，加密的主要收益（防「文件被拷走」）不成立，而密钥管理会带来
新的故障模式（密钥丢失、重装后无法解密、启动索要口令）。完整的备选方案、
DPAPI 的信任边界、以及实现时必须一并处理的四点，记录在
[TODO.md](./TODO.md) 的「历史快照中的凭证为明文」一节，重新考虑时无需
重新调研。

---

## 阶段六 · 其余按需

以下不排期，按实际使用中的痛点取舍。

| 项 | 工时 | 触发条件 |
| -- | ---- | -------- |
| 凭证加密（方案 B，首选 DPAPI） | 1.5 | 数据库可能被复制走 / 多人共用 / 部署到不受信网络 |
| 同步目录警告 + 导出提示（加密的廉价替代） | 0.1 | 担心导出文件或备份被误发 |
| SSE 支持 | 1 ～ 1.5 | 需要调试流式接口（**建议下一项**） |
| Mock 转发请求体 | 0.5 ～ 1 | 需要「透传代理」而非「固定响应」 |
| 请求链式调用 | 2 ～ 3 | 需要「先登录再调用」的自动化 |
| WebSocket 支持 | 3 ～ 5 | 有明确的 WS 调试需求 |

原表里的「race detector 验证」已移出：CI 每次推送都会跑 `go test -race`。

---

## 执行约定

**每项开始前**

1. 确认 `git status` 干净
2. 在 TODO 里把该项标为 `[~]`

**每项完成后**

1. `go vet ./... && go test ./...`（改动后端时）
2. `cd web && pnpm exec tsc -b && pnpm lint && pnpm test`（改动前端时）
3. 用 `mocktest` 手工验证至少一条相关路径

   ```bash
   go run ./mocktest        # 监听 127.0.0.1:5250
   ```

   覆盖状态码、内容类型、重定向、慢响应、错误形态等场景，浏览器打开
   `http://127.0.0.1:5250` 有索引页。
4. 更新 `docs/history/CHANGELOG.md` 与 `docs/history/WORKLOG.md`
5. 在 TODO 里标为 `[x]`，或移到「已修复」区
6. 提交信息说明**为什么**，不只是改了什么

**涉及数据库时**

- 迁移只追加，不修改已发布的语句块
- 造一个旧库验证：老数据能读、新字段有默认值

**涉及兼容分支时**

- 改动是否也适用于 `legacy/win2008r2`？若适用，需要单独改一次
- 该分支不得引入 Go 1.21+ 特性，否则失去存在意义
- 在该分支的 CI 里，`.\scripts\check-legacy-pins.ps1` 与 `legacyguard` 会
  自动执行；本地改完也应各跑一次

---

## 里程碑

| 里程碑 | 包含 | 累计工时 | 状态 |
| ------ | ---- | -------- | ---- |
| M1 可交付 | 阶段一 | 0.5 天 | ✅ |
| M2 有回归保护 | M1 + 阶段二 | 2 ～ 2.5 天 | ✅ |
| M3 面向非英语用户 | M2 + 阶段三 | 3 ～ 4 天 | ✅ |
| M4 排障能力完整 | M3 + 阶段四 | 3.5 ～ 5 天 | ✅ |
| M5 可持续交付 | M4 + 阶段五 + 1.4 | 5 ～ 6 天 | ✅（`v0.3.0`） |
| M6 旧系统可交付 | 兼容分支 CI/CD + `v0.1.2-win2008r2` | +0.5 天 | ✅（2008 R2 三项修复待复验） |

M2 是关键节点：从它开始，后续每一步都有自动化兜底。M5 之后，验证与发版
都不再依赖某一台开发机；M6 把这一点扩展到了旧系统。

---

## 下一步

没有排期项。按需从阶段六里挑，优先 **SSE 支持**：它是剩余功能里最容易
起手、且不引入长期状态管理负担的一项。

唯一带外部依赖的待办是**2008 R2 SP1 实机验证**：产物已发布，需要一台
真机。步骤见 `legacy/win2008r2` 分支的 `docs/planning/TODO.md`。
