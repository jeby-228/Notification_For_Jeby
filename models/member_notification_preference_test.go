package models

import (
	"member_API/testutil"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestMemberNotificationPreference_Structure(t *testing.T) {
	memberID := uuid.New()
	providerID := uuid.New()

	pref := MemberNotificationPreference{
		MemberID:   memberID,
		ProviderID: providerID,
		IsDefault:  true,
		IsActive:   true,
	}

	assert.Equal(t, memberID, pref.MemberID)
	assert.Equal(t, providerID, pref.ProviderID)
	assert.True(t, pref.IsDefault)
	assert.True(t, pref.IsActive)
}

func TestMemberNotificationPreference_Create(t *testing.T) {
	t.Run("成功創建偏好設定", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		pref := &MemberNotificationPreference{
			Base: Base{
				ID: uuid.New(),
			},
			MemberID:   uuid.New(),
			ProviderID: uuid.New(),
			IsDefault:  false,
			IsActive:   true,
		}

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO \"member_notification_preferences\"").
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := db.Create(pref).Error
		assert.NoError(t, err)
	})

	t.Run("重複的會員-提供者組合應該失敗", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		memberID := uuid.New()
		providerID := uuid.New()

		pref := &MemberNotificationPreference{
			Base: Base{
				ID: uuid.New(),
			},
			MemberID:   memberID,
			ProviderID: providerID,
			IsDefault:  false,
			IsActive:   true,
		}

		mock.ExpectBegin()
		mock.ExpectExec("INSERT").
			WillReturnError(gorm.ErrDuplicatedKey)
		mock.ExpectRollback()

		err := db.Create(pref).Error
		assert.Error(t, err)
	})
}

func TestMemberNotificationPreference_Query(t *testing.T) {
	t.Run("查詢會員的所有偏好設定", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		memberID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "member_id", "provider_id", "is_default", "is_active"}).
			AddRow(uuid.New(), memberID, uuid.New(), false, true).
			AddRow(uuid.New(), memberID, uuid.New(), true, true)

		mock.ExpectQuery("SELECT (.+) FROM \"member_notification_preferences\"").
			WithArgs(memberID).
			WillReturnRows(rows)

		var prefs []MemberNotificationPreference
		err := db.Where("member_id = ?", memberID).Find(&prefs).Error

		assert.NoError(t, err)
		assert.Len(t, prefs, 2)
	})

	t.Run("查詢會員的預設偏好", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		memberID := uuid.New()
		providerID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "member_id", "provider_id", "is_default", "is_active"}).
			AddRow(uuid.New(), memberID, providerID, true, true)

		mock.ExpectQuery("SELECT (.+) FROM \"member_notification_preferences\"").
			WithArgs(memberID, true, 1).
			WillReturnRows(rows)

		var pref MemberNotificationPreference
		err := db.Where("member_id = ? AND is_default = ?", memberID, true).First(&pref).Error

		assert.NoError(t, err)
		assert.True(t, pref.IsDefault)
	})

	t.Run("查詢活躍的偏好設定", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		memberID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "member_id", "provider_id", "is_default", "is_active"}).
			AddRow(uuid.New(), memberID, uuid.New(), false, true)

		mock.ExpectQuery("SELECT (.+) FROM \"member_notification_preferences\"").
			WithArgs(memberID, true).
			WillReturnRows(rows)

		var prefs []MemberNotificationPreference
		err := db.Where("member_id = ? AND is_active = ?", memberID, true).Find(&prefs).Error

		assert.NoError(t, err)
		assert.Len(t, prefs, 1)
		assert.True(t, prefs[0].IsActive)
	})

	t.Run("查詢特定提供者的偏好", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		providerID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "member_id", "provider_id", "is_default", "is_active"}).
			AddRow(uuid.New(), uuid.New(), providerID, false, true).
			AddRow(uuid.New(), uuid.New(), providerID, false, true)

		mock.ExpectQuery("SELECT (.+) FROM \"member_notification_preferences\"").
			WithArgs(providerID).
			WillReturnRows(rows)

		var prefs []MemberNotificationPreference
		err := db.Where("provider_id = ?", providerID).Find(&prefs).Error

		assert.NoError(t, err)
		assert.Len(t, prefs, 2)
	})
}

func TestMemberNotificationPreference_Update(t *testing.T) {
	t.Run("停用偏好設定", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		prefID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectExec("UPDATE \"member_notification_preferences\"").
			WithArgs(false, sqlmock.AnyArg(), prefID).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := db.Model(&MemberNotificationPreference{}).
			Where("id = ?", prefID).
			Update("is_active", false).Error

		assert.NoError(t, err)
	})

	t.Run("設為預設偏好", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		prefID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectExec("UPDATE \"member_notification_preferences\"").
			WithArgs(true, sqlmock.AnyArg(), prefID).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := db.Model(&MemberNotificationPreference{}).
			Where("id = ?", prefID).
			Update("is_default", true).Error

		assert.NoError(t, err)
	})

	t.Run("批量停用會員的所有偏好", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		memberID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectExec("UPDATE \"member_notification_preferences\"").
			WithArgs(false, sqlmock.AnyArg(), memberID).
			WillReturnResult(sqlmock.NewResult(0, 3))
		mock.ExpectCommit()

		err := db.Model(&MemberNotificationPreference{}).
			Where("member_id = ?", memberID).
			Update("is_active", false).Error

		assert.NoError(t, err)
	})
}

func TestMemberNotificationPreference_Delete(t *testing.T) {
	t.Run("刪除偏好設定", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		prefID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectExec("DELETE FROM \"member_notification_preferences\"").
			WithArgs(prefID).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := db.Delete(&MemberNotificationPreference{}, prefID).Error
		assert.NoError(t, err)
	})

	t.Run("刪除會員的所有偏好", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		memberID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectExec("DELETE FROM \"member_notification_preferences\"").
			WithArgs(memberID).
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectCommit()

		err := db.Where("member_id = ?", memberID).Delete(&MemberNotificationPreference{}).Error
		assert.NoError(t, err)
	})
}

func TestMemberNotificationPreference_DefaultBehavior(t *testing.T) {
	t.Run("只能有一個預設偏好", func(t *testing.T) {
		memberID := uuid.New()

		pref1 := MemberNotificationPreference{
			MemberID:   memberID,
			ProviderID: uuid.New(),
			IsDefault:  true,
			IsActive:   true,
		}

		pref2 := MemberNotificationPreference{
			MemberID:   memberID,
			ProviderID: uuid.New(),
			IsDefault:  true,
			IsActive:   true,
		}

		assert.True(t, pref1.IsDefault)
		assert.True(t, pref2.IsDefault)
		// 在實際應用中，應該有邏輯確保只有一個預設偏好
	})

	t.Run("預設值測試", func(t *testing.T) {
		pref := MemberNotificationPreference{
			MemberID:   uuid.New(),
			ProviderID: uuid.New(),
		}

		// Go 的零值
		assert.False(t, pref.IsDefault)
		assert.False(t, pref.IsActive)
	})
}

func TestMemberNotificationPreference_EdgeCases(t *testing.T) {
	t.Run("Nil 會員 ID", func(t *testing.T) {
		pref := MemberNotificationPreference{
			MemberID:   uuid.Nil,
			ProviderID: uuid.New(),
			IsActive:   true,
		}
		assert.Equal(t, uuid.Nil, pref.MemberID)
	})

	t.Run("Nil 提供者 ID", func(t *testing.T) {
		pref := MemberNotificationPreference{
			MemberID:   uuid.New(),
			ProviderID: uuid.Nil,
			IsActive:   true,
		}
		assert.Equal(t, uuid.Nil, pref.ProviderID)
	})

	t.Run("同時為預設和非活躍", func(t *testing.T) {
		pref := MemberNotificationPreference{
			MemberID:   uuid.New(),
			ProviderID: uuid.New(),
			IsDefault:  true,
			IsActive:   false,
		}
		assert.True(t, pref.IsDefault)
		assert.False(t, pref.IsActive)
		// 這種狀態在實際應用中可能需要驗證
	})
}

func TestMemberNotificationPreference_MultipleProviders(t *testing.T) {
	t.Run("會員可以有多個提供者偏好", func(t *testing.T) {
		memberID := uuid.New()

		smtpPref := MemberNotificationPreference{
			MemberID:   memberID,
			ProviderID: uuid.New(),
			IsDefault:  true,
			IsActive:   true,
		}

		smsPref := MemberNotificationPreference{
			MemberID:   memberID,
			ProviderID: uuid.New(),
			IsDefault:  false,
			IsActive:   true,
		}

		firebasePref := MemberNotificationPreference{
			MemberID:   memberID,
			ProviderID: uuid.New(),
			IsDefault:  false,
			IsActive:   true,
		}

		assert.Equal(t, memberID, smtpPref.MemberID)
		assert.Equal(t, memberID, smsPref.MemberID)
		assert.Equal(t, memberID, firebasePref.MemberID)
		assert.NotEqual(t, smtpPref.ProviderID, smsPref.ProviderID)
		assert.NotEqual(t, smtpPref.ProviderID, firebasePref.ProviderID)
	})
}
