package models

import (
	"time"

	"github.com/google/uuid"
)

// MemberDeviceToken 儲存會員的 Firebase Device Token
type MemberDeviceToken struct {
	MemberID    uuid.UUID  `gorm:"type:uuid;uniqueIndex:idx_member_device_token" json:"member_id"`            // 會員ID
	DeviceToken string     `gorm:"size:255;uniqueIndex:idx_member_device_token" json:"device_token"`          // 設備Token
	DeviceType  string     `gorm:"size:50;check:device_type IN ('ios', 'android', 'web')" json:"device_type"` // 設備類型: ios, android, web
	IsActive    bool       `gorm:"default:true" json:"is_active"`                                             // 是否啟用
	LastUsedAt  *time.Time `json:"last_used_at"`                                                              // 最後使用時間
	Base
}
