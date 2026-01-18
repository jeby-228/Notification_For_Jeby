package services

import (
	"encoding/json"
	"errors"
	"time"

	"member_API/auth"
	"member_API/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NotificationProviderService struct {
	DB *gorm.DB
}

func NewNotificationProviderService(db *gorm.DB) *NotificationProviderService {
	return &NotificationProviderService{DB: db}
}

// encryptConfig 加密配置中的敏感資料
func (s *NotificationProviderService) encryptConfig(providerType models.ProviderType, configJSON string) (string, error) {
	if configJSON == "" {
		return "", nil
	}

	var config map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return "", err
	}

	// 根據提供者類型加密敏感欄位
	switch providerType {
	case models.ProviderSMTP:
		if password, ok := config["password"].(string); ok && password != "" {
			encrypted, err := auth.Encrypt(password)
			if err != nil {
				return "", err
			}
			config["password"] = encrypted
		}
	case models.ProviderSMS:
		if authToken, ok := config["auth_token"].(string); ok && authToken != "" {
			encrypted, err := auth.Encrypt(authToken)
			if err != nil {
				return "", err
			}
			config["auth_token"] = encrypted
		}
	case models.ProviderFirebase:
		if credentialJSON, ok := config["credential_json"].(string); ok && credentialJSON != "" {
			encrypted, err := auth.Encrypt(credentialJSON)
			if err != nil {
				return "", err
			}
			config["credential_json"] = encrypted
		}
		if serverKey, ok := config["server_key"].(string); ok && serverKey != "" {
			encrypted, err := auth.Encrypt(serverKey)
			if err != nil {
				return "", err
			}
			config["server_key"] = encrypted
		}
	}

	encryptedJSON, err := json.Marshal(config)
	if err != nil {
		return "", err
	}

	return string(encryptedJSON), nil
}

// decryptConfig 解密配置中的敏感資料
func (s *NotificationProviderService) decryptConfig(providerType models.ProviderType, configJSON string) (string, error) {
	if configJSON == "" {
		return "", nil
	}

	var config map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return "", err
	}

	// 根據提供者類型解密敏感欄位
	switch providerType {
	case models.ProviderSMTP:
		if password, ok := config["password"].(string); ok && password != "" {
			decrypted, err := auth.Decrypt(password)
			if err != nil {
				return "", err
			}
			config["password"] = decrypted
		}
	case models.ProviderSMS:
		if authToken, ok := config["auth_token"].(string); ok && authToken != "" {
			decrypted, err := auth.Decrypt(authToken)
			if err != nil {
				return "", err
			}
			config["auth_token"] = decrypted
		}
	case models.ProviderFirebase:
		if credentialJSON, ok := config["credential_json"].(string); ok && credentialJSON != "" {
			decrypted, err := auth.Decrypt(credentialJSON)
			if err != nil {
				return "", err
			}
			config["credential_json"] = decrypted
		}
		if serverKey, ok := config["server_key"].(string); ok && serverKey != "" {
			decrypted, err := auth.Decrypt(serverKey)
			if err != nil {
				return "", err
			}
			config["server_key"] = decrypted
		}
	}

	decryptedJSON, err := json.Marshal(config)
	if err != nil {
		return "", err
	}

	return string(decryptedJSON), nil
}

// CreateProvider 建立新的通知提供者
func (s *NotificationProviderService) CreateProvider(tenantsID uuid.UUID, name string, providerType models.ProviderType, config string, creatorID uuid.UUID) (*models.NotificationProvider, error) {
	// 加密敏感資料
	encryptedConfig, err := s.encryptConfig(providerType, config)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	provider := &models.NotificationProvider{
		Base: models.Base{
			CreationTime: now,
			CreatorId:    creatorID,
			IsDeleted:    false,
		},
		TenantsID: tenantsID,
		Name:      name,
		Type:      providerType,
		Config:    encryptedConfig,
		IsActive:  true,
	}

	if err := s.DB.Create(provider).Error; err != nil {
		return nil, err
	}

	return provider, nil
}

// GetProviderByID 取得單一提供者
func (s *NotificationProviderService) GetProviderByID(id uuid.UUID) (*models.NotificationProvider, error) {
	var provider models.NotificationProvider
	if err := s.DB.Where("is_deleted = ?", false).First(&provider, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("提供者不存在")
		}
		return nil, err
	}

	return &provider, nil
}

// GetProviderByIDWithDecryption 取得單一提供者並解密敏感資料
func (s *NotificationProviderService) GetProviderByIDWithDecryption(id uuid.UUID) (*models.NotificationProvider, error) {
	provider, err := s.GetProviderByID(id)
	if err != nil {
		return nil, err
	}

	// 解密配置
	decryptedConfig, err := s.decryptConfig(provider.Type, provider.Config)
	if err != nil {
		return nil, err
	}
	provider.Config = decryptedConfig

	return provider, nil
}

// GetProvidersByTenantID 取得租戶的所有提供者
func (s *NotificationProviderService) GetProvidersByTenantID(tenantsID uuid.UUID) ([]models.NotificationProvider, error) {
	var providers []models.NotificationProvider
	if err := s.DB.Where("tenants_id = ? AND is_deleted = ?", tenantsID, false).Find(&providers).Error; err != nil {
		return nil, err
	}
	return providers, nil
}

// UpdateProvider 更新提供者
func (s *NotificationProviderService) UpdateProvider(id uuid.UUID, name string, providerType models.ProviderType, config string, isActive bool, modifierID uuid.UUID) (*models.NotificationProvider, error) {
	var provider models.NotificationProvider
	if err := s.DB.Where("is_deleted = ?", false).First(&provider, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("提供者不存在")
		}
		return nil, err
	}

	// 加密敏感資料
	encryptedConfig, err := s.encryptConfig(providerType, config)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	provider.Name = name
	provider.Type = providerType
	provider.Config = encryptedConfig
	provider.IsActive = isActive
	provider.LastModificationTime = &now
	provider.LastModifierId = modifierID

	if err := s.DB.Save(&provider).Error; err != nil {
		return nil, err
	}

	return &provider, nil
}

// DeleteProvider 軟刪除提供者
func (s *NotificationProviderService) DeleteProvider(id uuid.UUID, deleterID uuid.UUID) error {
	now := time.Now()
	result := s.DB.Model(&models.NotificationProvider{}).
		Where("id = ? AND is_deleted = ?", id, false).
		Updates(map[string]interface{}{
			"is_deleted":             true,
			"deleted_at":             &now,
			"last_modifier_id":       deleterID,
			"last_modification_time": &now,
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("提供者不存在或已被刪除")
	}

	return nil
}

// TestProvider 測試提供者配置是否有效
func (s *NotificationProviderService) TestProvider(id uuid.UUID) (bool, string, error) {
	provider, err := s.GetProviderByIDWithDecryption(id)
	if err != nil {
		return false, "", err
	}

	// 驗證配置格式
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(provider.Config), &config); err != nil {
		return false, "配置格式無效", nil
	}

	// 根據提供者類型驗證必要欄位
	switch provider.Type {
	case models.ProviderSMTP:
		requiredFields := []string{"host", "port", "username", "password", "from"}
		for _, field := range requiredFields {
			if _, ok := config[field]; !ok {
				return false, "缺少必要欄位: " + field, nil
			}
		}
	case models.ProviderSMS:
		requiredFields := []string{"provider", "account_id", "auth_token", "from_phone"}
		for _, field := range requiredFields {
			if _, ok := config[field]; !ok {
				return false, "缺少必要欄位: " + field, nil
			}
		}
	case models.ProviderFirebase:
		requiredFields := []string{"project_id", "credential_json"}
		for _, field := range requiredFields {
			if _, ok := config[field]; !ok {
				return false, "缺少必要欄位: " + field, nil
			}
		}
	}

	return true, "配置有效", nil
}
