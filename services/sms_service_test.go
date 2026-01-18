package services

import (
	"encoding/json"
	"errors"
	"member_API/models"
	"member_API/testutil"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

type MockFailingSMSProvider struct {
	FailCount int
	Calls     int
}

func (m *MockFailingSMSProvider) SendSMS(fromPhone, toPhone, body string) error {
	m.Calls++
	if m.Calls <= m.FailCount {
		return errors.New("mock SMS provider error")
	}
	return nil
}

func TestValidatePhoneNumber(t *testing.T) {
	tests := []struct {
		name    string
		phone   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "有效的國際電話號碼",
			phone:   "+886912345678",
			wantErr: false,
		},
		{
			name:    "有效的美國電話號碼",
			phone:   "+15551234567",
			wantErr: false,
		},
		{
			name:    "有效的本地電話號碼",
			phone:   "0912345678",
			wantErr: false,
		},
		{
			name:    "空電話號碼",
			phone:   "",
			wantErr: true,
			errMsg:  "電話號碼不能為空",
		},
		{
			name:    "無效格式 - 包含字母",
			phone:   "+886912abc678",
			wantErr: true,
			errMsg:  "電話號碼格式無效",
		},
		{
			name:    "無效格式 - 太短",
			phone:   "+1",
			wantErr: true,
			errMsg:  "電話號碼格式無效",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePhoneNumber(tt.phone)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateSMSLength(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "有效的簡訊內容",
			body:    "這是一則測試簡訊",
			wantErr: false,
		},
		{
			name:    "空內容",
			body:    "",
			wantErr: true,
			errMsg:  "簡訊內容不能為空",
		},
		{
			name:    "超過字數限制",
			body:    string(make([]byte, 161)),
			wantErr: true,
			errMsg:  "簡訊內容超過 160 字元限制",
		},
		{
			name:    "剛好160字元",
			body:    string(make([]byte, 160)),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSMSLength(tt.body)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSendSMS_Success(t *testing.T) {
	db, mock := testutil.SetupTestDB(t)
	service := NewSMSService(db, &MockSMSProvider{})

	memberID := uuid.New()
	providerID := uuid.New()
	recipientPhone := "+886912345678"
	body := "測試簡訊"

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

	log, err := service.SendSMS(memberID, providerID, recipientPhone, body)

	assert.NoError(t, err)
	assert.NotNil(t, log)
	assert.Equal(t, models.StatusSent, log.Status)
	assert.Equal(t, recipientPhone, log.RecipientPhone)
	assert.Equal(t, body, log.Body)
	assert.NotNil(t, log.SentAt)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestSendSMS_InvalidPhone(t *testing.T) {
	db, _ := testutil.SetupTestDB(t)
	service := NewSMSService(db, &MockSMSProvider{})

	memberID := uuid.New()
	providerID := uuid.New()
	invalidPhone := "invalid-phone"
	body := "測試簡訊"

	log, err := service.SendSMS(memberID, providerID, invalidPhone, body)

	assert.Error(t, err)
	assert.Nil(t, log)
	assert.Contains(t, err.Error(), "電話號碼格式無效")
}

func TestSendSMS_BodyTooLong(t *testing.T) {
	db, _ := testutil.SetupTestDB(t)
	service := NewSMSService(db, &MockSMSProvider{})

	memberID := uuid.New()
	providerID := uuid.New()
	recipientPhone := "+886912345678"
	longBody := string(make([]byte, 161))

	log, err := service.SendSMS(memberID, providerID, recipientPhone, longBody)

	assert.Error(t, err)
	assert.Nil(t, log)
	assert.Contains(t, err.Error(), "簡訊內容超過 160 字元限制")
}

func TestSendSMS_ProviderNotFound(t *testing.T) {
	db, mock := testutil.SetupTestDB(t)
	service := NewSMSService(db, &MockSMSProvider{})

	memberID := uuid.New()
	providerID := uuid.New()
	recipientPhone := "+886912345678"
	body := "測試簡訊"

	mock.ExpectQuery(`SELECT \* FROM "notification_providers"`).
		WithArgs(providerID, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	log, err := service.SendSMS(memberID, providerID, recipientPhone, body)

	assert.Error(t, err)
	assert.Nil(t, log)
	assert.Contains(t, err.Error(), "找不到通知提供者")

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestSendSMS_ProviderNotActive(t *testing.T) {
	db, mock := testutil.SetupTestDB(t)
	service := NewSMSService(db, &MockSMSProvider{})

	memberID := uuid.New()
	providerID := uuid.New()
	recipientPhone := "+886912345678"
	body := "測試簡訊"

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
		providerID, uuid.New(), "Test SMS Provider", models.ProviderSMS, string(configJSON), false,
	)

	mock.ExpectQuery(`SELECT \* FROM "notification_providers"`).
		WithArgs(providerID, 1).
		WillReturnRows(providerRows)

	log, err := service.SendSMS(memberID, providerID, recipientPhone, body)

	assert.Error(t, err)
	assert.Nil(t, log)
	assert.Contains(t, err.Error(), "SMS 提供者未啟用")

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestSendSMS_WrongProviderType(t *testing.T) {
	db, mock := testutil.SetupTestDB(t)
	service := NewSMSService(db, &MockSMSProvider{})

	memberID := uuid.New()
	providerID := uuid.New()
	recipientPhone := "+886912345678"
	body := "測試簡訊"

	smtpConfig := models.SMTPConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "user",
		Password: "pass",
		From:     "test@example.com",
		UseTLS:   true,
	}
	configJSON, _ := json.Marshal(smtpConfig)

	providerRows := sqlmock.NewRows([]string{
		"id", "tenants_id", "name", "type", "config", "is_active",
	}).AddRow(
		providerID, uuid.New(), "Test SMTP Provider", models.ProviderSMTP, string(configJSON), true,
	)

	mock.ExpectQuery(`SELECT \* FROM "notification_providers"`).
		WithArgs(providerID, 1).
		WillReturnRows(providerRows)

	log, err := service.SendSMS(memberID, providerID, recipientPhone, body)

	assert.Error(t, err)
	assert.Nil(t, log)
	assert.Contains(t, err.Error(), "提供者類型不是 SMS")

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestSendSMS_RetryMechanism(t *testing.T) {
	db, mock := testutil.SetupTestDB(t)
	failingProvider := &MockFailingSMSProvider{FailCount: 2}
	service := NewSMSService(db, failingProvider)

	memberID := uuid.New()
	providerID := uuid.New()
	recipientPhone := "+886912345678"
	body := "測試簡訊"

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

	log, err := service.SendSMS(memberID, providerID, recipientPhone, body)

	assert.NoError(t, err)
	assert.NotNil(t, log)
	assert.Equal(t, models.StatusSent, log.Status)
	assert.Equal(t, 3, failingProvider.Calls)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestSendSMS_AllRetriesFail(t *testing.T) {
	db, mock := testutil.SetupTestDB(t)
	failingProvider := &MockFailingSMSProvider{FailCount: 99}
	service := NewSMSService(db, failingProvider)

	memberID := uuid.New()
	providerID := uuid.New()
	recipientPhone := "+886912345678"
	body := "測試簡訊"

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

	log, err := service.SendSMS(memberID, providerID, recipientPhone, body)

	assert.Error(t, err)
	assert.Nil(t, log)
	assert.Contains(t, err.Error(), "發送簡訊失敗，已重試 3 次")
	assert.Equal(t, 3, failingProvider.Calls)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

func TestNewSMSService_WithNilProvider(t *testing.T) {
	db, _ := testutil.SetupTestDB(t)
	service := NewSMSService(db, nil)

	assert.NotNil(t, service)
	assert.NotNil(t, service.Provider)
	_, ok := service.Provider.(*MockSMSProvider)
	assert.True(t, ok, "應該使用 MockSMSProvider 作為預設提供者")
}

func TestMockSMSProvider_SendSMS(t *testing.T) {
	provider := &MockSMSProvider{}
	err := provider.SendSMS("+886900000000", "+886912345678", "測試簡訊")
	assert.NoError(t, err)
}
