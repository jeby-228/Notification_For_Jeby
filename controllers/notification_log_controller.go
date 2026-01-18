package controllers

import (
	"fmt"
	"net/http"
	"time"

	"member_API/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetNotificationLogs returns notification logs with filtering and pagination
// @Summary 查詢發送記錄
// @Description 查詢通知發送記錄，支援多條件篩選和分頁
// @Tags 通知日誌
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param status query string false "狀態篩選 (PENDING, SENT, FAILED)"
// @Param type query string false "類型篩選 (SMTP, SMS, FIREBASE)"
// @Param start_date query string false "開始時間 (RFC3339格式)"
// @Param end_date query string false "結束時間 (RFC3339格式)"
// @Param recipient query string false "接收者篩選 (email或phone)"
// @Param page query int false "頁碼" default(1)
// @Param page_size query int false "每頁數量" default(20)
// @Success 200 {object} map[string]interface{} "查詢成功"
// @Failure 400 {object} map[string]string "請求參數錯誤"
// @Failure 401 {object} map[string]string "未認證"
// @Failure 500 {object} map[string]string "服務器錯誤"
// @Router /logs [get]
func GetNotificationLogs(c *gin.Context) {
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database connection not configured"})
		return
	}

	query := db.WithContext(c.Request.Context())

	// 狀態篩選
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	// 類型篩選
	if notifType := c.Query("type"); notifType != "" {
		query = query.Where("type = ?", notifType)
	}

	// 時間範圍篩選
	if startDate := c.Query("start_date"); startDate != "" {
		if parsedTime, err := time.Parse(time.RFC3339, startDate); err == nil {
			query = query.Where("creation_time >= ?", parsedTime)
		}
	}

	if endDate := c.Query("end_date"); endDate != "" {
		if parsedTime, err := time.Parse(time.RFC3339, endDate); err == nil {
			query = query.Where("creation_time <= ?", parsedTime)
		}
	}

	// 接收者篩選
	if recipient := c.Query("recipient"); recipient != "" {
		query = query.Where("recipient_email LIKE ? OR recipient_phone LIKE ?", "%"+recipient+"%", "%"+recipient+"%")
	}

	// 分頁參數
	page := 1
	pageSize := 20
	if p, ok := c.GetQuery("page"); ok {
		if val, err := parsePositiveInt(p); err == nil && val > 0 {
			page = val
		}
	}
	if ps, ok := c.GetQuery("page_size"); ok {
		if val, err := parsePositiveInt(ps); err == nil && val > 0 && val <= 100 {
			pageSize = val
		}
	}

	// 計算總數
	var total int64
	if err := query.Model(&models.NotificationLog{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 查詢記錄
	var logs []models.NotificationLog
	offset := (page - 1) * pageSize
	if err := query.Order("creation_time DESC").Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": logs,
		"pagination": gin.H{
			"page":       page,
			"page_size":  pageSize,
			"total":      total,
			"total_page": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// GetNotificationLogByID returns a single notification log by ID
// @Summary 單一記錄詳情
// @Description 根據 ID 獲取單個通知發送記錄的詳細信息
// @Tags 通知日誌
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "記錄 ID (UUID)"
// @Success 200 {object} map[string]interface{} "查詢成功"
// @Failure 400 {object} map[string]string "無效的記錄 ID"
// @Failure 401 {object} map[string]string "未認證"
// @Failure 404 {object} map[string]string "記錄不存在"
// @Failure 500 {object} map[string]string "服務器錯誤"
// @Router /logs/{id} [get]
func GetNotificationLogByID(c *gin.Context) {
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database connection not configured"})
		return
	}

	idStr := c.Param("id")
	logID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid log id"})
		return
	}

	var log models.NotificationLog
	if err := db.WithContext(c.Request.Context()).Where("id = ?", logID).First(&log).Error; err != nil {
		if err.Error() == "record not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "log not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": log})
}

// GetNotificationLogStats returns statistics for notification logs
// @Summary 統計資訊
// @Description 獲取通知發送記錄的統計資訊，包括成功率、失敗率和各通道使用量
// @Tags 通知日誌
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param start_date query string false "開始時間 (RFC3339格式)"
// @Param end_date query string false "結束時間 (RFC3339格式)"
// @Success 200 {object} map[string]interface{} "統計成功"
// @Failure 401 {object} map[string]string "未認證"
// @Failure 500 {object} map[string]string "服務器錯誤"
// @Router /logs/stats [get]
func GetNotificationLogStats(c *gin.Context) {
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database connection not configured"})
		return
	}

	query := db.WithContext(c.Request.Context()).Model(&models.NotificationLog{})

	// 時間範圍篩選
	if startDate := c.Query("start_date"); startDate != "" {
		if parsedTime, err := time.Parse(time.RFC3339, startDate); err == nil {
			query = query.Where("creation_time >= ?", parsedTime)
		}
	}

	if endDate := c.Query("end_date"); endDate != "" {
		if parsedTime, err := time.Parse(time.RFC3339, endDate); err == nil {
			query = query.Where("creation_time <= ?", parsedTime)
		}
	}

	// 計算總數
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 按狀態統計
	type StatusCount struct {
		Status string
		Count  int64
	}
	var statusCounts []StatusCount
	if err := query.Select("status, COUNT(*) as count").Group("status").Scan(&statusCounts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	statusStats := make(map[string]int64)
	for _, sc := range statusCounts {
		statusStats[sc.Status] = sc.Count
	}

	// 按類型統計
	type TypeCount struct {
		Type  string
		Count int64
	}
	var typeCounts []TypeCount
	if err := query.Select("type, COUNT(*) as count").Group("type").Scan(&typeCounts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	typeStats := make(map[string]int64)
	for _, tc := range typeCounts {
		typeStats[tc.Type] = tc.Count
	}

	// 計算成功率和失敗率
	sent := statusStats["SENT"]
	failed := statusStats["FAILED"]
	successRate := 0.0
	failureRate := 0.0
	if total > 0 {
		successRate = float64(sent) / float64(total) * 100
		failureRate = float64(failed) / float64(total) * 100
	}

	c.JSON(http.StatusOK, gin.H{
		"total":         total,
		"success_rate":  successRate,
		"failure_rate":  failureRate,
		"status_stats":  statusStats,
		"channel_stats": typeStats,
	})
}

// parsePositiveInt parses a string to a positive integer
func parsePositiveInt(s string) (int, error) {
	var val int
	if _, err := fmt.Sscanf(s, "%d", &val); err != nil {
		return 0, err
	}
	return val, nil
}
