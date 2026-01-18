# 通知系統開發清單

## 階段一：基礎建設

### 1. 資料庫模型補充
- [ ] 新增 `MemberDeviceToken` Model
  - 儲存使用者的 Firebase Device Token
  - 支援多設備（iOS/Android/Web）
  - Token 過期管理

### 2. 環境變數配置管理
- [ ] NotificationProvider CRUD API
  - [ ] POST `/api/providers` - 新增通知提供者設定
  - [ ] GET `/api/providers` - 查詢租戶的所有設定
  - [ ] PUT `/api/providers/:id` - 更新設定
  - [ ] DELETE `/api/providers/:id` - 刪除設定
  - [ ] GET `/api/providers/:id/test` - 測試設定是否有效

- [ ] 配置內容：
  - SMTP Config (host, port, username, password, from, use_tls)
  - SMS Config (provider, account_id, auth_token, from_phone)
  - Firebase Config (project_id, credential_json, server_key)

### 3. API Key 驗證模組
- [ ] API Key 中間件 (Middleware)
  - [ ] 驗證 Header 中的 `X-API-Key` 或 `Authorization: ApiKey {key}`
  - [ ] 查詢 Member 表確認 API Key 有效
  - [ ] 載入 Member 和 Tenant 資訊到 Context
  - [ ] 錯誤處理：無效/過期/缺失的 API Key

- [ ] API Key 管理 API
  - [ ] POST `/api/auth/regenerate-key` - 重新生成 API Key
  - [ ] GET `/api/auth/verify-key` - 驗證 API Key 是否有效

## 階段二：通知推播模組

### 4. SMTP 郵件服務
- [ ] SMTP Service 實作
  - [ ] 連線池管理
  - [ ] TLS/SSL 支援
  - [ ] HTML 郵件支援
  - [ ] 附件支援（選配）
  - [ ] 錯誤重試機制

- [ ] SMTP API
  - [ ] POST `/api/notifications/email` - 發送郵件
    - 參數：recipient_email, recipient_name, subject, body (支援 HTML)

### 5. SMS 簡訊服務
- [ ] SMS Service 實作
  - [ ] 整合 SMS Provider (Twilio/其他)
  - [ ] 電話號碼驗證
  - [ ] 字數限制檢查
  - [ ] 錯誤重試機制

- [ ] SMS API
  - [ ] POST `/api/notifications/sms` - 發送簡訊
    - 參數：recipient_phone, body

### 6. Firebase 推播服務
- [ ] Firebase Service 實作
  - [ ] Firebase Admin SDK 初始化
  - [ ] 多 Tenant 的 Firebase App 管理
  - [ ] Device Token 有效性檢查
  - [ ] 批次推播支援（一次發送給多設備）
  - [ ] 錯誤處理（無效 Token 自動停用）

- [ ] Firebase API
  - [ ] POST `/api/notifications/push` - 發送推播
    - 參數：member_id 或 device_token, title, body, data (額外資料)
  - [ ] POST `/api/devices/register` - 註冊設備 Token
  - [ ] DELETE `/api/devices/:token` - 刪除設備 Token
  - [ ] GET `/api/devices` - 查詢使用者的所有設備

## 階段三：進階功能

### 7. 統一通知 API
- [ ] Unified Notification Service
  - [ ] POST `/api/notifications/send` - 統一發送接口
    - 自動根據 Member Preference 選擇通道
    - 支援多通道同時發送（Email + SMS + Push）
    - 優先級機制（預設 > 自訂）

### 8. 通知記錄與追蹤
- [ ] NotificationLog 查詢 API
  - [ ] GET `/api/logs` - 查詢發送記錄
    - 篩選：狀態、類型、時間範圍、接收者
  - [ ] GET `/api/logs/:id` - 單一記錄詳情
  - [ ] GET `/api/logs/stats` - 統計資訊
    - 成功率、失敗率、各通道使用量

### 9. 使用者偏好設定
- [ ] Member Preference API
  - [ ] GET `/api/preferences` - 查詢偏好設定
  - [ ] PUT `/api/preferences` - 更新偏好設定
  - [ ] POST `/api/preferences/default` - 設定預設通道

### 10. Webhook 通知
- [ ] Webhook Service (選配)
  - [ ] 發送狀態回調 (成功/失敗)
  - [ ] 設定 Webhook URL
  - [ ] 驗證簽名機制

## 階段四：測試與部署

### 11. 單元測試
- [ ] Models 測試
- [ ] Services 測試 (使用 Mock)
- [ ] Controllers 測試
- [ ] Middleware 測試
- [ ] API 整合測試

### 12. 文件與部署
- [ ] Swagger API 文件更新
- [ ] README 使用說明
- [ ] 部署腳本
- [ ] 環境變數範例 (.env.example)
- [ ] Docker Compose 設定

## 額外建議補充

### 13. 安全性
- [ ] Rate Limiting (防止濫用)
  - 每個 API Key 的發送頻率限制
  - IP 限制（選配）
- [ ] 敏感資料加密
  - Provider Config 加密儲存
  - API Key 使用 Hash

### 14. 效能優化
- [ ] 非同步發送機制
  - 使用 Queue (Redis/RabbitMQ)
  - Worker Pool 處理
- [ ] 快取機制
  - Provider Config 快取
  - API Key 驗證快取

### 15. 監控與告警
- [ ] Prometheus Metrics
  - 發送成功率
  - API 請求量
  - 錯誤率
- [ ] 日誌系統
  - 結構化日誌
  - 錯誤追蹤

### 16. 成本控制
- [ ] 配額管理
  - 每個 Tenant 的發送配額
  - 超額告警
- [ ] 計費統計
  - 各通道用量統計
  - 費用估算

---

## 開發優先順序建議

### P0 (核心功能 - 必須完成)
1. 環境變數配置管理 API
2. API Key 驗證模組
3. SMTP 郵件服務
4. SMS 簡訊服務
5. Firebase 推播服務
6. 通知記錄查詢

### P1 (重要功能)
7. 統一通知 API
8. 使用者偏好設定
9. Device Token 管理
10. 單元測試

### P2 (進階功能)
11. 非同步發送機制
12. Rate Limiting
13. 統計與監控
14. Webhook 通知

### P3 (優化功能)
15. 配額管理
16. 快取機制
17. 效能優化
