package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"member_API/models"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/google/uuid"
	"google.golang.org/api/option"
	"gorm.io/gorm"
)

// FirebaseService 管理 Firebase 推播服務
type FirebaseService struct {
	DB       *gorm.DB
	apps     map[uuid.UUID]*firebase.App // tenant_id -> firebase app
	clients  map[uuid.UUID]*messaging.Client
	mu       sync.RWMutex
}

// NewFirebaseService 建立新的 Firebase 服務
func NewFirebaseService(db *gorm.DB) *FirebaseService {
	return &FirebaseService{
		DB:      db,
		apps:    make(map[uuid.UUID]*firebase.App),
		clients: make(map[uuid.UUID]*messaging.Client),
	}
}

// getOrCreateClient 取得或建立 Firebase Client
func (s *FirebaseService) getOrCreateClient(ctx context.Context, tenantID uuid.UUID) (*messaging.Client, error) {
	s.mu.RLock()
	if client, exists := s.clients[tenantID]; exists {
		s.mu.RUnlock()
		return client, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	// 再次檢查，避免重複建立
	if client, exists := s.clients[tenantID]; exists {
		return client, nil
	}

	// 從資料庫載入 Firebase 設定
	var provider models.NotificationProvider
	if err := s.DB.Where("tenants_id = ? AND type = ? AND is_active = ?", tenantID, models.ProviderFirebase, true).
		First(&provider).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("firebase provider not found for tenant")
		}
		return nil, err
	}

	// 解析 Firebase 設定
	var config models.FirebaseConfig
	if err := json.Unmarshal([]byte(provider.Config), &config); err != nil {
		return nil, fmt.Errorf("failed to parse firebase config: %w", err)
	}

	// 建立 Firebase App
	opt := option.WithCredentialsJSON([]byte(config.CredentialJSON))
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return nil, fmt.Errorf("failed to create firebase app: %w", err)
	}

	// 建立 Messaging Client
	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create messaging client: %w", err)
	}

	s.apps[tenantID] = app
	s.clients[tenantID] = client

	return client, nil
}

// ValidateToken 驗證 Device Token 是否有效
func (s *FirebaseService) ValidateToken(ctx context.Context, tenantID uuid.UUID, token string) (bool, error) {
	client, err := s.getOrCreateClient(ctx, tenantID)
	if err != nil {
		return false, err
	}

	// 嘗試發送空的測試訊息來驗證 token
	message := &messaging.Message{
		Token: token,
		Data: map[string]string{
			"test": "validation",
		},
	}

	// 使用 DryRun 模式測試
	_, err = client.Send(ctx, message)
	if err != nil {
		// 檢查錯誤類型
		if messaging.IsInvalidArgument(err) || messaging.IsRegistrationTokenNotRegistered(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// SendToToken 發送推播給單一設備
func (s *FirebaseService) SendToToken(ctx context.Context, tenantID uuid.UUID, memberID uuid.UUID, token, title, body string, data map[string]string) error {
	client, err := s.getOrCreateClient(ctx, tenantID)
	if err != nil {
		return err
	}

	message := &messaging.Message{
		Token: token,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: data,
	}

	// 發送訊息
	response, err := client.Send(ctx, message)
	
	now := time.Now()
	notificationLog := models.NotificationLog{
		MemberID:   memberID,
		Type:       models.ProviderFirebase,
		Subject:    title,
		Body:       body,
		Status:     models.StatusSent,
		SentAt:     &now,
	}

	// 取得 provider ID
	var provider models.NotificationProvider
	if err := s.DB.Where("tenants_id = ? AND type = ?", tenantID, models.ProviderFirebase).First(&provider).Error; err == nil {
		notificationLog.ProviderID = provider.ID
	}

	if err != nil {
		notificationLog.Status = models.StatusFailed
		notificationLog.ErrorMsg = err.Error()
		
		// 如果是無效 token，自動停用
		if messaging.IsInvalidArgument(err) || messaging.IsRegistrationTokenNotRegistered(err) {
			s.DeactivateToken(token)
		}
		
		// 記錄失敗日誌
		s.DB.Create(&notificationLog)
		return err
	}

	log.Printf("Successfully sent message: %s", response)
	
	// 記錄成功日誌
	s.DB.Create(&notificationLog)
	
	return nil
}

// SendToMultipleTokens 批次發送推播給多個設備
func (s *FirebaseService) SendToMultipleTokens(ctx context.Context, tenantID uuid.UUID, memberID uuid.UUID, tokens []string, title, body string, data map[string]string) (*messaging.BatchResponse, error) {
	client, err := s.getOrCreateClient(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// 建立多播訊息
	message := &messaging.MulticastMessage{
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data:   data,
		Tokens: tokens,
	}

	// 批次發送
	batchResponse, err := client.SendEachForMulticast(ctx, message)
	if err != nil {
		return nil, err
	}

	// 處理失敗的 token
	if batchResponse.FailureCount > 0 {
		for idx, response := range batchResponse.Responses {
			if !response.Success {
				// 檢查是否為無效 token
				if messaging.IsInvalidArgument(response.Error) || messaging.IsRegistrationTokenNotRegistered(response.Error) {
					s.DeactivateToken(tokens[idx])
				}
			}
		}
	}

	// 記錄通知日誌
	now := time.Now()
	var provider models.NotificationProvider
	s.DB.Where("tenants_id = ? AND type = ?", tenantID, models.ProviderFirebase).First(&provider)

	for _, response := range batchResponse.Responses {
		notificationLog := models.NotificationLog{
			MemberID:   memberID,
			ProviderID: provider.ID,
			Type:       models.ProviderFirebase,
			Subject:    title,
			Body:       body,
			Status:     models.StatusSent,
			SentAt:     &now,
		}

		if !response.Success {
			notificationLog.Status = models.StatusFailed
			notificationLog.ErrorMsg = response.Error.Error()
		}

		s.DB.Create(&notificationLog)
	}

	log.Printf("Successfully sent %d messages, %d failures", batchResponse.SuccessCount, batchResponse.FailureCount)

	return batchResponse, nil
}

// SendToMember 發送推播給會員的所有設備
func (s *FirebaseService) SendToMember(ctx context.Context, tenantID uuid.UUID, memberID uuid.UUID, title, body string, data map[string]string) error {
	// 查詢會員的所有啟用設備
	var deviceTokens []models.MemberDeviceToken
	if err := s.DB.Where("member_id = ? AND is_active = ?", memberID, true).Find(&deviceTokens).Error; err != nil {
		return err
	}

	if len(deviceTokens) == 0 {
		return errors.New("no active devices found for member")
	}

	// 提取 token 列表
	tokens := make([]string, 0, len(deviceTokens))
	for _, dt := range deviceTokens {
		tokens = append(tokens, dt.DeviceToken)
	}

	// 批次發送
	_, err := s.SendToMultipleTokens(ctx, tenantID, memberID, tokens, title, body, data)
	return err
}

// DeactivateToken 停用無效的 Device Token
func (s *FirebaseService) DeactivateToken(token string) error {
	result := s.DB.Model(&models.MemberDeviceToken{}).
		Where("device_token = ?", token).
		Update("is_active", false)
	
	if result.Error != nil {
		return result.Error
	}
	
	if result.RowsAffected > 0 {
		log.Printf("Deactivated invalid token: %s", token)
	}
	
	return nil
}

// RegisterDeviceToken 註冊新的設備 Token
func (s *FirebaseService) RegisterDeviceToken(memberID uuid.UUID, deviceToken, deviceType string) error {
	now := time.Now()
	
	// 使用 upsert 邏輯
	var existingToken models.MemberDeviceToken
	err := s.DB.Where("member_id = ? AND device_token = ?", memberID, deviceToken).First(&existingToken).Error
	
	if err == nil {
		// Token 已存在，更新狀態和最後使用時間
		existingToken.IsActive = true
		existingToken.LastUsedAt = &now
		existingToken.DeviceType = deviceType
		return s.DB.Save(&existingToken).Error
	}
	
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	
	// 建立新 Token
	newToken := models.MemberDeviceToken{
		MemberID:    memberID,
		DeviceToken: deviceToken,
		DeviceType:  deviceType,
		IsActive:    true,
		LastUsedAt:  &now,
	}
	
	return s.DB.Create(&newToken).Error
}

// DeleteDeviceToken 刪除設備 Token
func (s *FirebaseService) DeleteDeviceToken(token string) error {
	result := s.DB.Where("device_token = ?", token).Delete(&models.MemberDeviceToken{})
	
	if result.Error != nil {
		return result.Error
	}
	
	if result.RowsAffected == 0 {
		return errors.New("device token not found")
	}
	
	return nil
}

// GetMemberDevices 取得會員的所有設備
func (s *FirebaseService) GetMemberDevices(memberID uuid.UUID) ([]models.MemberDeviceToken, error) {
	var devices []models.MemberDeviceToken
	if err := s.DB.Where("member_id = ?", memberID).Find(&devices).Error; err != nil {
		return nil, err
	}
	return devices, nil
}
