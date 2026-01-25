package controllers

import (
	"member_API/models"
	"member_API/services"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var smsService *services.SMSService

func SetupSMSController(service *services.SMSService) {
	smsService = service
}

type SendSMSRequest struct {
	RecipientPhone string `json:"recipient_phone" binding:"required" example:"+886912345678"`
	Body           string `json:"body" binding:"required" example:"您的驗證碼是 123456"`
	ProviderID     string `json:"provider_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
}

type SendSMSResponse struct {
	Success bool                    `json:"success" example:"true"`
	Message string                  `json:"message" example:"簡訊發送成功"`
	Log     *models.NotificationLog `json:"log,omitempty"`
}

// SendSMS 發送簡訊
// @Summary 發送簡訊
// @Description 發送簡訊通知到指定電話號碼
// @Tags 通知
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body SendSMSRequest true "發送簡訊請求"
// @Success 200 {object} SendSMSResponse "發送成功"
// @Failure 400 {object} map[string]string "參數錯誤"
// @Failure 401 {object} map[string]string "未授權"
// @Failure 500 {object} map[string]string "發送失敗"
// @Router /notifications/sms [post]
func SendSMS(c *gin.Context) {
	if smsService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "SMS 服務未初始化",
		})
		return
	}

	var req SendSMSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "參數錯誤: " + err.Error(),
		})
		return
	}

	providerID, err := uuid.Parse(req.ProviderID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Provider ID 格式錯誤",
		})
		return
	}

	memberIDValue, exists := c.Get("member_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "無法取得會員資訊",
		})
		return
	}

	memberID, ok := memberIDValue.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "會員 ID 格式錯誤",
		})
		return
	}

	log, err := smsService.SendSMS(memberID, providerID, req.RecipientPhone, req.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, SendSMSResponse{
		Success: true,
		Message: "簡訊發送成功",
		Log:     log,
	})
}
