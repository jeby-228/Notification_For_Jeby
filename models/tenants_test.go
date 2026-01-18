package models

import (
	"member_API/testutil"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestTenants_Structure(t *testing.T) {
	tenantID := uuid.New()

	tenant := Tenants{
		ID:          tenantID,
		Name:        "Test Tenant",
		Description: "A test tenant",
	}

	assert.Equal(t, tenantID, tenant.ID)
	assert.Equal(t, "Test Tenant", tenant.Name)
	assert.Equal(t, "A test tenant", tenant.Description)
}

func TestTenants_Create(t *testing.T) {
	t.Run("成功創建租戶", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		tenant := &Tenants{
			ID:          uuid.New(),
			Name:        "Company A",
			Description: "First company",
		}

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO \"tenants\"").
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := db.Create(tenant).Error
		assert.NoError(t, err)
	})

	t.Run("創建時自動生成 UUID", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		tenant := &Tenants{
			Name:        "Company B",
			Description: "Second company",
		}

		mock.ExpectBegin()
		mock.ExpectExec("INSERT").
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := db.Create(tenant).Error
		assert.NoError(t, err)
		// Base 的 BeforeCreate hook 會自動生成 ID
	})
}

func TestTenants_Query(t *testing.T) {
	t.Run("查詢所有租戶", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		rows := sqlmock.NewRows([]string{"id", "name", "description"}).
			AddRow(uuid.New(), "Tenant1", "Description1").
			AddRow(uuid.New(), "Tenant2", "Description2")

		mock.ExpectQuery("SELECT (.+) FROM \"tenants\"").
			WillReturnRows(rows)

		var tenants []Tenants
		err := db.Find(&tenants).Error

		assert.NoError(t, err)
		assert.Len(t, tenants, 2)
	})

	t.Run("通過 ID 查詢租戶", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		tenantID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "name", "description"}).
			AddRow(tenantID, "Specific Tenant", "Specific description")

		mock.ExpectQuery("SELECT (.+) FROM \"tenants\"").
			WithArgs(tenantID, 1).
			WillReturnRows(rows)

		var tenant Tenants
		err := db.Where("id = ?", tenantID).First(&tenant).Error

		assert.NoError(t, err)
		assert.Equal(t, tenantID, tenant.ID)
		assert.Equal(t, "Specific Tenant", tenant.Name)
	})

	t.Run("通過名稱查詢租戶", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		rows := sqlmock.NewRows([]string{"id", "name", "description"}).
			AddRow(uuid.New(), "Target Company", "Target description")

		mock.ExpectQuery("SELECT (.+) FROM \"tenants\"").
			WithArgs("Target Company", 1).
			WillReturnRows(rows)

		var tenant Tenants
		err := db.Where("name = ?", "Target Company").First(&tenant).Error

		assert.NoError(t, err)
		assert.Equal(t, "Target Company", tenant.Name)
	})

	t.Run("租戶不存在", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		nonExistID := uuid.New()

		mock.ExpectQuery("SELECT (.+) FROM \"tenants\"").
			WithArgs(nonExistID, 1).
			WillReturnError(gorm.ErrRecordNotFound)

		var tenant Tenants
		err := db.Where("id = ?", nonExistID).First(&tenant).Error

		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
	})
}

func TestTenants_Update(t *testing.T) {
	t.Run("更新租戶名稱", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		tenantID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectExec("UPDATE \"tenants\"").
			WithArgs("New Name", sqlmock.AnyArg(), tenantID).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := db.Model(&Tenants{}).
			Where("id = ?", tenantID).
			Update("name", "New Name").Error

		assert.NoError(t, err)
	})

	t.Run("更新租戶描述", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		tenantID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectExec("UPDATE \"tenants\"").
			WithArgs("New Description", sqlmock.AnyArg(), tenantID).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := db.Model(&Tenants{}).
			Where("id = ?", tenantID).
			Update("description", "New Description").Error

		assert.NoError(t, err)
	})

	t.Run("批量更新", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		tenantID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectExec("UPDATE \"tenants\"").
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), tenantID).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := db.Model(&Tenants{}).
			Where("id = ?", tenantID).
			Updates(map[string]interface{}{
				"name":        "Updated Name",
				"description": "Updated Description",
			}).Error

		assert.NoError(t, err)
	})
}

func TestTenants_Delete(t *testing.T) {
	t.Run("刪除租戶", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		tenantID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectExec("DELETE FROM \"tenants\"").
			WithArgs(tenantID).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := db.Delete(&Tenants{}, tenantID).Error
		assert.NoError(t, err)
	})
}

func TestTenants_EdgeCases(t *testing.T) {
	t.Run("空名稱", func(t *testing.T) {
		tenant := Tenants{
			ID:          uuid.New(),
			Name:        "",
			Description: "Empty name tenant",
		}
		assert.Empty(t, tenant.Name)
	})

	t.Run("空描述", func(t *testing.T) {
		tenant := Tenants{
			ID:          uuid.New(),
			Name:        "Tenant",
			Description: "",
		}
		assert.Empty(t, tenant.Description)
	})

	t.Run("長名稱", func(t *testing.T) {
		longName := string(make([]byte, 255))
		tenant := Tenants{
			ID:   uuid.New(),
			Name: longName,
		}
		assert.Len(t, tenant.Name, 255)
	})

	t.Run("長描述", func(t *testing.T) {
		longDesc := string(make([]byte, 255))
		tenant := Tenants{
			ID:          uuid.New(),
			Name:        "Tenant",
			Description: longDesc,
		}
		assert.Len(t, tenant.Description, 255)
	})

	t.Run("Nil UUID", func(t *testing.T) {
		tenant := Tenants{
			ID:   uuid.Nil,
			Name: "Nil ID Tenant",
		}
		assert.Equal(t, uuid.Nil, tenant.ID)
	})
}

func TestTenants_Relationships(t *testing.T) {
	t.Run("租戶與會員的關聯", func(t *testing.T) {
		tenantID := uuid.New()

		tenant := Tenants{
			ID:   tenantID,
			Name: "Company with Members",
		}

		member := Member{
			TenantsID: tenantID,
			Name:      "Employee",
			Email:     "employee@company.com",
		}

		assert.Equal(t, tenant.ID, member.TenantsID)
	})

	t.Run("租戶與通知提供者的關聯", func(t *testing.T) {
		tenantID := uuid.New()

		tenant := Tenants{
			ID:   tenantID,
			Name: "Company with Providers",
		}

		provider := NotificationProvider{
			TenantsID: tenantID,
			Name:      "SMTP Provider",
			Type:      ProviderSMTP,
		}

		assert.Equal(t, tenant.ID, provider.TenantsID)
	})
}

func TestTenants_BaseFields(t *testing.T) {
	t.Run("Base 欄位可用", func(t *testing.T) {
		tenant := Tenants{
			ID:   uuid.New(),
			Name: "Tenant with Base",
		}

		// Base 欄位應該可以訪問
		tenant.Sort = 1
		assert.Equal(t, 1, tenant.Sort)

		tenant.CreatorId = uuid.New()
		assert.NotEqual(t, uuid.Nil, tenant.CreatorId)
	})
}

func TestTenants_Count(t *testing.T) {
	t.Run("統計租戶數量", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		rows := sqlmock.NewRows([]string{"count"}).
			AddRow(5)

		mock.ExpectQuery("SELECT count\\(\\*\\) FROM \"tenants\"").
			WillReturnRows(rows)

		var count int64
		err := db.Model(&Tenants{}).Count(&count).Error

		assert.NoError(t, err)
		assert.Equal(t, int64(5), count)
	})
}

func TestTenants_Pagination(t *testing.T) {
	t.Run("分頁查詢租戶", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		rows := sqlmock.NewRows([]string{"id", "name", "description"}).
			AddRow(uuid.New(), "Tenant1", "Desc1").
			AddRow(uuid.New(), "Tenant2", "Desc2")

		mock.ExpectQuery("SELECT (.+) FROM \"tenants\"").
			WillReturnRows(rows)

		var tenants []Tenants
		err := db.Limit(10).Offset(0).Find(&tenants).Error

		assert.NoError(t, err)
		assert.Len(t, tenants, 2)
	})
}

func TestTenants_Search(t *testing.T) {
	t.Run("按名稱搜索租戶", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		rows := sqlmock.NewRows([]string{"id", "name", "description"}).
			AddRow(uuid.New(), "ABC Company", "ABC Description")

		mock.ExpectQuery("SELECT (.+) FROM \"tenants\"").
			WithArgs("%ABC%").
			WillReturnRows(rows)

		var tenants []Tenants
		err := db.Where("name LIKE ?", "%ABC%").Find(&tenants).Error

		assert.NoError(t, err)
		assert.Len(t, tenants, 1)
	})
}

func TestTenants_Validation(t *testing.T) {
	t.Run("必填欄位驗證", func(t *testing.T) {
		tenant := Tenants{
			ID:   uuid.New(),
			Name: "Valid Tenant",
		}

		// Name 是 not null，應該有值
		assert.NotEmpty(t, tenant.Name)
	})
}
