package services

import (
	"errors"
	"member_API/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type APIKeyService struct {
	DB *gorm.DB
}

func NewAPIKeyService(db *gorm.DB) *APIKeyService {
	return &APIKeyService{DB: db}
}

func (s *APIKeyService) RegenerateAPIKey(memberID uuid.UUID) (*models.Member, error) {
	var member models.Member
	if err := s.DB.Where("id = ? AND is_deleted = ?", memberID, false).First(&member).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("會員不存在")
		}
		return nil, err
	}

	newKey, err := GenerateAPIKey()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	member.APIKey = newKey
	member.LastModificationTime = &now
	member.LastModifierId = memberID

	if err := s.DB.Save(&member).Error; err != nil {
		return nil, err
	}

	return &member, nil
}

func (s *APIKeyService) ValidateAPIKey(apiKey string) (*models.Member, *models.Tenants, error) {
	var member models.Member
	if err := s.DB.Where("api_key = ? AND is_deleted = ?", apiKey, false).First(&member).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, errors.New("無效的 API Key")
		}
		return nil, nil, err
	}

	if member.TenantsID == uuid.Nil {
		return &member, nil, nil
	}

	var tenant models.Tenants
	if err := s.DB.First(&tenant, member.TenantsID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &member, nil, nil
		}
		return &member, nil, err
	}

	return &member, &tenant, nil
}
