package models

import (
	"github.com/google/uuid"
	"time"
)

type NotificationStatus string

const (
	StatusPending NotificationStatus = "PENDING"
	StatusSent    NotificationStatus = "SENT"
	StatusFailed  NotificationStatus = "FAILED"
)

type NotificationLog struct {
	MemberID       uuid.UUID          `gorm:"type:uuid;not null;index" json:"member_id"`
	ProviderID     uuid.UUID          `gorm:"type:uuid;not null;index" json:"provider_id"`
	Type           ProviderType       `gorm:"size:50;not null;index" json:"type"`
	RecipientName  string             `gorm:"size:255" json:"recipient_name"`
	RecipientEmail string             `gorm:"size:255;index" json:"recipient_email"`
	RecipientPhone string             `gorm:"size:50;index" json:"recipient_phone"`
	Subject        string             `gorm:"size:500" json:"subject"`
	Body           string             `gorm:"type:text;not null" json:"body"`
	Status         NotificationStatus `gorm:"size:50;not null;index" json:"status"`
	ErrorMsg       string             `gorm:"type:text" json:"error_msg"`
	SentAt         *time.Time         `json:"sent_at"`
	Base
}
