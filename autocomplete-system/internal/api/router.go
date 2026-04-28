package api

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, h *Handler) {
	// middlewares
	r.Use(CORSMiddleware())
	r.Use(LoggingMiddleware())

	// serve UI
	r.StaticFile("/", "./ui/index.html")

	// routes matching old UI
	r.GET("/search", h.Search)
	r.POST("/query", h.LogSelection)
	// r.GET("/size", h.Size)
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
