package controllers

import (
	"bytes"
	"encoding/json"
	"member_API/models"
	"member_API/services"
	"member_API/testutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSendSMS_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, mock := testutil.SetupTestDB(t)
	service := services.NewSMSService(db, &services.MockSMSProvider{})
	SetupSMSController(service)

	memberID := uuid.New()
	providerID := uuid.New()

	smsConfig := models.SMSConfig{
		Provider:  "mock",
		AccountID: "test-account",
		AuthToken: "test-token",
		FromPhone: "+886900000000",
	}
	configJSON, _ := json.Marshal(smsConfig)

	providerRows := sqlmock.NewRows([]string{
		"id", "tenants_id", "name", "type", "config", "is_active",
	}).AddRow(
		providerID, uuid.New(), "Test SMS Provider", models.ProviderSMS, string(configJSON), true,
	)

	mock.ExpectQuery(`SELECT \* FROM "notification_providers"`).
		WithArgs(providerID, 1).
		WillReturnRows(providerRows)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "notification_logs"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "notification_logs"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	router := gin.New()
	router.POST("/sms", func(c *gin.Context) {
		c.Set("member_id", memberID)
		SendSMS(c)
	})

	reqBody := SendSMSRequest{
		RecipientPhone: "+886912345678",
		Body:           "測試簡訊",
		ProviderID:     providerID.String(),
	}
	jsonData, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/sms", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response SendSMSResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
	assert.Equal(t, "簡訊發送成功", response.Message)
	assert.NotNil(t, response.Log)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestSendSMS_MissingRecipientPhone(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, _ := testutil.SetupTestDB(t)
	service := services.NewSMSService(db, &services.MockSMSProvider{})
	SetupSMSController(service)

	memberID := uuid.New()

	router := gin.New()
	router.POST("/sms", func(c *gin.Context) {
		c.Set("member_id", memberID)
		SendSMS(c)
	})

	reqBody := SendSMSRequest{
		Body:       "測試簡訊",
		ProviderID: uuid.New().String(),
	}
	jsonData, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/sms", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response["success"].(bool))
	assert.Contains(t, response["error"].(string), "參數錯誤")
}

func TestSendSMS_MissingBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, _ := testutil.SetupTestDB(t)
	service := services.NewSMSService(db, &services.MockSMSProvider{})
	SetupSMSController(service)

	memberID := uuid.New()

	router := gin.New()
	router.POST("/sms", func(c *gin.Context) {
		c.Set("member_id", memberID)
		SendSMS(c)
	})

	reqBody := SendSMSRequest{
		RecipientPhone: "+886912345678",
		ProviderID:     uuid.New().String(),
	}
	jsonData, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/sms", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response["success"].(bool))
	assert.Contains(t, response["error"].(string), "參數錯誤")
}

func TestSendSMS_InvalidProviderID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, _ := testutil.SetupTestDB(t)
	service := services.NewSMSService(db, &services.MockSMSProvider{})
	SetupSMSController(service)

	memberID := uuid.New()

	router := gin.New()
	router.POST("/sms", func(c *gin.Context) {
		c.Set("member_id", memberID)
		SendSMS(c)
	})

	reqBody := SendSMSRequest{
		RecipientPhone: "+886912345678",
		Body:           "測試簡訊",
		ProviderID:     "invalid-uuid",
	}
	jsonData, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/sms", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response["success"].(bool))
	assert.Contains(t, response["error"].(string), "Provider ID 格式錯誤")
}

func TestSendSMS_NoMemberID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, _ := testutil.SetupTestDB(t)
	service := services.NewSMSService(db, &services.MockSMSProvider{})
	SetupSMSController(service)

	router := gin.New()
	router.POST("/sms", SendSMS)

	reqBody := SendSMSRequest{
		RecipientPhone: "+886912345678",
		Body:           "測試簡訊",
		ProviderID:     uuid.New().String(),
	}
	jsonData, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/sms", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response["success"].(bool))
	assert.Contains(t, response["error"].(string), "無法取得會員資訊")
}

func TestSendSMS_InvalidPhoneNumber(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, _ := testutil.SetupTestDB(t)
	service := services.NewSMSService(db, &services.MockSMSProvider{})
	SetupSMSController(service)

	memberID := uuid.New()
	providerID := uuid.New()

	router := gin.New()
	router.POST("/sms", func(c *gin.Context) {
		c.Set("member_id", memberID)
		SendSMS(c)
	})

	reqBody := SendSMSRequest{
		RecipientPhone: "invalid-phone",
		Body:           "測試簡訊",
		ProviderID:     providerID.String(),
	}
	jsonData, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/sms", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response["success"].(bool))
	assert.Contains(t, response["error"].(string), "電話號碼格式無效")
}

func TestSendSMS_ServiceNotInitialized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	SetupSMSController(nil)

	memberID := uuid.New()

	router := gin.New()
	router.POST("/sms", func(c *gin.Context) {
		c.Set("member_id", memberID)
		SendSMS(c)
	})

	reqBody := SendSMSRequest{
		RecipientPhone: "+886912345678",
		Body:           "測試簡訊",
		ProviderID:     uuid.New().String(),
	}
	jsonData, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest(http.MethodPost, "/sms", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response["success"].(bool))
	assert.Contains(t, response["error"].(string), "SMS 服務未初始化")
}
