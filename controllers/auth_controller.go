package controllers

import (
	"errors"
	"net/http"

	"member_API/auth"
	"member_API/models"
	"member_API/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LoginRequest struct {
	Tenants  models.Tenants
	Email    string `json:"email" binding:"required,email" example:"user@example.com"`
	Password string `json:"password" binding:"required,min=6" example:"password123"`
}

type RegisterRequest struct {
	Name     string `json:"name" binding:"required" example:"張三"`
	Email    string `json:"email" binding:"required,email" example:"user@example.com"`
	Password string `json:"password" binding:"required,min=6" example:"password123"`
}

type AuthResponse struct {
	Token string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	User  User   `json:"user"`
}

func respondError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}

func checkDB(c *gin.Context) bool {
	if db == nil {
		respondError(c, http.StatusInternalServerError, "數據庫連接未配置")
		return false
	}
	return true
}

func getUserFromContext(c *gin.Context) (int64, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}

	idValue, ok := userID.(int64)
	if !ok {
		return 0, false
	}

	return idValue, true
}

func memberToUser(m *models.Member) User {
	return User{
		ID:    int64(m.ID.ID()),
		Name:  m.Name,
		Email: m.Email,
	}
}

func generateAuthResponse(member *models.Member) (*AuthResponse, error) {
	user := memberToUser(member)
	token, err := auth.GenerateToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}
	return &AuthResponse{Token: token, User: user}, nil
}

// Register 用戶註冊
// @Summary 用戶註冊
// @Description 註冊新用戶，返回 JWT token 和用戶信息
// @Tags 認證
// @Accept json
// @Produce json
// @Param register body RegisterRequest true "註冊信息"
// @Success 201 {object} AuthResponse "註冊成功"
// @Failure 400 {object} map[string]string "請求參數錯誤"
// @Failure 409 {object} map[string]string "該電子郵件已被註冊"
// @Failure 500 {object} map[string]string "服務器錯誤"
// @Router /register [post]
func Register(c *gin.Context) {
	if !checkDB(c) {
		return
	}

	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	svc := services.NewMemberService(db)
	member, err := svc.CreateMember(req.Name, req.Email, req.Password, uuid.Nil)
	if err != nil {
		if err.Error() == "email 已被使用" {
			respondError(c, http.StatusConflict, "該電子郵件已被註冊")
			return
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	response, err := generateAuthResponse(member)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Token 生成失敗")
		return
	}

	c.JSON(http.StatusCreated, response)
}

// Login 用戶登入
// @Summary 用戶登入
// @Description 用戶登入，驗證郵件和密碼後返回 JWT token 和用戶信息
// @Tags 認證
// @Accept json
// @Produce json
// @Param login body LoginRequest true "登入信息"
// @Success 200 {object} AuthResponse "登入成功"
// @Failure 400 {object} map[string]string "請求參數錯誤"
// @Failure 401 {object} map[string]string "電子郵件或密碼錯誤"
// @Failure 500 {object} map[string]string "服務器錯誤"
// @Router /login [post]
func Login(c *gin.Context) {
	if !checkDB(c) {
		return
	}

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	var member models.Member
	err := db.WithContext(c.Request.Context()).
		Where("email = ? AND tenants_id = ?", req.Email, req.Tenants.ID).
		First(&member).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(c, http.StatusUnauthorized, "電子郵件或密碼錯誤")
			return
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	if !auth.CheckPassword(req.Password, member.PasswordHash) {
		respondError(c, http.StatusUnauthorized, "電子郵件或密碼錯誤")
		return
	}

	response, err := generateAuthResponse(&member)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Token 生成失敗")
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetProfile 獲取當前用戶信息（需要認證）
// @Summary 獲取當前用戶信息
// @Description 獲取當前登入用戶的詳細信息，需要 JWT 認證
// @Tags 用戶
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]User "獲取成功"
// @Failure 401 {object} map[string]string "未認證"
// @Failure 404 {object} map[string]string "用戶不存在"
// @Failure 500 {object} map[string]string "服務器錯誤"
// @Router /profile [get]
func GetProfile(c *gin.Context) {
	idValue, ok := getUserFromContext(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "未認證")
		return
	}

	if !checkDB(c) {
		return
	}

	var member models.Member
	if err := db.WithContext(c.Request.Context()).
		Select("id", "name", "email").
		First(&member, idValue).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(c, http.StatusNotFound, "用戶不存在")
			return
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": memberToUser(&member)})
}

// RegenerateAPIKey 重新生成 API Key
// @Summary 重新生成 API Key
// @Description 重新生成當前用戶的 API Key，舊的 API Key 將失效
// @Tags 認證
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]string "重新生成成功"
// @Failure 401 {object} map[string]string "未認證"
// @Failure 500 {object} map[string]string "服務器錯誤"
// @Router /auth/regenerate-key [post]
func RegenerateAPIKey(c *gin.Context) {
	idValue, ok := getUserFromContext(c)
	if !ok {
		respondError(c, http.StatusUnauthorized, "未認證")
		return
	}

	if !checkDB(c) {
		return
	}

	memberID, err := uuid.Parse(string(rune(idValue)))
	if err != nil {
		var member models.Member
		if err := db.WithContext(c.Request.Context()).First(&member, idValue).Error; err != nil {
			respondError(c, http.StatusInternalServerError, "無法找到會員")
			return
		}
		memberID = member.ID
	}

	svc := services.NewAPIKeyService(db)
	member, err := svc.RegenerateAPIKey(memberID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "API Key 已重新生成",
		"api_key": member.APIKey,
	})
}

// VerifyAPIKey 驗證 API Key
// @Summary 驗證 API Key 是否有效
// @Description 驗證提供的 API Key 是否有效並返回會員資訊
// @Tags 認證
// @Accept json
// @Produce json
// @Param X-API-Key header string false "API Key"
// @Param Authorization header string false "API Key (格式: ApiKey {key})"
// @Success 200 {object} map[string]interface{} "API Key 有效"
// @Failure 401 {object} map[string]string "無效的 API Key"
// @Failure 500 {object} map[string]string "服務器錯誤"
// @Router /auth/verify-key [get]
func VerifyAPIKey(c *gin.Context) {
	member, exists := c.Get("member")
	if !exists {
		respondError(c, http.StatusUnauthorized, "無效的 API Key")
		return
	}

	memberData := member.(*models.Member)
	response := gin.H{
		"valid": true,
		"member": gin.H{
			"id":    memberData.ID,
			"name":  memberData.Name,
			"email": memberData.Email,
		},
	}

	if tenant, exists := c.Get("tenant"); exists {
		tenantData := tenant.(*models.Tenants)
		response["tenant"] = gin.H{
			"id":          tenantData.ID,
			"name":        tenantData.Name,
			"description": tenantData.Description,
		}
	}

	c.JSON(http.StatusOK, response)
}
