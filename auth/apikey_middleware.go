package auth

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"member_API/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type cacheEntry struct {
	member    *models.Member
	tenant    *models.Tenants
	expiresAt time.Time
}

var (
	apiKeyCache        = sync.Map{}
	cacheTTL           = 5 * time.Minute
	apiKeyMiddlewareDB *gorm.DB
)

func SetAPIKeyMiddlewareDB(db *gorm.DB) {
	apiKeyMiddlewareDB = db
}

func extractAPIKey(c *gin.Context) string {
	if key := c.GetHeader("X-API-Key"); key != "" {
		return key
	}

	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && parts[0] == "ApiKey" {
			return parts[1]
		}
	}

	return ""
}

func getCachedEntry(apiKey string) (*cacheEntry, bool) {
	if val, ok := apiKeyCache.Load(apiKey); ok {
		entry := val.(*cacheEntry)
		if time.Now().Before(entry.expiresAt) {
			return entry, true
		}
		apiKeyCache.Delete(apiKey)
	}
	return nil, false
}

func setCacheEntry(apiKey string, member *models.Member, tenant *models.Tenants) {
	entry := &cacheEntry{
		member:    member,
		tenant:    tenant,
		expiresAt: time.Now().Add(cacheTTL),
	}
	apiKeyCache.Store(apiKey, entry)
}

func APIKeyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := extractAPIKey(c)

		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "缺少 API Key"})
			c.Abort()
			return
		}

		if entry, ok := getCachedEntry(apiKey); ok {
			c.Set("member", entry.member)
			c.Set("member_id", entry.member.ID)
			c.Set("member_email", entry.member.Email)
			if entry.tenant != nil {
				c.Set("tenant", entry.tenant)
				c.Set("tenant_id", entry.tenant.ID)
			}
			c.Next()
			return
		}

		if apiKeyMiddlewareDB == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "數據庫未配置"})
			c.Abort()
			return
		}

		var member models.Member
		if err := apiKeyMiddlewareDB.Where("api_key = ? AND is_deleted = ?", apiKey, false).First(&member).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "無效的 API Key"})
			c.Abort()
			return
		}

		var tenant models.Tenants
		var tenantPtr *models.Tenants
		if member.TenantsID != uuid.Nil {
			if err := apiKeyMiddlewareDB.First(&tenant, member.TenantsID).Error; err == nil {
				tenantPtr = &tenant
			}
		}

		setCacheEntry(apiKey, &member, tenantPtr)

		c.Set("member", &member)
		c.Set("member_id", member.ID)
		c.Set("member_email", member.Email)
		if tenantPtr != nil {
			c.Set("tenant", tenantPtr)
			c.Set("tenant_id", tenantPtr.ID)
		}

		c.Next()
	}
}
