package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"member_API/models"
	"regexp"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	MaxSMSLength  = 160
	MaxRetries    = 3
	RetryDelayMs  = 1000
	PhoneRegexStr = `^(\+?[1-9]\d{1,14}|0\d{9,14})$`
)

type SMSProvider interface {
	SendSMS(fromPhone, toPhone, body string) error
}

type MockSMSProvider struct{}

func (m *MockSMSProvider) SendSMS(fromPhone, toPhone, body string) error {
	return nil
}

type SMSService struct {
	DB       *gorm.DB
	Provider SMSProvider
}

func NewSMSService(db *gorm.DB, provider SMSProvider) *SMSService {
	if provider == nil {
		provider = &MockSMSProvider{}
	}
	return &SMSService{
		DB:       db,
		Provider: provider,
	}
}

func ValidatePhoneNumber(phone string) error {
	if phone == "" {
		return errors.New("電話號碼不能為空")
	}

	phoneRegex := regexp.MustCompile(PhoneRegexStr)
	if !phoneRegex.MatchString(phone) {
		return errors.New("電話號碼格式無效")
	}

	return nil
}

func ValidateSMSLength(body string) error {
	if body == "" {
		return errors.New("簡訊內容不能為空")
	}

	if len(body) > MaxSMSLength {
		return fmt.Errorf("簡訊內容超過 %d 字元限制", MaxSMSLength)
	}

	return nil
}

func (s *SMSService) SendSMS(memberID uuid.UUID, providerID uuid.UUID, recipientPhone string, body string) (*models.NotificationLog, error) {
	if err := ValidatePhoneNumber(recipientPhone); err != nil {
		return nil, err
	}

	if err := ValidateSMSLength(body); err != nil {
		return nil, err
	}

	var provider models.NotificationProvider
	if err := s.DB.First(&provider, providerID).Error; err != nil {
		return nil, errors.New("找不到通知提供者")
	}

	if provider.Type != models.ProviderSMS {
		return nil, errors.New("提供者類型不是 SMS")
	}

	if !provider.IsActive {
		return nil, errors.New("SMS 提供者未啟用")
	}

	var smsConfig models.SMSConfig
	if err := json.Unmarshal([]byte(provider.Config), &smsConfig); err != nil {
		return nil, errors.New("SMS 配置無效")
	}

	log := &models.NotificationLog{
		MemberID:       memberID,
		ProviderID:     providerID,
		Type:           models.ProviderSMS,
		RecipientPhone: recipientPhone,
		Body:           body,
		Status:         models.StatusPending,
	}

	if err := s.DB.Create(log).Error; err != nil {
		return nil, err
	}

	var lastErr error
	for i := 0; i < MaxRetries; i++ {
		if i > 0 {
			time.Sleep(time.Duration(RetryDelayMs) * time.Millisecond)
		}

		err := s.Provider.SendSMS(smsConfig.FromPhone, recipientPhone, body)
		if err == nil {
			now := time.Now()
			log.Status = models.StatusSent
			log.SentAt = &now
			s.DB.Save(log)
			return log, nil
		}

		lastErr = err
	}

	log.Status = models.StatusFailed
	log.ErrorMsg = fmt.Sprintf("發送失敗: %v", lastErr)
	s.DB.Save(log)

	return nil, fmt.Errorf("發送簡訊失敗，已重試 %d 次: %v", MaxRetries, lastErr)
}
