# API Key 驗證中間件

## 功能說明

API Key 驗證中間件提供基於 API Key 的身份驗證機制，適用於服務間通訊或程式化 API 存取。

## 特性

- 支援兩種 Header 格式：
  - `X-API-Key: {api_key}`
  - `Authorization: ApiKey {api_key}`
- 自動載入 Member 和 Tenant 資訊到 Context
- 內建快取機制（5分鐘 TTL）
- 完整的錯誤處理

## 使用方式

### 1. 註冊時自動生成 API Key

當使用者透過 `/api/v1/register` 註冊時，系統會自動生成 API Key。

```bash
curl -X POST http://localhost:9876/api/v1/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "測試用戶",
    "email": "test@example.com",
    "password": "password123"
  }'
```

回應會包含 JWT token 和用戶資訊（API Key 不會在註冊時直接返回）。

### 2. 重新生成 API Key

使用 JWT token 驗證後，可以重新生成 API Key：

```bash
curl -X POST http://localhost:9876/api/v1/auth/regenerate-key \
  -H "Authorization: Bearer {jwt_token}"
```

回應：
```json
{
  "message": "API Key 已重新生成",
  "api_key": "ak_1234567890abcdef..."
}
```

### 3. 驗證 API Key

使用 API Key 驗證：

```bash
# 使用 X-API-Key header
curl http://localhost:9876/api/v1/auth/verify-key \
  -H "X-API-Key: ak_1234567890abcdef..."

# 或使用 Authorization header
curl http://localhost:9876/api/v1/auth/verify-key \
  -H "Authorization: ApiKey ak_1234567890abcdef..."
```

回應：
```json
{
  "valid": true,
  "member": {
    "id": "uuid",
    "name": "測試用戶",
    "email": "test@example.com"
  },
  "tenant": {
    "id": "uuid",
    "name": "租戶名稱",
    "description": "租戶描述"
  }
}
```

### 4. 在路由中使用中間件

```go
import "member_API/auth"

apiKeyProtected := router.Group("/api/v1")
apiKeyProtected.Use(auth.APIKeyMiddleware())
{
    apiKeyProtected.GET("/protected-endpoint", handler)
}
```

### 5. 在 Handler 中取得驗證資訊

```go
func handler(c *gin.Context) {
    // 取得 Member 資訊
    member, _ := c.Get("member")
    memberData := member.(*models.Member)
    
    // 取得 Member ID
    memberID, _ := c.Get("member_id")
    
    // 取得 Tenant 資訊（可能為 nil）
    tenant, exists := c.Get("tenant")
    if exists {
        tenantData := tenant.(*models.Tenants)
    }
}
```

## API Key 格式

API Key 格式為 `ak_` 加上 64 個十六進制字元，例如：
```
ak_1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef
```

## 快取機制

- 驗證過的 API Key 會被快取 5 分鐘
- 快取會在過期後自動清除
- 快取使用 `sync.Map` 實現，線程安全

## 錯誤處理

| 錯誤情況 | HTTP 狀態碼 | 錯誤訊息 |
|---------|-----------|---------|
| 缺少 API Key | 401 | 缺少 API Key |
| 無效的 API Key | 401 | 無效的 API Key |
| 資料庫未配置 | 500 | 數據庫未配置 |

## 安全建議

1. 定期更換 API Key
2. 使用 HTTPS 傳輸
3. 不要在客戶端程式碼中暴露 API Key
4. 妥善保管 API Key，避免提交到版本控制系統
