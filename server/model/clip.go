package model

import (
	"path/filepath"
	"strings"
	"time"
)

// Supported clip types.
const (
	TypeText  = "text"
	TypeImage = "image"
	TypeFile  = "file"
)

// IsValidClipType reports whether t is a supported clip type.
func IsValidClipType(t string) bool {
	switch t {
	case TypeText, TypeImage, TypeFile:
		return true
	default:
		return false
	}
}

// Clip represents a clipboard entry — text, image, or file.
type Clip struct {
	ID        string    `json:"id"`                 // UUID v7
	Type      string    `json:"type"`               // "text", "image", "file"
	Content   string    `json:"content"`            // text content (for type=text)
	FileName  string    `json:"fileName"`           // original filename
	FilePath  string    `json:"-"`                  // server storage path (never exposed to client)
	FileSize  int64     `json:"fileSize"`           // file size in bytes
	MimeType  string    `json:"mimeType"`           // MIME type
	HasThumb  bool      `json:"hasThumb"`           // whether a thumbnail exists
	Pinned    bool      `json:"pinned"`             // pinned/favorited
	CreatedAt time.Time `json:"createdAt"`          // creation timestamp
	Uploader  string    `json:"uploader,omitempty"` // operating system and browser derived from the upload request
	FileURL   string    `json:"fileUrl,omitempty"`  // computed download URL (not stored)
	ThumbURL  string    `json:"thumbUrl,omitempty"` // computed thumbnail URL (not stored)
}

// OfficePreviewPath returns the PDF cache path for convertible documents.
func (c Clip) OfficePreviewPath(dataDir string) string {
	if c.Type == TypeFile && c.FilePath != "" {
		switch strings.ToLower(filepath.Ext(c.FileName)) {
		case ".docx", ".pptx":
			return filepath.Join(dataDir, "previews", c.ID+".pdf")
		}
	}
	return ""
}
