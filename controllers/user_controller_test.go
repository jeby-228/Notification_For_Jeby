package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"member_API/testutil"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestGetUsers(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(sqlmock.Sqlmock)
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name: "成功取得用戶列表",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "name", "email"}).
					AddRow(uuid.New(), "User 1", "user1@example.com").
					AddRow(uuid.New(), "User 2", "user2@example.com")

				mock.ExpectQuery(`SELECT (.+) FROM "members"`).
					WillReturnRows(rows)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Contains(t, resp, "users")
				users := resp["users"].([]interface{})
				assert.Len(t, users, 2)
			},
		},
		{
			name: "返回空列表",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "name", "email"})
				mock.ExpectQuery(`SELECT (.+) FROM "members"`).
					WillReturnRows(rows)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Contains(t, resp, "users")
				users := resp["users"].([]interface{})
				assert.Len(t, users, 0)
			},
		},
		{
			name: "數據庫錯誤",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT (.+) FROM "members"`).
					WillReturnError(gorm.ErrInvalidDB)
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Contains(t, resp, "error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gormDB, mock := testutil.SetupTestDB(t)
			SetupUserController(gormDB)
			defer SetupUserController(nil)

			tt.setupMock(mock)

			router := setupTestRouter()
			router.GET("/users", GetUsers)

			req := httptest.NewRequest(http.MethodGet, "/users", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err == nil {
				tt.checkResponse(t, response)
			}
		})
	}
}

func TestGetUsersWithoutDB(t *testing.T) {
	SetupUserController(nil)

	router := setupTestRouter()
	router.GET("/users", GetUsers)

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response, "users")
	assert.Contains(t, response, "message")
	assert.Equal(t, "database connection not configured", response["message"])
}

func TestGetUserByID(t *testing.T) {
	memberID := uuid.New()

	tests := []struct {
		name           string
		userID         string
		setupMock      func(sqlmock.Sqlmock)
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:   "成功取得單一用戶",
			userID: "1",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "name", "email"}).
					AddRow(memberID, "Test User", "test@example.com")

				mock.ExpectQuery(`SELECT (.+) FROM "members"`).
					WithArgs(uint64(1), 1).
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
			name:   "用戶不存在",
			userID: "999",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT (.+) FROM "members"`).
					WithArgs(uint64(999), 1).
					WillReturnError(gorm.ErrRecordNotFound)
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Contains(t, resp, "error")
				assert.Equal(t, "user not found", resp["error"])
			},
		},
		{
			name:           "無效的用戶ID",
			userID:         "invalid",
			setupMock:      func(mock sqlmock.Sqlmock) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Contains(t, resp, "error")
				assert.Equal(t, "invalid user id", resp["error"])
			},
		},
		{
			name:   "數據庫錯誤",
			userID: "1",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT (.+) FROM "members"`).
					WithArgs(uint64(1), 1).
					WillReturnError(gorm.ErrInvalidDB)
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Contains(t, resp, "error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gormDB, mock := testutil.SetupTestDB(t)
			SetupUserController(gormDB)
			defer SetupUserController(nil)

			tt.setupMock(mock)

			router := setupTestRouter()
			router.GET("/user/:id", GetUserByID)

			req := httptest.NewRequest(http.MethodGet, "/user/"+tt.userID, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			_ = json.Unmarshal(w.Body.Bytes(), &response)
			tt.checkResponse(t, response)
		})
	}
}

func TestGetUserByIDWithoutDB(t *testing.T) {
	SetupUserController(nil)

	router := setupTestRouter()
	router.GET("/user/:id", GetUserByID)

	req := httptest.NewRequest(http.MethodGet, "/user/1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response, "user")
	assert.Contains(t, response, "message")
	assert.Equal(t, "database connection not configured", response["message"])
}

func TestDeleteUserByID(t *testing.T) {
	tests := []struct {
		name           string
		userID         string
		setupMock      func(sqlmock.Sqlmock)
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:   "成功刪除用戶",
			userID: "1",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`DELETE FROM "members"`).
					WithArgs(uint64(1)).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Contains(t, resp, "message")
				assert.Equal(t, "user deleted successfully", resp["message"])
			},
		},
		{
			name:           "無效的用戶ID",
			userID:         "invalid",
			setupMock:      func(mock sqlmock.Sqlmock) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Contains(t, resp, "error")
				assert.Equal(t, "invalid user id", resp["error"])
			},
		},
		{
			name:   "數據庫錯誤",
			userID: "1",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`DELETE FROM "members"`).
					WithArgs(uint64(1)).
					WillReturnError(gorm.ErrInvalidDB)
				mock.ExpectRollback()
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Contains(t, resp, "error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gormDB, mock := testutil.SetupTestDB(t)
			SetupUserController(gormDB)
			defer SetupUserController(nil)

			tt.setupMock(mock)

			router := setupTestRouter()
			router.DELETE("/user/:id", DeleteUserByID)

			req := httptest.NewRequest(http.MethodDelete, "/user/"+tt.userID, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			_ = json.Unmarshal(w.Body.Bytes(), &response)
			tt.checkResponse(t, response)
		})
	}
}

func TestDeleteUserByIDWithoutDB(t *testing.T) {
	SetupUserController(nil)

	router := setupTestRouter()
	router.DELETE("/user/:id", DeleteUserByID)

	req := httptest.NewRequest(http.MethodDelete, "/user/1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response, "error")
	assert.Equal(t, "database connection not configured", response["error"])
}
