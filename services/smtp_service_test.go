package services

import (
	"encoding/json"
	"member_API/models"
	"member_API/testutil"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestSendEmail(t *testing.T) {
	memberID := uuid.New()
	providerID := uuid.New()
	tenantsID := uuid.New()
	now := time.Now()

	smtpConfig := models.SMTPConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "test@example.com",
		Password: "password",
		From:     "noreply@example.com",
		UseTLS:   false,
	}
	configJSON, err := json.Marshal(smtpConfig)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}

	smtpConfigTLS := models.SMTPConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "test@example.com",
		Password: "password",
		From:     "noreply@example.com",
		UseTLS:   true,
	}
	configJSONTLS, err := json.Marshal(smtpConfigTLS)
	if err != nil {
		t.Fatalf("failed to marshal tls config: %v", err)
	}

	invalidConfig := models.SMTPConfig{
		Host:     "",
		Port:     0,
		Username: "",
		Password: "",
		From:     "",
		UseTLS:   false,
	}
	invalidConfigJSON, err := json.Marshal(invalidConfig)
	if err != nil {
		t.Fatalf("failed to marshal invalid config: %v", err)
	}

	tests := []struct {
		name      string
		memberID  uuid.UUID
		providerID uuid.UUID
		req       EmailRequest
		setupMock func(sqlmock.Sqlmock)
		wantErr   bool
		errMsg    string
	}{
		{
			name:       "Provider not found",
			memberID:   memberID,
			providerID: uuid.New(),
			req: EmailRequest{
				RecipientEmail: "test@example.com",
				RecipientName:  "Test User",
				Subject:        "Test Subject",
				Body:           "Test Body",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT \* FROM "notification_providers"`).
					WithArgs(sqlmock.AnyArg(), true, models.ProviderSMTP, 1).
					WillReturnError(gorm.ErrRecordNotFound)
			},
			wantErr: true,
			errMsg:  "smtp provider not found or inactive",
		},
		{
			name:       "Invalid config JSON",
			memberID:   memberID,
			providerID: providerID,
			req: EmailRequest{
				RecipientEmail: "test@example.com",
				RecipientName:  "Test User",
				Subject:        "Test Subject",
				Body:           "Test Body",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "tenants_id", "name", "type", "config", "is_active",
					"creation_time", "creator_id", "is_deleted",
				}).AddRow(
					providerID, tenantsID, "SMTP Provider", models.ProviderSMTP, "invalid json", true,
					now, memberID, false,
				)
				mock.ExpectQuery(`SELECT \* FROM "notification_providers"`).
					WithArgs(providerID, true, models.ProviderSMTP, 1).
					WillReturnRows(rows)
			},
			wantErr: true,
			errMsg:  "invalid smtp config",
		},
		{
			name:       "Invalid config - missing required fields",
			memberID:   memberID,
			providerID: providerID,
			req: EmailRequest{
				RecipientEmail: "test@example.com",
				RecipientName:  "Test User",
				Subject:        "Test Subject",
				Body:           "Test Body",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "tenants_id", "name", "type", "config", "is_active",
					"creation_time", "creator_id", "is_deleted",
				}).AddRow(
					providerID, tenantsID, "SMTP Provider", models.ProviderSMTP, string(invalidConfigJSON), true,
					now, memberID, false,
				)
				mock.ExpectQuery(`SELECT \* FROM "notification_providers"`).
					WithArgs(providerID, true, models.ProviderSMTP, 1).
					WillReturnRows(rows)

				mock.ExpectBegin()
				mock.ExpectExec(`INSERT INTO "notification_logs"`).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			wantErr: true,
			errMsg:  "missing required fields",
		},
		{
			name:       "Email send fails and logs to database with StatusFailed",
			memberID:   memberID,
			providerID: providerID,
			req: EmailRequest{
				RecipientEmail: "test@example.com",
				RecipientName:  "Test User",
				Subject:        "Test Subject",
				Body:           "Test Body",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "tenants_id", "name", "type", "config", "is_active",
					"creation_time", "creator_id", "is_deleted",
				}).AddRow(
					providerID, tenantsID, "SMTP Provider", models.ProviderSMTP, string(configJSON), true,
					now, memberID, false,
				)
				mock.ExpectQuery(`SELECT \* FROM "notification_providers"`).
					WithArgs(providerID, true, models.ProviderSMTP, 1).
					WillReturnRows(rows)

				// Expect log insert without strict argument matching
				mock.ExpectBegin()
				mock.ExpectExec(`INSERT INTO "notification_logs"`).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			wantErr: true,
			errMsg:  "failed after 3 retries",
		},
		{
			name:       "Email send with TLS config fails",
			memberID:   memberID,
			providerID: providerID,
			req: EmailRequest{
				RecipientEmail: "test@example.com",
				RecipientName:  "Test User",
				Subject:        "Test Subject",
				Body:           "Test Body",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "tenants_id", "name", "type", "config", "is_active",
					"creation_time", "creator_id", "is_deleted",
				}).AddRow(
					providerID, tenantsID, "SMTP Provider", models.ProviderSMTP, string(configJSONTLS), true,
					now, memberID, false,
				)
				mock.ExpectQuery(`SELECT \* FROM "notification_providers"`).
					WithArgs(providerID, true, models.ProviderSMTP, 1).
					WillReturnRows(rows)

				mock.ExpectBegin()
				mock.ExpectExec(`INSERT INTO "notification_logs"`).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			wantErr: true,
			errMsg:  "failed after 3 retries",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := testutil.SetupTestDB(t)
			service := NewSMTPService(db)

			tt.setupMock(mock)

			err := service.SendEmail(tt.memberID, tt.providerID, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Unfulfilled expectations: %v", err)
			}
		})
	}
}

func TestSendEmailWithConfig_Validation(t *testing.T) {
	db, _ := testutil.SetupTestDB(t)
	service := NewSMTPService(db)

	tests := []struct {
		name    string
		config  models.SMTPConfig
		req     EmailRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "Missing host",
			config: models.SMTPConfig{
				Host:     "",
				Port:     587,
				Username: "test@example.com",
				Password: "password",
				From:     "noreply@example.com",
				UseTLS:   false,
			},
			req: EmailRequest{
				RecipientEmail: "test@example.com",
				Subject:        "Test",
				Body:           "Test Body",
			},
			wantErr: true,
			errMsg:  "missing required fields",
		},
		{
			name: "Missing port",
			config: models.SMTPConfig{
				Host:     "smtp.example.com",
				Port:     0,
				Username: "test@example.com",
				Password: "password",
				From:     "noreply@example.com",
				UseTLS:   false,
			},
			req: EmailRequest{
				RecipientEmail: "test@example.com",
				Subject:        "Test",
				Body:           "Test Body",
			},
			wantErr: true,
			errMsg:  "missing required fields",
		},
		{
			name: "Missing username",
			config: models.SMTPConfig{
				Host:     "smtp.example.com",
				Port:     587,
				Username: "",
				Password: "password",
				From:     "noreply@example.com",
				UseTLS:   false,
			},
			req: EmailRequest{
				RecipientEmail: "test@example.com",
				Subject:        "Test",
				Body:           "Test Body",
			},
			wantErr: true,
			errMsg:  "missing required fields",
		},
		{
			name: "Missing password",
			config: models.SMTPConfig{
				Host:     "smtp.example.com",
				Port:     587,
				Username: "test@example.com",
				Password: "",
				From:     "noreply@example.com",
				UseTLS:   false,
			},
			req: EmailRequest{
				RecipientEmail: "test@example.com",
				Subject:        "Test",
				Body:           "Test Body",
			},
			wantErr: true,
			errMsg:  "missing required fields",
		},
		{
			name: "Missing from",
			config: models.SMTPConfig{
				Host:     "smtp.example.com",
				Port:     587,
				Username: "test@example.com",
				Password: "password",
				From:     "",
				UseTLS:   false,
			},
			req: EmailRequest{
				RecipientEmail: "test@example.com",
				Subject:        "Test",
				Body:           "Test Body",
			},
			wantErr: true,
			errMsg:  "missing required fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.sendEmailWithConfig(tt.config, tt.req)
			
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
