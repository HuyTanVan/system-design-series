package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"url-shortener/internal/store"
)

type Handler struct {
	store          *store.Store
	baseURL        string
	defaultTTLDays int
}

func NewHandler(s *store.Store, baseURL string, defaultTTLDays int) *Handler {
	return &Handler{store: s, baseURL: baseURL, defaultTTLDays: defaultTTLDays}
}

type UrlShortenRequest struct {
	URL     string `json:"url"      binding:"required,url"`
	Alias   string `json:"alias"`
	TTLDays int    `json:"ttl_days"`
}

// POST api/url/shorten
func (h *Handler) Shorten(c *gin.Context) {
	var req UrlShortenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.store.Shorten(c.Request.Context(), h.baseURL, store.ShortenRequest{
		Original:       req.URL,
		Alias:          req.Alias,
		TTLDays:        req.TTLDays,
		DefaultTTLDays: h.defaultTTLDays,
	})
	if errors.Is(err, store.ErrAliasConflict) {
		c.JSON(http.StatusConflict, gin.H{"error": "alias already in use"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not shorten url"})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// GET /:code
func (h *Handler) Redirect(c *gin.Context) {
	code := c.Param("code")

	url, err := h.store.Resolve(c.Request.Context(), code)
	if errors.Is(err, store.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "short link not found"})
		return
	}
	if errors.Is(err, store.ErrExpired) {
		c.JSON(http.StatusGone, gin.H{"error": "short link has expired"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not resolve url"})
		return
	}

	go h.store.RecordClick(url.ID, c.ClientIP(), c.Request.UserAgent(), c.Request.Referer())

	c.Redirect(http.StatusMovedPermanently, url.Original)
}

// GET /stats/:code
func (h *Handler) Stats(c *gin.Context) {
	code := c.Param("code")

	url, summary, err := h.store.GetStats(c.Request.Context(), code)
	if errors.Is(err, store.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "short link not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not get stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":       url.Code,
		"original":   url.Original,
		"alias":      url.Alias,
		"expires_at": url.ExpiresAt,
		"created_at": url.CreatedAt.Format(time.RFC3339),
		"clicks":     summary,
	})
}
