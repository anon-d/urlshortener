// Package handler содержит HTTP-обработчики для сервиса сокращения URL.
// Обработчики работают поверх фреймворка Gin и реализуют CRUD-операции
// над короткими ссылками.
package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/anon-d/urlshortener/internal/audit"
	"github.com/anon-d/urlshortener/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// APIRequest — тело запроса для эндпоинта POST /api/shorten.
// Содержит оригинальный URL, который необходимо сократить.
type APIRequest struct {
	URL string `json:"url" binding:"required"`
}

// APIResponse — тело ответа для эндпоинта POST /api/shorten.
// Содержит результирующий короткий URL.
type APIResponse struct {
	Result string `json:"result"`
}

// ItemBatchRequest — элемент массива в теле запроса для пакетного сокращения
// (POST /api/shorten/batch). Связывает correlation_id с оригинальным URL.
type ItemBatchRequest struct {
	CorrelationID string `json:"correlation_id,omitzero"`
	OriginalURL   string `json:"original_url,omitzero"`
}

// ItemBatchResponse — элемент массива в теле ответа для пакетного сокращения.
// Связывает correlation_id с результирующим коротким URL.
type ItemBatchResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

// UserURLResponse — элемент ответа для эндпоинта GET /api/user/urls.
// Содержит пару короткий URL — оригинальный URL.
type UserURLResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// URLHandler содержит зависимости и методы для обработки HTTP-запросов
// к сервису сокращения URL.
type URLHandler struct {
	Service       *service.Service
	URLAddr       string
	logger        *zap.SugaredLogger
	DeleteChannel chan<- DeleteRequest
	audit         *audit.Publisher
}

// DeleteRequest представляет запрос на асинхронное удаление URL.
// Передаётся через канал в DeleteWorker для пакетной обработки.
type DeleteRequest struct {
	UserID   string
	ShortURL string
}

// NewURLHandler создаёт новый экземпляр URLHandler.
// Принимает сервис бизнес-логики, базовый адрес для формирования коротких ссылок,
// логгер, канал для отправки запросов на удаление и издателя аудита.
func NewURLHandler(service *service.Service, urlAddr string, logger *zap.SugaredLogger, deleteChan chan<- DeleteRequest, auditPublisher *audit.Publisher) *URLHandler {
	return &URLHandler{
		Service:       service,
		URLAddr:       urlAddr,
		logger:        logger,
		DeleteChannel: deleteChan,
		audit:         auditPublisher,
	}
}

// NotAllowed возвращает 405 Method Not Allowed.
func (u *URLHandler) NotAllowed(c *gin.Context) {
	c.JSON(http.StatusMethodNotAllowed, gin.H{
		"status":  "Error",
		"message": "Method not allowed",
	})
}

// HealthCheck возвращает 200 OK для проверки работоспособности сервиса.
func (u *URLHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "Success",
		"message": "Health check passed",
	})
}

// NotFound возвращает 404 Not Found.
func (u *URLHandler) NotFound(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{
		"status":  "Error",
		"message": "Not found",
	})
}

// PostURL принимает запрос на создание короткой ссылки по корневому пути,
// оригинальный URL берется из тела запроса.
// Возвращается baseURL+shortURL.
func (u *URLHandler) PostURL(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	if len(body) == 0 {
		c.String(http.StatusBadRequest, "empty request body")
		return
	}

	userID, _ := getUserID(c)
	id, err := u.Service.ShortenURL(c, string(body), userID)
	if err != nil {
		var conflictErr *service.ConflictError
		if errors.As(err, &conflictErr) {
			u.logger.Warnw("URL already exists", "error", conflictErr)
			// URL уже существует, возвращаем 409
			shortURL, joinErr := url.JoinPath(u.URLAddr, id)
			if joinErr != nil {
				u.logger.Errorw("failed to join URL path", "error", joinErr)
				c.String(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
				return
			}
			c.String(http.StatusConflict, shortURL)
			return
		}
		u.logger.Errorw("failed to shorten URL", "error", err)
		c.String(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}
	shortURL, err := url.JoinPath(u.URLAddr, id)
	if err != nil {
		u.logger.Errorw("failed to join URL path", "error", err)
		c.String(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}
	u.publishAudit("shorten", userID, string(body))
	c.String(http.StatusCreated, shortURL)
}

// GetURL принимает запрос на получение оригинальной ссылки по корневому пути,
// shortURL берется из параметра запроса (id).
// Оригинальный URL возвращается в заголовке Location со статусом 307.
// Если URL помечен как удаленный, возвращается 410 Gone.
func (u *URLHandler) GetURL(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.String(http.StatusBadRequest, "missing id parameter")
		return
	}

	// Получаем полные данные URL
	data, err := u.Service.GetURLByShortURL(c, id)
	if err != nil {
		u.logger.Errorw("failed to get URL", "error", err, "short_url", id)
		c.String(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
		return
	}

	// Проверка на is_deleted
	if data.IsDeleted {
		c.String(http.StatusGone, http.StatusText(http.StatusGone))
		return
	}

	userID, _ := getUserID(c)
	u.publishAudit("follow", userID, data.OriginalURL)
	c.Redirect(http.StatusTemporaryRedirect, data.OriginalURL)
}

// Shorten принимает запрос на создание короткой ссылки по пути "/api/shorten",
// оригинальный URL берется из тела запроса по ключу "url".
// Возвращается baseURL+shortURL в теле ответа по ключу "result".
func (u *URLHandler) Shorten(c *gin.Context) {
	var request APIRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.String(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
		return
	}

	userID, _ := getUserID(c)
	targetURL := request.URL
	id, err := u.Service.ShortenURL(c, targetURL, userID)

	shortURL, joinErr := url.JoinPath(u.URLAddr, id)
	if joinErr != nil {
		u.logger.Errorw("failed to join URL path", "error", joinErr)
		c.String(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}

	c.Writer.Header().Set("Content-Type", "application/json")

	if err != nil {
		var conflictErr *service.ConflictError
		if errors.As(err, &conflictErr) {
			// URL уже существует, возвращаем 409
			c.Writer.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(c.Writer).Encode(APIResponse{Result: shortURL})
			return
		}
		u.logger.Errorw("failed to shorten URL", "error", err)
		c.String(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}

	u.publishAudit("shorten", userID, targetURL)
	c.Writer.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(c.Writer).Encode(APIResponse{Result: shortURL})
}

// BatchShorten принимает запрос на пакетное сокращение URL по пути "/api/shorten/batch",
// оригинальные URL берутся из тела запроса.
// Возвращается correlation_id, short_url (baseURL+shortURL) в теле ответа.
func (u *URLHandler) BatchShorten(c *gin.Context) {
	var request []ItemBatchRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.String(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
		return
	}

	if len(request) == 0 {
		c.String(http.StatusBadRequest, "empty batch")
		return
	}

	userID, _ := getUserID(c)

	batchURLsMap := make(map[string]string, len(request))
	for _, item := range request {
		batchURLsMap[item.CorrelationID] = item.OriginalURL
	}
	batchURLsMap, err := u.Service.ShortenBatchURL(c, batchURLsMap, userID)
	if err != nil {
		u.logger.Errorw("failed to shorten batch URLs", "error", err)
		c.String(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}

	response := make([]ItemBatchResponse, 0, len(request))
	for _, item := range request {
		shortURL, err := url.JoinPath(u.URLAddr, batchURLsMap[item.CorrelationID])
		if err != nil {
			u.logger.Errorw("failed to join URL path", "error", err)
			c.String(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
			return
		}
		response = append(response, ItemBatchResponse{
			CorrelationID: item.CorrelationID,
			ShortURL:      shortURL,
		})
	}
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(c.Writer).Encode(response); err != nil {
		u.logger.Errorw("failed to encode response", "error", err)
		c.String(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}
}

// PingDB проверяет доступность хранилища данных.
// Возвращает 200 OK при успешном подключении, 500 Internal Server Error в противном случае.
func (u *URLHandler) PingDB(c *gin.Context) {
	if u.Service.Storage == nil {
		u.logger.Warnw("storage is not initialized")
		c.String(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}
	if err := u.Service.Storage.Ping(c); err != nil {
		u.logger.Warnw("storage ping failed", "error", err)
		c.String(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}
	c.String(http.StatusOK, http.StatusText(http.StatusOK))
}

// GetUserURLs возвращает все URL, созданные пользователем
func (u *URLHandler) GetUserURLs(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	urls, err := u.Service.GetURLsByUser(c, userID)
	if err != nil {
		u.logger.Errorw("failed to get user URLs", "error", err)
		c.String(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		return
	}

	if len(urls) == 0 {
		c.AbortWithStatus(http.StatusNoContent)
		return
	}

	response := make([]UserURLResponse, 0, len(urls))
	for _, item := range urls {
		shortURL, err := url.JoinPath(u.URLAddr, item.ShortURL)
		if err != nil {
			u.logger.Errorw("failed to join URL path", "error", err)
			c.String(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
			return
		}
		response = append(response, UserURLResponse{
			ShortURL:    shortURL,
			OriginalURL: item.OriginalURL,
		})
	}

	c.JSON(http.StatusOK, response)
}

// DeleteURLs принимает запрос на асинхронное удаление URL
func (u *URLHandler) DeleteURLs(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	// Парсим массив shortURL из тела запроса
	var shortURLs []string
	if err := c.ShouldBindJSON(&shortURLs); err != nil {
		u.logger.Errorw("failed to parse delete request", "error", err)
		c.String(http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
		return
	}

	if len(shortURLs) == 0 {
		c.String(http.StatusBadRequest, "empty URL list")
		return
	}

	// Отправляем запросы на удаление в канал для асинхронной обработки
	// Блокирующая отправка
	for _, shortURL := range shortURLs {
		req := DeleteRequest{
			UserID:   userID,
			ShortURL: shortURL,
		}
		u.DeleteChannel <- req
	}

	c.Status(http.StatusAccepted)
}

// publishAudit отправляет событие аудита во все зарегистрированные приёмники.
func (u *URLHandler) publishAudit(action, userID, originalURL string) {
	if u.audit == nil {
		return
	}
	u.audit.Publish(audit.AuditEvent{
		Timestamp: time.Now().Unix(),
		Action:    action,
		UserID:    userID,
		URL:       originalURL,
	})
}
