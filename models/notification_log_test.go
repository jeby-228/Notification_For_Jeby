package models

import (
	"member_API/testutil"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNotificationStatus_Constants(t *testing.T) {
	assert.Equal(t, NotificationStatus("PENDING"), StatusPending)
	assert.Equal(t, NotificationStatus("SENT"), StatusSent)
	assert.Equal(t, NotificationStatus("FAILED"), StatusFailed)
}

func TestNotificationLog_Structure(t *testing.T) {
	memberID := uuid.New()
	providerID := uuid.New()
	sentAt := time.Now()

	log := NotificationLog{
		MemberID:       memberID,
		ProviderID:     providerID,
		Type:           ProviderSMTP,
		RecipientName:  "John Doe",
		RecipientEmail: "john@example.com",
		RecipientPhone: "+1234567890",
		Subject:        "Test Subject",
		Body:           "Test Body",
		Status:         StatusPending,
		ErrorMsg:       "",
		SentAt:         &sentAt,
	}

	assert.Equal(t, memberID, log.MemberID)
	assert.Equal(t, providerID, log.ProviderID)
	assert.Equal(t, ProviderSMTP, log.Type)
	assert.Equal(t, "John Doe", log.RecipientName)
	assert.Equal(t, "john@example.com", log.RecipientEmail)
	assert.Equal(t, StatusPending, log.Status)
	assert.NotNil(t, log.SentAt)
}

func TestNotificationLog_Create(t *testing.T) {
	t.Run("成功創建通知記錄", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		log := &NotificationLog{
			Base: Base{
				ID: uuid.New(),
			},
			MemberID:       uuid.New(),
			ProviderID:     uuid.New(),
			Type:           ProviderSMTP,
			RecipientEmail: "test@example.com",
			Subject:        "Test",
			Body:           "Test body",
			Status:         StatusPending,
		}

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO \"notification_logs\"").
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := db.Create(log).Error
		assert.NoError(t, err)
	})
}

func TestNotificationLog_Query(t *testing.T) {
	t.Run("查詢會員的所有通知記錄", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		memberID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "member_id", "provider_id", "type", "status", "body"}).
			AddRow(uuid.New(), memberID, uuid.New(), "SMTP", "SENT", "body1").
			AddRow(uuid.New(), memberID, uuid.New(), "SMS", "PENDING", "body2")

		mock.ExpectQuery("SELECT (.+) FROM \"notification_logs\"").
			WithArgs(memberID).
			WillReturnRows(rows)

		var logs []NotificationLog
		err := db.Where("member_id = ?", memberID).Find(&logs).Error

		assert.NoError(t, err)
		assert.Len(t, logs, 2)
	})

	t.Run("查詢特定狀態的通知", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		rows := sqlmock.NewRows([]string{"id", "member_id", "status", "body"}).
			AddRow(uuid.New(), uuid.New(), "PENDING", "body")

		mock.ExpectQuery("SELECT (.+) FROM \"notification_logs\"").
			WithArgs("PENDING").
			WillReturnRows(rows)

		var logs []NotificationLog
		err := db.Where("status = ?", StatusPending).Find(&logs).Error

		assert.NoError(t, err)
		assert.Len(t, logs, 1)
		assert.Equal(t, StatusPending, logs[0].Status)
	})

	t.Run("查詢特定類型的通知", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		rows := sqlmock.NewRows([]string{"id", "member_id", "type", "status", "body"}).
			AddRow(uuid.New(), uuid.New(), "SMTP", "SENT", "body")

		mock.ExpectQuery("SELECT (.+) FROM \"notification_logs\"").
			WithArgs("SMTP").
			WillReturnRows(rows)

		var logs []NotificationLog
		err := db.Where("type = ?", ProviderSMTP).Find(&logs).Error

		assert.NoError(t, err)
		assert.Len(t, logs, 1)
		assert.Equal(t, ProviderSMTP, logs[0].Type)
	})

	t.Run("查詢失敗的通知", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		rows := sqlmock.NewRows([]string{"id", "status", "error_msg", "body"}).
			AddRow(uuid.New(), "FAILED", "Connection timeout", "body")

		mock.ExpectQuery("SELECT (.+) FROM \"notification_logs\"").
			WithArgs("FAILED").
			WillReturnRows(rows)

		var logs []NotificationLog
		err := db.Where("status = ?", StatusFailed).Find(&logs).Error

		assert.NoError(t, err)
		assert.Len(t, logs, 1)
		assert.Equal(t, StatusFailed, logs[0].Status)
		assert.NotEmpty(t, logs[0].ErrorMsg)
	})
}

func TestNotificationLog_Update(t *testing.T) {
	t.Run("更新通知狀態為已發送", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		logID := uuid.New()
		sentAt := time.Now()

		mock.ExpectBegin()
		mock.ExpectExec("UPDATE \"notification_logs\"").
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), logID).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := db.Model(&NotificationLog{}).
			Where("id = ?", logID).
			Updates(map[string]interface{}{
				"status":  StatusSent,
				"sent_at": sentAt,
			}).Error

		assert.NoError(t, err)
	})

	t.Run("更新通知狀態為失敗", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		logID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectExec("UPDATE \"notification_logs\"").
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), logID).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := db.Model(&NotificationLog{}).
			Where("id = ?", logID).
			Updates(map[string]interface{}{
				"status":    StatusFailed,
				"error_msg": "SMTP connection failed",
			}).Error

		assert.NoError(t, err)
	})
}

func TestNotificationLog_Types(t *testing.T) {
	t.Run("SMTP 通知", func(t *testing.T) {
		log := NotificationLog{
			Type:           ProviderSMTP,
			RecipientEmail: "test@example.com",
			Subject:        "Email Subject",
			Body:           "Email body",
		}

		assert.Equal(t, ProviderSMTP, log.Type)
		assert.NotEmpty(t, log.RecipientEmail)
		assert.NotEmpty(t, log.Subject)
	})

	t.Run("SMS 通知", func(t *testing.T) {
		log := NotificationLog{
			Type:           ProviderSMS,
			RecipientPhone: "+1234567890",
			Body:           "SMS body",
		}

		assert.Equal(t, ProviderSMS, log.Type)
		assert.NotEmpty(t, log.RecipientPhone)
	})

	t.Run("Firebase 通知", func(t *testing.T) {
		log := NotificationLog{
			Type:    ProviderFirebase,
			Subject: "Push notification title",
			Body:    "Push notification body",
		}

		assert.Equal(t, ProviderFirebase, log.Type)
		assert.NotEmpty(t, log.Subject)
	})
}

func TestNotificationLog_StatusTransitions(t *testing.T) {
	t.Run("從 PENDING 到 SENT", func(t *testing.T) {
		log := NotificationLog{
			Status: StatusPending,
		}
		assert.Equal(t, StatusPending, log.Status)

		log.Status = StatusSent
		sentAt := time.Now()
		log.SentAt = &sentAt

		assert.Equal(t, StatusSent, log.Status)
		assert.NotNil(t, log.SentAt)
	})

	t.Run("從 PENDING 到 FAILED", func(t *testing.T) {
		log := NotificationLog{
			Status: StatusPending,
		}

		log.Status = StatusFailed
		log.ErrorMsg = "Failed to send"

		assert.Equal(t, StatusFailed, log.Status)
		assert.NotEmpty(t, log.ErrorMsg)
	})
}

func TestNotificationLog_EdgeCases(t *testing.T) {
	t.Run("空收件人名稱", func(t *testing.T) {
		log := NotificationLog{
			RecipientName:  "",
			RecipientEmail: "test@example.com",
		}
		assert.Empty(t, log.RecipientName)
		assert.NotEmpty(t, log.RecipientEmail)
	})

	t.Run("長主題", func(t *testing.T) {
		longSubject := string(make([]byte, 500))
		log := NotificationLog{
			Subject: longSubject,
		}
		assert.Len(t, log.Subject, 500)
	})

	t.Run("長內容", func(t *testing.T) {
		longBody := string(make([]byte, 10000))
		log := NotificationLog{
			Body: longBody,
		}
		assert.Len(t, log.Body, 10000)
	})

	t.Run("Nil 發送時間", func(t *testing.T) {
		log := NotificationLog{
			Status: StatusPending,
			SentAt: nil,
		}
		assert.Nil(t, log.SentAt)
	})

	t.Run("空錯誤訊息", func(t *testing.T) {
		log := NotificationLog{
			Status:   StatusSent,
			ErrorMsg: "",
		}
		assert.Empty(t, log.ErrorMsg)
	})
}

func TestNotificationLog_Recipient(t *testing.T) {
	t.Run("完整的收件人資訊", func(t *testing.T) {
		log := NotificationLog{
			RecipientName:  "John Doe",
			RecipientEmail: "john@example.com",
			RecipientPhone: "+1234567890",
		}

		assert.Equal(t, "John Doe", log.RecipientName)
		assert.Equal(t, "john@example.com", log.RecipientEmail)
		assert.Equal(t, "+1234567890", log.RecipientPhone)
	})

	t.Run("只有 Email", func(t *testing.T) {
		log := NotificationLog{
			Type:           ProviderSMTP,
			RecipientEmail: "email@example.com",
		}

		assert.NotEmpty(t, log.RecipientEmail)
		assert.Empty(t, log.RecipientPhone)
	})

	t.Run("只有電話", func(t *testing.T) {
		log := NotificationLog{
			Type:           ProviderSMS,
			RecipientPhone: "+1234567890",
		}

		assert.NotEmpty(t, log.RecipientPhone)
		assert.Empty(t, log.RecipientEmail)
	})
}

func TestNotificationLog_ErrorHandling(t *testing.T) {
	t.Run("記錄錯誤訊息", func(t *testing.T) {
		errorMessages := []string{
			"Connection timeout",
			"Invalid recipient",
			"Authentication failed",
			"Rate limit exceeded",
		}

		for _, errMsg := range errorMessages {
			log := NotificationLog{
				Status:   StatusFailed,
				ErrorMsg: errMsg,
			}
			assert.Equal(t, StatusFailed, log.Status)
			assert.Equal(t, errMsg, log.ErrorMsg)
		}
	})
}

func TestNotificationLog_TimeTracking(t *testing.T) {
	t.Run("記錄發送時間", func(t *testing.T) {
		before := time.Now()
		time.Sleep(10 * time.Millisecond)

		sentAt := time.Now()
		log := NotificationLog{
			Status: StatusSent,
			SentAt: &sentAt,
		}

		assert.NotNil(t, log.SentAt)
		assert.True(t, log.SentAt.After(before))
	})

	t.Run("未發送無發送時間", func(t *testing.T) {
		log := NotificationLog{
			Status: StatusPending,
		}

		assert.Nil(t, log.SentAt)
	})
}

func TestNotificationLog_Indexing(t *testing.T) {
	t.Run("會員 ID 索引查詢效率", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		memberID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "member_id", "body"}).
			AddRow(uuid.New(), memberID, "body")

		mock.ExpectQuery("SELECT (.+) FROM \"notification_logs\"").
			WithArgs(memberID).
			WillReturnRows(rows)

		var logs []NotificationLog
		err := db.Where("member_id = ?", memberID).Find(&logs).Error

		assert.NoError(t, err)
	})

	t.Run("狀態索引查詢效率", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		rows := sqlmock.NewRows([]string{"id", "status", "body"}).
			AddRow(uuid.New(), "FAILED", "body")

		mock.ExpectQuery("SELECT (.+) FROM \"notification_logs\"").
			WithArgs("FAILED").
			WillReturnRows(rows)

		var logs []NotificationLog
		err := db.Where("status = ?", StatusFailed).Find(&logs).Error

		assert.NoError(t, err)
	})
}
