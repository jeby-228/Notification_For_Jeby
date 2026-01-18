package auth

import (
	"member_API/models"
	"member_API/testutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestAPIKeyMiddleware_MissingAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	APIKeyMiddleware()(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "缺少 API Key")
	assert.True(t, c.IsAborted())
}

func TestAPIKeyMiddleware_InvalidAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock := testutil.SetupTestDB(t)
	SetAPIKeyMiddlewareDB(db)

	mock.ExpectQuery(`SELECT \* FROM "members"`).
		WithArgs("invalid_key", false, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("X-API-Key", "invalid_key")

	APIKeyMiddleware()(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "無效的 API Key")
	assert.True(t, c.IsAborted())

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestAPIKeyMiddleware_ValidAPIKey_XAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock := testutil.SetupTestDB(t)
	SetAPIKeyMiddlewareDB(db)

	memberID := uuid.New()
	tenantID := uuid.New()
	apiKey := "ak_test_key_123"

	memberRows := sqlmock.NewRows([]string{"id", "name", "email", "api_key", "tenants_id", "is_deleted"}).
		AddRow(memberID, "Test User", "test@example.com", apiKey, tenantID, false)

	mock.ExpectQuery(`SELECT \* FROM "members"`).
		WithArgs(apiKey, false, 1).
		WillReturnRows(memberRows)

	tenantRows := sqlmock.NewRows([]string{"id", "name", "description"}).
		AddRow(tenantID, "Test Tenant", "Test Description")

	mock.ExpectQuery(`SELECT \* FROM "tenants"`).
		WithArgs(tenantID, 1).
		WillReturnRows(tenantRows)

	router := gin.New()
	nextCalled := false
	router.Use(APIKeyMiddleware())
	router.GET("/", func(c *gin.Context) {
		nextCalled = true

		member, exists := c.Get("member")
		assert.True(t, exists)
		assert.NotNil(t, member)

		memberID, exists := c.Get("member_id")
		assert.True(t, exists)
		assert.NotEqual(t, uuid.Nil, memberID)

		memberEmail, exists := c.Get("member_email")
		assert.True(t, exists)
		assert.Equal(t, "test@example.com", memberEmail)

		tenant, exists := c.Get("tenant")
		assert.True(t, exists)
		assert.NotNil(t, tenant)

		tenantID, exists := c.Get("tenant_id")
		assert.True(t, exists)
		assert.NotEqual(t, uuid.Nil, tenantID)

		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", apiKey)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, nextCalled)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestAPIKeyMiddleware_ValidAPIKey_Authorization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock := testutil.SetupTestDB(t)
	SetAPIKeyMiddlewareDB(db)

	memberID := uuid.New()
	tenantID := uuid.New()
	apiKey := "ak_test_key_456"

	memberRows := sqlmock.NewRows([]string{"id", "name", "email", "api_key", "tenants_id", "is_deleted"}).
		AddRow(memberID, "Test User", "test@example.com", apiKey, tenantID, false)

	mock.ExpectQuery(`SELECT \* FROM "members"`).
		WithArgs(apiKey, false, 1).
		WillReturnRows(memberRows)

	tenantRows := sqlmock.NewRows([]string{"id", "name", "description"}).
		AddRow(tenantID, "Test Tenant", "Test Description")

	mock.ExpectQuery(`SELECT \* FROM "tenants"`).
		WithArgs(tenantID, 1).
		WillReturnRows(tenantRows)

	router := gin.New()
	nextCalled := false
	router.Use(APIKeyMiddleware())
	router.GET("/", func(c *gin.Context) {
		nextCalled = true
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "ApiKey "+apiKey)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, nextCalled)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestAPIKeyMiddleware_Cache(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock := testutil.SetupTestDB(t)
	SetAPIKeyMiddlewareDB(db)

	memberID := uuid.New()
	tenantID := uuid.New()
	apiKey := "ak_test_key_cache"

	memberRows := sqlmock.NewRows([]string{"id", "name", "email", "api_key", "tenants_id", "is_deleted"}).
		AddRow(memberID, "Test User", "test@example.com", apiKey, tenantID, false)

	mock.ExpectQuery(`SELECT \* FROM "members"`).
		WithArgs(apiKey, false, 1).
		WillReturnRows(memberRows)

	tenantRows := sqlmock.NewRows([]string{"id", "name", "description"}).
		AddRow(tenantID, "Test Tenant", "Test Description")

	mock.ExpectQuery(`SELECT \* FROM "tenants"`).
		WithArgs(tenantID, 1).
		WillReturnRows(tenantRows)

	router := gin.New()
	router.Use(APIKeyMiddleware())
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", apiKey)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.Header.Set("X-API-Key", apiKey)
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestAPIKeyMiddleware_CacheExpiration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldTTL := cacheTTL
	cacheTTL = 100 * time.Millisecond
	defer func() { cacheTTL = oldTTL }()

	apiKey := "ak_test_key_expire"
	memberID := uuid.New()
	tenantID := uuid.New()

	member := &models.Member{
		Base:      models.Base{ID: memberID},
		Name:      "Test User",
		Email:     "test@example.com",
		APIKey:    apiKey,
		TenantsID: tenantID,
	}
	tenant := &models.Tenants{
		ID:   tenantID,
		Name: "Test Tenant",
	}

	setCacheEntry(apiKey, member, tenant)

	entry, ok := getCachedEntry(apiKey)
	assert.True(t, ok)
	assert.NotNil(t, entry)

	time.Sleep(150 * time.Millisecond)

	entry, ok = getCachedEntry(apiKey)
	assert.False(t, ok)
	assert.Nil(t, entry)
}

func TestAPIKeyMiddleware_NoTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, mock := testutil.SetupTestDB(t)
	SetAPIKeyMiddlewareDB(db)

	memberID := uuid.New()
	apiKey := "ak_test_key_no_tenant"

	memberRows := sqlmock.NewRows([]string{"id", "name", "email", "api_key", "tenants_id", "is_deleted"}).
		AddRow(memberID, "Test User", "test@example.com", apiKey, uuid.Nil, false)

	mock.ExpectQuery(`SELECT \* FROM "members"`).
		WithArgs(apiKey, false, 1).
		WillReturnRows(memberRows)

	router := gin.New()
	nextCalled := false
	router.Use(APIKeyMiddleware())
	router.GET("/", func(c *gin.Context) {
		nextCalled = true

		member, exists := c.Get("member")
		assert.True(t, exists)
		assert.NotNil(t, member)

		_, exists = c.Get("tenant")
		assert.False(t, exists)

		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", apiKey)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, nextCalled)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}
