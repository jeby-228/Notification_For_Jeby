# Member API

一個使用 Go、Gin 框架和 PostgreSQL 構建的 RESTful 和 GraphQL API 服務，提供會員管理功能和 JWT 認證。

## 功能特性

- ✅ RESTful API 端點
- ✅ GraphQL API 支援
- ✅ JWT 認證機制
- ✅ 用戶註冊與登入
- ✅ 密碼加密（bcrypt）
- ✅ PostgreSQL 資料庫
- ✅ Swagger API 文檔
- ✅ Docker 支援

## 技術棧

- **語言**: Go 1.24+
- **框架**: Gin Web Framework
- **資料庫**: PostgreSQL
- **ORM**: GORM
- **認證**: JWT (golang-jwt/jwt)
- **密碼加密**: bcrypt
- **API 文檔**: Swagger/OpenAPI
- **GraphQL**: graphql-go/graphql

## 快速開始

### 1. 環境需求

- Go 1.24 或更高版本
- PostgreSQL 12 或更高版本
- Git

### 2. 克隆專案

```bash
git clone https://github.com/jebylinjbjob/member_API.git
cd member_API
```

### 3. 環境變數設定

複製環境變數範例文件並根據需要修改：

```bash
cp .env.example .env
```

編輯 `.env` 文件設置以下變數：

- `POSTGRES_DSN`: PostgreSQL 資料庫連線字串
- `JWT_SECRET`: JWT 簽章密鑰（**生產環境務必更改**）

生成強隨機密鑰的方法：

```bash
# 使用 openssl 生成 32 字節的 base64 編碼密鑰
openssl rand -base64 32
```

### 4. 載入環境變數

```bash
# Linux/macOS
export $(cat .env | xargs)

# 或使用 source（如果使用 bash/zsh）
set -a
source .env
set +a
```

### 5. 安裝依賴

```bash
go mod download
```

### 6. 資料庫設置

確保 PostgreSQL 服務正在運行，然後創建資料庫：

```bash
# 連接到 PostgreSQL
psql -U postgres

# 創建資料庫
CREATE DATABASE member_api;

# 退出
\q
```

執行資料庫遷移：

```bash
psql -U postgres -d member_api -f database/migration.sql
```

### 7. 生成 Swagger 文檔

```bash
# 安裝 swag（如果尚未安裝）
go install github.com/swaggo/swag/cmd/swag@latest

# 生成文檔
swag init
```

### 8. 啟動服務

```bash
go run main.go
```

服務將在 `http://localhost:8080` 啟動。

## 使用 Docker


在本地建構 Docker 映像：
```bash
docker build -t member-api .
```

### 從 GitHub Container Registry 抓取映像 (推薦)

如果要使用已發布的映像：
```bash
# 先登入 GitHub Container Registry
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin

# 抓取最新版本
docker pull ghcr.io/jebylinjbjob/member_api:latest

# 或抓取特定版本
docker pull ghcr.io/jebylinjbjob/member_api:v1.0.0
```

### 運行容器

```bash
docker run -d \
  -p 8080:8080 \
  -e POSTGRES_DSN="postgres://postgres:postgres@host.docker.internal:5432/member_api?sslmode=disable" \
  -e JWT_SECRET="$(openssl rand -base64 32)" \
  --name member-api \
  member-api
```

## API 文檔

### Swagger UI

啟動服務後，訪問 Swagger UI：

```
http://localhost:8080/swagger/index.html
```

### GraphQL Playground

訪問 GraphQL Playground：

```
http://localhost:8080/graphql
```

### 健康檢查

```bash
curl http://localhost:8080/health
```

## API 端點

### 公開端點（無需認證）

- `POST /api/v1/register` - 用戶註冊
- `POST /api/v1/login` - 用戶登入
- `GET /health` - 健康檢查

### 受保護端點（需要 JWT 認證）

- `GET /api/v1/users` - 獲取所有會員
- `GET /api/v1/user/:id` - 獲取單個會員
- `GET /api/v1/profile` - 獲取當前用戶信息

## 認證使用範例

### 註冊新用戶

```bash
curl -X POST http://localhost:8080/api/v1/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "張三",
    "email": "zhangsan@example.com",
    "password": "password123"
  }'
```

### 用戶登入

```bash
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "zhangsan@example.com",
    "password": "password123"
  }'
```

### 訪問受保護的 API

```bash
curl -X GET http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

更多詳細資訊請參閱 [認證文檔](docs/AUTHENTICATION.md)。

## 專案結構

```
member_API/
├── auth/              # JWT 認證與密碼處理
├── controllers/       # 控制器層
├── database/          # 資料庫遷移腳本
├── docs/              # API 文檔
├── graphql/           # GraphQL schema 和 resolver
├── migrations/        # 資料庫版本遷移
├── models/            # 資料模型
├── routes/            # 路由設定
├── main.go            # 應用程式入口
├── Dockerfile         # Docker 構建文件
└── .env.example       # 環境變數範例
```

## 開發指南

### 運行測試

```bash
go test ./...
```

### 程式碼格式化

```bash
go fmt ./...
```

### 程式碼檢查

```bash
go vet ./...
```
