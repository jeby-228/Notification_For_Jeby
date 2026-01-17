package services

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}

	dialector := postgres.New(postgres.Config{
		Conn:       sqlDB,
		DriverName: "postgres",
	})

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm db: %v", err)
	}

	return db, mock
}

func TestGenerateAPIKey(t *testing.T) {
	db, _ := setupTestDB(t)
	svc := NewAPIKeyService(db)

	key, err := svc.GenerateAPIKey()
	assert.NoError(t, err)
	assert.NotEmpty(t, key)
	assert.Contains(t, key, "ak_")
	assert.Greater(t, len(key), 10)
}

func TestGenerateAPIKey_Uniqueness(t *testing.T) {
	db, _ := setupTestDB(t)
	svc := NewAPIKeyService(db)

	keys := make(map[string]bool)
	for i := 0; i < 100; i++ {
		key, err := svc.GenerateAPIKey()
		assert.NoError(t, err)
		assert.False(t, keys[key], "generated duplicate key")
		keys[key] = true
	}
}

func TestRegenerateAPIKey_Success(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewAPIKeyService(db)

	memberID := uuid.New()
	oldAPIKey := "ak_old_key"

	memberRows := sqlmock.NewRows([]string{"id", "name", "email", "api_key", "is_deleted"}).
		AddRow(memberID, "Test User", "test@example.com", oldAPIKey, false)

	mock.ExpectQuery(`SELECT \* FROM "members"`).
		WithArgs(memberID, false, 1).
		WillReturnRows(memberRows)

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "members"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	member, err := svc.RegenerateAPIKey(memberID)
	assert.NoError(t, err)
	assert.NotNil(t, member)
	assert.NotEqual(t, oldAPIKey, member.APIKey)
	assert.Contains(t, member.APIKey, "ak_")

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestRegenerateAPIKey_MemberNotFound(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewAPIKeyService(db)

	memberID := uuid.New()

	mock.ExpectQuery(`SELECT \* FROM "members"`).
		WithArgs(memberID, false, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	member, err := svc.RegenerateAPIKey(memberID)
	assert.Error(t, err)
	assert.Nil(t, member)
	assert.Equal(t, "會員不存在", err.Error())

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestValidateAPIKey_Success(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewAPIKeyService(db)

	memberID := uuid.New()
	tenantID := uuid.New()
	apiKey := "ak_valid_key"

	memberRows := sqlmock.NewRows([]string{"id", "name", "email", "api_key", "tenants_id", "is_deleted"}).
		AddRow(memberID, "Test User", "test@example.com", apiKey, tenantID, false)

	mock.ExpectQuery(`SELECT \* FROM "members"`).
		WithArgs(apiKey, false, 1).
		WillReturnRows(memberRows)

	tenantRows := sqlmock.NewRows([]string{"id", "name", "description"}).
		AddRow(tenantID, "Test Tenant", "Description")

	mock.ExpectQuery(`SELECT \* FROM "tenants"`).
		WithArgs(tenantID, 1).
		WillReturnRows(tenantRows)

	member, tenant, err := svc.ValidateAPIKey(apiKey)
	assert.NoError(t, err)
	assert.NotNil(t, member)
	assert.NotNil(t, tenant)
	assert.Equal(t, "test@example.com", member.Email)
	assert.Equal(t, "Test Tenant", tenant.Name)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestValidateAPIKey_InvalidKey(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewAPIKeyService(db)

	apiKey := "ak_invalid_key"

	mock.ExpectQuery(`SELECT \* FROM "members"`).
		WithArgs(apiKey, false, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	member, tenant, err := svc.ValidateAPIKey(apiKey)
	assert.Error(t, err)
	assert.Nil(t, member)
	assert.Nil(t, tenant)
	assert.Equal(t, "無效的 API Key", err.Error())

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestValidateAPIKey_NoTenant(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewAPIKeyService(db)

	memberID := uuid.New()
	apiKey := "ak_valid_key_no_tenant"

	memberRows := sqlmock.NewRows([]string{"id", "name", "email", "api_key", "tenants_id", "is_deleted"}).
		AddRow(memberID, "Test User", "test@example.com", apiKey, uuid.Nil, false)

	mock.ExpectQuery(`SELECT \* FROM "members"`).
		WithArgs(apiKey, false, 1).
		WillReturnRows(memberRows)

	member, tenant, err := svc.ValidateAPIKey(apiKey)
	assert.NoError(t, err)
	assert.NotNil(t, member)
	assert.Nil(t, tenant)
	assert.Equal(t, "test@example.com", member.Email)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestValidateAPIKey_DeletedMember(t *testing.T) {
	db, mock := setupTestDB(t)
	svc := NewAPIKeyService(db)

	apiKey := "ak_deleted_member_key"

	mock.ExpectQuery(`SELECT \* FROM "members"`).
		WithArgs(apiKey, false, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	member, tenant, err := svc.ValidateAPIKey(apiKey)
	assert.Error(t, err)
	assert.Nil(t, member)
	assert.Nil(t, tenant)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestAPIKeyFormat(t *testing.T) {
	key, err := generateSecureKey()
	assert.NoError(t, err)
	assert.Regexp(t, `^ak_[a-f0-9]{64}$`, key)
}
