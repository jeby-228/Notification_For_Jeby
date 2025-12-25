# Member API

一個使用 Go、Gin 框架和 PostgreSQL 構建的 RESTful 和 GraphQL API 服務，提供會員管理功能和 JWT 認證。

## 功能特性

- ✅ RESTful API 端點
- ✅ GraphQL API 支援
- ✅ JWT 認證機制
- ✅ 用戶註冊與登入
- ✅ 密碼加密（bcrypt）
- ✅ PostgreSQL 資料庫
- ✅ Swagger API 文件
- ✅ Docker 支援
- ✅ 自動更新部署（Watchtower）

## 技術棧

- **語言**: Go 1.24+
- **框架**: Gin Web Framework
- **資料庫**: PostgreSQL
- **ORM**: GORM
- **認證**: JWT (golang-jwt/jwt)
- **密碼加密**: bcrypt
- **API 文件**: Swagger/OpenAPI
- **GraphQL**: gqlgen
- **容器化**: Docker & Docker Compose
- **自動更新**: Watchtower

## 快速開始

### 方式一：使用 Docker Compose（推薦）

這是最簡單的部署方式，包含自動更新功能。

1. 確保已安裝 Docker 和 Docker Compose
2. 編輯 `deploy/docker-compose.yml` 設定資料庫連線和 JWT 密鑰
3. 啟動服務：

```bash
docker-compose -f deploy/docker-compose.yml up -d
```

4. 查看日誌：

```bash
docker-compose -f deploy/docker-compose.yml logs -f member_api
```

5. 停止服務：

```bash
docker-compose -f deploy/docker-compose.yml down
```

**自動更新機制**：

- Watchtower 每 5 分鐘檢查一次新版本
- 發現新版本時自動拉取並重啟容器
- 自動清理舊映像檔
- 無需手動干預

### 方式二：本地開發

#### 環境需求

- Go 1.24 或更高版本
- PostgreSQL 12 或更高版本
- Git

#### 環境變數設定

複製環境變數範例檔案並根據需要修改：

```bash
cp .env.example .env
```

編輯 `.env` 檔案設定以下變數：

- `POSTGRES_DSN`: PostgreSQL 資料庫連線字串
- `JWT_SECRET`: JWT 簽章密鑰（**生產環境務必更改**）

生成強隨機密鑰的方法：

```bash
# 使用 openssl 生成 32 位元組的 base64 編碼密鑰
openssl rand -base64 32
```

#### 安裝相依套件

```bash
go mod download
```

#### 生成 Swagger 文件

```bash
# 安裝 swag（如果尚未安裝）
go install github.com/swaggo/swag/cmd/swag@latest

# 生成文件
swag init
```

#### 啟動服務

```bash
go run main.go
```

服務將在 `http://localhost:9876` 啟動。

## Docker 部署

### 從 GitHub Container Registry 抓取映像檔

```bash
# 抓取最新版本
docker pull ghcr.io/jebylinjbjob/member_api:latest

# 或抓取特定版本
docker pull ghcr.io/jebylinjbjob/member_api:v1.0.0
```

### 本地建構映像檔

```bash
docker build -t member-api .
```

### 執行容器

```bash
docker run -d \
  -p 9876:9876 \
  -e POSTGRES_DSN="postgres://postgres:postgres@host.docker.internal:5432/member_api?sslmode=disable" \
  -e JWT_SECRET="$(openssl rand -base64 32)" \
  --name member-api \
  ghcr.io/jebylinjbjob/member_api:latest
```

## API 文件

### Swagger UI

啟動服務後，訪問 Swagger UI：

```
http://localhost:9876/swagger/index.html
```

### GraphQL Playground

訪問 GraphQL Playground：

```
http://localhost:9876/graphql
```

### 健康檢查

```bash
curl http://localhost:9876/health
```

## API 端點

### 公開端點（無需認證）

- `POST /api/v1/register` - 用戶註冊
- `POST /api/v1/login` - 用戶登入
- `GET /health` - 健康檢查

### 受保護端點（需要 JWT 認證）

- `GET /api/v1/users` - 獲取所有會員
- `GET /api/v1/user/:id` - 獲取單個會員
- `GET /api/v1/profile` - 獲取當前用戶資訊

### 使用 JWT 認證

1. 先登入獲取 token：

```bash
curl -X POST http://localhost:9876/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123"}'
```

2. 使用 token 訪問受保護的端點：

```bash
curl -X GET http://localhost:9876/api/v1/profile \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

## 專案結構

```
member_API/
├── auth/              # JWT 認證與密碼處理
│   ├── jwt.go
│   ├── middleware.go
│   └── password.go
├── controllers/       # 控制器層
│   ├── auth_controller.go
│   └── user_controller.go
├── deploy/            # 部署配置
│   └── docker-compose.yml
├── docs/              # API 文件
│   ├── docs.go
│   ├── swagger.json
│   └── swagger.yaml
├── graphql/           # GraphQL schema 和 resolver
│   ├── generated.go
│   ├── resolver.go
│   ├── schema.graphql
│   ├── schema.resolvers.go
│   ├── setup.go
│   └── model/
│       └── models_gen.go
├── models/            # 資料模型
│   └── member.go
├── routes/            # 路由設定
│   └── routes.go
├── Dockerfile         # Docker 建構檔案
├── go.mod             # Go 模組定義
├── gqlgen.yml         # GraphQL 程式碼產生器配置
├── main.go            # 應用程式入口
└── .env.example       # 環境變數範例
```

## 開發指南

### 執行測試

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

### 更新 Swagger 文件

當 API 端點有變更時，重新生成 Swagger 文件：

```bash
swag init
```

### GraphQL Schema 變更

編輯 `graphql/schema.graphql` 後，執行以下命令重新生成程式碼：

```bash
go run github.com/99designs/gqlgen generate
```

## 環境變數說明

| 變數名稱       | 說明                      | 範例                                                             |
| -------------- | ------------------------- | ---------------------------------------------------------------- |
| `POSTGRES_DSN` | PostgreSQL 資料庫連線字串 | `postgres://user:password@localhost:5432/dbname?sslmode=disable` |
| `JWT_SECRET`   | JWT 簽章密鑰              | `your-secret-key`                                                |

## 生產環境建議

1. **更改預設密鑰**：務必使用強隨機字串作為 `JWT_SECRET`
2. **啟用 SSL**：在資料庫連線字串中將 `sslmode=disable` 改為 `sslmode=require`
3. **設定防火牆**：僅開放必要的連接埠
4. **定期備份**：設定資料庫自動備份機制
5. **監控日誌**：使用日誌聚合工具監控應用程式狀態
6. **限流**：在生產環境中加入 API 限流機制

## 疑難排解

### 資料庫連線失敗

檢查 `POSTGRES_DSN` 是否正確，確保：

- 資料庫伺服器正在執行
- 連線字串中的使用者名稱、密碼、主機、連接埠、資料庫名稱都正確
- 網路連線正常

### Docker 容器無法啟動

```bash
# 查看容器日誌
docker logs member_api

# 檢查容器狀態
docker ps -a
```

### Watchtower 未自動更新

```bash
# 查看 Watchtower 日誌
docker logs member_api_watchtower

# 手動觸發更新
docker-compose -f deploy/docker-compose.yml pull
docker-compose -f deploy/docker-compose.yml up -d
```
