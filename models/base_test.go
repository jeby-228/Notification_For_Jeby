package models

import (
	"member_API/testutil"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestBase_Structure(t *testing.T) {
	base := Base{
		ID:             uuid.New(),
		Sort:           1,
		CreationTime:   time.Now(),
		CreatorId:      uuid.New(),
		LastModifierId: uuid.New(),
		IsDeleted:      false,
	}

	assert.NotEqual(t, uuid.Nil, base.ID)
	assert.Equal(t, 1, base.Sort)
	assert.False(t, base.IsDeleted)
	assert.Nil(t, base.DeletedAt)
}

func TestBase_BeforeCreate(t *testing.T) {
	t.Run("自動生成 UUID", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		base := Base{}
		assert.Equal(t, uuid.Nil, base.ID, "初始 ID 應該為 nil")

		mock.ExpectBegin()
		mock.ExpectExec("INSERT").
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := db.Create(&base).Error
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, base.ID, "創建後 ID 應該自動生成")
	})

	t.Run("保留已設定的 UUID", func(t *testing.T) {
		db, mock := testutil.SetupTestDB(t)

		existingID := uuid.New()
		base := Base{ID: existingID}

		mock.ExpectBegin()
		mock.ExpectExec("INSERT").
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := db.Create(&base).Error
		assert.NoError(t, err)
		assert.Equal(t, existingID, base.ID, "應該保留原有的 ID")
	})
}

func TestBase_Timestamps(t *testing.T) {
	t.Run("創建時間自動設置", func(t *testing.T) {
		before := time.Now()
		base := Base{}
		base.CreationTime = time.Now()
		after := time.Now()

		assert.True(t, base.CreationTime.After(before) || base.CreationTime.Equal(before))
		assert.True(t, base.CreationTime.Before(after) || base.CreationTime.Equal(after))
	})

	t.Run("最後修改時間可更新", func(t *testing.T) {
		base := Base{}
		modTime := time.Now()
		base.LastModificationTime = &modTime

		assert.NotNil(t, base.LastModificationTime)
		assert.Equal(t, modTime, *base.LastModificationTime)
	})

	t.Run("刪除時間初始為 nil", func(t *testing.T) {
		base := Base{}
		assert.Nil(t, base.DeletedAt)
	})

	t.Run("設置刪除時間", func(t *testing.T) {
		base := Base{}
		deleteTime := time.Now()
		base.DeletedAt = &deleteTime
		base.IsDeleted = true

		assert.NotNil(t, base.DeletedAt)
		assert.True(t, base.IsDeleted)
		assert.Equal(t, deleteTime, *base.DeletedAt)
	})
}

func TestBase_SoftDelete(t *testing.T) {
	t.Run("軟刪除標記", func(t *testing.T) {
		base := Base{
			ID:        uuid.New(),
			IsDeleted: false,
		}

		assert.False(t, base.IsDeleted)
		assert.Nil(t, base.DeletedAt)

		deleteTime := time.Now()
		base.IsDeleted = true
		base.DeletedAt = &deleteTime

		assert.True(t, base.IsDeleted)
		assert.NotNil(t, base.DeletedAt)
	})
}

func TestBase_CreatorAndModifier(t *testing.T) {
	t.Run("設置創建者和修改者", func(t *testing.T) {
		creatorID := uuid.New()
		modifierID := uuid.New()

		base := Base{
			CreatorId:      creatorID,
			LastModifierId: modifierID,
		}

		assert.Equal(t, creatorID, base.CreatorId)
		assert.Equal(t, modifierID, base.LastModifierId)
	})

	t.Run("創建者和修改者可以相同", func(t *testing.T) {
		userID := uuid.New()

		base := Base{
			CreatorId:      userID,
			LastModifierId: userID,
		}

		assert.Equal(t, base.CreatorId, base.LastModifierId)
	})
}

func TestBase_Sort(t *testing.T) {
	t.Run("排序欄位", func(t *testing.T) {
		tests := []struct {
			name string
			sort int
		}{
			{"零值", 0},
			{"正數", 100},
			{"負數", -1},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				base := Base{Sort: tt.sort}
				assert.Equal(t, tt.sort, base.Sort)
			})
		}
	})
}

func TestBase_DefaultValues(t *testing.T) {
	t.Run("預設值", func(t *testing.T) {
		base := Base{}

		assert.Equal(t, uuid.Nil, base.ID)
		assert.Equal(t, 0, base.Sort)
		assert.False(t, base.IsDeleted)
		assert.Nil(t, base.DeletedAt)
		assert.Nil(t, base.LastModificationTime)
	})
}

func TestBase_JSONSerialization(t *testing.T) {
	t.Run("IsDeleted 不應該序列化", func(t *testing.T) {
		base := Base{
			ID:        uuid.New(),
			IsDeleted: true,
		}

		// IsDeleted 有 json:"-" tag，不應該被序列化
		assert.True(t, base.IsDeleted)
	})

	t.Run("DeletedAt 不應該序列化", func(t *testing.T) {
		deleteTime := time.Now()
		base := Base{
			ID:        uuid.New(),
			DeletedAt: &deleteTime,
		}

		// DeletedAt 有 json:"-" tag，不應該被序列化
		assert.NotNil(t, base.DeletedAt)
	})
}
