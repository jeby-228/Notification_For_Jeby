package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Base 基礎模型結構，包含通用的審計欄位
type Base struct {
	ID                   uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	Sort                 int        `json:"sort"`
	CreationTime         time.Time  `gorm:"autoCreateTime" json:"created_at"`
	CreatorId            uuid.UUID  `json:"creator_id"`
	LastModificationTime *time.Time `gorm:"autoUpdateTime" json:"last_modification_time"`
	LastModifierId       uuid.UUID  `json:"last_modifier_id"`
	IsDeleted            bool       `gorm:"default:false" json:"-"`
	DeletedAt            *time.Time `gorm:"index" json:"-"`
}

// BeforeCreate hook to auto-generate UUID
func (b *Base) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}
