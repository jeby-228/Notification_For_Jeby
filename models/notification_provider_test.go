package models

import (
	"encoding/json"
	"member_API/testutil"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestProviderType_Constants(t *testing.T) {
	assert.Equal(t, ProviderType("SMTP"), ProviderSMTP)
	assert.Equal(t, ProviderType("SMS"), ProviderSMS)
	assert.Equal(t, ProviderType("FIREBASE"), ProviderFirebase)
}

func TestNotificationProvider_Structure(t *testing.T) {
	tenantID := uuid.New()

	provider := NotificationProvider{
		TenantsID: tenantID,
		Name:      "Test Provider",
		Type:      ProviderSMTP,
		Config:    `{"host":"smtp.example.com"}`,
		IsActive:  true,
	}

	assert.Equal(t, tenantID, provider.TenantsID)
	assert.Equal(t, "Test Provider", provider.Name)
	assert.Equal(t, ProviderSMTP, provider.Type)
	assert.NotEmpty(t, provider.Config)
	assert.True(t, provider.IsActive)
}

func TestNotificationProvider_Create(t *testing.T) {
	t.Run("成功創建 SMTP 提供者", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		provider := &NotificationProvider{
			Base: Base{
				ID: uuid.New(),
			},
			TenantsID: uuid.New(),
			Name:      "SMTP Provider",
			Type:      ProviderSMTP,
			Config:    `{"host":"smtp.gmail.com","port":587}`,
			IsActive:  true,
		}

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO \"notification_providers\"").
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := db.Create(provider).Error
		assert.NoError(t, err)
	})

	t.Run("成功創建 SMS 提供者", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		provider := &NotificationProvider{
			Base: Base{
				ID: uuid.New(),
			},
			TenantsID: uuid.New(),
			Name:      "SMS Provider",
			Type:      ProviderSMS,
			Config:    `{"provider":"twilio"}`,
			IsActive:  true,
		}

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO \"notification_providers\"").
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := db.Create(provider).Error
		assert.NoError(t, err)
	})

	t.Run("成功創建 Firebase 提供者", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		provider := &NotificationProvider{
			Base: Base{
				ID: uuid.New(),
			},
			TenantsID: uuid.New(),
			Name:      "Firebase Provider",
			Type:      ProviderFirebase,
			Config:    `{"project_id":"my-project"}`,
			IsActive:  true,
		}

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO \"notification_providers\"").
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := db.Create(provider).Error
		assert.NoError(t, err)
	})
}

func TestNotificationProvider_Query(t *testing.T) {
	t.Run("查詢租戶的所有提供者", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		tenantID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "tenants_id", "name", "type", "is_active"}).
			AddRow(uuid.New(), tenantID, "Provider1", "SMTP", true).
			AddRow(uuid.New(), tenantID, "Provider2", "SMS", true)

		mock.ExpectQuery("SELECT (.+) FROM \"notification_providers\"").
			WithArgs(tenantID).
			WillReturnRows(rows)

		var providers []NotificationProvider
		err := db.Where("tenants_id = ?", tenantID).Find(&providers).Error

		assert.NoError(t, err)
		assert.Len(t, providers, 2)
	})

	t.Run("查詢活躍的提供者", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		rows := sqlmock.NewRows([]string{"id", "name", "is_active"}).
			AddRow(uuid.New(), "Active Provider", true)

		mock.ExpectQuery("SELECT (.+) FROM \"notification_providers\"").
			WithArgs(true).
			WillReturnRows(rows)

		var providers []NotificationProvider
		err := db.Where("is_active = ?", true).Find(&providers).Error

		assert.NoError(t, err)
		assert.Len(t, providers, 1)
		assert.True(t, providers[0].IsActive)
	})

	t.Run("查詢特定類型的提供者", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		rows := sqlmock.NewRows([]string{"id", "name", "type"}).
			AddRow(uuid.New(), "SMTP Provider", "SMTP")

		mock.ExpectQuery("SELECT (.+) FROM \"notification_providers\"").
			WithArgs("SMTP").
			WillReturnRows(rows)

		var providers []NotificationProvider
		err := db.Where("type = ?", ProviderSMTP).Find(&providers).Error

		assert.NoError(t, err)
		assert.Len(t, providers, 1)
		assert.Equal(t, ProviderSMTP, providers[0].Type)
	})
}

func TestNotificationProvider_Update(t *testing.T) {
	t.Run("停用提供者", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		providerID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectExec("UPDATE \"notification_providers\"").
			WithArgs(false, sqlmock.AnyArg(), providerID).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := db.Model(&NotificationProvider{}).
			Where("id = ?", providerID).
			Update("is_active", false).Error

		assert.NoError(t, err)
	})

	t.Run("更新配置", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		providerID := uuid.New()
		newConfig := `{"host":"new.smtp.com","port":465}`

		mock.ExpectBegin()
		mock.ExpectExec("UPDATE \"notification_providers\"").
			WithArgs(newConfig, sqlmock.AnyArg(), providerID).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := db.Model(&NotificationProvider{}).
			Where("id = ?", providerID).
			Update("config", newConfig).Error

		assert.NoError(t, err)
	})
}

func TestNotificationProvider_Delete(t *testing.T) {
	t.Run("刪除提供者", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		providerID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectExec("DELETE FROM \"notification_providers\"").
			WithArgs(providerID).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := db.Delete(&NotificationProvider{}, providerID).Error
		assert.NoError(t, err)
	})
}

func TestSMTPConfig_JSON(t *testing.T) {
	t.Run("序列化 SMTP 配置", func(t *testing.T) {
		config := SMTPConfig{
			Host:     "smtp.gmail.com",
			Port:     587,
			Username: "user@gmail.com",
			Password: "password",
			From:     "noreply@example.com",
			UseTLS:   true,
		}

		jsonData, err := json.Marshal(config)
		assert.NoError(t, err)
		assert.NotEmpty(t, jsonData)

		var decoded SMTPConfig
		err = json.Unmarshal(jsonData, &decoded)
		assert.NoError(t, err)
		assert.Equal(t, config.Host, decoded.Host)
		assert.Equal(t, config.Port, decoded.Port)
		assert.Equal(t, config.UseTLS, decoded.UseTLS)
	})

	t.Run("從 JSON 反序列化", func(t *testing.T) {
		jsonStr := `{"host":"smtp.example.com","port":465,"username":"user","password":"pass","from":"sender@example.com","use_tls":true}`

		var config SMTPConfig
		err := json.Unmarshal([]byte(jsonStr), &config)
		assert.NoError(t, err)
		assert.Equal(t, "smtp.example.com", config.Host)
		assert.Equal(t, 465, config.Port)
		assert.True(t, config.UseTLS)
	})
}

func TestSMSConfig_JSON(t *testing.T) {
	t.Run("序列化 SMS 配置", func(t *testing.T) {
		config := SMSConfig{
			Provider:  "twilio",
			AccountID: "AC123",
			AuthToken: "token123",
			FromPhone: "+1234567890",
		}

		jsonData, err := json.Marshal(config)
		assert.NoError(t, err)
		assert.NotEmpty(t, jsonData)

		var decoded SMSConfig
		err = json.Unmarshal(jsonData, &decoded)
		assert.NoError(t, err)
		assert.Equal(t, config.Provider, decoded.Provider)
		assert.Equal(t, config.AccountID, decoded.AccountID)
	})

	t.Run("從 JSON 反序列化", func(t *testing.T) {
		jsonStr := `{"provider":"nexmo","account_id":"ACC456","auth_token":"auth456","from_phone":"+9876543210"}`

		var config SMSConfig
		err := json.Unmarshal([]byte(jsonStr), &config)
		assert.NoError(t, err)
		assert.Equal(t, "nexmo", config.Provider)
		assert.Equal(t, "ACC456", config.AccountID)
	})
}

func TestFirebaseConfig_JSON(t *testing.T) {
	t.Run("序列化 Firebase 配置", func(t *testing.T) {
		config := FirebaseConfig{
			ProjectID:      "my-project",
			CredentialJSON: `{"type":"service_account"}`,
			ServerKey:      "server_key_123",
		}

		jsonData, err := json.Marshal(config)
		assert.NoError(t, err)
		assert.NotEmpty(t, jsonData)

		var decoded FirebaseConfig
		err = json.Unmarshal(jsonData, &decoded)
		assert.NoError(t, err)
		assert.Equal(t, config.ProjectID, decoded.ProjectID)
		assert.Equal(t, config.ServerKey, decoded.ServerKey)
	})

	t.Run("從 JSON 反序列化", func(t *testing.T) {
		jsonStr := `{"project_id":"firebase-project","credential_json":"{}","server_key":"key789"}`

		var config FirebaseConfig
		err := json.Unmarshal([]byte(jsonStr), &config)
		assert.NoError(t, err)
		assert.Equal(t, "firebase-project", config.ProjectID)
		assert.Equal(t, "key789", config.ServerKey)
	})
}

func TestNotificationProvider_ConfigTypes(t *testing.T) {
	t.Run("SMTP 提供者使用 SMTP 配置", func(t *testing.T) {
		smtpConfig := SMTPConfig{
			Host: "smtp.example.com",
			Port: 587,
		}
		configJSON, _ := json.Marshal(smtpConfig)

		provider := NotificationProvider{
			Type:   ProviderSMTP,
			Config: string(configJSON),
		}

		assert.Equal(t, ProviderSMTP, provider.Type)

		var decoded SMTPConfig
		err := json.Unmarshal([]byte(provider.Config), &decoded)
		assert.NoError(t, err)
		assert.Equal(t, "smtp.example.com", decoded.Host)
	})

	t.Run("SMS 提供者使用 SMS 配置", func(t *testing.T) {
		smsConfig := SMSConfig{
			Provider: "twilio",
		}
		configJSON, _ := json.Marshal(smsConfig)

		provider := NotificationProvider{
			Type:   ProviderSMS,
			Config: string(configJSON),
		}

		assert.Equal(t, ProviderSMS, provider.Type)

		var decoded SMSConfig
		err := json.Unmarshal([]byte(provider.Config), &decoded)
		assert.NoError(t, err)
		assert.Equal(t, "twilio", decoded.Provider)
	})

	t.Run("Firebase 提供者使用 Firebase 配置", func(t *testing.T) {
		firebaseConfig := FirebaseConfig{
			ProjectID: "my-firebase",
		}
		configJSON, _ := json.Marshal(firebaseConfig)

		provider := NotificationProvider{
			Type:   ProviderFirebase,
			Config: string(configJSON),
		}

		assert.Equal(t, ProviderFirebase, provider.Type)

		var decoded FirebaseConfig
		err := json.Unmarshal([]byte(provider.Config), &decoded)
		assert.NoError(t, err)
		assert.Equal(t, "my-firebase", decoded.ProjectID)
	})
}

func TestNotificationProvider_EdgeCases(t *testing.T) {
	t.Run("空配置", func(t *testing.T) {
		provider := NotificationProvider{
			Name:   "Empty Config",
			Config: "",
		}
		assert.Empty(t, provider.Config)
	})

	t.Run("無效的 JSON 配置", func(t *testing.T) {
		provider := NotificationProvider{
			Name:   "Invalid Config",
			Config: "not a json",
		}

		var config map[string]interface{}
		err := json.Unmarshal([]byte(provider.Config), &config)
		assert.Error(t, err)
	})

	t.Run("Nil 租戶 ID", func(t *testing.T) {
		provider := NotificationProvider{
			TenantsID: uuid.Nil,
			Name:      "No Tenant",
		}
		assert.Equal(t, uuid.Nil, provider.TenantsID)
	})

	t.Run("預設為非活躍", func(t *testing.T) {
		provider := NotificationProvider{
			Name: "Inactive",
		}
		assert.False(t, provider.IsActive)
	})
}

func TestNotificationProvider_MultipleProviders(t *testing.T) {
	t.Run("租戶可以有多個不同類型的提供者", func(t *testing.T) {
		tenantID := uuid.New()

		smtp := NotificationProvider{
			TenantsID: tenantID,
			Name:      "SMTP",
			Type:      ProviderSMTP,
		}

		sms := NotificationProvider{
			TenantsID: tenantID,
			Name:      "SMS",
			Type:      ProviderSMS,
		}

		firebase := NotificationProvider{
			TenantsID: tenantID,
			Name:      "Firebase",
			Type:      ProviderFirebase,
		}

		assert.Equal(t, tenantID, smtp.TenantsID)
		assert.Equal(t, tenantID, sms.TenantsID)
		assert.Equal(t, tenantID, firebase.TenantsID)
		assert.NotEqual(t, smtp.Type, sms.Type)
		assert.NotEqual(t, smtp.Type, firebase.Type)
	})
}
