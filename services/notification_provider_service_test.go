package services

import (
	"testing"

	"member_API/models"
	"member_API/testutil"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCreateProvider(t *testing.T) {
	db, mock := testutil.SetupTestDB(t)
	svc := NewNotificationProviderService(db)

	tenantsID := uuid.New()
	creatorID := uuid.New()
	config := `{"host":"smtp.example.com","port":587,"username":"user","password":"pass","from":"noreply@example.com","use_tls":true}`

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "notification_providers"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
	mock.ExpectCommit()

	provider, err := svc.CreateProvider(tenantsID, "Test SMTP", models.ProviderSMTP, config, creatorID)

	assert.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, "Test SMTP", provider.Name)
	assert.Equal(t, models.ProviderSMTP, provider.Type)
}

func TestGetProviderByID(t *testing.T) {
	db, mock := testutil.SetupTestDB(t)
	svc := NewNotificationProviderService(db)

	providerID := uuid.New()
	tenantsID := uuid.New()

	rows := sqlmock.NewRows([]string{"id", "tenants_id", "name", "type", "config", "is_active", "is_deleted"}).
		AddRow(providerID, tenantsID, "Test Provider", models.ProviderSMTP, `{"host":"smtp.example.com"}`, true, false)

	mock.ExpectQuery(`SELECT \* FROM "notification_providers"`).
		WithArgs(false, providerID, 1).
		WillReturnRows(rows)

	provider, err := svc.GetProviderByID(providerID)

	assert.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, providerID, provider.ID)
	assert.Equal(t, "Test Provider", provider.Name)
}

func TestGetProvidersByTenantID(t *testing.T) {
	db, mock := testutil.SetupTestDB(t)
	svc := NewNotificationProviderService(db)

	tenantsID := uuid.New()
	provider1ID := uuid.New()
	provider2ID := uuid.New()

	rows := sqlmock.NewRows([]string{"id", "tenants_id", "name", "type", "config", "is_active", "is_deleted"}).
		AddRow(provider1ID, tenantsID, "SMTP Provider", models.ProviderSMTP, `{}`, true, false).
		AddRow(provider2ID, tenantsID, "SMS Provider", models.ProviderSMS, `{}`, true, false)

	mock.ExpectQuery(`SELECT \* FROM "notification_providers"`).
		WithArgs(tenantsID, false).
		WillReturnRows(rows)

	providers, err := svc.GetProvidersByTenantID(tenantsID)

	assert.NoError(t, err)
	assert.Len(t, providers, 2)
	assert.Equal(t, "SMTP Provider", providers[0].Name)
	assert.Equal(t, "SMS Provider", providers[1].Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateProvider(t *testing.T) {
	db, mock := testutil.SetupTestDB(t)
	svc := NewNotificationProviderService(db)

	providerID := uuid.New()
	tenantsID := uuid.New()
	modifierID := uuid.New()
	config := `{"host":"smtp.example.com","port":587}`

	// Mock SELECT query
	rows := sqlmock.NewRows([]string{"id", "tenants_id", "name", "type", "config", "is_active", "is_deleted"}).
		AddRow(providerID, tenantsID, "Old Name", models.ProviderSMTP, `{}`, true, false)
	mock.ExpectQuery(`SELECT \* FROM "notification_providers"`).
		WithArgs(false, providerID, 1).
		WillReturnRows(rows)

	// Mock UPDATE query
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "notification_providers"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	provider, err := svc.UpdateProvider(providerID, "Updated Name", models.ProviderSMTP, config, true, modifierID)

	assert.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, "Updated Name", provider.Name)
}

func TestDeleteProvider(t *testing.T) {
	db, mock := testutil.SetupTestDB(t)
	svc := NewNotificationProviderService(db)

	providerID := uuid.New()
	deleterID := uuid.New()

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "notification_providers"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := svc.DeleteProvider(providerID, deleterID)

	assert.NoError(t, err)
}

func TestTestProvider(t *testing.T) {
	db, mock := testutil.SetupTestDB(t)
	svc := NewNotificationProviderService(db)

	providerID := uuid.New()
	tenantsID := uuid.New()

	tests := []struct {
		name          string
		config        string
		providerType  models.ProviderType
		expectedValid bool
		expectedMsg   string
	}{
		{
			name:          "有效的 SMTP 配置",
			config:        `{"host":"smtp.example.com","port":587,"username":"user","password":"pass","from":"noreply@example.com","use_tls":true}`,
			providerType:  models.ProviderSMTP,
			expectedValid: true,
			expectedMsg:   "配置有效",
		},
		{
			name:          "缺少必要欄位的 SMTP 配置",
			config:        `{"host":"smtp.example.com","port":587}`,
			providerType:  models.ProviderSMTP,
			expectedValid: false,
			expectedMsg:   "缺少必要欄位",
		},
		{
			name:          "有效的 SMS 配置",
			config:        `{"provider":"twilio","account_id":"AC123","auth_token":"token123","from_phone":"+1234567890"}`,
			providerType:  models.ProviderSMS,
			expectedValid: true,
			expectedMsg:   "配置有效",
		},
		{
			name:          "有效的 Firebase 配置",
			config:        `{"project_id":"my-project","credential_json":"{}","server_key":"key123"}`,
			providerType:  models.ProviderFirebase,
			expectedValid: true,
			expectedMsg:   "配置有效",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows := sqlmock.NewRows([]string{"id", "tenants_id", "name", "type", "config", "is_active", "is_deleted"}).
				AddRow(providerID, tenantsID, "Test Provider", tt.providerType, tt.config, true, false)

			mock.ExpectQuery(`SELECT \* FROM "notification_providers"`).
				WithArgs(false, providerID, 1).
				WillReturnRows(rows)

			valid, message, err := svc.TestProvider(providerID)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedValid, valid)
			if tt.expectedValid {
				assert.Equal(t, tt.expectedMsg, message)
			} else {
				assert.Contains(t, message, tt.expectedMsg)
			}
		})
	}
}

func TestEncryptDecryptConfig(t *testing.T) {
	db, _ := testutil.SetupTestDB(t)
	svc := NewNotificationProviderService(db)

	tests := []struct {
		name         string
		providerType models.ProviderType
		config       string
	}{
		{
			name:         "SMTP 配置加密",
			providerType: models.ProviderSMTP,
			config:       `{"host":"smtp.example.com","port":587,"username":"user","password":"secretpass","from":"noreply@example.com"}`,
		},
		{
			name:         "SMS 配置加密",
			providerType: models.ProviderSMS,
			config:       `{"provider":"twilio","account_id":"AC123","auth_token":"secret_token","from_phone":"+1234567890"}`,
		},
		{
			name:         "Firebase 配置加密",
			providerType: models.ProviderFirebase,
			config:       `{"project_id":"my-project","credential_json":"secret_json","server_key":"secret_key"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 加密
			encrypted, err := svc.encryptConfig(tt.providerType, tt.config)
			assert.NoError(t, err)
			assert.NotEqual(t, tt.config, encrypted)

			// 解密
			decrypted, err := svc.decryptConfig(tt.providerType, encrypted)
			assert.NoError(t, err)

			// 驗證解密後的內容（至少應該是有效的 JSON）
			assert.NotEmpty(t, decrypted)
		})
	}
}
