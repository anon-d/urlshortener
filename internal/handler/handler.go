package handler

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	myError "github.com/anon-d/urlshortener/internal/error"
	"github.com/anon-d/urlshortener/internal/service/url"
	"github.com/gin-gonic/gin"
)

type URLHandler struct {
	URLService *url.URLService
	URLAddr    string
}

func NewURLHandler(urlService *url.URLService, urlAddr string) *URLHandler {
	return &URLHandler{
		URLService: urlService,
		URLAddr:    urlAddr,
	}
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
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	shortURL := fmt.Sprintf("%s/%s", u.URLAddr, id)
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
		if errors.Is(err, myError.ErrNotFound) {
			c.String(http.StatusNotFound, err.Error())
			return
		}
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	c.Redirect(http.StatusTemporaryRedirect, urlLong)
}
