package models

import (
	"time"

	"github.com/google/uuid"
)

// MemberDeviceToken 儲存會員的 Firebase Device Token
type MemberDeviceToken struct {
	MemberID    uuid.UUID  `gorm:"type:uuid;index" json:"member_id"`      // 會員ID
	DeviceToken string     `gorm:"size:255;uniqueIndex" json:"device_token"` // 設備Token
	DeviceType  string     `gorm:"size:50" json:"device_type"`            // 設備類型: ios, android, web
	IsActive    bool       `gorm:"default:true" json:"is_active"`         // 是否啟用
	LastUsedAt  *time.Time `json:"last_used_at"`                          // 最後使用時間
	Base
}
