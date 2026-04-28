package api

import (
	"autocomplete/internal/events"
	"autocomplete/internal/kafka"
	"autocomplete/internal/trie"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Trie     *trie.AutoComplete
	Producer *kafka.Producer // write search logs to Kafka
	Bus      *events.EventBus
}

func NewHandler(t *trie.AutoComplete) *Handler {
	return &Handler{Trie: t}
}

// GET /v1/search?q=family
func (h *Handler) Search(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "query is required",
		})
		return
	}

	results := h.Trie.Search(q)
	c.JSON(http.StatusOK,
		results,
	)
}

// POST /v1/query
// Body: { "query": "some text" }
func (h *Handler) LogSelection(c *gin.Context) {
	var body struct {
		Query string `json:"query"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	if body.Query == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "query is required",
		})
		return
	}

	// NON-BLOCKING
	h.Bus.Emit(body.Query)
	c.JSON(http.StatusOK, gin.H{
		"message": "selection logged",
	})
}
