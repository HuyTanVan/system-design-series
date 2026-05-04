package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
}

func NewHandler() *Handler {
	return &Handler{}
}

// GET api/v1/url?q=some+query
func (h *Handler) GetURL(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "query is required",
		})
		return
	}

	// 1. Validate query length
	// 2. query url from cache
	// 3. not in cache, query from main db
	// 4. update cache return result
	c.JSON(http.StatusOK,
		results,
	)
}

// POST api/v1/url
// Body: { "url": "long url", "custom_alias": "optional custom alias" }
func (h *Handler) LogSelection(c *gin.Context) {
	var body struct {
		URL         string `json:"url"`
		CustomAlias string `json:"custom_alias,omitempty"`
	}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}
	// 1. validate url
	// 2. custom alias is provided, check if it is available
	// 3. generate short url, save to db and cache
	c.JSON(http.StatusOK, gin.H{
		"short_url": "http://short.url/abc123",
	})
}
