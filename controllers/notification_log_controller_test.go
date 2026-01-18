package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"member_API/testutil"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestGetNotificationLogs(t *testing.T) {
	logID1 := uuid.New()
	logID2 := uuid.New()
	memberID := uuid.New()
	providerID := uuid.New()
	now := time.Now()

	tests := []struct {
		name           string
		queryParams    string
		setupMock      func(sqlmock.Sqlmock)
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:        "成功查詢記錄列表",
			queryParams: "",
			setupMock: func(mock sqlmock.Sqlmock) {
				// Count query
				mock.ExpectQuery(`SELECT count\(\*\) FROM "notification_logs"`).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

				// Data query
				rows := sqlmock.NewRows([]string{"id", "member_id", "provider_id", "type", "status", "creation_time"}).
					AddRow(logID1, memberID, providerID, "SMTP", "SENT", now).
					AddRow(logID2, memberID, providerID, "SMS", "FAILED", now)

				mock.ExpectQuery(`SELECT \* FROM "notification_logs" ORDER BY creation_time DESC LIMIT \$1`).
					WithArgs(20).
					WillReturnRows(rows)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Contains(t, resp, "data")
				assert.Contains(t, resp, "pagination")
				data := resp["data"].([]interface{})
				assert.Len(t, data, 2)
			},
		},
		{
			name:        "按狀態篩選",
			queryParams: "?status=SENT",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT count\(\*\) FROM "notification_logs" WHERE status = \$1`).
					WithArgs("SENT").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

				rows := sqlmock.NewRows([]string{"id", "member_id", "provider_id", "type", "status", "creation_time"}).
					AddRow(logID1, memberID, providerID, "SMTP", "SENT", now)

				mock.ExpectQuery(`SELECT \* FROM "notification_logs" WHERE status = \$1 ORDER BY creation_time DESC LIMIT \$2`).
					WithArgs("SENT", 20).
					WillReturnRows(rows)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Contains(t, resp, "data")
				data := resp["data"].([]interface{})
				assert.Len(t, data, 1)
			},
		},
		{
			name:        "按類型篩選",
			queryParams: "?type=SMS",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT count\(\*\) FROM "notification_logs" WHERE type = \$1`).
					WithArgs("SMS").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

				rows := sqlmock.NewRows([]string{"id", "member_id", "provider_id", "type", "status", "creation_time"}).
					AddRow(logID2, memberID, providerID, "SMS", "FAILED", now)

				mock.ExpectQuery(`SELECT \* FROM "notification_logs" WHERE type = \$1 ORDER BY creation_time DESC LIMIT \$2`).
					WithArgs("SMS", 20).
					WillReturnRows(rows)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Contains(t, resp, "data")
			},
		},
		{
			name:        "分頁查詢",
			queryParams: "?page=2&page_size=10",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT count\(\*\) FROM "notification_logs"`).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(25))

				rows := sqlmock.NewRows([]string{"id", "member_id", "provider_id", "type", "status", "creation_time"})
				mock.ExpectQuery(`SELECT \* FROM "notification_logs" ORDER BY creation_time DESC LIMIT \$1 OFFSET \$2`).
					WithArgs(10, 10).
					WillReturnRows(rows)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				pagination := resp["pagination"].(map[string]interface{})
				assert.Equal(t, float64(2), pagination["page"])
				assert.Equal(t, float64(10), pagination["page_size"])
			},
		},
		{
			name:        "數據庫錯誤",
			queryParams: "",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT count\(\*\) FROM "notification_logs"`).
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
			router.GET("/logs", GetNotificationLogs)

			req := httptest.NewRequest(http.MethodGet, "/logs"+tt.queryParams, nil)
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

func TestGetNotificationLogsWithoutDB(t *testing.T) {
	SetupUserController(nil)

	router := setupTestRouter()
	router.GET("/logs", GetNotificationLogs)

	req := httptest.NewRequest(http.MethodGet, "/logs", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response, "error")
	assert.Equal(t, "database connection not configured", response["error"])
}

func TestGetNotificationLogByID(t *testing.T) {
	logID := uuid.New()
	memberID := uuid.New()
	providerID := uuid.New()
	now := time.Now()

	tests := []struct {
		name           string
		logID          string
		setupMock      func(sqlmock.Sqlmock)
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:  "成功獲取單一記錄",
			logID: logID.String(),
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "member_id", "provider_id", "type", "status", "creation_time"}).
					AddRow(logID, memberID, providerID, "SMTP", "SENT", now)

				mock.ExpectQuery(`SELECT \* FROM "notification_logs" WHERE id = \$1 ORDER BY "notification_logs"."id" LIMIT \$2`).
					WithArgs(logID, 1).
					WillReturnRows(rows)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Contains(t, resp, "data")
				data := resp["data"].(map[string]interface{})
				assert.Equal(t, logID.String(), data["id"])
			},
		},
		{
			name:  "記錄不存在",
			logID: uuid.New().String(),
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT \* FROM "notification_logs" WHERE id = \$1`).
					WillReturnError(fmt.Errorf("record not found"))
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Contains(t, resp, "error")
				assert.Equal(t, "log not found", resp["error"])
			},
		},
		{
			name:           "無效的記錄ID",
			logID:          "invalid-uuid",
			setupMock:      func(mock sqlmock.Sqlmock) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Contains(t, resp, "error")
				assert.Equal(t, "invalid log id", resp["error"])
			},
		},
		{
			name:  "數據庫錯誤",
			logID: logID.String(),
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT \* FROM "notification_logs" WHERE id = \$1`).
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
			router.GET("/logs/:id", GetNotificationLogByID)

			req := httptest.NewRequest(http.MethodGet, "/logs/"+tt.logID, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			_ = json.Unmarshal(w.Body.Bytes(), &response)
			tt.checkResponse(t, response)
		})
	}
}

func TestGetNotificationLogByIDWithoutDB(t *testing.T) {
	SetupUserController(nil)

	router := setupTestRouter()
	router.GET("/logs/:id", GetNotificationLogByID)

	req := httptest.NewRequest(http.MethodGet, "/logs/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response, "error")
	assert.Equal(t, "database connection not configured", response["error"])
}

func TestGetNotificationLogStats(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		setupMock      func(sqlmock.Sqlmock)
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:        "成功獲取統計資訊",
			queryParams: "",
			setupMock: func(mock sqlmock.Sqlmock) {
				// Total count
				mock.ExpectQuery(`SELECT count\(\*\) FROM "notification_logs"`).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(100))

				// Status stats
				statusRows := sqlmock.NewRows([]string{"status", "count"}).
					AddRow("SENT", 70).
					AddRow("FAILED", 20).
					AddRow("PENDING", 10)
				mock.ExpectQuery(`SELECT status, COUNT\(\*\) as count FROM "notification_logs"`).
					WillReturnRows(statusRows)

				// Type stats
				typeRows := sqlmock.NewRows([]string{"type", "count"}).
					AddRow("SMTP", 50).
					AddRow("SMS", 30).
					AddRow("FIREBASE", 20)
				mock.ExpectQuery(`SELECT type, COUNT\(\*\) as count FROM "notification_logs"`).
					WillReturnRows(typeRows)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Contains(t, resp, "total")
				assert.Contains(t, resp, "success_rate")
				assert.Contains(t, resp, "failure_rate")
				assert.Contains(t, resp, "status_stats")
				assert.Contains(t, resp, "channel_stats")

				assert.Equal(t, float64(100), resp["total"])
				assert.Equal(t, 70.0, resp["success_rate"])
				assert.Equal(t, 20.0, resp["failure_rate"])
			},
		},
		{
			name:        "空資料統計",
			queryParams: "",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT count\(\*\) FROM "notification_logs"`).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

				statusRows := sqlmock.NewRows([]string{"status", "count"})
				mock.ExpectQuery(`SELECT status, COUNT\(\*\) as count FROM "notification_logs"`).
					WillReturnRows(statusRows)

				typeRows := sqlmock.NewRows([]string{"type", "count"})
				mock.ExpectQuery(`SELECT type, COUNT\(\*\) as count FROM "notification_logs"`).
					WillReturnRows(typeRows)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				assert.Equal(t, float64(0), resp["total"])
				assert.Equal(t, 0.0, resp["success_rate"])
				assert.Equal(t, 0.0, resp["failure_rate"])
			},
		},
		{
			name:        "數據庫錯誤 - 計數失敗",
			queryParams: "",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT count\(\*\) FROM "notification_logs"`).
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
			router.GET("/logs/stats", GetNotificationLogStats)

			req := httptest.NewRequest(http.MethodGet, "/logs/stats"+tt.queryParams, nil)
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

func TestGetNotificationLogStatsWithoutDB(t *testing.T) {
	SetupUserController(nil)

	router := setupTestRouter()
	router.GET("/logs/stats", GetNotificationLogStats)

	req := httptest.NewRequest(http.MethodGet, "/logs/stats", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response, "error")
	assert.Equal(t, "database connection not configured", response["error"])
}
