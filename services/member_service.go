package services

import (
	"errors"
	"member_API/auth"
	"member_API/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MemberService struct {
	DB *gorm.DB
}

func NewMemberService(db *gorm.DB) *MemberService {
	return &MemberService{DB: db}
}

// CreateMember 建立新會員
func (s *MemberService) CreateMember(name, email, password string, creatorId uuid.UUID) (*models.Member, error) {
	// 檢查 email 是否已存在
	var exists models.Member
	if err := s.DB.Where("email = ? AND is_deleted = ?", email, false).First(&exists).Error; err == nil {
		return nil, errors.New("email 已被使用")
	}

	// 加密密碼
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	member := &models.Member{
		Base: models.Base{
			CreationTime: now,
			CreatorId:    creatorId,
			IsDeleted:    false,
		},
		Name:         name,
		Email:        email,
		PasswordHash: hash,
	}

	if err := s.DB.Create(member).Error; err != nil {
		return nil, err
	}

	return member, nil
}

// UpdateMember 更新會員資訊
func (s *MemberService) UpdateMember(id uuid.UUID, name, email string, modifierId uuid.UUID) (*models.Member, error) {
	var member models.Member
	if err := s.DB.Where("is_deleted = ?", false).First(&member, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("會員不存在")
		}
		return nil, err
	}

	now := time.Now()
	member.Name = name
	member.Email = email
	member.LastModificationTime = &now
	member.LastModifierId = modifierId

	if err := s.DB.Save(&member).Error; err != nil {
		return nil, err
	}

	return &member, nil
}

// DeleteMember 軟刪除會員
func (s *MemberService) DeleteMember(id uuid.UUID, deleterId uuid.UUID) error {
	now := time.Now()
	result := s.DB.Model(&models.Member{}).
		Where("id = ? AND is_deleted = ?", id, false).
		Updates(map[string]interface{}{
			"is_deleted":             true,
			"deleted_at":             &now,
			"last_modifier_id":       deleterId,
			"last_modification_time": &now,
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("會員不存在或已被刪除")
	}

	return nil
}

// GetMemberByID 取得單一會員
func (s *MemberService) GetMemberByID(id uuid.UUID) (*models.Member, error) {
	var member models.Member
	if err := s.DB.Where("is_deleted = ?", false).First(&member, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("會員不存在")
		}
		return nil, err
	}
	return &member, nil
}

// GetMembers 取得會員列表
func (s *MemberService) GetMembers(limit int) ([]models.Member, error) {
	var members []models.Member
	if err := s.DB.Where("is_deleted = ?", false).Limit(limit).Find(&members).Error; err != nil {
		return nil, err
	}
	return members, nil
}
