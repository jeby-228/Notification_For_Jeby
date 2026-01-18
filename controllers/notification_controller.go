package controllers

import (
	"net/http"

	"member_API/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var notificationDB *gorm.DB

func SetupNotificationController(database *gorm.DB) {
	notificationDB = database
}

// SendEmail sends an email notification
// @Summary 發送郵件通知
// @Description 透過 SMTP 發送郵件通知，需要 JWT 認證
// @Tags 通知
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param provider_id query string true "通知提供者 ID"
// @Param body body services.EmailRequest true "郵件內容"
// @Success 200 {object} map[string]string "發送成功"
// @Failure 400 {object} map[string]string "請求參數錯誤"
// @Failure 401 {object} map[string]string "未認證"
// @Failure 500 {object} map[string]string "發送失敗"
// @Router /notifications/email [post]
func SendEmail(c *gin.Context) {
	if notificationDB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database connection not configured"})
		return
	}

	memberIDValue, exists := c.Get("member_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	memberID, ok := memberIDValue.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid member id"})
		return
	}

	providerIDStr := c.Query("provider_id")
	if providerIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider_id is required"})
		return
	}

	providerID, err := uuid.Parse(providerIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid provider_id"})
		return
	}

	var req services.EmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	smtpService := services.NewSMTPService(notificationDB)
	if err := smtpService.SendEmail(memberID, providerID, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "email sent successfully"})
}
