package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncryptDecrypt(t *testing.T) {
	tests := []struct {
		name      string
		plaintext string
		wantErr   bool
	}{
		{
			name:      "加密和解密普通字符串",
			plaintext: "Hello, World!",
			wantErr:   false,
		},
		{
			name:      "加密和解密敏感資料",
			plaintext: "password123!@#",
			wantErr:   false,
		},
		{
			name:      "加密和解密空字符串",
			plaintext: "",
			wantErr:   false,
		},
		{
			name:      "加密和解密長字符串",
			plaintext: "This is a very long string that contains a lot of characters and should be encrypted and decrypted properly without any issues at all. Let's make it even longer to test the encryption properly!",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 加密
			encrypted, err := Encrypt(tt.plaintext)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			// 空字符串應該返回空字符串
			if tt.plaintext == "" {
				assert.Equal(t, "", encrypted)
				return
			}

			// 加密後的字符串應該與原始字符串不同
			assert.NotEqual(t, tt.plaintext, encrypted)

			// 解密
			decrypted, err := Decrypt(encrypted)
			assert.NoError(t, err)
			assert.Equal(t, tt.plaintext, decrypted)
		})
	}
}

func TestDecryptInvalidData(t *testing.T) {
	tests := []struct {
		name       string
		ciphertext string
	}{
		{
			name:       "解密無效的 base64",
			ciphertext: "invalid-base64!!!",
		},
		{
			name:       "解密過短的密文",
			ciphertext: "YWJj",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decrypt(tt.ciphertext)
			assert.Error(t, err)
		})
	}
}

func TestDecryptEmptyString(t *testing.T) {
	decrypted, err := Decrypt("")
	assert.NoError(t, err)
	assert.Equal(t, "", decrypted)
}
