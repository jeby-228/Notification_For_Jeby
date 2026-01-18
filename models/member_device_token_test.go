package models

import (
	"member_API/testutil"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestMemberDeviceToken_Structure(t *testing.T) {
	// 測試基本結構
	memberID := uuid.New()
	deviceToken := "test-firebase-token-12345"
	deviceType := "ios"
	lastUsedAt := time.Now()

	token := MemberDeviceToken{
		MemberID:    memberID,
		DeviceToken: deviceToken,
		DeviceType:  deviceType,
		IsActive:    true,
		LastUsedAt:  &lastUsedAt,
	}

	assert.Equal(t, memberID, token.MemberID, "MemberID 應該正確設置")
	assert.Equal(t, deviceToken, token.DeviceToken, "DeviceToken 應該正確設置")
	assert.Equal(t, deviceType, token.DeviceType, "DeviceType 應該正確設置")
	assert.True(t, token.IsActive, "IsActive 應該為 true")
	assert.NotNil(t, token.LastUsedAt, "LastUsedAt 不應該為 nil")
	assert.Equal(t, lastUsedAt, *token.LastUsedAt, "LastUsedAt 應該正確設置")
}

func TestMemberDeviceToken_DeviceTypes(t *testing.T) {
	// 測試不同設備類型
	deviceTypes := []string{"ios", "android", "web"}

	for _, deviceType := range deviceTypes {
		t.Run(deviceType, func(t *testing.T) {
			token := MemberDeviceToken{
				MemberID:    uuid.New(),
				DeviceToken: "test-token",
				DeviceType:  deviceType,
				IsActive:    true,
			}

			assert.Equal(t, deviceType, token.DeviceType, "DeviceType 應該正確設置")
		})
	}
}

func TestMemberDeviceToken_IsActive(t *testing.T) {
	tests := []struct {
		name     string
		isActive bool
	}{
		{"啟用狀態", true},
		{"停用狀態", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := MemberDeviceToken{
				MemberID:    uuid.New(),
				DeviceToken: "test-token",
				DeviceType:  "ios",
				IsActive:    tt.isActive,
			}

			assert.Equal(t, tt.isActive, token.IsActive, "IsActive 應該正確設置")
		})
	}
}

func TestMemberDeviceToken_LastUsedAt(t *testing.T) {
	t.Run("有最後使用時間", func(t *testing.T) {
		lastUsedAt := time.Now()
		token := MemberDeviceToken{
			MemberID:    uuid.New(),
			DeviceToken: "test-token",
			DeviceType:  "android",
			IsActive:    true,
			LastUsedAt:  &lastUsedAt,
		}

		assert.NotNil(t, token.LastUsedAt, "LastUsedAt 不應該為 nil")
		assert.Equal(t, lastUsedAt, *token.LastUsedAt, "LastUsedAt 應該正確設置")
	})

	t.Run("無最後使用時間", func(t *testing.T) {
		token := MemberDeviceToken{
			MemberID:    uuid.New(),
			DeviceToken: "test-token",
			DeviceType:  "web",
			IsActive:    true,
			LastUsedAt:  nil,
		}

		assert.Nil(t, token.LastUsedAt, "LastUsedAt 應該為 nil")
	})
}

func TestMemberDeviceToken_BaseFields(t *testing.T) {
	// 測試 Base 欄位是否正確嵌入
	token := MemberDeviceToken{
		MemberID:    uuid.New(),
		DeviceToken: "test-token",
		DeviceType:  "ios",
		IsActive:    true,
	}

	// Base 的 ID 欄位應該可以訪問
	token.ID = uuid.New()
	assert.NotEqual(t, uuid.Nil, token.ID, "ID 應該可以設置")

	// Base 的 CreationTime 欄位應該可以訪問
	token.CreationTime = time.Now()
	assert.False(t, token.CreationTime.IsZero(), "CreationTime 應該可以設置")
}

func TestMemberDeviceToken_MultipleDevices(t *testing.T) {
	// 測試同一個會員可以有多個設備
	memberID := uuid.New()

	iosToken := MemberDeviceToken{
		MemberID:    memberID,
		DeviceToken: "ios-token-123",
		DeviceType:  "ios",
		IsActive:    true,
	}

	androidToken := MemberDeviceToken{
		MemberID:    memberID,
		DeviceToken: "android-token-456",
		DeviceType:  "android",
		IsActive:    true,
	}

	webToken := MemberDeviceToken{
		MemberID:    memberID,
		DeviceToken: "web-token-789",
		DeviceType:  "web",
		IsActive:    true,
	}

	assert.Equal(t, memberID, iosToken.MemberID, "iOS token 應該有相同的 MemberID")
	assert.Equal(t, memberID, androidToken.MemberID, "Android token 應該有相同的 MemberID")
	assert.Equal(t, memberID, webToken.MemberID, "Web token 應該有相同的 MemberID")

	assert.NotEqual(t, iosToken.DeviceToken, androidToken.DeviceToken, "不同設備應該有不同的 token")
	assert.NotEqual(t, iosToken.DeviceToken, webToken.DeviceToken, "不同設備應該有不同的 token")
	assert.NotEqual(t, androidToken.DeviceToken, webToken.DeviceToken, "不同設備應該有不同的 token")
}

func TestMemberDeviceToken_DatabaseConstraints(t *testing.T) {
	// 測試資料庫層級的約束條件
	t.Run("Invalid device type should be rejected", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		memberID := uuid.New()
		invalidToken := &MemberDeviceToken{
			MemberID:    memberID,
			DeviceToken: "test-token",
			DeviceType:  "invalid", // 無效的設備類型
			IsActive:    true,
		}

		// 預期 INSERT 會失敗，因為 device_type 不符合 CHECK 約束
		mock.ExpectBegin()
		mock.ExpectExec("INSERT").
			WillReturnError(gorm.ErrCheckConstraintViolated)
		mock.ExpectRollback()

		err := db.Create(invalidToken).Error
		assert.Error(t, err, "無效的 device_type 應該被拒絕")
	})

	t.Run("Composite unique constraint prevents duplicate tokens per member", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		memberID := uuid.New()
		deviceToken := "duplicate-token"

		// 第一次插入成功
		token1 := &MemberDeviceToken{
			Base: Base{
				ID: uuid.New(),
			},
			MemberID:    memberID,
			DeviceToken: deviceToken,
			DeviceType:  "ios",
			IsActive:    true,
		}

		mock.ExpectBegin()
		mock.ExpectExec("INSERT").
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := db.Create(token1).Error
		assert.NoError(t, err, "第一次插入應該成功")

		// 相同會員相同 token 的第二次插入應該失敗
		token2 := &MemberDeviceToken{
			Base: Base{
				ID: uuid.New(),
			},
			MemberID:    memberID,
			DeviceToken: deviceToken,
			DeviceType:  "android",
			IsActive:    true,
		}

		mock.ExpectBegin()
		mock.ExpectExec("INSERT").
			WillReturnError(gorm.ErrDuplicatedKey)
		mock.ExpectRollback()

		err = db.Create(token2).Error
		assert.Error(t, err, "相同會員的重複 token 應該被拒絕")
	})

	t.Run("Same token allowed across different members", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		deviceToken := "shared-token"
		member1ID := uuid.New()
		member2ID := uuid.New()

		// 會員1插入成功
		token1 := &MemberDeviceToken{
			Base: Base{
				ID: uuid.New(),
			},
			MemberID:    member1ID,
			DeviceToken: deviceToken,
			DeviceType:  "ios",
			IsActive:    true,
		}

		mock.ExpectBegin()
		mock.ExpectExec("INSERT").
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := db.Create(token1).Error
		assert.NoError(t, err, "會員1插入應該成功")

		// 會員2使用相同 token 應該也能成功
		token2 := &MemberDeviceToken{
			Base: Base{
				ID: uuid.New(),
			},
			MemberID:    member2ID,
			DeviceToken: deviceToken,
			DeviceType:  "android",
			IsActive:    true,
		}

		mock.ExpectBegin()
		mock.ExpectExec("INSERT").
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err = db.Create(token2).Error
		assert.NoError(t, err, "不同會員使用相同 token 應該成功")
	})

	t.Run("Valid device types are accepted", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		validTypes := []string{"ios", "android", "web"}

		for _, deviceType := range validTypes {
			token := &MemberDeviceToken{
				Base: Base{
					ID: uuid.New(),
				},
				MemberID:    uuid.New(),
				DeviceToken: "test-token-" + deviceType,
				DeviceType:  deviceType,
				IsActive:    true,
			}

			mock.ExpectBegin()
			mock.ExpectExec("INSERT").
				WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()

			err := db.Create(token).Error
			assert.NoError(t, err, "有效的 device_type '%s' 應該被接受", deviceType)
		}
	})
}

func TestMemberDeviceToken_Query(t *testing.T) {
	t.Run("查詢特定會員的所有設備", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		memberID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "member_id", "device_token", "device_type", "is_active"}).
			AddRow(uuid.New(), memberID, "token-1", "ios", true).
			AddRow(uuid.New(), memberID, "token-2", "android", true)

		mock.ExpectQuery("SELECT (.+) FROM \"member_device_tokens\"").
			WithArgs(memberID).
			WillReturnRows(rows)

		var tokens []MemberDeviceToken
		err := db.Where("member_id = ?", memberID).Find(&tokens).Error

		assert.NoError(t, err)
		assert.Len(t, tokens, 2, "應該查詢到2個設備")
	})

	t.Run("查詢活躍的設備", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		memberID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "member_id", "device_token", "device_type", "is_active"}).
			AddRow(uuid.New(), memberID, "active-token", "ios", true)

		mock.ExpectQuery("SELECT (.+) FROM \"member_device_tokens\"").
			WithArgs(memberID, true).
			WillReturnRows(rows)

		var tokens []MemberDeviceToken
		err := db.Where("member_id = ? AND is_active = ?", memberID, true).Find(&tokens).Error

		assert.NoError(t, err)
		assert.Len(t, tokens, 1, "應該只查詢到活躍的設備")
		assert.True(t, tokens[0].IsActive)
	})

	t.Run("查詢特定設備類型", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		memberID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "member_id", "device_token", "device_type", "is_active"}).
			AddRow(uuid.New(), memberID, "ios-token", "ios", true)

		mock.ExpectQuery("SELECT (.+) FROM \"member_device_tokens\"").
			WithArgs(memberID, "ios").
			WillReturnRows(rows)

		var tokens []MemberDeviceToken
		err := db.Where("member_id = ? AND device_type = ?", memberID, "ios").Find(&tokens).Error

		assert.NoError(t, err)
		assert.Len(t, tokens, 1)
		assert.Equal(t, "ios", tokens[0].DeviceType)
	})
}

func TestMemberDeviceToken_Update(t *testing.T) {
	t.Run("更新設備活躍狀態", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		tokenID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectExec("UPDATE \"member_device_tokens\"").
			WithArgs(false, sqlmock.AnyArg(), tokenID).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := db.Model(&MemberDeviceToken{}).
			Where("id = ?", tokenID).
			Update("is_active", false).Error

		assert.NoError(t, err, "更新活躍狀態應該成功")
	})

	t.Run("更新最後使用時間", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		tokenID := uuid.New()
		lastUsedAt := time.Now()

		mock.ExpectBegin()
		mock.ExpectExec("UPDATE \"member_device_tokens\"").
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), tokenID).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := db.Model(&MemberDeviceToken{}).
			Where("id = ?", tokenID).
			Update("last_used_at", lastUsedAt).Error

		assert.NoError(t, err, "更新最後使用時間應該成功")
	})

	t.Run("批量停用舊設備", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		memberID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectExec("UPDATE \"member_device_tokens\"").
			WithArgs(false, sqlmock.AnyArg(), memberID, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectCommit()

		cutoffTime := time.Now().AddDate(0, -6, 0)
		err := db.Model(&MemberDeviceToken{}).
			Where("member_id = ? AND last_used_at < ?", memberID, cutoffTime).
			Update("is_active", false).Error

		assert.NoError(t, err, "批量停用舊設備應該成功")
	})
}

func TestMemberDeviceToken_Delete(t *testing.T) {
	t.Run("刪除特定設備", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		tokenID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectExec("DELETE FROM \"member_device_tokens\"").
			WithArgs(tokenID).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := db.Delete(&MemberDeviceToken{}, tokenID).Error

		assert.NoError(t, err, "刪除設備應該成功")
	})

	t.Run("刪除會員的所有設備", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		memberID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectExec("DELETE FROM \"member_device_tokens\"").
			WithArgs(memberID).
			WillReturnResult(sqlmock.NewResult(0, 3))
		mock.ExpectCommit()

		err := db.Where("member_id = ?", memberID).Delete(&MemberDeviceToken{}).Error

		assert.NoError(t, err, "刪除會員的所有設備應該成功")
	})
}

func TestMemberDeviceToken_EdgeCases(t *testing.T) {
	t.Run("設備 token 長度邊界", func(t *testing.T) {
		token := MemberDeviceToken{
			MemberID:    uuid.New(),
			DeviceToken: "a", // 最短
			DeviceType:  "ios",
			IsActive:    true,
		}
		assert.Equal(t, "a", token.DeviceToken)

		longToken := MemberDeviceToken{
			MemberID:    uuid.New(),
			DeviceToken: string(make([]byte, 255)), // 最長
			DeviceType:  "android",
			IsActive:    true,
		}
		assert.Len(t, longToken.DeviceToken, 255)
	})

	t.Run("空設備類型", func(t *testing.T) {
		token := MemberDeviceToken{
			MemberID:    uuid.New(),
			DeviceToken: "test-token",
			DeviceType:  "", // 空字串
			IsActive:    true,
		}
		assert.Empty(t, token.DeviceType, "設備類型可以是空字串")
	})

	t.Run("nil UUID", func(t *testing.T) {
		token := MemberDeviceToken{
			MemberID:    uuid.Nil, // nil UUID
			DeviceToken: "test-token",
			DeviceType:  "web",
			IsActive:    true,
		}
		assert.Equal(t, uuid.Nil, token.MemberID)
	})

	t.Run("預設值測試", func(t *testing.T) {
		token := MemberDeviceToken{
			MemberID:    uuid.New(),
			DeviceToken: "test-token",
			DeviceType:  "ios",
		}

		// IsActive 沒設定時，Go 預設為 false
		assert.False(t, token.IsActive, "未設定的 IsActive 應該為 false")
		assert.Nil(t, token.LastUsedAt, "未設定的 LastUsedAt 應該為 nil")
	})
}

func TestMemberDeviceToken_TimestampUpdates(t *testing.T) {
	t.Run("記錄最後使用時間", func(t *testing.T) {
		before := time.Now()
		time.Sleep(10 * time.Millisecond)

		lastUsedAt := time.Now()
		token := MemberDeviceToken{
			MemberID:    uuid.New(),
			DeviceToken: "test-token",
			DeviceType:  "android",
			IsActive:    true,
			LastUsedAt:  &lastUsedAt,
		}

		assert.NotNil(t, token.LastUsedAt)
		assert.True(t, token.LastUsedAt.After(before), "最後使用時間應該在之前的時間之後")
	})

	t.Run("更新最後使用時間", func(t *testing.T) {
		firstTime := time.Now().Add(-1 * time.Hour)
		token := MemberDeviceToken{
			MemberID:    uuid.New(),
			DeviceToken: "test-token",
			DeviceType:  "web",
			IsActive:    true,
			LastUsedAt:  &firstTime,
		}

		assert.Equal(t, firstTime, *token.LastUsedAt)

		// 更新時間
		newTime := time.Now()
		token.LastUsedAt = &newTime

		assert.Equal(t, newTime, *token.LastUsedAt)
		assert.True(t, token.LastUsedAt.After(firstTime), "新時間應該晚於舊時間")
	})
}
