package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"lan-clipboard/config"
	"lan-clipboard/model"
	"lan-clipboard/store"
	"lan-clipboard/ws"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ClipHandler handles CRUD operations for clipboard entries.
type ClipHandler struct {
	store *store.SQLiteStore
	hub   *ws.Hub
	cfg   *config.Config
}

// NewClipHandler creates a new ClipHandler.
func NewClipHandler(s *store.SQLiteStore, hub *ws.Hub, cfg *config.Config) *ClipHandler {
	return &ClipHandler{store: s, hub: hub, cfg: cfg}
}

// CreateClip handles POST /api/clips — create a new text clip.
func (h *ClipHandler) CreateClip(c *gin.Context) {
	var body struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if len(body.Content) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content is required"})
		return
	}

	if len(body.Content) > h.cfg.MaxTextSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content exceeds maximum size of " + strconv.Itoa(h.cfg.MaxTextSize) + " bytes"})
		return
	}

	id, err := uuid.NewV7()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate ID"})
		return
	}

	clip := &model.Clip{
		ID:        id.String(),
		Type:      "text",
		Content:   body.Content,
		CreatedAt: time.Now().UTC(),
		Uploader:  uploaderFromUA(c.Request.UserAgent()),
	}

	if err := h.store.CreateClip(clip); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create clip"})
		return
	}

	// Broadcast via WebSocket
	h.hub.BroadcastMessage(ws.Message{Type: "clip_created", Data: clip})

	c.JSON(http.StatusCreated, clip)
}

// ListClips handles GET /api/clips — list non-pinned clips with cursor pagination.
//
// A cursor pages toward older clips. The after cursor pages toward the nearest
// newer clips. Both directions return clips in descending chronological order.
func (h *ClipHandler) ListClips(c *gin.Context) {
	rawCursor := strings.TrimSpace(c.Query("cursor"))
	after := strings.TrimSpace(c.Query("after"))
	if rawCursor != "" && after != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cursor and after are mutually exclusive"})
		return
	}
	if after != "" {
		rawCursor = after
	}
	cursor, err := parseCursor(rawCursor)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	clipType := c.Query("type")
	if clipType != "" && !model.IsValidClipType(clipType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid clip type"})
		return
	}

	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit > 50 {
		limit = 50
	}

	var clips []model.Clip
	var hasMore bool
	if after != "" {
		clips, hasMore, err = h.store.ListNewerClips(*cursor, limit, clipType)
	} else {
		clips, hasMore, err = h.store.ListClips(cursor, limit, clipType)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list clips"})
		return
	}

	// Populate computed file URLs
	populateFileURLs(clips)

	nextCursor := ""
	if hasMore && len(clips) > 0 {
		boundary := clips[len(clips)-1]
		if after != "" {
			boundary = clips[0]
		}
		nextCursor = boundary.CreatedAt.UTC().Format(time.RFC3339Nano)
	}

	c.JSON(http.StatusOK, gin.H{
		"clips":      clips,
		"hasMore":    hasMore,
		"nextCursor": nextCursor,
	})
}

// ListClipDates indexes the days in the regular timeline in the viewer's time zone.
func (h *ClipHandler) ListClipDates(c *gin.Context) {
	location, err := time.LoadLocation(c.DefaultQuery("timezone", "UTC"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid time zone"})
		return
	}
	clipType := c.Query("type")
	if clipType != "" && !model.IsValidClipType(clipType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid clip type"})
		return
	}
	dates, err := h.store.ListClipDates(c.Request.Context(), location, clipType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list clip dates"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"dates": dates})
}

// parseCursor parses an opaque pagination cursor. An empty cursor returns nil.
// Timestamps are accepted in RFC3339Nano (what we emit) and in a couple of legacy
// layouts so old bookmarked cursors keep working.
func parseCursor(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			t = t.UTC()
			return &t, nil
		}
	}

	return nil, fmt.Errorf("invalid cursor %q", raw)
}

// ListPinnedClips handles GET /api/clips/pinned — list all pinned clips.
func (h *ClipHandler) ListPinnedClips(c *gin.Context) {
	clips, err := h.store.ListPinnedClips()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list pinned clips"})
		return
	}

	populateFileURLs(clips)

	c.JSON(http.StatusOK, gin.H{"clips": clips})
}

// GetClip handles GET /api/clips/:id — get a single clip.
func (h *ClipHandler) GetClip(c *gin.Context) {
	id := c.Param("id")
	clip, err := h.store.GetClip(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get clip"})
		return
	}
	if clip == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "clip not found"})
		return
	}

	populateFileURL(clip)

	c.JSON(http.StatusOK, clip)
}

// UpdateClip handles PATCH /api/clips/:id — toggle pinned status.
func (h *ClipHandler) UpdateClip(c *gin.Context) {
	id := c.Param("id")

	var body struct {
		Pinned *bool `json:"pinned"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Pinned == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pinned field is required"})
		return
	}

	if err := h.store.UpdateClip(id, *body.Pinned); err != nil {
		if err.Error() == "clip not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "clip not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update clip"})
		return
	}

	// Fetch updated clip to broadcast
	clip, err := h.store.GetClip(id)
	if err != nil || clip == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load updated clip"})
		return
	}
	populateFileURL(clip)
	h.hub.BroadcastMessage(ws.Message{Type: "clip_updated", Data: clip})

	c.JSON(http.StatusOK, clip)
}

// DeleteClip handles DELETE /api/clips/:id — delete a single clip and its files.
func (h *ClipHandler) DeleteClip(c *gin.Context) {
	id := c.Param("id")

	clip, err := h.store.DeleteClip(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete clip"})
		return
	}
	if clip == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "clip not found"})
		return
	}

	// Clean up files from disk
	deleteClipFiles(h.cfg.DataDir, clip)

	h.hub.BroadcastMessage(ws.Message{Type: "clip_deleted", Data: gin.H{"id": id}})

	c.JSON(http.StatusOK, gin.H{"message": "clip deleted"})
}

// DeleteAllClips handles DELETE /api/clips — clear all clips and their files.
func (h *ClipHandler) DeleteAllClips(c *gin.Context) {
	clips, err := h.store.DeleteAllClips()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete all clips"})
		return
	}

	// Clean up all files from disk
	for i := range clips {
		deleteClipFiles(h.cfg.DataDir, &clips[i])
	}

	h.hub.BroadcastMessage(ws.Message{Type: "clips_cleared", Data: nil})

	c.JSON(http.StatusOK, gin.H{"message": "all clips deleted", "count": len(clips)})
}

// GetStats handles GET /api/stats — return storage statistics.
func (h *ClipHandler) GetStats(c *gin.Context) {
	totalClips, totalSize, err := h.store.GetStorageStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"totalClips": totalClips,
		"totalSize":  totalSize,
	})
}
