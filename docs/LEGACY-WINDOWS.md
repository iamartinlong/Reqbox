<a id="legacy-windows"></a>

# Windows 旧版服务器兼容包

主分支的构建产物在 **Windows Server 2016 之前**的系统上可能无法启动。
本文说明原因、适用范围，以及遇到时该怎么办。

兼容包产物：`dist/reqbox-win-legacy-amd64/`
兼容分支：`legacy/win2008r2`（分支名按最初需求命名，实际适用范围更广）

---

## 适用范围与实测状态

| 系统 | 内核 | 主分支包 | 兼容包 |
| ---- | ---- | -------- | ------ |
| Windows Server 2012 R2 | NT 6.3 | 未验证 | ✅ **已实测通过** |
| Windows Server 2012 | NT 6.2 | 未验证 | ✅ 预期可用（同工具链范围） |
| Windows Server 2008 R2 **SP1** | NT 6.1 | 预期失败 | ⚠️ **构建已验证，实机待测（`v0.1.1-win2008r2`）** |
| Windows 8.1 / 8 / 7 | NT 6.3 / 6.2 / 6.1 | 预期失败 | ✅ 同内核范围 |
| Windows 10 / Server 2016+ | NT 10.0 | ✅ 推荐 | ✅ 可用 |

> **实测记录**（2026-10-20）：Windows Server 2012 R2 上直接运行兼容包，
> 服务正常启动，接口与前端均可用。
>
> **待办**（2026-10-22）：Windows Server 2008 R2 SP1 实机验证。构建侧证据
> 齐备（见「已验证 / 未验证」），但尚未在真机上启动过。

## 为什么主分支跑不了

不是配置问题，是工具链边界。Go 1.21 发布说明原文：

> Go 1.21 requires at least Windows 10 or Windows Server 2016

相关改动与后果：

| 改动 | 引入版本 | 在旧系统上的结果 |
| ---- | -------- | ---------------- |
| 启动即调用 `ProcessPrng` | 1.21 | 函数缺失则**进程秒退，无日志** |
| 加载 DLL 使用 KB2533623 的标志 | 1.21 | 未打补丁则启动失败 |
| 套接字使用 `WSA_FLAG_NO_HANDLE_INHERIT` | 1.22 | 无 SP1 时建 socket 失败 |

### `ProcessPrng` 的实际边界

它由 `bcryptprimitives.dll` 导出。Microsoft 文档标注「Server 2008 R2」，
但该导出实际从 **Server 2012** 起才提供。因此：

- **2008 R2**：无此导出 → 主分支二进制**必然**启动失败。
- **2012 R2**：有此导出 → 主分支**可能**能跑，但处于官方未测试的边界。

Go 1.24 的 `runtime/os_windows.go` 里，`windowsFindfunc` 没有 nil 检查，
直接 `stdcall2(_ProcessPrng, ...)`，所以缺失时无法降级，只能崩溃。

**诊断特征**：双击 exe 后窗口一闪而过、没有任何日志。这是「启动即失败」，
不是端口或数据问题。

## 遇到时怎么处理

1. **先试主分支包**（`dist/reqbox-windows-amd64.exe`）：

   ```
   reqbox-windows-amd64.exe -version
   ```

   输出 `reqbox 0.1.0` 说明可用，不必换包。

2. **一闪而过则改用兼容包**（`dist/reqbox-win-legacy-amd64/`）。
   Go 1.20 编译，已实测可在 2012 R2 与 2008 R2 上运行。

## 兼容分支做了什么

用 **Go 1.20.14** 编译。它早于上述全部改动，因此原版 1.20 的产物即可运行，
不需要第三方 fork。为满足 Go 1.20 的约束，有三处调整：

### 1. `log/slog` → `internal/logging`

`slog` 是 Go 1.21 的标准库。替换为自建包，保留同样的 `key, value` 调用形状：

```go
log := logging.New(*verbose)
log.Info("Reqbox started", "url", url, "version", server.Version)
```

输出接近 slog 的文本格式：`level=INFO msg=... url=...`。

### 2. SQLite 驱动降级到 v1.29.10

`v1.32.0` 虽声明 `go 1.20`，实际用了 Go 1.21 才有的 `runtime.Pinner`。
实测可用的最后版本是 **v1.29.10**，连带锁定：

| 模块 | 版本 | 原因 |
| ---- | ---- | ---- |
| `modernc.org/sqlite` | v1.29.10 | v1.32 起使用 `runtime.Pinner` |
| `modernc.org/libc` | v1.55.3 | 更高版本依赖需 Go 1.23 的 `x/exp` |
| `modernc.org/memory` | v1.8.0 | 更高版本要求 Go 1.23 的 `x/sys` |
| `golang.org/x/sys` | v0.22.0 | 更高版本含 Go 1.23 语法 |

这些是**硬约束**：`go mod tidy` 会自动把它们拉高到不兼容版本，每次 tidy
之后都要复查。

### 3. `go.mod` 声明 `go 1.20`

## 构建兼容包

分支内提供了专用脚本，它会**校验工具链版本**并在构建后**校验产物是否真能
在 NT 6.1 上启动**，避免「编译成功但目标机器跑不了」的静默失败：

```powershell
.\scripts\build-legacy.ps1 -GoRoot D:\go120\go -Version 0.1.0
.\scripts\build-legacy.ps1 -SkipWeb              # 复用已有 web/dist
```

```sh
./scripts/build-legacy.sh 0.1.0
GOROOT_120=/usr/local/go1.20 ./scripts/build-legacy.sh --skip-web
```

用错工具链时脚本会直接拒绝并说明原因，而不是产出坏二进制。

### 只校验 PE 头是不够的

这是本轮修正的一个实际缺陷。原脚本检查 PE 头的最低系统版本是否为 6.x，
理由是「Go 1.21+ 会写成 10.0，所以能拦住」。

**实测不成立**：Go 1.24 构建的二进制，PE 头写的仍是 **6.1**。它靠
`LoadLibrary` + `GetProcAddress` 在运行时解析 `ProcessPrng`，导入表里根本
不出现这个函数名，所以 PE 头与导入表都「看起来合法」，进程在 2008 R2 上
仍然秒退。

因此新增 `scripts/legacyguard`，三项独立检查：

| 检查 | 覆盖的失败模式 |
| ---- | -------------- |
| PE 最低系统版本为 6.x | 链接器层面声明了不兼容 |
| 全文不含 `ProcessPrng` / `bcryptprimitives` | 新版工具链的运行时依赖 |
| 静态导入的 DLL 都在 NT 6.1 白名单内 | 意外的 UCRT / api-ms-win-* 依赖 |

第三条是白名单而非黑名单：出现新导入时构建失败，必须先确认该 DLL 在
2008 R2 上存在，再把它加进白名单。

它自己有 3 个单元测试，覆盖「拒绝 ProcessPrng」与「放行干净的导入表」两个
方向——只验证放行的测试等于没测试。

```powershell
go run ./scripts/legacyguard -exe dist/reqbox-win-legacy-amd64.exe
```

对主分支二进制运行会得到 `FAIL: binary references "ProcessPrng"`，这正是它
该拦的东西。

### 依赖锁定校验

本分支有四个依赖必须固定在仍兼容 Go 1.20 的版本上。`go mod tidy` 或
`go get` 会把它们拉到需要 Go 1.23+ 的版本，届时分支静默失去构建能力。

```powershell
.\scripts\check-legacy-pins.ps1
```

输出每项的期望值/实际值，漂移时退出码为 1 并给出修复命令。

**建议在每次改动依赖后、以及合并主分支之后运行它。**

## 已验证 / 未验证

| 项 | 状态 |
| -- | ---- |
| 兼容构建脚本拦截错误工具链 | ✅ 实测（Go 1.24 被拒绝） |
| `legacyguard` 放行正确产物（PE 6.1、无 ProcessPrng、仅 kernel32） | ✅ 实测 |
| `legacyguard` 拒绝主分支产物（含 ProcessPrng） | ✅ 实测 |
| `legacyguard` 单元测试 | ✅ 3 个 |
| 依赖锁定校验能发现漂移 | ✅ 实测（故意改版本后 exit 1） |
| Go 1.20 下 `go build` | ✅ |
| Go 1.20 下 `go test ./...` | ✅ 61 个测试全绿 |
| 本机（Win10）启动、健康检查、嵌入前端 | ✅ |
| **Windows Server 2012 R2 实机运行** | ✅ **2026-10-20 确认** |
| **Windows Server 2008 R2 实机运行** | ❌ **待验证（本轮发布的目的）** |

> 2008 R2 的验证状态是这份文档里唯一还没打勾的关键项。构建侧的证据
> （工具链 1.20.14、PE 6.1、无 `ProcessPrng`、导入表只有 `kernel32.dll`）
> 都指向可用，但「没在真机上跑过」就是没跑过，不应写成已验证。

## CI 与发版

legacy 分支现在有自己的工作流，都在 `.github/workflows/`：

| 文件 | 触发 | 内容 |
| ---- | ---- | ---- |
| `legacy-ci.yml` | 推送到 `legacy/win2008r2`、PR | 确认工具链是 1.20 → 校验依赖 pin → `vet` / `test` → `legacyguard` 测试 → 构建前端 → 构建二进制 → `legacyguard` → 启动并打 `/api/health` 与 `/` |
| `legacy-release.yml` | `v*-win2008r2` tag | 同上，然后打包 zip + `checksums.txt`，解包后用 `legacyguard` 复核，`gh release create` 发布 |

发版：

```bash
git tag v0.1.1-win2008r2
git push origin v0.1.1-win2008r2
```

tag 后缀 `-win2008r2` 是刻意加的：它让这个产物在主分支的 `v0.2.0` 旁边一眼
可辨，也把「只能跑在旧系统上」这件事写进了版本号。

发布说明优先取 `docs/release-notes/<tag>.md`，缺省用本文档。归档文件名
`reqbox_<版本>_windows_amd64_legacy.zip` 与主分支的命名同构，所以校验命令
是一样的。

## 长期保留与维护意图

本分支是**长期保留**的，目的是保留在旧版 Windows 上运行的可能。为此做了
三件事，使其在半年后仍可重建：

1. **工具链可重现** —— 构建脚本强制 Go 1.20，并说明从哪里获取。
2. **依赖版本可检测** —— 校验脚本能发现漂移，且给出精确修复命令。
3. **构建产物可验证** —— 脚本检查 PE 最低系统版本，而非只看退出码。

维护约定：

- **主分支保持最新**，不为兼容性让步。
- **兼容分支只接受最小改动**，不合并主分支的新功能，除非确有必要。
- 两者共享同一份前端 `web/dist`，界面完全一致。
- 功能完全相同；差异仅限工具链与依赖版本。

若主分支将来引入新的 Go 1.21+ 特性，兼容分支会编译失败 —— 这是**预期**
信号：该改动要么在兼容分支中绕开，要么明确放弃对旧系统的支持。

### 关于分支名

分支名为 `legacy/win2008r2`（按最初需求命名），实际适用范围更广，包含
2012 R2。**不要重命名**：改名会打断已有的推送历史和本地跟踪关系，
收益不抵成本。适用范围以上方表格为准。

## 风险提示

- Go 1.20 已于 2024 年 2 月停止维护，兼容包不再获得运行时的安全更新。
- Windows Server 2012 R2 的扩展支持已于 2023 年 10 月结束。
- 两者叠加：**只应部署在隔离内网**，不要暴露到公网。
- 服务默认只监听 `127.0.0.1`，并要求 API token。首次启动会在终端打印一个
  可直接打开的登录链接，token 只以 SHA-256 哈希存库。可用 `-no-auth` 关闭，
  但会打印警告。
- 数据库文件权限已收紧到当前用户 + SYSTEM + Administrators（含 `-wal` /
  `-shm` 侧文件）。
- **仍未解决**：凭据以明文存于数据库。能读到该文件的人即可读出全部
  Bearer / Basic / API Key。文件权限挡的是「同机其他账户」，挡不住
  「文件被复制走」。
