package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"lan-clipboard/config"
	"lan-clipboard/handler"
	"lan-clipboard/middleware"
	"lan-clipboard/store"
	"lan-clipboard/ws"

	"github.com/gin-gonic/gin"
)

// maxMultipartMemory is the amount of a multipart upload kept in RAM before
// spilling to a temporary file on disk. It must NOT be as large as the maximum
// file size, otherwise every big upload would be buffered in memory.
const maxMultipartMemory = 32 << 20 // 32 MB

// newRouter wires all API, WebSocket and static-file routes onto a Gin engine.
func newRouter(cfg *config.Config, db *store.SQLiteStore, hub *ws.Hub) *gin.Engine {
	clipHandler := handler.NewClipHandler(db, hub, cfg)
	fileHandler := handler.NewFileHandler(db, hub, cfg)
	officePreview := handler.NewOfficePreviewHandler(db, cfg)
	searchHandler := handler.NewSearchHandler(db)
	wsHandler := handler.NewWSHandler(hub, cfg.PublicOrigin)

	router := gin.Default()
	router.MaxMultipartMemory = maxMultipartMemory
	router.Use(middleware.AllowedHost(cfg.PublicOrigin))

	// API routes — specific routes must be registered before parameterised ones.
	api := router.Group("/api", middleware.SameOrigin(cfg.PublicOrigin))
	{
		api.POST("/clips", clipHandler.CreateClip)
		api.GET("/clips", clipHandler.ListClips)
		api.GET("/clips/dates", clipHandler.ListClipDates)
		api.GET("/clips/pinned", clipHandler.ListPinnedClips)
		api.GET("/clips/search", searchHandler.Search)
		api.POST("/clips/upload", fileHandler.Upload)
		api.GET("/clips/:id", clipHandler.GetClip)
		api.GET("/clips/:id/preview", officePreview.ServePDF)
		api.PATCH("/clips/:id", clipHandler.UpdateClip)
		api.DELETE("/clips/:id", clipHandler.DeleteClip)
		api.DELETE("/clips", clipHandler.DeleteAllClips)

		// File serving
		api.GET("/files/*filepath", fileHandler.ServeFile)
		api.GET("/thumbnails/*filepath", fileHandler.ServeThumbnail)

		// Stats
		api.GET("/stats", clipHandler.GetStats)
	}

	// WebSocket endpoint
	router.GET("/ws", wsHandler.Handle)

	// Built frontend (optional — API keeps working without it)
	registerStaticRoutes(router, cfg.StaticDir)

	return router
}

// registerStaticRoutes serves the built SPA from staticDir with an
// index.html fallback for client-side routes.
func registerStaticRoutes(router *gin.Engine, staticDir string) {
	indexPath := filepath.Join(staticDir, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		log.Printf("Static frontend not found at %s — serving API only (run `make build` to build the web UI)", staticDir)
		router.NoRoute(func(c *gin.Context) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		})
		return
	}

	if assetsDir := filepath.Join(staticDir, "assets"); pathExists(assetsDir) {
		router.Static("/assets", assetsDir)
	}
	if favicon := filepath.Join(staticDir, "favicon.ico"); pathExists(favicon) {
		router.StaticFile("/favicon.ico", favicon)
	}

	router.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		// Never let the SPA fallback swallow API or WebSocket paths.
		if path == "/api" || strings.HasPrefix(path, "/api/") || path == "/ws" {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.File(indexPath)
	})
}

// pathExists reports whether path exists (file or directory).
func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
