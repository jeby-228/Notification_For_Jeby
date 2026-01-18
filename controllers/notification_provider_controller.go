package controllers

import (
	"net/http"

	"member_API/models"
	"member_API/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CreateProviderRequest struct {
	TenantsID uuid.UUID           `json:"tenants_id" binding:"required"`
	Name      string              `json:"name" binding:"required"`
	Type      models.ProviderType `json:"type" binding:"required"`
	Config    string              `json:"config" binding:"required"`
}

type UpdateProviderRequest struct {
	Name     string              `json:"name" binding:"required"`
	Type     models.ProviderType `json:"type" binding:"required"`
	Config   string              `json:"config" binding:"required"`
	IsActive bool                `json:"is_active"`
}

// CreateProvider 建立通知提供者
// @Summary 建立通知提供者
// @Description 建立新的通知提供者配置，需要 JWT 認證
// @Tags 通知提供者
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param provider body CreateProviderRequest true "提供者配置"
// @Success 201 {object} models.NotificationProvider "建立成功"
// @Failure 400 {object} map[string]string "請求參數錯誤"
// @Failure 401 {object} map[string]string "未認證"
// @Failure 500 {object} map[string]string "服務器錯誤"
// @Router /providers [post]
func CreateProvider(c *gin.Context) {
	if !checkDB(c) {
		return
	}

	var req CreateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	userID, ok := getUserFromContext(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "未認證")
		return
	}

	// 從資料庫取得完整的 member 資訊以獲取 UUID
	var member models.Member
	if err := db.WithContext(c.Request.Context()).First(&member, userID).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "無法找到會員")
		return
	}

	svc := services.NewNotificationProviderService(db)
	provider, err := svc.CreateProvider(req.TenantsID, req.Name, req.Type, req.Config, member.ID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusCreated, provider)
}

// GetProviders 取得租戶的所有提供者
// @Summary 取得租戶的所有提供者
// @Description 取得指定租戶的所有通知提供者配置，需要 JWT 認證
// @Tags 通知提供者
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param tenants_id query string true "租戶 ID"
// @Success 200 {object} map[string][]models.NotificationProvider "取得成功"
// @Failure 400 {object} map[string]string "請求參數錯誤"
// @Failure 401 {object} map[string]string "未認證"
// @Failure 500 {object} map[string]string "服務器錯誤"
// @Router /providers [get]
func GetProviders(c *gin.Context) {
	if !checkDB(c) {
		return
	}

	tenantsIDStr := c.Query("tenants_id")
	if tenantsIDStr == "" {
		respondError(c, http.StatusBadRequest, "缺少 tenants_id 參數")
		return
	}

	tenantsID, err := uuid.Parse(tenantsIDStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "無效的 tenants_id")
		return
	}

	svc := services.NewNotificationProviderService(db)
	providers, err := svc.GetProvidersByTenantID(tenantsID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"providers": providers})
}

// GetProvider 取得單一提供者
// @Summary 取得單一提供者
// @Description 取得指定 ID 的通知提供者配置，需要 JWT 認證
// @Tags 通知提供者
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "提供者 ID"
// @Success 200 {object} models.NotificationProvider "取得成功"
// @Failure 400 {object} map[string]string "請求參數錯誤"
// @Failure 401 {object} map[string]string "未認證"
// @Failure 404 {object} map[string]string "提供者不存在"
// @Failure 500 {object} map[string]string "服務器錯誤"
// @Router /providers/{id} [get]
func GetProvider(c *gin.Context) {
	if !checkDB(c) {
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "無效的 ID")
		return
	}

	svc := services.NewNotificationProviderService(db)
	provider, err := svc.GetProviderByID(id)
	if err != nil {
		if err.Error() == "提供者不存在" {
			respondError(c, http.StatusNotFound, err.Error())
			return
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, provider)
}

// UpdateProvider 更新提供者
// @Summary 更新提供者
// @Description 更新指定 ID 的通知提供者配置，需要 JWT 認證
// @Tags 通知提供者
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "提供者 ID"
// @Param provider body UpdateProviderRequest true "提供者配置"
// @Success 200 {object} models.NotificationProvider "更新成功"
// @Failure 400 {object} map[string]string "請求參數錯誤"
// @Failure 401 {object} map[string]string "未認證"
// @Failure 404 {object} map[string]string "提供者不存在"
// @Failure 500 {object} map[string]string "服務器錯誤"
// @Router /providers/{id} [put]
func UpdateProvider(c *gin.Context) {
	if !checkDB(c) {
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "無效的 ID")
		return
	}

	var req UpdateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	userID, ok := getUserFromContext(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "未認證")
		return
	}

	// 從資料庫取得完整的 member 資訊以獲取 UUID
	var member models.Member
	if err := db.WithContext(c.Request.Context()).First(&member, userID).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "無法找到會員")
		return
	}

	svc := services.NewNotificationProviderService(db)
	provider, err := svc.UpdateProvider(id, req.Name, req.Type, req.Config, req.IsActive, member.ID)
	if err != nil {
		if err.Error() == "提供者不存在" {
			respondError(c, http.StatusNotFound, err.Error())
			return
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, provider)
}

// DeleteProvider 刪除提供者
// @Summary 刪除提供者
// @Description 刪除指定 ID 的通知提供者配置，需要 JWT 認證
// @Tags 通知提供者
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "提供者 ID"
// @Success 200 {object} map[string]string "刪除成功"
// @Failure 400 {object} map[string]string "請求參數錯誤"
// @Failure 401 {object} map[string]string "未認證"
// @Failure 404 {object} map[string]string "提供者不存在"
// @Failure 500 {object} map[string]string "服務器錯誤"
// @Router /providers/{id} [delete]
func DeleteProvider(c *gin.Context) {
	if !checkDB(c) {
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "無效的 ID")
		return
	}

	userID, ok := getUserFromContext(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "未認證")
		return
	}

	// 從資料庫取得完整的 member 資訊以獲取 UUID
	var member models.Member
	if err := db.WithContext(c.Request.Context()).First(&member, userID).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "無法找到會員")
		return
	}

	svc := services.NewNotificationProviderService(db)
	if err := svc.DeleteProvider(id, member.ID); err != nil {
		if err.Error() == "提供者不存在或已被刪除" {
			respondError(c, http.StatusNotFound, err.Error())
			return
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "刪除成功"})
}

// TestProvider 測試提供者配置
// @Summary 測試提供者配置
// @Description 測試指定 ID 的通知提供者配置是否有效，需要 JWT 認證
// @Tags 通知提供者
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "提供者 ID"
// @Success 200 {object} map[string]interface{} "測試結果"
// @Failure 400 {object} map[string]string "請求參數錯誤"
// @Failure 401 {object} map[string]string "未認證"
// @Failure 404 {object} map[string]string "提供者不存在"
// @Failure 500 {object} map[string]string "服務器錯誤"
// @Router /providers/{id}/test [get]
func TestProvider(c *gin.Context) {
	if !checkDB(c) {
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "無效的 ID")
		return
	}

	svc := services.NewNotificationProviderService(db)
	valid, message, err := svc.TestProvider(id)
	if err != nil {
		if err.Error() == "提供者不存在" {
			respondError(c, http.StatusNotFound, err.Error())
			return
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":   valid,
		"message": message,
	})
}
