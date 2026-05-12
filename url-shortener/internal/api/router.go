package api

import (
	"log"
	"net/http"
	"time"
	"url-shortener/internal/config"
	"url-shortener/internal/store"

	"github.com/gin-gonic/gin"
)

func NewRouter(s *store.Store, cfg *config.Config) *gin.Engine {
	r := gin.Default()

	h := NewHandler(s, cfg.BaseURL, cfg.DefaultTTLDays)

	r.StaticFile("/", "./ui/index.html")
	r.POST("/urls", h.Shorten)
	r.GET("/stats/:code", h.Stats) // must be before /:code
	r.GET("/:code", h.Redirect)

	return r
}
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST")
		c.Header("Access-Control-Allow-Headers", "Content-Type")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusOK)
			return
		}

		c.Next()
	}
}

func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		if c.Request.URL.RawQuery != "" {
			log.Printf("%s %s?%s — %s", c.Request.Method, c.Request.URL.Path, c.Request.URL.RawQuery, time.Since(start))
		} else {
			log.Printf("%s %s — %s", c.Request.Method, c.Request.URL.Path, time.Since(start))
		}
	}
}
