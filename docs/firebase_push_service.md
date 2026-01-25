# Firebase 推播服務實作說明

## 概述

本專案實作了完整的 Firebase Cloud Messaging (FCM) 推播服務，支援多租戶（Multi-Tenant）架構，提供設備管理和批次推播功能。

## 實作功能

### 1. Firebase Admin SDK 初始化
- ✅ 自動載入租戶的 Firebase 配置
- ✅ 支援多個 Firebase App 實例（每個租戶一個）
- ✅ 使用單例模式和互斥鎖確保執行緒安全

### 2. 多 Tenant 的 Firebase App 管理
- ✅ 根據 `tenant_id` 動態載入 Firebase 配置
- ✅ 快取 Firebase App 和 Messaging Client
- ✅ 從資料庫 `NotificationProvider` 表讀取配置

### 3. Device Token 有效性檢查
- ✅ `ValidateToken()` 方法驗證 token 有效性
- ✅ 發送推播時自動檢測無效 token
- ✅ 自動停用無效的 device token

### 4. 批次推播支援
- ✅ `SendToToken()` - 發送給單一設備
- ✅ `SendToMultipleTokens()` - 批次發送給多個設備
- ✅ `SendToMember()` - 發送給會員的所有設備

### 5. 錯誤處理
- ✅ 自動檢測 `InvalidArgument` 和 `RegistrationTokenNotRegistered` 錯誤
- ✅ 無效 token 自動設定 `is_active = false`
- ✅ 所有推播都記錄到 `NotificationLog` 表

## API 端點

### 1. POST `/api/v1/notifications/push` - 發送推播
**需要認證**: Bearer Token

**請求參數**:
```json
{
  "member_id": "uuid-string",        // 或使用 device_token (擇一)
  "device_token": "firebase-token",  // 或使用 member_id (擇一)
  "title": "通知標題",
  "body": "通知內容",
  "data": {                          // 可選的自訂資料
    "key1": "value1",
    "key2": "value2"
  }
}
```

**回應**:
```json
{
  "message": "notification sent to all member devices"
}
```

### 2. POST `/api/v1/devices/register` - 註冊設備 Token
**需要認證**: Bearer Token

**請求參數**:
```json
{
  "device_token": "firebase-device-token",
  "device_type": "ios"  // 可選值: ios, android, web
}
```

**回應**:
```json
{
  "message": "device token registered successfully"
}
```

### 3. DELETE `/api/v1/devices/:token` - 刪除設備 Token
**需要認證**: Bearer Token

**路徑參數**: `token` - 設備 Token

**回應**:
```json
{
  "message": "device token deleted successfully"
}
```

### 4. GET `/api/v1/devices` - 查詢使用者的所有設備
**需要認證**: Bearer Token

**回應**:
```json
[
  {
    "member_id": "uuid",
    "device_token": "token1",
    "device_type": "ios",
    "is_active": true,
    "last_used_at": "2026-01-18T14:00:00Z"
  },
  {
    "member_id": "uuid",
    "device_token": "token2",
    "device_type": "android",
    "is_active": true,
    "last_used_at": "2026-01-18T13:00:00Z"
  }
]
```

## 檔案結構

```
member_API/
├── services/
│   ├── firebase_service.go       # Firebase 服務核心實作
│   └── firebase_service_test.go  # 單元測試（使用 Mock）
├── controllers/
│   ├── notification_controller.go      # API 控制器
│   └── notification_controller_test.go # 控制器測試
├── models/
│   ├── member_device_token.go          # 設備 Token 模型
│   ├── notification_log.go             # 通知日誌模型
│   └── notification_provider.go        # 通知提供商模型
└── routes/
    └── routes.go                        # 路由定義
```

## 核心方法說明

### FirebaseService

#### `NewFirebaseService(db *gorm.DB) *FirebaseService`
建立新的 Firebase 服務實例。

#### `SendToToken(ctx, tenantID, memberID, token, title, body, data) error`
發送推播給單一設備。

#### `SendToMultipleTokens(ctx, tenantID, memberID, tokens, title, body, data) (*messaging.BatchResponse, error)`
批次發送推播給多個設備。

#### `SendToMember(ctx, tenantID, memberID, title, body, data) error`
發送推播給會員的所有啟用設備。

#### `ValidateToken(ctx, tenantID, token) (bool, error)`
驗證 Device Token 是否有效。

#### `DeactivateToken(token) error`
停用無效的 Device Token。

#### `RegisterDeviceToken(memberID, deviceToken, deviceType) error`
註冊或更新設備 Token。

#### `DeleteDeviceToken(token) error`
刪除設備 Token。

#### `GetMemberDevices(memberID) ([]MemberDeviceToken, error)`
取得會員的所有設備。

## 測試

### 執行所有測試
```bash
go test ./...
```

### 執行 Firebase 服務測試
```bash
go test ./services -v -run TestFirebase
```

### 執行控制器測試
```bash
go test ./controllers -v
```

### 測試覆蓋率
```bash
go test -cover ./...
```

## 設定說明

### 資料庫設定

確保以下表格已建立：
- `members` - 會員表
- `tenants` - 租戶表
- `notification_providers` - 通知提供商表
- `member_device_tokens` - 會員設備 Token 表
- `notification_logs` - 通知日誌表

### Firebase 設定

在 `notification_providers` 表中新增 Firebase 配置：

```sql
INSERT INTO notification_providers (
  tenants_id,
  name,
  type,
  config,
  is_active
) VALUES (
  'your-tenant-uuid',
  'Firebase Push Service',
  'FIREBASE',
  '{
    "project_id": "your-firebase-project",
    "credential_json": "{\"type\":\"service_account\",...}",
    "server_key": "your-server-key"
  }',
  true
);
```

## 驗收標準

### ✅ 完成項目

1. **單元測試通過**
   - Firebase 服務測試: 7/7 通過
   - 控制器測試: 7/7 通過
   - 所有測試使用 Mock，不需要真實的 Firebase 連接

2. **多設備推播**
   - `SendToMultipleTokens()` 支援批次發送
   - `SendToMember()` 自動找出會員的所有設備

3. **無效 Token 自動停用**
   - 發送失敗時檢查錯誤類型
   - 自動呼叫 `DeactivateToken()`
   - 設定 `is_active = false`

4. **記錄到 NotificationLog**
   - 每次發送都建立日誌記錄
   - 記錄成功/失敗狀態
   - 失敗時記錄錯誤訊息

## 安全性考量

1. **認證要求**: 所有 API 端點都需要 JWT Bearer Token
2. **租戶隔離**: 每個租戶使用獨立的 Firebase App
3. **設備綁定**: Device Token 綁定到特定會員
4. **自動清理**: 無效 token 自動停用

## 注意事項

1. **Context 依賴**: 控制器假設認證中間件會設定 `tenant_id` 和 `member_id`
2. **Firebase 憑證**: 需要正確的 Firebase Service Account JSON
3. **批次限制**: Firebase FCM 每次批次最多 500 個 token
4. **速率限制**: 注意 Firebase 的 API 配額限制

## 後續優化建議

1. 新增推播排程功能
2. 實作推播模板管理
3. 新增推播統計和分析
4. 支援推播優先級設定
5. 新增推播重試機制
6. 實作推播 A/B 測試
