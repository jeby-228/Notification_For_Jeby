package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"member_API/models"
	"member_API/testutil"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	return router
}

func TestCreateProvider(t *testing.T) {
	mockDB, mock := testutil.SetupTestDB(t)
	db = mockDB
	defer func() { db = nil }()

	router := setupTestRouter()
	router.POST("/providers", CreateProvider)

	tenantsID := uuid.New()
	userID := uuid.New()
	config := `{"host":"smtp.example.com","port":587,"username":"user","password":"pass","from":"noreply@example.com"}`

	reqBody := CreateProviderRequest{
		TenantsID: tenantsID,
		Name:      "Test SMTP",
		Type:      models.ProviderSMTP,
		Config:    config,
	}

	body, _ := json.Marshal(reqBody)

	// Mock for getting member
	memberRows := sqlmock.NewRows([]string{"id"}).AddRow(userID)
	mock.ExpectQuery(`SELECT \* FROM "members"`).WillReturnRows(memberRows)

	// Mock for creating provider
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "notification_providers"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
	mock.ExpectCommit()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/providers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// 模擬認證中間件設置的 user_id
	router.Use(func(c *gin.Context) {
		c.Set("user_id", int64(1))
		c.Next()
	})
	router.POST("/providers-test", CreateProvider)

	req, _ = http.NewRequest("POST", "/providers-test", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestGetProviders(t *testing.T) {
	mockDB, mock := testutil.SetupTestDB(t)
	db = mockDB
	defer func() { db = nil }()

	router := setupTestRouter()
	router.GET("/providers", GetProviders)

	tenantsID := uuid.New()
	provider1ID := uuid.New()
	provider2ID := uuid.New()

	rows := sqlmock.NewRows([]string{"id", "tenants_id", "name", "type", "config", "is_active", "is_deleted"}).
		AddRow(provider1ID, tenantsID, "SMTP Provider", models.ProviderSMTP, `{}`, true, false).
		AddRow(provider2ID, tenantsID, "SMS Provider", models.ProviderSMS, `{}`, true, false)

	mock.ExpectQuery(`SELECT \* FROM "notification_providers"`).
		WithArgs(tenantsID, false).
		WillReturnRows(rows)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/providers?tenants_id="+tenantsID.String(), nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response, "providers")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetProvider(t *testing.T) {
	mockDB, mock := testutil.SetupTestDB(t)
	db = mockDB
	defer func() { db = nil }()

	router := setupTestRouter()
	router.GET("/providers/:id", GetProvider)

	providerID := uuid.New()
	tenantsID := uuid.New()

	rows := sqlmock.NewRows([]string{"id", "tenants_id", "name", "type", "config", "is_active", "is_deleted"}).
		AddRow(providerID, tenantsID, "Test Provider", models.ProviderSMTP, `{}`, true, false)

	mock.ExpectQuery(`SELECT \* FROM "notification_providers"`).
		WithArgs(false, providerID).
		WillReturnRows(rows)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/providers/"+providerID.String(), nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var provider models.NotificationProvider
	json.Unmarshal(w.Body.Bytes(), &provider)
	assert.Equal(t, providerID, provider.ID)
	assert.Equal(t, "Test Provider", provider.Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateProvider(t *testing.T) {
	mockDB, mock := testutil.SetupTestDB(t)
	db = mockDB
	defer func() { db = nil }()

	router := setupTestRouter()

	providerID := uuid.New()
	tenantsID := uuid.New()
	userID := uuid.New()
	config := `{"host":"smtp.example.com","port":587}`

	reqBody := UpdateProviderRequest{
		Name:     "Updated Provider",
		Type:     models.ProviderSMTP,
		Config:   config,
		IsActive: true,
	}

	body, _ := json.Marshal(reqBody)

	// Mock for getting member
	memberRows := sqlmock.NewRows([]string{"id"}).AddRow(userID)
	mock.ExpectQuery(`SELECT \* FROM "members"`).WillReturnRows(memberRows)

	// Mock for getting provider
	providerRows := sqlmock.NewRows([]string{"id", "tenants_id", "name", "type", "config", "is_active", "is_deleted"}).
		AddRow(providerID, tenantsID, "Old Name", models.ProviderSMTP, `{}`, true, false)
	mock.ExpectQuery(`SELECT \* FROM "notification_providers"`).
		WithArgs(false, providerID).
		WillReturnRows(providerRows)

	// Mock for updating provider
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "notification_providers"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	router.Use(func(c *gin.Context) {
		c.Set("user_id", int64(1))
		c.Next()
	})
	router.PUT("/providers/:id", UpdateProvider)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/providers/"+providerID.String(), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDeleteProvider(t *testing.T) {
	mockDB, mock := testutil.SetupTestDB(t)
	db = mockDB
	defer func() { db = nil }()

	router := setupTestRouter()

	providerID := uuid.New()
	userID := uuid.New()

	// Mock for getting member
	memberRows := sqlmock.NewRows([]string{"id"}).AddRow(userID)
	mock.ExpectQuery(`SELECT \* FROM "members"`).WillReturnRows(memberRows)

	// Mock for deleting provider
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "notification_providers"`).
		WithArgs(true, sqlmock.AnyArg(), userID, sqlmock.AnyArg(), providerID, false).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	router.Use(func(c *gin.Context) {
		c.Set("user_id", int64(1))
		c.Next()
	})
	router.DELETE("/providers/:id", DeleteProvider)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/providers/"+providerID.String(), nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "刪除成功", response["message"])
}

func TestTestProvider(t *testing.T) {
	mockDB, mock := testutil.SetupTestDB(t)
	db = mockDB
	defer func() { db = nil }()

	router := setupTestRouter()
	router.GET("/providers/:id/test", TestProvider)

	providerID := uuid.New()
	tenantsID := uuid.New()
	config := `{"host":"smtp.example.com","port":587,"username":"user","password":"pass","from":"noreply@example.com","use_tls":true}`

	rows := sqlmock.NewRows([]string{"id", "tenants_id", "name", "type", "config", "is_active", "is_deleted"}).
		AddRow(providerID, tenantsID, "Test Provider", models.ProviderSMTP, config, true, false)

	mock.ExpectQuery(`SELECT \* FROM "notification_providers"`).
		WithArgs(false, providerID).
		WillReturnRows(rows)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/providers/"+providerID.String()+"/test", nil)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response, "valid")
	assert.Contains(t, response, "message")
	assert.NoError(t, mock.ExpectationsWereMet())
}
