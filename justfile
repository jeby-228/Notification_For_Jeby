# 會員 API 專案的 Justfile
default:
    @just --list

# 本地運行
run:
    go run main.go

# 編譯
build:
    go build

# 測試
test:
    go test -v ./...

# 格式化
fmt:
    go fmt ./...

# 整理依賴
tidy:
    go mod tidy

# 測試覆蓋率
test-coverage:
    go test -cover ./...

# 測試覆蓋率產生報告
test-cov:
    go test -v -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html

# 清理
clean:
    rm -rf bin/ coverage.out coverage.html

# 靜態代碼分析
lint:
    $(go env GOPATH)/bin/golangci-lint run

# 一鍵檢查
auto_check:
    just tidy
    just fmt
    just lint
    just build
    just test
    just test-cov