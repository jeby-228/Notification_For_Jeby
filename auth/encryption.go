package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"os"
)

// GetEncryptionKey 從環境變數獲取加密金鑰，如果不存在則使用預設值（僅供開發環境）
func GetEncryptionKey() []byte {
	key := os.Getenv("ENCRYPTION_KEY")
	if key == "" {
		// 預設金鑰（32 bytes for AES-256），生產環境應該使用環境變數
		key = "default-32-byte-key-for-dev!!"
	}
	// 確保金鑰長度為 32 bytes
	keyBytes := []byte(key)
	if len(keyBytes) < 32 {
		// 補齊到 32 bytes
		padding := make([]byte, 32-len(keyBytes))
		keyBytes = append(keyBytes, padding...)
	} else if len(keyBytes) > 32 {
		// 截斷到 32 bytes
		keyBytes = keyBytes[:32]
	}
	return keyBytes
}

// Encrypt 使用 AES-256-GCM 加密數據
func Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	key := GetEncryptionKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 使用 AES-256-GCM 解密數據
func Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	key := GetEncryptionKey()
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, cipherData := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
