package controllers

import (
	"member_API/services"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var firebaseService *services.FirebaseService

// SetupFirebaseController 初始化 Firebase 控制器
func SetupFirebaseController(db *gorm.DB) {
	firebaseService = services.NewFirebaseService(db)
}

// SendPushNotificationRequest 發送推播請求結構
type SendPushNotificationRequest struct {
	MemberID    *string           `json:"member_id"`
	DeviceToken *string           `json:"device_token"`
	Title       string            `json:"title" binding:"required"`
	Body        string            `json:"body" binding:"required"`
	Data        map[string]string `json:"data"`
}

// RegisterDeviceTokenRequest 註冊設備 Token 請求結構
type RegisterDeviceTokenRequest struct {
	DeviceToken string `json:"device_token" binding:"required"`
	DeviceType  string `json:"device_type" binding:"required,oneof=ios android web"`
}

// SendPushNotification 發送推播
// @Summary 發送推播通知
// @Description 發送推播通知給指定會員或設備
// @Tags 通知
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param request body SendPushNotificationRequest true "推播請求"
// @Success 200 {object} map[string]string "發送成功"
// @Failure 400 {object} map[string]string "請求錯誤"
// @Failure 401 {object} map[string]string "未授權"
// @Failure 500 {object} map[string]string "內部錯誤"
// @Router /notifications/push [post]
// @Security BearerAuth
func SendPushNotification(c *gin.Context) {
	if firebaseService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Firebase service not initialized"})
		return
	}

	var req SendPushNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 必須提供 member_id 或 device_token 其中之一
	if req.MemberID == nil && req.DeviceToken == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "either member_id or device_token is required"})
		return
	}

	// 從 context 取得 tenant_id (假設在 auth middleware 中設定)
	tenantIDStr, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id not found"})
		return
	}
	tenantID, err := uuid.Parse(tenantIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
		return
	}

	ctx := c.Request.Context()

	// 如果提供 member_id，發送給該會員所有設備
	if req.MemberID != nil {
		memberID, err := uuid.Parse(*req.MemberID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid member_id"})
			return
		}

		if err := firebaseService.SendToMember(ctx, tenantID, memberID, req.Title, req.Body, req.Data); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "notification sent to all member devices"})
		return
	}

	// 如果提供 device_token，發送給特定設備
	if req.DeviceToken != nil {
		// 需要找到對應的 member_id
		memberIDValue, exists := c.Get("member_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "member_id not found in context"})
			return
		}
		memberID, err := uuid.Parse(memberIDValue.(string))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid member_id in context"})
			return
		}

		if err := firebaseService.SendToToken(ctx, tenantID, memberID, *req.DeviceToken, req.Title, req.Body, req.Data); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "notification sent to device"})
		return
	}
}

// RegisterDeviceToken 註冊設備 Token
// @Summary 註冊設備 Token
// @Description 註冊或更新會員的設備 Token
// @Tags 設備
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param request body RegisterDeviceTokenRequest true "設備 Token 請求"
// @Success 200 {object} map[string]string "註冊成功"
// @Failure 400 {object} map[string]string "請求錯誤"
// @Failure 401 {object} map[string]string "未授權"
// @Failure 500 {object} map[string]string "內部錯誤"
// @Router /devices/register [post]
// @Security BearerAuth
func RegisterDeviceToken(c *gin.Context) {
	if firebaseService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Firebase service not initialized"})
		return
	}

	var req RegisterDeviceTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 從 context 取得 member_id
	memberIDValue, exists := c.Get("member_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "member_id not found"})
		return
	}
	memberID, err := uuid.Parse(memberIDValue.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid member_id"})
		return
	}

	if err := firebaseService.RegisterDeviceToken(memberID, req.DeviceToken, req.DeviceType); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "device token registered successfully"})
}

// DeleteDeviceToken 刪除設備 Token
// @Summary 刪除設備 Token
// @Description 刪除指定的設備 Token
// @Tags 設備
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param token path string true "設備 Token"
// @Success 200 {object} map[string]string "刪除成功"
// @Failure 400 {object} map[string]string "請求錯誤"
// @Failure 401 {object} map[string]string "未授權"
// @Failure 404 {object} map[string]string "Token 不存在"
// @Failure 500 {object} map[string]string "內部錯誤"
// @Router /devices/{token} [delete]
// @Security BearerAuth
func DeleteDeviceToken(c *gin.Context) {
	if firebaseService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Firebase service not initialized"})
		return
	}

	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
		return
	}

	if err := firebaseService.DeleteDeviceToken(token); err != nil {
		if err.Error() == "device token not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "device token deleted successfully"})
}

// GetMemberDevices 取得會員的所有設備
// @Summary 取得會員設備列表
// @Description 取得當前登入會員的所有設備 Token
// @Tags 設備
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Success 200 {object} []models.MemberDeviceToken "設備列表"
// @Failure 401 {object} map[string]string "未授權"
// @Failure 500 {object} map[string]string "內部錯誤"
// @Router /devices [get]
// @Security BearerAuth
func GetMemberDevices(c *gin.Context) {
	if firebaseService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Firebase service not initialized"})
		return
	}

	// 從 context 取得 member_id
	memberIDValue, exists := c.Get("member_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "member_id not found"})
		return
	}
	memberID, err := uuid.Parse(memberIDValue.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid member_id"})
		return
	}

	devices, err := firebaseService.GetMemberDevices(memberID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, devices)
}
