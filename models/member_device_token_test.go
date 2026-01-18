package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
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
	token.Base.ID = uuid.New()
	assert.NotEqual(t, uuid.Nil, token.Base.ID, "Base.ID 應該可以設置")

	// Base 的 CreationTime 欄位應該可以訪問
	token.Base.CreationTime = time.Now()
	assert.False(t, token.Base.CreationTime.IsZero(), "Base.CreationTime 應該可以設置")
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
