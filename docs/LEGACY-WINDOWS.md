<a id="legacy-windows"></a>

# Windows 旧版服务器兼容包

主分支的构建产物在 **Windows Server 2016 之前**的系统上可能无法启动。
本文说明原因、适用范围，以及遇到时该怎么办。

> **本文件在 `main` 上是简版**。完整版本（构建细节、CI、发版、`legacyguard`
> 的三项检查）在 `legacy/win2008r2` 分支的 `docs/LEGACY-WINDOWS.md`。两边
> 的差异是刻意的：主分支读者只需要知道「这是怎么回事、该下哪个包」。

兼容包下载：<https://github.com/iamartinlong/Reqbox/releases/tag/v0.1.1-win2008r2>
（文件名 `reqbox_0.1.1_windows_amd64_legacy.zip`）
兼容分支：`legacy/win2008r2`（分支名按最初需求命名，实际适用范围更广）

---

## 适用范围与实测状态

| 系统 | 内核 | 主分支包 | 兼容包 |
| ---- | ---- | -------- | ------ |
| Windows Server 2012 R2 | NT 6.3 | 未验证 | ✅ **已实测通过** |
| Windows Server 2012 | NT 6.2 | 未验证 | ✅ 预期可用（同工具链范围） |
| Windows Server 2008 R2 **SP1** | NT 6.1 | 预期失败 | ⚠️ **构建已验证，实机待测** |
| Windows 8.1 / 8 / 7 | NT 6.3 / 6.2 / 6.1 | 预期失败 | ✅ 同内核范围 |
| Windows 10 / Server 2016+ | NT 10.0 | ✅ 推荐 | ✅ 可用 |

> **实测记录**（2026-10-20）：Windows Server 2012 R2 上直接运行兼容包，
> 服务正常启动，接口与前端均可用。

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

1. **先试主分支包**（`reqbox_<版本>_windows_amd64.zip` 里的 `reqbox.exe`）：

   ```
   reqbox.exe -version
   ```

   有输出就说明可用，不必换包。

2. **一闪而过则改用兼容包**
   （`reqbox_<版本>_windows_amd64_legacy.zip`，见上方的 Release 链接）。
   Go 1.20.14 编译；2012 R2 已实测通过，2008 R2 的实机验证仍在等结果。

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

```bash
# 需要 Go 1.20（本机位于 D:\go120）
export PATH="D:/go120/go/bin:$PATH"
export GOTOOLCHAIN=local    # 不可省略

go build -trimpath \
  -ldflags "-s -w -X reqbox/internal/server.Version=0.1.0" \
  -o reqbox.exe ./cmd/reqbox
```

`GOTOOLCHAIN=local` 必须设置：否则新版 go 命令读到 `go 1.20` 指令后会自动
下载更新的工具链，等于白降级。

**但只验证 PE 头是不够的。** Go 1.24 构建的二进制，PE 头**仍然是 6.1**；
`ProcessPrng` 是运行时通过 `LoadLibrary` 解析的，导入表里没有这个符号。
所以「PE 头是 6.x」不能证明兼容。

要用 `legacyguard`（在 `legacy/win2008r2` 分支上），它检查三项：PE 头、
全文禁用符号、导入 DLL 白名单，并且有自己的单元测试：

```powershell
git checkout legacy/win2008r2
go run ./scripts/legacyguard -exe reqbox.exe
```

对主分支的二进制运行会得到：

```
legacyguard: FAIL
  binary references "ProcessPrng". Go 1.21+ requires ProcessPrng, ...
```

对兼容包则放行。构建脚本（`build-legacy.ps1` / `.sh`）与 CI 都已接上，
**不要绕过它**。

## 已验证 / 未验证

| 项 | 状态 |
| -- | ---- |
| Go 1.20 下 `go build` | ✅ |
| Go 1.20 下 `go test ./...` | ✅ 61 个测试全绿 |
| PE 头声明 6.1 | ✅ |
| 全文无 `ProcessPrng` / `bcryptprimitives` | ✅ `legacyguard` |
| 静态导入仅 `kernel32.dll` | ✅ `legacyguard` |
| 本机（Win10）启动、健康检查、嵌入前端 | ✅ |
| `v0.1.1-win2008r2` 发布产物下载回验（校验和、启动） | ✅ 2026-10-22 |
| **Windows Server 2012 R2 实机运行** | ✅ **2026-10-20 确认** |
| **Windows Server 2008 R2 SP1 实机运行** | ❌ **待验证** |

> 2008 R2 是唯一还没打勾的关键项。构建侧证据齐备，但「没在真机上跑过」
> 就是没跑过，不应写成已验证。

## 维护策略

- **主分支保持最新**，不为兼容性让步。
- **兼容分支只接受最小改动**，不合并主分支的新功能，除非确有必要。
- 两者共享同一份前端 `web/dist`，界面完全一致。
- 功能完全相同；差异仅限工具链与依赖版本。
- 两个分支各有自己的 CI 与发版工作流，详见该分支的文档。

若主分支将来引入新的 Go 1.21+ 特性，兼容分支会编译失败 —— 这是**预期**
信号：该改动要么在兼容分支中绕开，要么明确放弃对旧系统的支持。

## 风险提示

- Go 1.20 已于 2024 年 2 月停止维护，兼容包不再获得运行时的安全更新。
- Windows Server 2008 R2 的扩展支持已于 2020 年 1 月结束；
  2012 R2 于 2023 年 10 月结束。
- 两者叠加：**只应部署在隔离内网**，不要暴露到公网。
- 服务默认只监听 `127.0.0.1`，并要求 API token（首次启动打印一次性登录链接）。
  可用 `-no-auth` 关闭，但会打印警告。
- 凭据仍以明文存于数据库。文件权限挡「同机其他账户」，挡不住「文件被复制走」。
