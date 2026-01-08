package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/anon-d/urlshortener/internal/logger"
	serviceDB "github.com/anon-d/urlshortener/internal/service/storage"
	serviceURL "github.com/anon-d/urlshortener/internal/service/url"
	"github.com/gin-gonic/gin"
)

type APIRequest struct {
	URL string `json:"url" binding:"required"`
}
type APIResponse struct {
	Result string `json:"result"`
}

type URLHandler struct {
	URLService *serviceURL.URLService
	DBService  *serviceDB.DBService
	URLAddr    string
	logger     *logger.Logger
}

func NewURLHandler(urlService *serviceURL.URLService, dbService *serviceDB.DBService, urlAddr string, logger *logger.Logger) *URLHandler {
	return &URLHandler{
		URLService: urlService,
		DBService:  dbService,
		URLAddr:    urlAddr,
		logger:     logger,
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
	id, err := u.URLService.ShortenURL(body)
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

func (u *URLHandler) GetURL(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.String(http.StatusBadRequest, "missing id parameter")
		return
	}
	urlLong, err := u.URLService.GetURL(id)
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

func (u *URLHandler) Shorten(c *gin.Context) {
	var request APIRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.String(http.StatusBadRequest, http.StatusText(400))
		return
	}
	targetURL := request.URL
	id, err := u.URLService.ShortenURL([]byte(targetURL))
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
	if err := u.DBService.DB.Ping(c); err != nil {
		c.String(http.StatusInternalServerError, http.StatusText(500))
		return
	}
	c.String(http.StatusOK, http.StatusText(200))
}
