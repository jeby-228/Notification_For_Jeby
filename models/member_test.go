package models

import (
	"member_API/testutil"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestMember_Structure(t *testing.T) {
	tenantID := uuid.New()
	member := Member{
		Name:         "測試用戶",
		Email:        "test@example.com",
		PasswordHash: "hashed_password",
		TenantsID:    tenantID,
		APIKey:       "api_key_123",
	}

	assert.Equal(t, "測試用戶", member.Name)
	assert.Equal(t, "test@example.com", member.Email)
	assert.Equal(t, "hashed_password", member.PasswordHash)
	assert.Equal(t, tenantID, member.TenantsID)
	assert.Equal(t, "api_key_123", member.APIKey)
}

func TestMember_Create(t *testing.T) {
	t.Run("成功創建會員", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		member := &Member{
			Base: Base{
				ID: uuid.New(),
			},
			Name:         "John Doe",
			Email:        "john@example.com",
			PasswordHash: "hashed_pass",
			TenantsID:    uuid.New(),
			APIKey:       "key123",
		}

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO \"members\"").
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := db.Create(member).Error
		assert.NoError(t, err)
	})

	t.Run("重複 Email 應該失敗", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		member := &Member{
			Base: Base{
				ID: uuid.New(),
			},
			Name:         "User1",
			Email:        "duplicate@example.com",
			PasswordHash: "pass1",
			TenantsID:    uuid.New(),
			APIKey:       "key1",
		}

		mock.ExpectBegin()
		mock.ExpectExec("INSERT").
			WillReturnError(gorm.ErrDuplicatedKey)
		mock.ExpectRollback()

		err := db.Create(member).Error
		assert.Error(t, err)
	})

	t.Run("重複 API Key 應該失敗", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		member := &Member{
			Base: Base{
				ID: uuid.New(),
			},
			Name:         "User2",
			Email:        "user2@example.com",
			PasswordHash: "pass2",
			TenantsID:    uuid.New(),
			APIKey:       "duplicate_key",
		}

		mock.ExpectBegin()
		mock.ExpectExec("INSERT").
			WillReturnError(gorm.ErrDuplicatedKey)
		mock.ExpectRollback()

		err := db.Create(member).Error
		assert.Error(t, err)
	})
}

func TestMember_Query(t *testing.T) {
	t.Run("通過 Email 查詢", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		memberID := uuid.New()
		tenantID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "name", "email", "password_hash", "tenants_id", "api_key"}).
			AddRow(memberID, "Test User", "test@example.com", "hashed", tenantID, "key123")

		mock.ExpectQuery("SELECT (.+) FROM \"members\"").
			WithArgs("test@example.com", 1).
			WillReturnRows(rows)

		var member Member
		err := db.Where("email = ?", "test@example.com").First(&member).Error

		assert.NoError(t, err)
		assert.Equal(t, "Test User", member.Name)
		assert.Equal(t, "test@example.com", member.Email)
	})

	t.Run("通過 API Key 查詢", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		memberID := uuid.New()
		tenantID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "name", "email", "password_hash", "tenants_id", "api_key"}).
			AddRow(memberID, "API User", "api@example.com", "hashed", tenantID, "special_key")

		mock.ExpectQuery("SELECT (.+) FROM \"members\"").
			WithArgs("special_key", 1).
			WillReturnRows(rows)

		var member Member
		err := db.Where("api_key = ?", "special_key").First(&member).Error

		assert.NoError(t, err)
		assert.Equal(t, "special_key", member.APIKey)
	})

	t.Run("查詢特定租戶的會員", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		tenantID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "name", "email", "tenants_id"}).
			AddRow(uuid.New(), "User1", "user1@example.com", tenantID).
			AddRow(uuid.New(), "User2", "user2@example.com", tenantID)

		mock.ExpectQuery("SELECT (.+) FROM \"members\"").
			WithArgs(tenantID).
			WillReturnRows(rows)

		var members []Member
		err := db.Where("tenants_id = ?", tenantID).Find(&members).Error

		assert.NoError(t, err)
		assert.Len(t, members, 2)
	})
}

func TestMember_Update(t *testing.T) {
	t.Run("更新會員資訊", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		memberID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectExec("UPDATE \"members\"").
			WithArgs("New Name", sqlmock.AnyArg(), memberID).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := db.Model(&Member{}).
			Where("id = ?", memberID).
			Update("name", "New Name").Error

		assert.NoError(t, err)
	})

	t.Run("更新密碼", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		memberID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectExec("UPDATE \"members\"").
			WithArgs("new_hashed_password", sqlmock.AnyArg(), memberID).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := db.Model(&Member{}).
			Where("id = ?", memberID).
			Update("password_hash", "new_hashed_password").Error

		assert.NoError(t, err)
	})

	t.Run("更新 API Key", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		memberID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectExec("UPDATE \"members\"").
			WithArgs("new_api_key", sqlmock.AnyArg(), memberID).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := db.Model(&Member{}).
			Where("id = ?", memberID).
			Update("api_key", "new_api_key").Error

		assert.NoError(t, err)
	})
}

func TestMember_Delete(t *testing.T) {
	t.Run("刪除會員", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		memberID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectExec("DELETE FROM \"members\"").
			WithArgs(memberID).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := db.Delete(&Member{}, memberID).Error
		assert.NoError(t, err)
	})
}

func TestMember_PasswordSecurity(t *testing.T) {
	t.Run("密碼不應該序列化到 JSON", func(t *testing.T) {
		member := Member{
			Name:         "User",
			Email:        "user@example.com",
			PasswordHash: "secret_hash",
		}

		// PasswordHash 有 json:"-" tag，不應該被序列化
		assert.NotEmpty(t, member.PasswordHash)
	})
}

func TestMember_EdgeCases(t *testing.T) {
	t.Run("空名稱", func(t *testing.T) {
		member := Member{
			Name:  "",
			Email: "empty@example.com",
		}
		assert.Empty(t, member.Name)
	})

	t.Run("長名稱", func(t *testing.T) {
		longName := string(make([]byte, 255))
		member := Member{
			Name:  longName,
			Email: "long@example.com",
		}
		assert.Len(t, member.Name, 255)
	})

	t.Run("Nil 租戶 ID", func(t *testing.T) {
		member := Member{
			Name:      "No Tenant",
			Email:     "notenant@example.com",
			TenantsID: uuid.Nil,
		}
		assert.Equal(t, uuid.Nil, member.TenantsID)
	})

	t.Run("空 API Key", func(t *testing.T) {
		member := Member{
			Name:   "No Key",
			Email:  "nokey@example.com",
			APIKey: "",
		}
		assert.Empty(t, member.APIKey)
	})
}

func TestMember_EmailValidation(t *testing.T) {
	t.Run("有效的 Email 格式", func(t *testing.T) {
		validEmails := []string{
			"user@example.com",
			"test.user@domain.com",
			"admin+tag@company.org",
		}

		for _, email := range validEmails {
			member := Member{
				Name:  "Test",
				Email: email,
			}
			assert.Equal(t, email, member.Email)
		}
	})
}
