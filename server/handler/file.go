package handler

import (
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lan-clipboard/config"
	"lan-clipboard/model"
	"lan-clipboard/store"
	"lan-clipboard/ws"

	"github.com/disintegration/imaging"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// FileHandler handles file upload and download operations.
type FileHandler struct {
	store *store.SQLiteStore
	hub   *ws.Hub
	cfg   *config.Config
}

// NewFileHandler creates a new FileHandler.
func NewFileHandler(s *store.SQLiteStore, hub *ws.Hub, cfg *config.Config) *FileHandler {
	return &FileHandler{store: s, hub: hub, cfg: cfg}
}

// Upload handles POST /api/clips/upload — upload one or more files.
func (h *FileHandler) Upload(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid multipart form"})
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		// Also try the "file" key for single-file uploads
		files = form.File["file"]
	}
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no files provided"})
		return
	}

	maxSize := h.cfg.MaxFileSizeBytes()
	var createdClips []model.Clip
	uploader := uploaderFromUA(c.Request.UserAgent())

	for _, fileHeader := range files {
		// Validate file size
		if fileHeader.Size > maxSize {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("file %s exceeds maximum size of %d MB", fileHeader.Filename, h.cfg.MaxFileSizeMB),
			})
			return
		}

		// Generate UUID v7 for this file
		id, err := uuid.NewV7()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate ID"})
			return
		}

		// Build storage path: {DATA_DIR}/uploads/YYYY/MM/DD/
		now := time.Now().UTC()
		dateDir := filepath.Join(h.cfg.DataDir, "uploads",
			fmt.Sprintf("%04d", now.Year()),
			fmt.Sprintf("%02d", int(now.Month())),
			fmt.Sprintf("%02d", now.Day()),
		)
		if err := os.MkdirAll(dateDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create upload directory"})
			return
		}

		// Sanitize filename: strip directories and control characters
		safeName := sanitizeFileName(fileHeader.Filename)
		storedName := id.String() + "_" + safeName
		filePath := filepath.Join(dateDir, storedName)

		// Save file to disk
		src, err := fileHeader.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read uploaded file"})
			return
		}

		dst, err := os.Create(filePath)
		if err != nil {
			src.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
			return
		}

		if _, err := io.Copy(dst, src); err != nil {
			src.Close()
			dst.Close()
			os.Remove(filePath)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write file"})
			return
		}
		src.Close()
		dst.Close()

		// Determine MIME type
		mimeType := fileHeader.Header.Get("Content-Type")
		if mimeType == "" || mimeType == "application/octet-stream" {
			// Try to detect from extension
			mimeType = mime.TypeByExtension(filepath.Ext(safeName))
			if mimeType == "" {
				mimeType = "application/octet-stream"
			}
		}

		// Determine clip type
		clipType := model.TypeFile
		if strings.HasPrefix(mimeType, "image/") {
			clipType = model.TypeImage
		}

		// Generate a thumbnail for images above the configured threshold.
		hasThumb := false
		thumbPath := ""
		if clipType == model.TypeImage && fileHeader.Size > h.cfg.ThumbnailThresholdBytes() {
			thumbPath = filepath.Join(dateDir, thumbFileName(id.String(), safeName))
			if err := generateThumbnail(filePath, thumbPath); err != nil {
				log.Printf("Warning: failed to generate thumbnail for %s: %v", safeName, err)
				thumbPath = ""
			} else {
				hasThumb = true
			}
		}

		// Build the relative path for URL construction
		// e.g., "2024/01/15/uuid_photo.jpg"
		uploadRoot := filepath.Join(h.cfg.DataDir, "uploads")
		relPath, _ := filepath.Rel(uploadRoot, filePath)

		clip := model.Clip{
			ID:        id.String(),
			Type:      clipType,
			FileName:  safeName,
			FilePath:  filePath,
			FileSize:  fileHeader.Size,
			MimeType:  mimeType,
			HasThumb:  hasThumb,
			Pinned:    false,
			CreatedAt: now,
			Uploader:  uploader,
			FileURL:   uploadURL("/api/files/", relPath),
		}

		if hasThumb {
			if thumbRelPath, err := filepath.Rel(uploadRoot, thumbPath); err == nil {
				clip.ThumbURL = uploadURL("/api/thumbnails/", thumbRelPath)
			}
		}

		if err := h.store.CreateClip(&clip); err != nil {
			// Roll back the file(s) so failed uploads don't leak disk space.
			os.Remove(filePath)
			if thumbPath != "" {
				os.Remove(thumbPath)
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create clip record"})
			return
		}

		// Broadcast each file as it's created
		h.hub.BroadcastMessage(ws.Message{Type: "clip_created", Data: clip})

		createdClips = append(createdClips, clip)
	}

	c.JSON(http.StatusCreated, gin.H{"clips": createdClips})
}

// ServeFile handles GET /api/files/*filepath — serve uploaded files.
func (h *FileHandler) ServeFile(c *gin.Context) {
	absPath, err := h.resolveUploadPath(c.Param("filepath"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}

	info, err := os.Stat(absPath)
	if err != nil || info.IsDir() {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	// Strip the UUID prefix for the download filename: "uuid_originalname" → "originalname"
	filename := filepath.Base(absPath)
	if parts := strings.SplitN(filename, "_", 2); len(parts) == 2 {
		filename = parts[1]
	}

	mimeType := mime.TypeByExtension(filepath.Ext(absPath))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	disposition := "attachment"
	if strings.HasPrefix(mimeType, "image/") {
		disposition = "inline"
	}
	if c.Query("preview") == "1" {
		if previewType := inlinePreviewMIME[strings.ToLower(filepath.Ext(filename))]; previewType != "" {
			if previewType == htmlPreviewMIME {
				if info.Size() > maxHTMLPreview {
					c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "HTML exceeds preview size limit"})
					return
				}
				c.Header("Content-Security-Policy", htmlPreviewPolicy)
				c.Header("Referrer-Policy", "no-referrer")
				c.Header("X-DNS-Prefetch-Control", "off")
			}
			mimeType = previewType
			disposition = "inline"
			c.Header("X-Content-Type-Options", "nosniff")
		}
	}
	c.Header("Content-Type", mimeType)
	if contentDisposition := mime.FormatMediaType(disposition, map[string]string{"filename": filename}); contentDisposition != "" {
		c.Header("Content-Disposition", contentDisposition)
	}

	c.File(absPath)
}

// ServeThumbnail handles GET /api/thumbnails/*filepath — serve thumbnail files.
func (h *FileHandler) ServeThumbnail(c *gin.Context) {
	absPath, err := h.resolveUploadPath(c.Param("filepath"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}

	info, err := os.Stat(absPath)
	if err != nil || info.IsDir() {
		c.JSON(http.StatusNotFound, gin.H{"error": "thumbnail not found"})
		return
	}

	c.Header("Content-Type", "image/jpeg")
	c.Header("Content-Disposition", "inline")
	c.File(absPath)
}

// resolveUploadPath safely maps a wildcard request path to a file below
// {DataDir}/uploads, rejecting any path that escapes the upload root.
func (h *FileHandler) resolveUploadPath(reqPath string) (string, error) {
	// Reject explicit ".." path segments before cleaning (Clean would silently
	// collapse them and could make a traversal look harmless).
	for _, segment := range strings.FieldsFunc(reqPath, func(r rune) bool {
		return r == '/' || r == '\\'
	}) {
		if segment == ".." {
			return "", fmt.Errorf("invalid path")
		}
	}

	clean := filepath.Clean("/" + reqPath)
	clean = strings.TrimPrefix(clean, string(os.PathSeparator))

	base, err := filepath.Abs(filepath.Join(h.cfg.DataDir, "uploads"))
	if err != nil {
		return "", err
	}
	absPath, err := filepath.Abs(filepath.Join(base, clean))
	if err != nil {
		return "", err
	}
	if absPath != base && !strings.HasPrefix(absPath, base+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes upload directory")
	}
	return absPath, nil
}

// sanitizeFileName reduces an arbitrary uploaded filename to a safe base name.
func sanitizeFileName(name string) string {
	// Treat Windows-style separators like path separators on every platform.
	name = strings.ReplaceAll(strings.TrimSpace(name), "\\", "/")
	name = filepath.Base(name)
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f || r == '/' || r == '\\' {
			return -1
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return "file"
	}

	// Keep well below filesystem filename limits.
	const maxNameLen = 200
	if len(name) > maxNameLen {
		ext := filepath.Ext(name)
		if len(ext) > 20 {
			ext = ""
		}
		name = name[:maxNameLen-len(ext)] + ext
	}
	return name
}

// thumbFileName builds the stored thumbnail filename for an uploaded image.
func thumbFileName(id, originalName string) string {
	return id + "_" + stripExt(originalName) + "_thumb.jpg"
}

// generateThumbnail creates a JPEG thumbnail with a max width of 800px.
func generateThumbnail(srcPath, dstPath string) error {
	src, err := imaging.Open(srcPath, imaging.AutoOrientation(true))
	if err != nil {
		return fmt.Errorf("failed to open image: %w", err)
	}

	// Resize to 800px wide, preserving aspect ratio
	thumb := imaging.Resize(src, 800, 0, imaging.Lanczos)

	if err := imaging.Save(thumb, dstPath, imaging.JPEGQuality(85)); err != nil {
		return fmt.Errorf("failed to save thumbnail: %w", err)
	}

	return nil
}

// stripExt removes the file extension from a filename.
func stripExt(name string) string {
	ext := filepath.Ext(name)
	return name[:len(name)-len(ext)]
}

// populateFileURLs sets the FileURL and ThumbURL computed fields for a slice of clips.
func populateFileURLs(clips []model.Clip) {
	for i := range clips {
		populateFileURL(&clips[i])
	}
}

// populateFileURL sets the FileURL and ThumbURL computed fields for a single clip.
func populateFileURL(clip *model.Clip) {
	if clip.FilePath == "" {
		return
	}

	// Extract the relative path from the file path.
	// FilePath is like: /data/uploads/2024/01/15/uuid_file.txt
	// We need the portion after "uploads/"
	if idx := strings.Index(clip.FilePath, "uploads/"); idx != -1 {
		relPath := clip.FilePath[idx+len("uploads/"):]
		clip.FileURL = uploadURL("/api/files/", relPath)

		if clip.HasThumb {
			dir := filepath.Dir(relPath)
			base := filepath.Base(relPath)
			// Extract UUID and original name to build thumb filename
			// base is like "uuid_photo.jpg" → thumb is "uuid_photo_thumb.jpg"
			ext := filepath.Ext(base)
			nameWithoutExt := base[:len(base)-len(ext)]
			thumbName := nameWithoutExt + "_thumb.jpg"
			clip.ThumbURL = uploadURL("/api/thumbnails/", filepath.Join(dir, thumbName))
		}
	}
}

func uploadURL(prefix, path string) string {
	segments := strings.Split(filepath.ToSlash(path), "/")
	for i, segment := range segments {
		segments[i] = url.PathEscape(segment)
	}
	return prefix + strings.Join(segments, "/")
}

// deleteClipFiles removes the original and its previews from disk.
func deleteClipFiles(dataDir string, clip *model.Clip) {
	if clip.FilePath == "" {
		return
	}

	if err := os.Remove(clip.FilePath); err != nil && !os.IsNotExist(err) {
		log.Printf("Warning: failed to delete file %s: %v", clip.FilePath, err)
	}

	if cachePath := clip.OfficePreviewPath(dataDir); cachePath != "" {
		if err := os.Remove(cachePath); err != nil && !os.IsNotExist(err) {
			log.Printf("Warning: failed to delete office preview %s: %v", cachePath, err)
		}
	}

	if clip.HasThumb {
		ext := filepath.Ext(clip.FilePath)
		thumbPath := clip.FilePath[:len(clip.FilePath)-len(ext)] + "_thumb.jpg"
		if err := os.Remove(thumbPath); err != nil && !os.IsNotExist(err) {
			log.Printf("Warning: failed to delete thumbnail %s: %v", thumbPath, err)
		}
	}
}
