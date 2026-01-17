package auth

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// setupTest 統一的測試環境設置
func setupTest(t *testing.T) {
	t.Helper()
	if err := os.Setenv("JWT_SECRET", "test-secret-key"); err != nil {
		t.Fatalf("Failed to set JWT_SECRET: %v", err)
	}
	jwtSecret = []byte("test-secret-key")
}

// createTestClaims 創建測試用的 claims
func createTestClaims(userID int64, email string, expiresAt, issuedAt time.Time) *Claims {
	return &Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			Issuer:    "member-api",
		},
	}
}

func TestGenerateToken(t *testing.T) {
	setupTest(t)

	tests := []struct {
		name      string
		userID    int64
		email     string
		wantErr   bool
		validator func(t *testing.T, token string)
	}{
		{
			name:    "成功生成標準 token",
			userID:  1,
			email:   "user@example.com",
			wantErr: false,
			validator: func(t *testing.T, token string) {
				// 驗證 token 格式 (header.payload.signature)
				parts := strings.Split(token, ".")
				if len(parts) != 3 {
					t.Errorf("Token 應該有 3 個部分，實際: %d", len(parts))
				}
				// 驗證可以被解析
				claims, err := ValidateToken(token)
				if err != nil {
					t.Errorf("生成的 token 無法驗證: %v", err)
				}
				if claims.UserID != 1 {
					t.Errorf("UserID = %v, want 1", claims.UserID)
				}
				if claims.Email != "user@example.com" {
					t.Errorf("Email = %v, want user@example.com", claims.Email)
				}
			},
		},
		{
			name:    "生成大 UserID 的 token",
			userID:  9223372036854775807, // int64 最大值
			email:   "max@example.com",
			wantErr: false,
			validator: func(t *testing.T, token string) {
				if token == "" {
					t.Error("Token 不應為空")
				}
			},
		},
		{
			name:    "生成含特殊字符 email 的 token",
			userID:  999,
			email:   "user+test@sub.example.com",
			wantErr: false,
			validator: func(t *testing.T, token string) {
				claims, _ := ValidateToken(token)
				if claims.Email != "user+test@sub.example.com" {
					t.Errorf("Email = %v, want user+test@sub.example.com", claims.Email)
				}
			},
		},
		{
			name:    "生成空 email 的 token（技術上可行）",
			userID:  1,
			email:   "",
			wantErr: false,
			validator: func(t *testing.T, token string) {
				if token == "" {
					t.Error("即使 email 為空，也應該生成 token")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GenerateToken(tt.userID, tt.email)

			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if token == "" {
					t.Error("GenerateToken() 返回空 token")
				}
				if tt.validator != nil {
					tt.validator(t, token)
				}
			}
		})
	}
}

func TestGenerateTokenExpiration(t *testing.T) {
	setupTest(t)

	token, err := GenerateToken(1, "test@example.com")
	if err != nil {
		t.Fatalf("GenerateToken() failed: %v", err)
	}

	claims, err := ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken() failed: %v", err)
	}

	// 驗證過期時間應該是 24 小時後
	expectedExpiry := time.Now().Add(24 * time.Hour)
	actualExpiry := claims.ExpiresAt.Time

	// 允許 5 秒的誤差
	diff := actualExpiry.Sub(expectedExpiry)
	if diff < -5*time.Second || diff > 5*time.Second {
		t.Errorf("ExpiresAt = %v, want around %v (diff: %v)", actualExpiry, expectedExpiry, diff)
	}

	// 驗證 Issuer
	if claims.Issuer != "member-api" {
		t.Errorf("Issuer = %v, want member-api", claims.Issuer)
	}

	// 驗證 IssuedAt
	issuedAt := claims.IssuedAt.Time
	now := time.Now()
	diff = now.Sub(issuedAt)
	if diff < 0 || diff > 5*time.Second {
		t.Errorf("IssuedAt = %v, want around %v", issuedAt, now)
	}
}

func TestValidateToken(t *testing.T) {
	setupTest(t)

	// 準備有效的 token
	validToken, err := GenerateToken(123, "valid@example.com")
	if err != nil {
		t.Fatalf("Failed to generate valid token: %v", err)
	}

	tests := []struct {
		name           string
		token          string
		wantErr        bool
		wantUserID     int64
		wantEmail      string
		validateClaims bool
	}{
		{
			name:           "驗證有效 token",
			token:          validToken,
			wantErr:        false,
			wantUserID:     123,
			wantEmail:      "valid@example.com",
			validateClaims: true,
		},
		{
			name:    "空字串 token",
			token:   "",
			wantErr: true,
		},
		{
			name:    "完全無效的字串",
			token:   "not-a-jwt-token",
			wantErr: true,
		},
		{
			name:    "只有一個部分",
			token:   "onlyonepart",
			wantErr: true,
		},
		{
			name:    "只有兩個部分",
			token:   "two.parts",
			wantErr: true,
		},
		{
			name:    "Base64 但不是有效的 JWT",
			token:   "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.invalid",
			wantErr: true,
		},
		{
			name:    "格式正確但簽名無效",
			token:   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxLCJlbWFpbCI6InRlc3RAZXhhbXBsZS5jb20ifQ.invalidsignature",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := ValidateToken(tt.token)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if claims != nil {
					t.Error("ValidateToken() 錯誤時應返回 nil claims")
				}
			} else {
				if claims == nil {
					t.Fatal("ValidateToken() 成功時不應返回 nil claims")
				}
				if tt.validateClaims {
					if claims.UserID != tt.wantUserID {
						t.Errorf("UserID = %v, want %v", claims.UserID, tt.wantUserID)
					}
					if claims.Email != tt.wantEmail {
						t.Errorf("Email = %v, want %v", claims.Email, tt.wantEmail)
					}
				}
			}
		})
	}
}

func TestValidateTokenExpired(t *testing.T) {
	setupTest(t)

	tests := []struct {
		name       string
		expiryTime time.Time
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "過期 1 小時",
			expiryTime: time.Now().Add(-1 * time.Hour),
			wantErr:    true,
			errMsg:     "expired",
		},
		{
			name:       "過期 1 秒",
			expiryTime: time.Now().Add(-1 * time.Second),
			wantErr:    true,
			errMsg:     "expired",
		},
		{
			name:       "過期 1 天",
			expiryTime: time.Now().Add(-24 * time.Hour),
			wantErr:    true,
			errMsg:     "expired",
		},
		{
			name:       "還沒過期（未來 1 小時）",
			expiryTime: time.Now().Add(1 * time.Hour),
			wantErr:    false,
		},
		{
			name:       "剛好在邊界（未來 1 秒）",
			expiryTime: time.Now().Add(1 * time.Second),
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := createTestClaims(1, "test@example.com", tt.expiryTime, time.Now().Add(-1*time.Hour))
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			tokenString, err := token.SignedString(jwtSecret)
			if err != nil {
				t.Fatalf("Failed to create token: %v", err)
			}

			_, err = ValidateToken(tokenString)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateToken() error = %v, should contain %q", err, tt.errMsg)
				}
			}
		})
	}
}

func TestValidateTokenSignature(t *testing.T) {
	setupTest(t)

	tests := []struct {
		name        string
		setupToken  func() string
		wantErr     bool
		description string
	}{
		{
			name: "使用錯誤的 secret 簽名",
			setupToken: func() string {
				claims := createTestClaims(1, "test@example.com", time.Now().Add(24*time.Hour), time.Now())
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				tokenString, _ := token.SignedString([]byte("wrong-secret-key"))
				return tokenString
			},
			wantErr:     true,
			description: "使用不同的 secret 簽名應該驗證失敗",
		},
		{
			name: "篡改 payload",
			setupToken: func() string {
				validToken, _ := GenerateToken(1, "test@example.com")
				// 修改 token 的中間部分
				parts := strings.Split(validToken, ".")
				if len(parts) == 3 {
					// 修改 payload 的最後一個字符
					parts[1] = parts[1][:len(parts[1])-1] + "X"
					return strings.Join(parts, ".")
				}
				return validToken
			},
			wantErr:     true,
			description: "篡改 payload 應該導致簽名驗證失敗",
		},
		{
			name: "篡改 signature",
			setupToken: func() string {
				validToken, _ := GenerateToken(1, "test@example.com")
				parts := strings.Split(validToken, ".")
				if len(parts) == 3 {
					// 修改簽名部分
					parts[2] = parts[2][:len(parts[2])-1] + "X"
					return strings.Join(parts, ".")
				}
				return validToken
			},
			wantErr:     true,
			description: "篡改簽名應該導致驗證失敗",
		},
		{
			name: "使用正確的 secret",
			setupToken: func() string {
				token, _ := GenerateToken(100, "correct@example.com")
				return token
			},
			wantErr:     false,
			description: "正確的簽名應該通過驗證",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenString := tt.setupToken()
			claims, err := ValidateToken(tokenString)

			if (err != nil) != tt.wantErr {
				t.Errorf("%s\nValidateToken() error = %v, wantErr %v", tt.description, err, tt.wantErr)
			}

			if tt.wantErr && claims != nil {
				t.Error("驗證失敗時應返回 nil claims")
			}
		})
	}
}

func TestValidateTokenValid(t *testing.T) {
	setupTest(t)

	t.Run("測試 token.Valid 檢查邏輯", func(t *testing.T) {
		// 這個測試專門驗證 jwt.go 中的 if !token.Valid 條件分支

		// Case 1: 使用錯誤 secret - 應觸發 token.Valid = false
		claims := createTestClaims(1, "test@example.com", time.Now().Add(24*time.Hour), time.Now())
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		wrongSecretToken, _ := token.SignedString([]byte("wrong-secret"))

		result, err := ValidateToken(wrongSecretToken)
		if err == nil {
			t.Error("使用錯誤 secret 的 token 應該返回錯誤")
		}
		if result != nil {
			t.Error("驗證失敗時應返回 nil claims")
		}

		// Case 2: 過期的 token - 應觸發 err != nil 且 token.Valid = false
		expiredClaims := createTestClaims(2, "expired@example.com", time.Now().Add(-1*time.Hour), time.Now().Add(-2*time.Hour))
		expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
		expiredTokenString, _ := expiredToken.SignedString(jwtSecret)

		result, err = ValidateToken(expiredTokenString)
		if err == nil {
			t.Error("過期的 token 應該返回錯誤")
		}
		if result != nil {
			t.Error("驗證失敗時應返回 nil claims")
		}
		if !strings.Contains(err.Error(), "expired") {
			t.Errorf("錯誤訊息應該包含 'expired'，實際: %v", err)
		}
	})
}

func TestTokenRoundTrip(t *testing.T) {
	setupTest(t)

	tests := []struct {
		name   string
		userID int64
		email  string
	}{
		{
			name:   "標準用戶",
			userID: 42,
			email:  "roundtrip@example.com",
		},
		{
			name:   "管理員",
			userID: 1,
			email:  "admin@example.com",
		},
		{
			name:   "大 ID 用戶",
			userID: 9999999999,
			email:  "bigid@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 生成 token
			token, err := GenerateToken(tt.userID, tt.email)
			if err != nil {
				t.Fatalf("GenerateToken() failed: %v", err)
			}

			// 驗證 token
			claims, err := ValidateToken(token)
			if err != nil {
				t.Fatalf("ValidateToken() failed: %v", err)
			}

			// 驗證 claims 內容
			if claims.UserID != tt.userID {
				t.Errorf("UserID = %v, want %v", claims.UserID, tt.userID)
			}
			if claims.Email != tt.email {
				t.Errorf("Email = %v, want %v", claims.Email, tt.email)
			}
			if claims.Issuer != "member-api" {
				t.Errorf("Issuer = %v, want member-api", claims.Issuer)
			}

			// 驗證時間相關的 claims
			if claims.ExpiresAt == nil {
				t.Error("ExpiresAt should not be nil")
			}
			if claims.IssuedAt == nil {
				t.Error("IssuedAt should not be nil")
			}

			// 驗證過期時間在未來
			if time.Now().After(claims.ExpiresAt.Time) {
				t.Error("Token should not be expired")
			}

			// 驗證發行時間在過去
			if time.Now().Before(claims.IssuedAt.Time) {
				t.Error("IssuedAt should be in the past")
			}
		})
	}
}

func TestJWTSecretInitialization(t *testing.T) {
	t.Run("檢查 jwtSecret 不為空", func(t *testing.T) {
		setupTest(t)
		if len(jwtSecret) == 0 {
			t.Error("jwtSecret 不應為空")
		}
	})

	t.Run("使用環境變量設置的 secret", func(t *testing.T) {
		setupTest(t)
		expected := []byte("test-secret-key")
		if string(jwtSecret) != string(expected) {
			t.Errorf("jwtSecret = %s, want %s", string(jwtSecret), string(expected))
		}
	})
}
