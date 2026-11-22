package handler

import (
	"net/http"
	"strings"

	"lan-clipboard/model"
	"lan-clipboard/store"

	"github.com/gin-gonic/gin"
)

// SearchHandler handles full-text search operations.
type SearchHandler struct {
	store *store.SQLiteStore
}

// NewSearchHandler creates a new SearchHandler.
func NewSearchHandler(s *store.SQLiteStore) *SearchHandler {
	return &SearchHandler{store: s}
}

// Search handles GET /api/clips/search?q=keyword&type=text|image|file.
func (h *SearchHandler) Search(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter 'q' is required"})
		return
	}

	clipType := c.Query("type")
	if clipType != "" && !model.IsValidClipType(clipType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid clip type"})
		return
	}

	clips, err := h.store.SearchClips(query, clipType, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "search failed"})
		return
	}

	// Populate computed file URLs
	populateFileURLs(clips)

	c.JSON(http.StatusOK, gin.H{"clips": clips})
}
