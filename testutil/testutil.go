package testutil

import (
	"fmt"
	"member_API/models"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SetupTestDB creates an in-memory SQLite database for testing
func SetupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Auto migrate the schema
	err = db.AutoMigrate(
		&models.Member{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return db
}

// CleanupTestDB cleans up test database
func CleanupTestDB(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}
	return sqlDB.Close()
}

// CreateTestMember creates a test member in the database
func CreateTestMember(t *testing.T, db *gorm.DB, email string) *models.Member {
	member := &models.Member{
		Name:         "Test User",
		Email:        email,
		PasswordHash: "test-hash",
		Base: models.Base{
			CreatorId: 1,
			IsDeleted: false,
		},
	}

	if err := db.Create(member).Error; err != nil {
		t.Fatalf("Failed to create test member: %v", err)
	}

	return member
}


