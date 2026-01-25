package controllers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"member_API/testutil"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSendEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupContext   func(*gin.Context)
		providerID     string
		body           map[string]string
		expectedStatus int
		expectedMsg    string
	}{
		{
			name: "Missing member_id in context",
			setupContext: func(c *gin.Context) {
			},
			providerID:     uuid.New().String(),
			body:           map[string]string{},
			expectedStatus: http.StatusUnauthorized,
			expectedMsg:    "unauthorized",
		},
		{
			name: "Missing provider_id",
			setupContext: func(c *gin.Context) {
				c.Set("member_id", uuid.New())
			},
			providerID:     "",
			body:           map[string]string{},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "provider_id is required",
		},
		{
			name: "Invalid provider_id",
			setupContext: func(c *gin.Context) {
				c.Set("member_id", uuid.New())
			},
			providerID:     "invalid-uuid",
			body:           map[string]string{},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "invalid provider_id",
		},
		{
			name: "Invalid request body",
			setupContext: func(c *gin.Context) {
				c.Set("member_id", uuid.New())
			},
			providerID: uuid.New().String(),
			body: map[string]string{
				"recipient_email": "invalid-email",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, _ := testutil.SetupTestDB(t)
			SetupNotificationController(db)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			bodyJSON, _ := json.Marshal(tt.body)
			c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/notifications/email?provider_id="+tt.providerID, bytes.NewBuffer(bodyJSON))
			c.Request.Header.Set("Content-Type", "application/json")

			tt.setupContext(c)

			SendEmail(c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedMsg != "" {
				assert.Contains(t, w.Body.String(), tt.expectedMsg)
			}
		})
	}
}
