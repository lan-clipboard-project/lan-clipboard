package handler

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lan-clipboard/config"
	"lan-clipboard/model"
	"lan-clipboard/store"

	"github.com/gin-gonic/gin"
)

var inlinePreviewMIME = map[string]string{
	".aac": "audio/aac", ".flac": "audio/flac", ".m4a": "audio/mp4", ".mp3": "audio/mpeg",
	".oga": "audio/ogg", ".ogg": "audio/ogg", ".opus": "audio/ogg", ".wav": "audio/wav",
	".avi": "video/x-msvideo", ".m4v": "video/mp4", ".mkv": "video/x-matroska", ".mov": "video/quicktime",
	".mp4": "video/mp4", ".ogv": "video/ogg", ".webm": "video/webm",
	".pdf": "application/pdf",
	".htm": htmlPreviewMIME, ".html": htmlPreviewMIME,
}

const (
	maxOfficeInput    = 50 << 20
	maxOfficePDF      = 100 << 20
	maxHTMLPreview    = 5 << 20
	htmlPreviewMIME   = "text/html"
	htmlPreviewPolicy = "sandbox; default-src 'none'; style-src 'unsafe-inline'; img-src data:; font-src data:; media-src data:; base-uri 'none'; form-action 'none'; frame-ancestors 'self'"
)

type OfficePreviewHandler struct {
	store    *store.SQLiteStore
	dataDir  string
	endpoint string
	client   *http.Client
	convert  chan struct{}
}

func NewOfficePreviewHandler(db *store.SQLiteStore, cfg *config.Config) *OfficePreviewHandler {
	return &OfficePreviewHandler{
		store:    db,
		dataDir:  cfg.DataDir,
		endpoint: strings.TrimRight(strings.TrimSpace(cfg.OfficeConverterURL), "/"),
		client: &http.Client{
			Timeout: 45 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		convert: make(chan struct{}, 1),
	}
}

// ServePDF lazily creates a PDF for a stored Word document or PowerPoint deck.
func (h *OfficePreviewHandler) ServePDF(c *gin.Context) {
	clip, err := h.store.GetClip(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read clip"})
		return
	}
	if clip == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "clip not found"})
		return
	}
	cachePath := clip.OfficePreviewPath(h.dataDir)
	if cachePath == "" {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": "file type has no PDF preview"})
		return
	}
	if serveCachedPDF(c, cachePath) {
		return
	}
	if clip.FileSize > maxOfficeInput {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "document exceeds preview size limit"})
		return
	}
	if h.endpoint == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "document preview service is disabled"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 55*time.Second)
	defer cancel()
	// Serial conversion limits NAS memory use. Recheck the cache after waiting
	// so concurrent requests for the same document reuse the first result.
	select {
	case h.convert <- struct{}{}:
	case <-ctx.Done():
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "document preview service is busy"})
		return
	}
	if !cachedPDFExists(cachePath) {
		err = h.convertPDF(ctx, clip, cachePath)
	}
	<-h.convert
	if errors.Is(err, os.ErrNotExist) {
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
		return
	}
	if err != nil {
		log.Printf("Office preview for %s: %v", clip.ID, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "document preview conversion failed"})
		return
	}
	if !serveCachedPDF(c, cachePath) {
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
	}
}

func cachedPDFExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func serveCachedPDF(c *gin.Context, path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return false
	}
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", "inline")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Cache-Control", "private, no-cache")
	http.ServeContent(c.Writer, c.Request, filepath.Base(path), info.ModTime(), file)
	return true
}

func (h *OfficePreviewHandler) convertPDF(ctx context.Context, clip *model.Clip, cachePath string) error {
	source, err := os.Open(clip.FilePath)
	if err != nil {
		return err
	}
	defer source.Close()
	info, err := source.Stat()
	if err != nil {
		return err
	}
	if info.Size() > maxOfficeInput {
		return fmt.Errorf("source exceeds preview size limit")
	}

	// Keep only multipart framing in memory and stream the original from disk.
	var framing bytes.Buffer
	form := multipart.NewWriter(&framing)
	if _, err := form.CreateFormFile("files", clip.FileName); err != nil {
		return err
	}
	headerSize := framing.Len()
	if err := form.Close(); err != nil {
		return err
	}
	contents := io.MultiReader(bytes.NewReader(framing.Bytes()[:headerSize]), source, bytes.NewReader(framing.Bytes()[headerSize:]))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.endpoint+"/forms/libreoffice/convert", contents)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", form.FormDataContentType())
	req.ContentLength = info.Size() + int64(framing.Len())
	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("converter returned HTTP %d", resp.StatusCode)
	}
	pdf := bufio.NewReader(resp.Body)
	if signature, err := pdf.Peek(5); err != nil || string(signature) != "%PDF-" {
		return fmt.Errorf("converter returned invalid PDF data")
	}

	if err := os.MkdirAll(filepath.Dir(cachePath), 0700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(cachePath), ".preview-*.pdf")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	size, copyErr := io.Copy(tmp, io.LimitReader(pdf, maxOfficePDF+1))
	closeErr := tmp.Close()
	if copyErr != nil || closeErr != nil {
		return errors.Join(copyErr, closeErr)
	}
	if size > maxOfficePDF {
		return fmt.Errorf("converted PDF exceeds preview size limit")
	}
	if err := os.Rename(tmp.Name(), cachePath); err != nil {
		return err
	}
	// Delete/expiry removes the database row before unlinking the cache. A
	// conversion that finished after deletion must remove its output too.
	current, err := h.store.GetClip(clip.ID)
	if err != nil || current == nil {
		os.Remove(cachePath)
		if err != nil {
			return err
		}
		return os.ErrNotExist
	}
	return nil
}
