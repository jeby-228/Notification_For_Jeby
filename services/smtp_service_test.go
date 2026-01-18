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
	configJSON, _ := json.Marshal(smtpConfig)

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
			name:       "Email send fails and logs to database",
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
