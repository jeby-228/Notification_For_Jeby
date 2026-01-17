package services

import (
	"fmt"
	"member_API/testutil"
	"testing"

	"github.com/google/uuid"
)

func TestMemberService_CreateMember(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer func() {
		if err := testutil.CleanupTestDB(db); err != nil {
			t.Errorf("Failed to cleanup test database: %v", err)
		}
	}()

	service := NewMemberService(db)

	tests := []struct {
		name      string
		userName  string
		email     string
		password  string
		creatorId uuid.UUID
		wantErr   bool
	}{
		{
			name:      "Valid member creation",
			userName:  "John Doe",
			email:     "john@example.com",
			password:  "password123",
			creatorId: uuid.New(),
			wantErr:   false,
		},
		{
			name:      "Duplicate email",
			userName:  "Jane Doe",
			email:     "john@example.com",
			password:  "password123",
			creatorId: uuid.New(),
			wantErr:   true,
		},
		{
			name:      "Valid member with different email",
			userName:  "Jane Smith",
			email:     "jane@example.com",
			password:  "password456",
			creatorId: uuid.New(),
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			member, err := service.CreateMember(tt.userName, tt.email, tt.password, tt.creatorId)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateMember() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if member == nil {
					t.Error("CreateMember() returned nil member")
					return
				}
				if member.Email != tt.email {
					t.Errorf("CreateMember() Email = %v, want %v", member.Email, tt.email)
				}
				if member.Name != tt.userName {
					t.Errorf("CreateMember() Name = %v, want %v", member.Name, tt.userName)
				}
				if member.PasswordHash == "" {
					t.Error("CreateMember() PasswordHash is empty")
				}
				if member.PasswordHash == tt.password {
					t.Error("CreateMember() password should be hashed")
				}
			}
		})
	}
}

func TestMemberService_GetMemberByID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer func() {
		if err := testutil.CleanupTestDB(db); err != nil {
			t.Errorf("Failed to cleanup test database: %v", err)
		}
	}()

	service := NewMemberService(db)

	// Create a test member
	created, err := service.CreateMember("Test User", "test@example.com", "password", uuid.New())
	if err != nil {
		t.Fatalf("Failed to create test member: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "Existing member",
			id:      created.ID,
			wantErr: false,
		},
		{
			name:    "Non-existing member",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			member, err := service.GetMemberByID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMemberByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if member == nil {
					t.Error("GetMemberByID() returned nil member")
					return
				}
				if member.ID != tt.id {
					t.Errorf("GetMemberByID() ID = %v, want %v", member.ID, tt.id)
				}
			}
		})
	}
}

func TestMemberService_GetMembers(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer func() {
		if err := testutil.CleanupTestDB(db); err != nil {
			t.Errorf("Failed to cleanup test database: %v", err)
		}
	}()

	service := NewMemberService(db)

	// Create multiple test members
	for i := 0; i < 5; i++ {
		email := fmt.Sprintf("test%d@example.com", i)
		_, err := service.CreateMember("Test User", email, "password", uuid.New())
		if err != nil {
			t.Fatalf("Failed to create test member: %v", err)
		}
	}

	tests := []struct {
		name      string
		limit     int
		wantCount int
		wantErr   bool
	}{
		{
			name:      "Get all members",
			limit:     10,
			wantCount: 5,
			wantErr:   false,
		},
		{
			name:      "Get limited members",
			limit:     3,
			wantCount: 3,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			members, err := service.GetMembers(tt.limit)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMembers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(members) != tt.wantCount {
					t.Errorf("GetMembers() count = %v, want %v", len(members), tt.wantCount)
				}
			}
		})
	}
}

func TestMemberService_UpdateMember(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer func() {
		if err := testutil.CleanupTestDB(db); err != nil {
			t.Errorf("Failed to cleanup test database: %v", err)
		}
	}()

	service := NewMemberService(db)

	// Create a test member
	created, err := service.CreateMember("Test User", "test@example.com", "password", uuid.New())
	if err != nil {
		t.Fatalf("Failed to create test member: %v", err)
	}

	modifierID := uuid.New()

	tests := []struct {
		name       string
		id         uuid.UUID
		newName    string
		newEmail   string
		modifierId uuid.UUID
		wantErr    bool
	}{
		{
			name:       "Valid update",
			id:         created.ID,
			newName:    "Updated Name",
			newEmail:   "updated@example.com",
			modifierId: modifierID,
			wantErr:    false,
		},
		{
			name:       "Update non-existing member",
			id:         uuid.New(),
			newName:    "Name",
			newEmail:   "email@example.com",
			modifierId: modifierID,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			member, err := service.UpdateMember(tt.id, tt.newName, tt.newEmail, tt.modifierId)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateMember() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if member == nil {
					t.Error("UpdateMember() returned nil member")
					return
				}
				if member.Name != tt.newName {
					t.Errorf("UpdateMember() Name = %v, want %v", member.Name, tt.newName)
				}
				if member.Email != tt.newEmail {
					t.Errorf("UpdateMember() Email = %v, want %v", member.Email, tt.newEmail)
				}
				if member.LastModifierId != tt.modifierId {
					t.Errorf("UpdateMember() LastModifierId = %v, want %v", member.LastModifierId, tt.modifierId)
				}
				if member.LastModificationTime == nil {
					t.Error("UpdateMember() LastModificationTime is nil")
				}
			}
		})
	}
}

func TestMemberService_DeleteMember(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer func() {
		if err := testutil.CleanupTestDB(db); err != nil {
			t.Errorf("Failed to cleanup test database: %v", err)
		}
	}()

	service := NewMemberService(db)

	// Create a test member
	created, err := service.CreateMember("Test User", "test@example.com", "password", uuid.New())
	if err != nil {
		t.Fatalf("Failed to create test member: %v", err)
	}

	deleterID := uuid.New()

	tests := []struct {
		name      string
		id        uuid.UUID
		deleterId uuid.UUID
		wantErr   bool
	}{
		{
			name:      "Valid delete",
			id:        created.ID,
			deleterId: deleterID,
			wantErr:   false,
		},
		{
			name:      "Delete non-existing member",
			id:        uuid.New(),
			deleterId: deleterID,
			wantErr:   true,
		},
		{
			name:      "Delete already deleted member",
			id:        created.ID,
			deleterId: deleterID,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.DeleteMember(tt.id, tt.deleterId)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteMember() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Verify the member is marked as deleted (soft delete)
			if !tt.wantErr {
				member, err := service.GetMemberByID(tt.id)
				if err == nil {
					t.Errorf("DeleteMember() member should not be retrievable after deletion, got member: %+v", member)
				}
			}
		})
	}
}
