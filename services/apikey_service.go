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

// RegenerateAPIKey 重新生成指定會員的 API Key
// 參數：
//   - memberID: 會員的 UUID
//
// 回傳值：
//   - *models.Member: 更新後的會員資料，包含新的 API Key
//   - error: 錯誤資訊，可能的錯誤包括：
//   - "會員不存在": 找不到指定的會員或會員已被刪除
//   - 資料庫錯誤
//
// 注意：舊的 API Key 將立即失效
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

// ValidateAPIKey 驗證 API Key 並返回對應的會員和租戶資訊
// 參數：
//   - apiKey: 要驗證的 API Key 字串
//
// 回傳值：
//   - *models.Member: 對應的會員資料，若 API Key 無效則為 nil
//   - *models.Tenants: 對應的租戶資料，可能為 nil（會員沒有租戶或租戶不存在）
//   - error: 錯誤資訊，可能的錯誤包括：
//   - "無效的 API Key": 找不到對應的會員或會員已被刪除
//   - 資料庫錯誤
//
// 注意：
//   - 即使會員沒有關聯的租戶（TenantsID 為 uuid.Nil），也會成功返回會員資料
//   - 即使租戶不存在，也會成功返回會員資料（tenant 為 nil）
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
