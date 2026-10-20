<a id="legacy-windows"></a>

# Windows Server 2008 R2 兼容分支

主分支的构建产物**无法在 Windows Server 2008 R2 上运行**。本文说明原因、
兼容分支做了什么，以及如何维护它。

分支：`legacy/win2008r2`

---

## 为什么主分支跑不了

不是配置问题，是工具链边界。Go 官方支持表写得很明确：

| 系统 | 最后支持的 Go 版本 |
| ---- | ------------------ |
| Windows 10 / Server 2016+ | 当前版本 |
| Windows 8 / Server 2012 | Go 1.20 |
| **Windows 7 / Server 2008 R2** | **Go 1.20** |

主分支用 Go 1.24。从 Go 1.21 起，编译出的程序依赖 2008 R2 不具备的能力：

| 改动 | 引入版本 | 在 2008 R2 上的结果 |
| ---- | -------- | ------------------- |
| 启动即调用 `ProcessPrng` | 1.21（1.22 加强） | 找不到入口点，**进程秒退** |
| 加载 DLL 使用 KB2533623 引入的标志 | 1.21 | 未打该补丁则**启动失败** |
| 套接字使用 `WSA_FLAG_NO_HANDLE_INHERIT` | 1.22 | 无 SP1 时**建 socket 失败** |
| PE 头写入 Windows 10 最低版本 | 1.27 | 旧系统拒绝加载 |

诊断特征：**双击 exe 后窗口一闪而过，没有任何日志**。这正是「启动即失败」
的表现，而不是端口或数据问题。

## 兼容分支的取舍

用 **Go 1.20.14** 编译。它早于上述全部改动，因此原版 1.20 的产物可以直接
在 2008 R2 上运行 —— 不需要第三方 fork。

代价：为满足 Go 1.20 的语法与依赖约束，做了三处调整。

### 1. `log/slog` → `internal/logging`

`slog` 是 Go 1.21 的标准库。替换为自建包，保留同样的 `key, value` 调用
形状，调用点无需改写：

```go
log := logging.New(*verbose)
log.Info("ApiMock started", "url", url, "version", server.Version)
```

输出格式与 slog 的文本处理器相近：`level=INFO msg=... url=...`。

### 2. SQLite 驱动降级

`modernc.org/sqlite` 从 v1.30 起就要求更高的 Go 版本；v1.32.0 虽然声明
`go 1.20`，实际却使用了 Go 1.21 才有的 `runtime.Pinner`。

实测可用的最后一个版本是 **v1.29.10**。连带锁定：

| 模块 | 版本 | 原因 |
| ---- | ---- | ---- |
| `modernc.org/sqlite` | v1.29.10 | v1.32 起用 `runtime.Pinner` |
| `modernc.org/libc` | v1.55.3 | 更高版本依赖需 Go 1.23 的 `golang.org/x/exp` |
| `modernc.org/memory` | v1.8.0 | 更高版本要求 Go 1.23 的 `golang.org/x/sys` |
| `golang.org/x/sys` | v0.22.0 | 更高版本包含 Go 1.23 语法 |

这些是**硬约束**，`go mod tidy` 会自动把它们拉高到不兼容版本，因此每次
tidy 之后都要检查。

### 3. `go.mod` 声明

```
go 1.20
```

## 构建

```bash
# 需要 Go 1.20（本机放于 D:\go120）
export PATH="D:/go120/go/bin:$PATH"
export GOTOOLCHAIN=local    # 禁止 Go 自动切换到新工具链

go build -trimpath \
  -ldflags "-s -w -X apimock/internal/server.Version=0.1.0" \
  -o apimock.exe ./cmd/apimock
```

`GOTOOLCHAIN=local` 不可省略：否则新版 go 命令会因 `go 1.20` 指令自动下载
并使用更新的工具链，等于白降级。

**验证 PE 头**（应为 6.1，而不是 10.0）：

```powershell
$b = [IO.File]::ReadAllBytes("apimock.exe")
$pe = [BitConverter]::ToInt32($b, 0x3C); $opt = $pe + 24
"min OS: $([BitConverter]::ToUInt16($b,$opt+40)).$([BitConverter]::ToUInt16($b,$opt+42))"
```

## 已验证 / 未验证

| 项 | 状态 |
| -- | ---- |
| Go 1.20 下 `go build` | ✅ 通过 |
| Go 1.20 下 `go test ./...` | ✅ 41 个测试全绿 |
| PE 头声明 6.1 | ✅ |
| 在本机（Win10）启动、健康检查、嵌入前端、预置数据 | ✅ |
| **在真实 Windows Server 2008 R2 上运行** | ❌ **未验证** |

最后一项只能在实际服务器上确认。请在服务器上执行：

```
apimock.exe -version
apimock.exe -addr 127.0.0.1:5178
```

预期输出 `apimock 0.1.0`，然后 `http://127.0.0.1:5178/api/health` 返回
`{"status":"ok"}`。

**如果仍然秒退**，可能是系统缺少 Service Pack 1。Go 1.20 的部分网络代码
需要 SP1 提供的标志（`WSA_FLAG_NO_HANDLE_INHERIT`）。检查系统版本：

```
systeminfo | findstr /B /C:"OS Name" /C:"OS Version"
```

`6.1.7601` 是 SP1，`6.1.7600` 则不是。

## 维护策略

- **主分支保持最新**，不为兼容性让步。
- **兼容分支只在必要时接受最小改动**，不合并主分支的新功能，除非确有必要。
- 两者共享同一份前端 `web/dist`，界面完全一致。
- 功能上两者相同；差异仅限工具链与依赖版本。

若将来主分支引入新的 Go 1.21+ 特性，兼容分支会编译失败 —— 这是**预期**
的信号，说明该改动要么在兼容分支中绕开，要么明确放弃兼容。
