package api

import (
	events "autocomplete/internal/event"
	"autocomplete/internal/kafka"
	"autocomplete/internal/trie"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	Trie     *trie.AutoComplete
	Producer *kafka.Producer // write search logs to Kafka
	Queue    *events.AsyncEventQueue
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
	fmt.Printf("Trie size: %d bytes\n", h.Trie.TrieSize())
	// // total trie memory
	// total := trie.CalculateNodeSize(h.Trie.Root) // recursive version
	// fmt.Printf("Total trie: %s\n", formatBytes(total))

	// // single node memory
	// single := h.Trie.GetSingleNodeSize(q) // non-recursive version
	// fmt.Printf("Single node: %s\n", formatBytes(single))
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
	if len(body.Query) > 30 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "query is too long",
		})
		return
	}

	// NON-BLOCKING
	h.Queue.Emit(body.Query)
	c.JSON(http.StatusOK, gin.H{
		"message": "selection logged",
	})
}

func formatBytes(b uintptr) string {
	const unit = uintptr(1024)

	if b < unit {
		return fmt.Sprintf("%d B", b)
	}

	div, exp := unit, 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
