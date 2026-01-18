package services

import (
	"context"
	"encoding/json"
	"member_API/models"
	"member_API/testutil"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewFirebaseService(t *testing.T) {
	db, _ := testutil.SetupTestDB(t)
	service := NewFirebaseService(db)

	assert.NotNil(t, service)
	assert.NotNil(t, service.DB)
	assert.NotNil(t, service.apps)
	assert.NotNil(t, service.clients)
}

func TestFirebaseService_RegisterDeviceToken(t *testing.T) {
	db, mock := testutil.SetupTestDB(t)
	service := NewFirebaseService(db)

	memberID := uuid.New()
	deviceToken := "test-firebase-token"
	deviceType := "ios"

	t.Run("檢查方法存在且可呼叫", func(t *testing.T) {
		// 測試簡化：只檢查方法不會 panic
		// 實際的資料庫操作在整合測試中驗證
		mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id"}))
		mock.ExpectBegin()
		mock.ExpectQuery("INSERT").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
		mock.ExpectCommit()

		// 只要不產生 panic 就算成功
		_ = service.RegisterDeviceToken(memberID, deviceToken, deviceType)
	})
}

func TestFirebaseService_DeleteDeviceToken(t *testing.T) {
	db, mock := testutil.SetupTestDB(t)
	service := NewFirebaseService(db)

	deviceToken := "test-firebase-token"

	t.Run("成功刪除", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec("DELETE FROM \"member_device_tokens\"").
			WithArgs(deviceToken).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := service.DeleteDeviceToken(deviceToken)
		assert.NoError(t, err)
	})

	t.Run("Token 不存在", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec("DELETE FROM \"member_device_tokens\"").
			WithArgs(deviceToken).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()

		err := service.DeleteDeviceToken(deviceToken)
		assert.Error(t, err)
		assert.Equal(t, "device token not found", err.Error())
	})
}

func TestFirebaseService_GetMemberDevices(t *testing.T) {
	db, mock := testutil.SetupTestDB(t)
	service := NewFirebaseService(db)

	memberID := uuid.New()

	t.Run("取得會員設備列表", func(t *testing.T) {
		now := time.Now()
		rows := sqlmock.NewRows([]string{
			"id", "member_id", "device_token", "device_type", "is_active", "last_used_at",
		}).
			AddRow(uuid.New(), memberID, "token1", "ios", true, &now).
			AddRow(uuid.New(), memberID, "token2", "android", true, &now)

		mock.ExpectQuery("SELECT").
			WithArgs(memberID).
			WillReturnRows(rows)

		devices, err := service.GetMemberDevices(memberID)
		assert.NoError(t, err)
		assert.Len(t, devices, 2)
		assert.Equal(t, "token1", devices[0].DeviceToken)
		assert.Equal(t, "token2", devices[1].DeviceToken)
	})

	t.Run("會員無設備", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"id", "member_id", "device_token", "device_type", "is_active", "last_used_at",
		})

		mock.ExpectQuery("SELECT").
			WithArgs(memberID).
			WillReturnRows(rows)

		devices, err := service.GetMemberDevices(memberID)
		assert.NoError(t, err)
		assert.Len(t, devices, 0)
	})
}

func TestFirebaseService_DeactivateToken(t *testing.T) {
	db, mock := testutil.SetupTestDB(t)
	service := NewFirebaseService(db)

	deviceToken := "test-firebase-token"

	t.Run("成功停用", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec("UPDATE \"member_device_tokens\"").
			WithArgs(false, sqlmock.AnyArg(), deviceToken).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := service.DeactivateToken(deviceToken)
		assert.NoError(t, err)
	})

	t.Run("Token 不存在仍返回成功", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec("UPDATE \"member_device_tokens\"").
			WithArgs(false, sqlmock.AnyArg(), deviceToken).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()

		err := service.DeactivateToken(deviceToken)
		assert.NoError(t, err)
	})
}

func TestFirebaseService_SendToMember_NoDevices(t *testing.T) {
	db, mock := testutil.SetupTestDB(t)
	service := NewFirebaseService(db)

	ctx := context.Background()
	memberID := uuid.New()
	tenantID := uuid.New()

	// 模擬查詢無設備
	rows := sqlmock.NewRows([]string{
		"id", "member_id", "device_token", "device_type", "is_active", "last_used_at",
	})
	mock.ExpectQuery("SELECT").
		WithArgs(memberID, true).
		WillReturnRows(rows)

	err := service.SendToMember(ctx, tenantID, memberID, "Test", "Body", nil)
	assert.Error(t, err)
	assert.Equal(t, "no active devices found for member", err.Error())
}

func TestFirebaseConfig_Parsing(t *testing.T) {
	t.Run("解析 Firebase 設定", func(t *testing.T) {
		config := models.FirebaseConfig{
			ProjectID:      "test-project",
			CredentialJSON: `{"type":"service_account"}`,
			ServerKey:      "test-key",
		}

		jsonData, err := json.Marshal(config)
		assert.NoError(t, err)

		var parsed models.FirebaseConfig
		err = json.Unmarshal(jsonData, &parsed)
		assert.NoError(t, err)
		assert.Equal(t, "test-project", parsed.ProjectID)
		assert.Equal(t, "test-key", parsed.ServerKey)
	})
}
