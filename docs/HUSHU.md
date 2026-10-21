<a id="hushu"></a>

# Hushu 接口导入说明

从 Java 项目 `YiwumsHushuApi` 提取的接口，已作为 **Hushu** 合集导入 Reqbox。

- 源项目：`E:\Project\5-义乌有线\projects\YiwumsHushuApi`
- 生成脚本：[`scripts/import-hushu.ps1`](../scripts/import-hushu.ps1)
- 导入文件：[`scripts/hushu-import.json`](../scripts/hushu-import.json)

---

## 必须先做的一件事：填写服务地址

导入的环境变量 `hushuBase` 当前是占位值：

```
hushuBase = http://hushu-host:8080
```

**改成实际地址后再发送请求。** 之所以不用 `127.0.0.1:8080`：本机 8080
已被另一个无关进程占用，填本机地址会让请求静默打到那个服务上，返回一个
看起来像「接口不存在」的 404，很难排查。

端口依据是 `application.yml` 的 `server.port: 8080`。

> 本地没有部署这个 Java 服务，所以**导入未做真实的连通性验证**。
> 已验证的是：接口清单、字段与 VO 一致、变量能正确解析为完整 URL。

## 导入了哪一层，以及为什么

这个 Java 项目是**代理层**，不是最终上游：

```
调用方 → /ywapi/*  (本项目)  →  OSP 平台
           ↑ 换 OAuth2 token、AES 加解密、加签名
```

`utils/GetPost.java` 里能看到真实上游：

| 项 | 值 |
| -- | -- |
| 管控平台 | `http://125.210.49.180:32001/http` |
| Token 地址 | `http://125.210.49.180:32000/oauth2/oauth/token` |
| 鉴权方式 | `grant_type=client_credentials` + appCode/appSecret |
| 报文 | 请求体 AES-256 加密，响应也需解密 |
| 签名 | 请求体 MD5 再转十六进制大写，放 `sign` 头 |

**因此导入的是 `/ywapi/*` 这一层**，理由是：直接调 OSP 平台需要先取 token、
再对报文做 AES 加解密和签名 —— Reqbox 目前没有脚本或前置请求能力，硬编码
这些既不可行也不该把密钥写进请求定义。

导入代理层则开箱即用：由 Java 服务替我们完成鉴权与加解密。

## 7 个接口

| # | 名称 | 方法 | 路径 | 请求体 |
| - | ---- | ---- | ---- | ------ |
| 1 | 客户信息查询 | POST | `/ywapi/getCustomerInformation` | `CustInfoQueryVo`（21 字段） |
| 2 | 用户信息查询 | POST | `/ywapi/userInfo` | `UserInfoVo` |
| 3 | 套餐查询 | POST | `/ywapi/packageInquiry` | `PackageInquiryVo` |
| 4 | 订购判断 | POST | `/ywapi/orderingJudgment` | `OrderingJudgmentVo` |
| 5 | 套餐订购 | POST | `/ywapi/packageSubscription` | `PackageSubscriptionVo` |
| 6 | 共享接口 | POST | `/ywapi/share` | `ShareVo`（任意 apiCode 转发） |
| 7 | 测试 | GET | `/ywapi/test` | 无（返回 token） |

请求体按各 VO 的字段生成，含注释里的枚举说明，可直接改值发送。

### 几个字段的取值约定

- `objectType`：1 用户ID / 2 机顶盒号 / 3 智能卡 / 4 宽带登录名 / 5 客户ID /
  6 客户编号 / 7 账户ID / 8 账户编号 / 9 客户姓名_证件号码
- `operType`：订购与判断场景下 1 表示产品、2 表示套餐
- `channelId`：示例用 105（取自源项目注释中的调用样例）
- `isRegionFuzzySearch` / `isKongGuan` / `isFtth`：0 否、1 是
- `orgCode` 与 `channelId` 在客户信息查询中是**必选**

### 接口 6 是通用入口

`/ywapi/share` 接受 `{apiCode, apiVersion, param}`，`param` 是**字符串化的
JSON**（注意是字符串，不是嵌套对象）。源项目里各接口对应的 apiCode：

| 接口 | apiCode |
| ---- | ------- |
| 客户信息查询 | `ws_fx_cc_custInfoQuery` |
| 用户信息查询 | `ws_fx_oc_subInfoQuery` |
| 套餐订购 | `ws_fx_oc_subProdOrder` |
| 套餐查询 | `ws_fx_oc_queryProdSubscription` |
| 订购判断 | `ws_fx_oc_checkSubProdOrder` |

有专用端点时优先用专用的；`/ywapi/share` 适合调用上表以外的 apiCode。

## 如何重新导入

```powershell
# 1. 重新生成导入文件（会生成新的集合 id）
.\scripts\import-hushu.ps1

# 2. 导入
curl -X POST http://127.0.0.1:5178/api/import `
  -H "Content-Type: application/json" `
  --data-binary "@scripts/hushu-import.json"
```

**注意**：导入是**整体替换**整个工作区（这是 Reqbox 现有语义）。如果工作区里
已有其他请求，先导出备份：

```powershell
curl http://127.0.0.1:5178/api/export -o backup.json
```

## 已知限制

- `channelId` 等业务取值来自源码注释，是**示例值**，实际值以业务系统为准。
- 代理层返回的是上游原始 JSON 字符串，未做包装；`respCode`（`00000` 为成功）
  需要在使用时自行判断。
- 若上游要求特定日期格式，`bookingDate` / `startDate` / `endDate` 使用
  `yyyy-MM-dd HH:mm:ss`（源码注明可省略时分秒）。
