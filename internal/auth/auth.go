// Package auth содержит общую логику авторизации,
// переиспользуемую HTTP- и gRPC-хендлерами.
package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"strings"
)

// GenerateUserID генерирует уникальный идентификатор пользователя.
func GenerateUserID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// SignValue подписывает значение с помощью HMAC-SHA256.
func SignValue(value, secretKey string) string {
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(value))
	signature := base64.URLEncoding.EncodeToString(h.Sum(nil))
	return value + "." + signature
}

// ValidateSignedValue проверяет подпись и возвращает оригинальное значение.
func ValidateSignedValue(signedValue, secretKey string) (string, bool) {
	parts := strings.Split(signedValue, ".")
	if len(parts) != 2 {
		return "", false
	}

	value := parts[0]
	providedSignature := parts[1]

	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(value))
	expectedSignature := base64.URLEncoding.EncodeToString(h.Sum(nil))

	if hmac.Equal([]byte(expectedSignature), []byte(providedSignature)) {
		return value, true
	}

	return "", false
}
