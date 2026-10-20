# 生成 Hushu 合集导入文件
# 源：E:\Project\5-义乌有线\projects\YiwumsHushuApi
# 该 Java 服务是代理层：/ywapi/* 接收业务参数，内部换 OAuth2 token
# 并做 AES 加解密后转发到 OSP 平台。因此这里导入的是代理层端点，
# 即 ApiMock 可直接调用的那一层。

$ErrorActionPreference = "Stop"
$now = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ss.fffffffZ")

$collectionId = [guid]::NewGuid().ToString()
$envId = [guid]::NewGuid().ToString()

function Row($key, $value) {
    @{ id = [guid]::NewGuid().ToString(); key = $key; value = $value; enabled = $true; description = "" }
}

function Req {
    param(
        [string]$Name,
        [string]$Method,
        [string]$Path,
        [string]$Doc,
        $Params = @(),
        $Headers = @(),
        [string]$BodyType = "none",
        [string]$Body = ""
    )
    @{
        id           = [guid]::NewGuid().ToString()
        collectionId = $collectionId
        name         = $Name
        method       = $Method
        url          = "{{hushuBase}}$Path"
        params       = @($Params)
        headers      = @($Headers)
        bodyType     = $BodyType
        body         = $Body
        form         = @()
        authType     = "none"
        auth         = @{}
        sortIndex    = 0
        exposeSlug   = ""
        exposed      = $false
        createdAt    = $now
        updatedAt    = $now
    }
}

# 通用 JSON 头 + 说明
$jsonHeader = @(Row "Content-Type" "application/json")

$requests = @()

# --- 1. 客户信息查询 ---------------------------------------------------
$custBody = @'
{
  "orgCode": "1004",
  "customerCode": "",
  "customerId": "",
  "customerType": null,
  "accountCode": "",
  "customerName": "",
  "mobile": null,
  "phone1": null,
  "identityNo": "",
  "cardNo": "",
  "stbNumber": "",
  "billNumber": "",
  "regionName": "",
  "doorDesc": "",
  "isRegionFuzzySearch": 0,
  "isKongGuan": 0,
  "shNumber": "",
  "currentPage": 1,
  "pageSize": 20,
  "channelId": 105,
  "isFtth": 0
}
'@
$requests += Req -Name "客户信息查询" -Method "POST" -Path "/ywapi/getCustomerInformation" `
    -BodyType "json" -Body $custBody -Headers $jsonHeader

# --- 2. 用户信息查询 ---------------------------------------------------
$userBody = @'
{
  "objectType": "1",
  "objectId": "",
  "currentPage": "1",
  "pageSize": "20",
  "channelId": "105"
}
'@
$requests += Req -Name "用户信息查询" -Method "POST" -Path "/ywapi/userInfo" `
    -BodyType "json" -Body $userBody -Headers $jsonHeader

# --- 3. 套餐查询 -------------------------------------------------------
$inquiryBody = @'
{
  "objectType": "1",
  "objectId": "",
  "channelId": "105"
}
'@
$requests += Req -Name "套餐查询" -Method "POST" -Path "/ywapi/packageInquiry" `
    -BodyType "json" -Body $inquiryBody -Headers $jsonHeader

# --- 4. 订购判断 -------------------------------------------------------
$judgeBody = @'
{
  "objectType": "1",
  "objectId": "",
  "operType": "1",
  "prodIds": "",
  "offerIds": "",
  "channelId": "105"
}
'@
$requests += Req -Name "订购判断" -Method "POST" -Path "/ywapi/orderingJudgment" `
    -BodyType "json" -Body $judgeBody -Headers $jsonHeader

# --- 5. 套餐订购 -------------------------------------------------------
$subBody = @'
{
  "objectType": "1",
  "objectId": "",
  "operType": "1",
  "offerList": [
    {
      "offerIds": "",
      "startDate": "2026-10-20 00:00:00",
      "endDate": "2026-11-20 00:00:00"
    }
  ],
  "longflowFlag": "0",
  "bookingDate": "",
  "ticketMainType": "",
  "ticketSubType": "",
  "developerCode": "",
  "opCode": "",
  "channelId": "105"
}
'@
$requests += Req -Name "套餐订购" -Method "POST" -Path "/ywapi/packageSubscription" `
    -BodyType "json" -Body $subBody -Headers $jsonHeader

# --- 6. 共享接口（通用转发） -------------------------------------------
$shareBody = @'
{
  "apiCode": "ws_fx_cc_custInfoQuery",
  "apiVersion": "1.0.0",
  "param": "{\"orgCode\":\"1004\",\"channelId\":105}"
}
'@
$requests += Req -Name "共享接口（任意 apiCode 转发）" -Method "POST" -Path "/ywapi/share" `
    -BodyType "json" -Body $shareBody -Headers $jsonHeader

# --- 7. 测试（取 token） -----------------------------------------------
$requests += Req -Name "测试（获取 Token）" -Method "GET" -Path "/ywapi/test"

# sortIndex 按数组顺序
for ($i = 0; $i -lt $requests.Count; $i++) { $requests[$i].sortIndex = $i + 1 }

$export = @{
    version             = "1"
    exportedAt          = $now
    collections         = @(
        @{
            id        = $collectionId
            name      = "Hushu"
            sortIndex = 1
            createdAt = $now
            updatedAt = $now
        }
    )
    requests            = $requests
    environments        = @(
        @{
            id        = $envId
            name      = "Hushu Local"
            sortIndex = 1
            variables = @(
                # The Java service binds :8080 (application.yml), but nothing is
                # running locally, so the value is left as an obvious placeholder
                # rather than 127.0.0.1:8080 - that port happens to be held by an
                # unrelated process and a request would silently hit it instead.
                # Set this to wherever the service actually runs, then send.
                @{ id = [guid]::NewGuid().ToString(); key = "hushuBase"; value = "http://hushu-host:8080"; enabled = $true; secret = $false }
            )
            createdAt = $now
            updatedAt = $now
        }
    )
    activeEnvironmentId = $envId
}

$out = Join-Path $PSScriptRoot "hushu-import.json"
$export | ConvertTo-Json -Depth 12 | Set-Content -Path $out -Encoding utf8
Write-Output "written: $out"
Write-Output "collection: Hushu ($collectionId)"
Write-Output "requests: $($requests.Count)"
