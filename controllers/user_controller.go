package controllers

import (
	"errors"
	"net/http"
	"strconv"

	"member_API/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var db *gorm.DB

// SetupUserController stores the shared database handle for controller use.
func SetupUserController(database *gorm.DB) {
	db = database
}

// User represents a simplified member record.
type User struct {
	ID    int64  `json:"id" example:"1"`
	Name  string `json:"name" example:"張三"`
	Email string `json:"email" example:"user@example.com"`
}

// GetUsers returns a small collection of users from the database.
// @Summary 獲取所有會員
// @Description 獲取會員列表，最多返回 50 條記錄，需要 JWT 認證
// @Tags 用戶
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string][]User "獲取成功"
// @Failure 401 {object} map[string]string "未認證"
// @Failure 500 {object} map[string]string "服務器錯誤"
// @Router /users [get]
func GetUsers(c *gin.Context) {
	if db == nil {
		c.JSON(http.StatusOK, gin.H{
			"users":   []User{},
			"message": "database connection not configured",
		})
		return
	}

	var members []models.Member
	if err := db.WithContext(c.Request.Context()).
		Select("id", "name", "email").
		Limit(50).
		Find(&members).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	users := make([]User, len(members))
	for i, member := range members {
		users[i] = User{ID: int64(member.ID), Name: member.Name, Email: member.Email}
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

// GetUserByID returns a single user by ID from the database.
// @Summary 根據 ID 獲取會員
// @Description 根據會員 ID 獲取單個會員的詳細信息，需要 JWT 認證
// @Tags 用戶
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "會員 ID" example(1)
// @Success 200 {object} map[string]User "獲取成功"
// @Failure 401 {object} map[string]string "未認證"
// @Failure 404 {object} map[string]string "會員不存在"
// @Failure 500 {object} map[string]string "服務器錯誤"
// @Router /user/{id} [get]
func GetUserByID(c *gin.Context) {
	if db == nil {
		c.JSON(http.StatusOK, gin.H{
			"user":    nil,
			"message": "database connection not configured",
		})
		return
	}

	memberID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var member models.Member
	if err := db.WithContext(c.Request.Context()).
		Select("id", "name", "email").
		First(&member, memberID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": User{ID: int64(member.ID), Name: member.Name, Email: member.Email}})
}
