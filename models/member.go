package models

import "time"

// Member represents a user stored in PostgreSQL and managed by GORM.
type Member struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `gorm:"size:255;not null" json:"name"`
	Email        string    `gorm:"size:255;uniqueIndex;not null" json:"email"`
	PasswordHash string    `gorm:"size:255" json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	IsDeleted    bool      `gorm:"default:false" json:"-"`
	DeletedAt   *time.Time `json:"-"`
}
