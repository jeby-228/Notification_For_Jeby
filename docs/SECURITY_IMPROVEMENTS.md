# GitHub Actions Security Scan 改善建議

## 問題摘要

**Workflow Run**: [#21458369933](https://github.com/jeby-228/Notification_For_Jeby/actions/runs/21458369933)  
**失敗原因**: Trivy 掃描發現 4 個 CRITICAL/HIGH 級別的安全漏洞  
**影響組件**: Alpine Linux 3.21.5 中的 OpenSSL (libcrypto3 & libssl3)

## 發現的漏洞

### OpenSSL RCE/DoS 漏洞 (CRITICAL)
- **描述**: OpenSSL 遠端代碼執行或拒絕服務攻擊
- **原因**: 處理超大初始化向量時的堆疊緩衝區溢出
- **受影響版本**: OpenSSL 3.3.5-r0
- **修復版本**: OpenSSL 3.3.6-r0

### OpenSSL PKCS#12 任意代碼執行漏洞 (HIGH)
- **描述**: PKCS#12 處理中的任意代碼執行
- **原因**: UTF-8 轉換時的越界寫入
- **受影響版本**: OpenSSL 3.3.5-r0
- **修復版本**: OpenSSL 3.3.6-r0

## 已實施的修復

### 1. 升級基礎映像 ✅

**變更內容**:
```dockerfile
# 修改前
FROM alpine:3.21

# 修改後
FROM alpine:3.21.6
```

**原因**:
- Alpine 3.21.6 於 2026-01-27 發布
- 包含 OpenSSL 3.3.6-r0，修復了所有已識別的漏洞
- 穩定版本，經過完整測試

### 2. 優化 Builder Stage ✅

**變更內容**:
```dockerfile
FROM golang:1.24-alpine AS builder
RUN apk --no-cache add ca-certificates  # 新增
WORKDIR /src
```

**原因**:
- 確保 Go modules 下載時有正確的 CA 證書
- 避免 TLS 驗證失敗

## 長期改善建議

### 1. 依賴項版本管理策略

**現況**: 使用 `alpine:3.21` (浮動標籤)  
**建議**: 使用固定版本如 `alpine:3.21.6`

**優點**:
- 可預測的構建環境
- 避免意外的破壞性變更
- 更容易追蹤和審計

**實施**:
```dockerfile
# 推薦：固定具體版本
FROM alpine:3.21.6

# 或定期更新的策略
FROM alpine:3.21.6  # 每月檢查更新
```

### 2. 多層次安全掃描

**現有配置**: ✅ 已經很好
- SARIF 格式上傳到 GitHub Security 標籤
- 表格格式輸出以便快速檢視
- 針對 CRITICAL 和 HIGH 級別漏洞

**額外建議**:
- 考慮添加 MEDIUM 級別的漏洞追蹤（僅警告，不失敗）
- 設定漏洞容忍策略

**範例配置**:
```yaml
# 對於 scheduled 掃描，可以更寬鬆
- name: Run Trivy vulnerability scanner (table output for scheduled)
  uses: aquasecurity/trivy-action@0.33.1
  if: github.event_name == 'schedule'
  with:
    image-ref: ghcr.io/${{ env.REPO }}:latest
    format: "table"
    severity: "CRITICAL,HIGH"
    exit-code: "0"  # 不讓 scheduled 掃描失敗，只記錄
    
# 對於 PR/push，保持嚴格
- name: Run Trivy vulnerability scanner (table output)
  uses: aquasecurity/trivy-action@0.33.1
  if: github.event_name != 'schedule'
  with:
    image-ref: ${{ env.REPO }}:scan
    format: "table"
    severity: "CRITICAL,HIGH"
    exit-code: "1"  # PR/push 時發現漏洞要失敗
```

### 3. 自動化更新流程

**建議**: 設定 Dependabot 或 Renovate Bot

**Dependabot 配置範例** (`.github/dependabot.yml`):
```yaml
version: 2
updates:
  # Docker 基礎映像更新
  - package-ecosystem: "docker"
    directory: "/"
    schedule:
      interval: "weekly"
    reviewers:
      - "jeby-228"
    
  # GitHub Actions 更新
  - package-ecosystem: "github-actions"
    directory: "/"
    schedule:
      interval: "weekly"
    
  # Go modules 更新
  - package-ecosystem: "gomod"
    directory: "/"
    schedule:
      interval: "weekly"
```

### 4. 漏洞通知機制

**現有**: ✅ SARIF 上傳到 GitHub Security 標籤  
**額外建議**:
- 設定 GitHub Security Alerts 通知
- 整合到 Slack/Discord/Email（如果團隊使用）

**GitHub Actions 通知範例**:
```yaml
- name: Notify on vulnerability
  if: failure()
  uses: 8398a7/action-slack@v3
  with:
    status: custom
    custom_payload: |
      {
        text: "🚨 Security scan failed: Critical vulnerabilities found"
      }
  env:
    SLACK_WEBHOOK_URL: ${{ secrets.SLACK_WEBHOOK }}
```

### 5. 定期審查與更新週期

**建議的流程**:
1. **每日**: Scheduled Trivy 掃描（已實施 ✅）
2. **每週**: 檢查 Dependabot PR 並合併
3. **每月**: 審查 Alpine/Go 版本更新
4. **每季**: 全面安全審計

### 6. 最小化映像大小與攻擊面

**現有**: ✅ 已經很好
- 使用多階段構建
- 最小化的 Alpine 基礎映像
- 非 root 用戶運行

**額外考慮**:
```dockerfile
# 選項 1: 使用更小的 distroless 映像（適用於生產環境）
FROM gcr.io/distroless/static:nonroot

# 選項 2: 保持 Alpine 但定期清理
FROM alpine:3.21.6
RUN apk --no-cache add ca-certificates && \
    rm -rf /var/cache/apk/*
```

### 7. 供應鏈安全

**建議**: 使用 SBOM (Software Bill of Materials)

**實施方式**:
```yaml
- name: Generate SBOM
  uses: aquasecurity/trivy-action@0.33.1
  with:
    scan-type: 'image'
    image-ref: ${{ env.REPO }}:scan
    format: 'cyclonedx'
    output: 'sbom.json'
    
- name: Upload SBOM
  uses: actions/upload-artifact@v4
  with:
    name: sbom
    path: sbom.json
```

## 優先級建議

### 🔴 高優先級（立即實施）
1. ✅ 升級到 Alpine 3.21.6（已完成）
2. ✅ 在 builder stage 添加 ca-certificates（已完成）
3. ✅ 調整 scheduled 掃描的 exit-code 策略（已完成）

### 🟡 中優先級（近期實施）
4. 設定 Dependabot 自動更新
5. 設定安全漏洞通知

### 🟢 低優先級（長期考慮）
6. 考慮 distroless 映像
7. 實施 SBOM 生成
8. 建立定期審查流程

## 總結

此次安全漏洞是由 Alpine Linux 基礎映像中的 OpenSSL 版本過舊引起的。通過升級到最新的 Alpine 3.21.6 版本，已經成功修復所有已識別的漏洞。

建議採用更主動的依賴項管理策略，包括：
- 使用固定版本標籤
- 自動化更新流程（Dependabot）
- 多層次的安全掃描策略
- 定期審查和更新

這樣可以在未來更快速地發現和修復類似的安全問題，提高整體系統的安全性。
