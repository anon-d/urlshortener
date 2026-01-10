package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/anon-d/urlshortener/internal/logger"
	"github.com/anon-d/urlshortener/internal/service"
	"github.com/gin-gonic/gin"
)

type APIRequest struct {
	URL string `json:"url" binding:"required"`
}
type APIResponse struct {
	Result string `json:"result"`
}

type URLHandler struct {
	Service *service.Service
	URLAddr string
	logger  *logger.Logger
}

func NewURLHandler(service *service.Service, urlAddr string, logger *logger.Logger) *URLHandler {
	return &URLHandler{
		Service: service,
		URLAddr: urlAddr,
		logger:  logger,
	}
}

func (u *URLHandler) NotAllowed(c *gin.Context) {
	c.JSON(405, gin.H{
		"status":  "Error",
		"message": "Method not allowed",
	})
}

func (u *URLHandler) HealthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":  "Success",
		"message": "Health check passed",
	})
}

func (u *URLHandler) NotFound(c *gin.Context) {
	c.JSON(404, gin.H{
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
	id, err := u.Service.ShortenURL(c, body)
	if err != nil {
		c.String(http.StatusInternalServerError, http.StatusText(500))
		return
	}
	shortURL, err := url.JoinPath(u.URLAddr, string(id))
	if err != nil {
		c.String(http.StatusInternalServerError, http.StatusText(500))
		return
	}
	c.String(http.StatusCreated, shortURL)
}

// GetURL принимает запрос на получение оригинальной ссылки по корневому пути,
// shortURL берется из параметра запроса (id).
// Оригинальный URL возвращается в заголовке Location со статусом 307.
func (u *URLHandler) GetURL(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.String(http.StatusBadRequest, "missing id parameter")
		return
	}
	urlLong, err := u.Service.GetURL(c, id)
	if err != nil {
		if errors.Is(err, errors.New("not found")) {
			c.String(http.StatusNotFound, err.Error())
			return
		}
		c.String(http.StatusBadRequest, http.StatusText(400))
		return
	}
	c.Redirect(http.StatusTemporaryRedirect, urlLong)
}

// Shorten принимает запрос на создание короткой ссылки по пути "/api/shorten",
// оригинальный URL берется из тела запроса по ключу "url".
// Возвращается baseURL+shortURL в теле ответа по ключу "result".
func (u *URLHandler) Shorten(c *gin.Context) {
	var request APIRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.String(http.StatusBadRequest, http.StatusText(400))
		return
	}
	targetURL := request.URL
	id, err := u.Service.ShortenURL(c, []byte(targetURL))
	if err != nil {
		c.String(http.StatusInternalServerError, http.StatusText(500))
		return
	}
	shortURL, err := url.JoinPath(u.URLAddr, string(id))
	if err != nil {
		c.String(http.StatusInternalServerError, http.StatusText(500))
		return
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(c.Writer).Encode(APIResponse{Result: shortURL}); err != nil {
		c.String(http.StatusInternalServerError, http.StatusText(500))
		return
	}
}

func (u *URLHandler) PingDB(c *gin.Context) {
	if u.Service.DB == nil {
		c.String(http.StatusInternalServerError, http.StatusText(500))
		return
	}
	if err := u.Service.DB.Ping(c); err != nil {
		c.String(http.StatusInternalServerError, http.StatusText(500))
		return
	}
	c.String(http.StatusOK, http.StatusText(200))
}
