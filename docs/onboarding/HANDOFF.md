# 交接说明（兼容分支）

面向接手 `legacy/win2008r2` 的人。主分支的交接文档在 `main` 的
`docs/onboarding/HANDOFF.md`，这里只写**这个分支特有**的部分。

**最后更新**：2026-10-22

---

## 一、这个分支为什么存在

Go 1.21 起，构建产物在**启动时**调用 `bcryptprimitives.dll` 的
`ProcessPrng`。这个导出在 Windows Server 2008 R2 上**不存在**（Microsoft
文档标注有，实际从 2012 起才提供）。

结果是：主分支的 exe 在 2008 R2 上双击后**窗口一闪而过，没有任何日志**。
不是配置问题，不是端口问题，是加载期就失败了。

本分支用 **Go 1.20.14** 编译，因此没有这个依赖。功能与主分支相同，差异
只在工具链和四个依赖的版本。

---

## 二、分支与版本

| 项 | 值 |
| -- | -- |
| 分支 | `legacy/win2008r2` |
| 工具链 | **Go 1.20.14**（硬要求，见下） |
| 目标系统 | Windows Server 2008 R2 SP1（NT 6.1）及更新 |
| 最新版本 | `v0.1.2-win2008r2` |
| 产物 | `reqbox_<版本>_windows_amd64_legacy.zip` |

**tag 命名带 `-win2008r2` 后缀**：它在 `main` 的 `v0.2.0` 旁边一眼可辨，
也把「只能跑在旧系统上」写进了版本号。发版工作流只认这个后缀
（`v*-win2008r2`），故意不会误触发主分支的发布流程。

---

## 三、五条不能违反的规则

### 1. 只能用 Go 1.20 构建

用新版 Go 编译**会成功**，产出的二进制在 2008 R2 上**秒退、无日志**。
这是这个仓库里最难排查的一类失败。

```bash
GOTOOLCHAIN=go1.20.14 go build ./cmd/reqbox
```

构建脚本（`scripts/build-legacy.ps1` / `.sh`）会拒绝非 1.20 的工具链。

### 2. 不要跑 `go mod tidy`

它会把 `go 1.20` 抬到 1.21+，并把四个依赖升到需要 Go 1.23 的版本。
需要改依赖时用 `go mod edit -require=...`。

改完必须验证：

```powershell
.\scripts\check-legacy-pins.ps1     # 5 项 pin 全部 OK 才算过
```

CI 的第一批步骤里就有这一项。

### 3. 只校验 PE 头是不够的

Go 1.24 构建的二进制，PE 头写的**仍然是 6.1**。`ProcessPrng` 是运行时通过
`LoadLibrary` + `GetProcAddress` 解析的，导入表里根本没有这个符号。

所以「PE 头是 6.x」不能证明兼容。用 `legacyguard`：

```powershell
go run ./scripts/legacyguard -exe dist/reqbox-win-legacy-amd64.exe
```

它检查 PE 头、全文禁用符号、导入 DLL 白名单三项。构建脚本与 CI 都已接上，
**不要绕过它**。

### 4. 不合并主分支的新功能

本分支只接受兼容性所需的最小改动。主分支的前端测试（Vitest）、新功能、
依赖升级都不往这里带——它们会把新的 Go 版本要求或依赖树带进来。

需要新功能时，用主分支的构建（Windows Server 2016+ 可用）。

### 5. 2008 R2 还没在真机上验证过

构建侧证据齐备（1.20.14、PE 6.1、无 `ProcessPrng`、仅导入 `kernel32.dll`），
**但没有在真实 2008 R2 上启动过**。2012 R2 已于 2026-10-20 实测通过。

验证步骤见 [TODO.md](../planning/TODO.md) 的 P1。验证通过后记得改两处状态表：
`docs/planning/TODO.md` 与 `docs/development/LEGACY-WINDOWS.md`。

---

## 四、常用命令

```powershell
# 构建（含工具链校验、legacyguard）
.\scripts\build-legacy.ps1 -Version 0.1.1 -SkipWeb

# 只构建，复用已有 web/dist
.\scripts\build-legacy.ps1 -Version 0.1.1 -SkipWeb

# 校验依赖 pin 没有漂移
.\scripts\check-legacy-pins.ps1

# 校验产物的兼容性
go run ./scripts/legacyguard -exe dist/reqbox-win-legacy-amd64.exe

# 测试（必须用 1.20 工具链）
$env:GOTOOLCHAIN="go1.20.14"
go test ./...
```

发版：

```bash
git tag v0.1.2-win2008r2
git push origin v0.1.2-win2008r2
```

CI 与发版的细节见 [LEGACY-WINDOWS.md](../development/LEGACY-WINDOWS.md) 的「CI 与发版」。

---

## 五、CI 跑什么

| 工作流 | 触发 | 关键步骤 |
| ------ | ---- | -------- |
| `legacy-ci.yml` | 推送到本分支、PR | 工具链 = 1.20 → pin 校验 → `vet` / `test` → `legacyguard` 单测 → 构建前端与二进制 → `legacyguard` → 启动并打 `/api/health` 与 `/` |
| `legacy-release.yml` | `v*-win2008r2` tag | 同上，然后打 zip + `checksums.txt`，**解包后**再跑一次 `legacyguard`，用 `gh release create` 发布 |

两个工作流都跑在 `windows-latest` 上——目标平台就是 Windows，在 Windows 上
冒烟比在 Linux 上交叉编译更能说明问题。

`-race` 不在本分支的 CI 里：它需要 cgo 与 C 编译器，且与 2008 R2 的兼容性
无关。主分支的 CI 已覆盖。

---

## 六、踩过的坑

### 包里的 `LICENSE` 曾经不存在

本分支从主分支分出来时没带 `LICENSE`，打包脚本 `Copy-Item LICENSE` 直接失败。
已补上（`2026-10-22`）。若再次出现「打包失败但看不出原因」，先确认这三个
文件都在：`LICENSE`、`README.md`、`docs/development/LEGACY-WINDOWS.md`。

### `$args` 是 PowerShell 的自动变量

发版工作流最初写 `$args = @(...)` 再把参数传给 `gh`，这会静默出错。
已改名为 `$ghArgs`。写 PowerShell 脚本时避开 `$args`、`$input`、`$error`
这类自动变量。

### `Compress-Archive` 的进度输出会淹没日志

CI 里它默认输出大量进度对象。本仓库的脚本里没有抑制它（需要的话加
`-Confirm:$false` 与 `$ProgressPreference = 'SilentlyContinue'`），日志会长
但无害。

---

## 七、风险

- **Go 1.20 已于 2024 年 2 月停止维护**，不再有运行时安全修复。
- **Windows Server 2008 R2 的扩展支持已于 2020 年 1 月结束**。
- 两者叠加：**只应部署在隔离内网**，不要暴露到公网。
- 凭据以明文存于数据库（与主分支相同的决定）。文件权限挡「同机其他账户」，
  挡不住「文件被复制走」。

这些不是可以「以后优化」的事项，而是选型时就接受的约束。若哪天需要更好的
安全性，答案是换系统 + 用主分支构建，而不是继续加固这个分支。
