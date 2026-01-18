package services

import (
	"member_API/testutil"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestCreateMember(t *testing.T) {
	tests := []struct {
		name      string
		inputName string
		email     string
		password  string
		creatorID uuid.UUID
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "成功建立會員",
			inputName: "Test User",
			email:     "test@example.com",
			password:  "password123",
			creatorID: uuid.New(),
			wantErr:   false,
		},
		{
			name:      "Email 已存在",
			inputName: "Test User",
			email:     "existing@example.com",
			password:  "password123",
			creatorID: uuid.New(),
			wantErr:   true,
			errMsg:    "email 已被使用",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := testutil.SetupTestDB(t)
			service := NewMemberService(db)

			if tt.wantErr && tt.errMsg == "email 已被使用" {
				rows := sqlmock.NewRows([]string{"id", "email"}).
					AddRow(uuid.New(), tt.email)
				mock.ExpectQuery(`SELECT \* FROM "members"`).
					WithArgs(tt.email, false, 1).
					WillReturnRows(rows)
			} else {
				mock.ExpectQuery(`SELECT \* FROM "members"`).
					WithArgs(tt.email, false, 1).
					WillReturnError(gorm.ErrRecordNotFound)

				mock.ExpectBegin()
				mock.ExpectExec(`INSERT INTO "members"`).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			}

			member, err := service.CreateMember(tt.inputName, tt.email, tt.password, tt.creatorID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, member)
			} else {
				assert.NoError(t, err)
				if member != nil {
					assert.Equal(t, tt.inputName, member.Name)
					assert.Equal(t, tt.email, member.Email)
					assert.NotEmpty(t, member.PasswordHash)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Unfulfilled expectations: %v", err)
			}
		})
	}
}

func TestUpdateMember(t *testing.T) {
	memberID := uuid.New()
	modifierID := uuid.New()
	now := time.Now()

	tests := []struct {
		name       string
		memberID   uuid.UUID
		inputName  string
		email      string
		modifierID uuid.UUID
		setupMock  func(sqlmock.Sqlmock)
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "成功更新會員",
			memberID:   memberID,
			inputName:  "Updated Name",
			email:      "updated@example.com",
			modifierID: modifierID,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "name", "email", "password_hash", "tenants_id", "api_key",
					"creation_time", "creator_id", "is_deleted",
				}).AddRow(
					memberID, "Old Name", "old@example.com", "hash", uuid.New(), "key",
					now, uuid.New(), false,
				)
				mock.ExpectQuery(`SELECT \* FROM "members"`).
					WithArgs(false, memberID, 1).
					WillReturnRows(rows)

				mock.ExpectBegin()
				mock.ExpectExec(`UPDATE "members"`).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			wantErr: false,
		},
		{
			name:       "會員不存在",
			memberID:   uuid.New(),
			inputName:  "Test",
			email:      "test@example.com",
			modifierID: modifierID,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT \* FROM "members"`).
					WithArgs(false, sqlmock.AnyArg(), 1).
					WillReturnError(gorm.ErrRecordNotFound)
			},
			wantErr: true,
			errMsg:  "會員不存在",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := testutil.SetupTestDB(t)
			service := NewMemberService(db)

			tt.setupMock(mock)

			member, err := service.UpdateMember(tt.memberID, tt.inputName, tt.email, tt.modifierID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, member)
			} else {
				assert.NoError(t, err)
				if member != nil {
					assert.Equal(t, tt.inputName, member.Name)
					assert.Equal(t, tt.email, member.Email)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Unfulfilled expectations: %v", err)
			}
		})
	}
}

func TestDeleteMember(t *testing.T) {
	memberID := uuid.New()
	deleterID := uuid.New()

	tests := []struct {
		name      string
		memberID  uuid.UUID
		deleterID uuid.UUID
		setupMock func(sqlmock.Sqlmock)
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "成功刪除會員",
			memberID:  memberID,
			deleterID: deleterID,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`UPDATE "members"`).
					WithArgs(sqlmock.AnyArg(), true, sqlmock.AnyArg(), deleterID, memberID, false).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			wantErr: false,
		},
		{
			name:      "會員不存在或已被刪除",
			memberID:  uuid.New(),
			deleterID: deleterID,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`UPDATE "members"`).
					WithArgs(sqlmock.AnyArg(), true, sqlmock.AnyArg(), deleterID, sqlmock.AnyArg(), false).
					WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectCommit()
			},
			wantErr: true,
			errMsg:  "會員不存在或已被刪除",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := testutil.SetupTestDB(t)
			service := NewMemberService(db)

			tt.setupMock(mock)

			err := service.DeleteMember(tt.memberID, tt.deleterID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetMemberByID(t *testing.T) {
	memberID := uuid.New()
	now := time.Now()

	tests := []struct {
		name      string
		memberID  uuid.UUID
		setupMock func(sqlmock.Sqlmock)
		wantErr   bool
		errMsg    string
	}{
		{
			name:     "成功取得會員",
			memberID: memberID,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "name", "email", "password_hash", "tenants_id", "api_key",
					"creation_time", "creator_id", "is_deleted",
				}).AddRow(
					memberID, "Test User", "test@example.com", "hash", uuid.New(), "key",
					now, uuid.New(), false,
				)
				mock.ExpectQuery(`SELECT \* FROM "members"`).
					WithArgs(false, memberID, 1).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name:     "會員不存在",
			memberID: uuid.New(),
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT \* FROM "members"`).
					WithArgs(false, sqlmock.AnyArg(), 1).
					WillReturnError(gorm.ErrRecordNotFound)
			},
			wantErr: true,
			errMsg:  "會員不存在",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := testutil.SetupTestDB(t)
			service := NewMemberService(db)

			tt.setupMock(mock)

			member, err := service.GetMemberByID(tt.memberID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				assert.Nil(t, member)
			} else {
				assert.NoError(t, err)
				if member != nil {
					assert.Equal(t, tt.memberID, member.ID)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Unfulfilled expectations: %v", err)
			}
		})
	}
}

func TestGetMembers(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		limit     int
		setupMock func(sqlmock.Sqlmock)
		wantErr   bool
		wantCount int
	}{
		{
			name:  "成功取得會員列表",
			limit: 10,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "name", "email", "password_hash", "tenants_id", "api_key",
					"creation_time", "creator_id", "is_deleted",
				}).
					AddRow(uuid.New(), "User 1", "user1@example.com", "hash", uuid.New(), "key1", now, uuid.New(), false).
					AddRow(uuid.New(), "User 2", "user2@example.com", "hash", uuid.New(), "key2", now, uuid.New(), false)

				mock.ExpectQuery(`SELECT \* FROM "members"`).
					WithArgs(false, 10).
					WillReturnRows(rows)
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:  "取得空列表",
			limit: 10,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "name", "email", "password_hash", "tenants_id", "api_key",
					"creation_time", "creator_id", "is_deleted",
				})
				mock.ExpectQuery(`SELECT \* FROM "members"`).
					WithArgs(false, 10).
					WillReturnRows(rows)
			},
			wantErr:   false,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := testutil.SetupTestDB(t)
			service := NewMemberService(db)

			tt.setupMock(mock)

			members, err := service.GetMembers(tt.limit)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, members)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, members)
				assert.Equal(t, tt.wantCount, len(members))
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Unfulfilled expectations: %v", err)
			}
		})
	}
}
