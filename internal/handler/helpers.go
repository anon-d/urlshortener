package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// getUserID извлекает user_id из контекста Gin
// Возвращает userID и true если успешно, пустую строку и false если нет
func getUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return "", false
	}

	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		return "", false
	}

	return userIDStr, true
}

// requireUserID проверяет наличие user_id в контексте и возвращает его
// Если user_id отсутствует, отправляет 401
func requireUserID(c *gin.Context) (string, bool) {
	userID, ok := getUserID(c)
	if !ok {
		c.String(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
		return "", false
	}
	return userID, true
}
