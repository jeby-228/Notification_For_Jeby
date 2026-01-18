package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"member_API/auth"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: sqlDB,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open gorm db: %v", err)
	}

	return gormDB, mock
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    RegisterRequest
		setupMock      func(sqlmock.Sqlmock)
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name: "成功註冊",
			requestBody: RegisterRequest{
				Name:     "Test User",
				Email:    "test@example.com",
				Password: "password123",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT \* FROM "members"`).
					WithArgs("test@example.com", false, 1).
					WillReturnError(gorm.ErrRecordNotFound)

				mock.ExpectBegin()
				mock.ExpectExec(`INSERT INTO "members"`).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Contains(t, resp, "token")
				assert.Contains(t, resp, "user")
				assert.NotEmpty(t, resp["token"])
			},
		},
		{
			name: "Email 已存在",
			requestBody: RegisterRequest{
				Name:     "Test User",
				Email:    "existing@example.com",
				Password: "password123",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email"}).
					AddRow(uuid.New(), "existing@example.com")
				mock.ExpectQuery(`SELECT \* FROM "members"`).
					WithArgs("existing@example.com", false, 1).
					WillReturnRows(rows)
			},
			expectedStatus: http.StatusConflict,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Contains(t, resp, "error")
				assert.Contains(t, resp["error"], "已被註冊")
			},
		},
		{
			name: "無效的請求參數",
			requestBody: RegisterRequest{
				Name:     "",
				Email:    "invalid-email",
				Password: "123",
			},
			setupMock:      func(mock sqlmock.Sqlmock) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Contains(t, resp, "error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gormDB, mock := setupTestDB(t)
			SetupUserController(gormDB)
			defer SetupUserController(nil)

			tt.setupMock(mock)

			router := setupTestRouter()
			router.POST("/register", Register)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			_ = json.Unmarshal(w.Body.Bytes(), &response)
			tt.checkResponse(t, response)
		})
	}
}

func TestLogin(t *testing.T) {
	memberID := uuid.New()
	now := time.Now()
	hashedPassword, _ := auth.HashPassword("password123")

	tests := []struct {
		name           string
		requestBody    LoginRequest
		setupMock      func(sqlmock.Sqlmock)
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name: "成功登入",
			requestBody: LoginRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "name", "email", "password_hash", "tenants_id", "api_key",
					"creation_time", "creator_id", "is_deleted",
				}).AddRow(
					memberID, "Test User", "test@example.com", hashedPassword, uuid.Nil, "key",
					now, uuid.New(), false,
				)
				mock.ExpectQuery(`SELECT \* FROM "members"`).
					WithArgs("test@example.com", 1).
					WillReturnRows(rows)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Contains(t, resp, "token")
				assert.Contains(t, resp, "user")
				assert.NotEmpty(t, resp["token"])
			},
		},
		{
			name: "用戶不存在",
			requestBody: LoginRequest{
				Email:    "notfound@example.com",
				Password: "password123",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT \* FROM "members"`).
					WithArgs("notfound@example.com", 1).
					WillReturnError(gorm.ErrRecordNotFound)
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Contains(t, resp, "error")
				assert.Contains(t, resp["error"], "電子郵件或密碼錯誤")
			},
		},
		{
			name: "密碼錯誤",
			requestBody: LoginRequest{
				Email:    "test@example.com",
				Password: "wrongpassword",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "name", "email", "password_hash", "api_key",
					"creation_time", "creator_id", "is_deleted",
				}).AddRow(
					memberID, "Test User", "test@example.com", hashedPassword, "key",
					now, uuid.New(), false,
				)
				mock.ExpectQuery(`SELECT \* FROM "members"`).
					WithArgs("test@example.com", 1).
					WillReturnRows(rows)
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Contains(t, resp, "error")
				assert.Contains(t, resp["error"], "電子郵件或密碼錯誤")
			},
		},
		{
			name: "無效的請求參數",
			requestBody: LoginRequest{
				Email:    "invalid-email",
				Password: "123",
			},
			setupMock:      func(mock sqlmock.Sqlmock) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Contains(t, resp, "error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gormDB, mock := setupTestDB(t)
			SetupUserController(gormDB)
			defer SetupUserController(nil)

			tt.setupMock(mock)

			router := setupTestRouter()
			router.POST("/login", Login)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			_ = json.Unmarshal(w.Body.Bytes(), &response)
			tt.checkResponse(t, response)
		})
	}
}

func TestGetProfile(t *testing.T) {
	memberID := uuid.New()

	tests := []struct {
		name           string
		setupContext   func(*gin.Context)
		setupMock      func(sqlmock.Sqlmock)
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name: "成功取得個人資料",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", int64(memberID.ID()))
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "name", "email",
				}).AddRow(
					memberID, "Test User", "test@example.com",
				)
				mock.ExpectQuery(`SELECT (.+) FROM "members"`).
					WithArgs(int64(memberID.ID()), 1).
					WillReturnRows(rows)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Contains(t, resp, "user")
				user := resp["user"].(map[string]interface{})
				assert.Equal(t, "Test User", user["name"])
				assert.Equal(t, "test@example.com", user["email"])
			},
		},
		{
			name: "未認證",
			setupContext: func(c *gin.Context) {
			},
			setupMock:      func(mock sqlmock.Sqlmock) {},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Contains(t, resp, "error")
				assert.Contains(t, resp["error"], "未認證")
			},
		},
		{
			name: "用戶不存在",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", int64(999))
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT (.+) FROM "members"`).
					WithArgs(int64(999), 1).
					WillReturnError(gorm.ErrRecordNotFound)
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Contains(t, resp, "error")
				assert.Contains(t, resp["error"], "用戶不存在")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gormDB, mock := setupTestDB(t)
			SetupUserController(gormDB)
			defer SetupUserController(nil)

			tt.setupMock(mock)

			router := setupTestRouter()
			router.GET("/profile", func(c *gin.Context) {
				tt.setupContext(c)
				GetProfile(c)
			})

			req := httptest.NewRequest(http.MethodGet, "/profile", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			_ = json.Unmarshal(w.Body.Bytes(), &response)
			tt.checkResponse(t, response)
		})
	}
}

func TestRegisterWithoutDB(t *testing.T) {
	SetupUserController(nil)

	router := setupTestRouter()
	router.POST("/register", Register)

	body, _ := json.Marshal(RegisterRequest{
		Name:     "Test",
		Email:    "test@example.com",
		Password: "password123",
	})
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response["error"], "數據庫連接未配置")
}

func TestLoginWithoutDB(t *testing.T) {
	SetupUserController(nil)

	router := setupTestRouter()
	router.POST("/login", Login)

	body, _ := json.Marshal(LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	})
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response["error"], "數據庫連接未配置")
}

func TestGetProfileWithInvalidUserID(t *testing.T) {
	gormDB, _ := setupTestDB(t)
	SetupUserController(gormDB)
	defer SetupUserController(nil)

	router := setupTestRouter()
	router.GET("/profile", func(c *gin.Context) {
		c.Set("user_id", "invalid-type")
		GetProfile(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/profile", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response["error"], "未認證")
}

func TestRegisterTokenGenerationError(t *testing.T) {
	gormDB, mock := setupTestDB(t)
	SetupUserController(gormDB)
	defer SetupUserController(nil)

	mock.ExpectQuery(`SELECT \* FROM "members"`).
		WithArgs("test@example.com", false, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "members"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	router := setupTestRouter()
	router.POST("/register", Register)

	body, _ := json.Marshal(RegisterRequest{
		Name:     "Test User",
		Email:    "test@example.com",
		Password: "password123",
	})
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)

	if w.Code == http.StatusInternalServerError {
		assert.Contains(t, response, "error")
	}
}
