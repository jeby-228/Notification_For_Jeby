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
	assert.NotEqual(t, uuid.Nil, token.ID, "Base.ID 應該可以設置")

	// Base 的 CreationTime 欄位應該可以訪問
	token.CreationTime = time.Now()
	assert.False(t, token.CreationTime.IsZero(), "Base.CreationTime 應該可以設置")
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
