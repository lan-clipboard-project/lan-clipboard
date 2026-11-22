package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
)

// Config holds all application configuration values,
// populated from environment variables with sensible defaults.
type Config struct {
	ListenHost         string   // HTTP listener address before any Docker port mapping
	Port               int      // HTTP server port
	PublicOrigin       *url.URL // Optional browser origin at the reverse proxy
	DataDir            string   // Base directory for DB and uploaded files
	StaticDir          string   // Directory containing the built frontend (index.html + assets)
	MaxFileSizeMB      int      // Maximum upload file size in megabytes
	MaxTextSize        int      // Maximum text content size in bytes
	RetentionDays      int      // Days to keep non-pinned clips before auto-deletion
	ThumbnailThreshold int      // Generate thumbnail for images larger than this (MB)
	OfficeConverterURL string   // Base URL of an optional local Gotenberg service
}

// Load reads configuration from environment variables with defaults.
func Load() (*Config, error) {
	var publicOrigin *url.URL
	if raw := os.Getenv("CLIP_PUBLIC_ORIGIN"); raw != "" {
		var err error
		publicOrigin, err = ParseOrigin(raw)
		if err != nil {
			return nil, fmt.Errorf("CLIP_PUBLIC_ORIGIN must be an http or https origin such as https://clip.example.test")
		}
	}
	return &Config{
		ListenHost:         getEnvStr("CLIP_LISTEN_HOST", "127.0.0.1"),
		Port:               getEnvInt("CLIP_PORT", 8080),
		PublicOrigin:       publicOrigin,
		DataDir:            getEnvStr("CLIP_DATA_DIR", "./data"),
		StaticDir:          getEnvStr("CLIP_STATIC_DIR", "./static"),
		MaxFileSizeMB:      getEnvInt("CLIP_MAX_FILE_SIZE", 300),
		MaxTextSize:        getEnvInt("CLIP_MAX_TEXT_SIZE", 307200),
		RetentionDays:      getEnvInt("CLIP_RETENTION_DAYS", 30),
		ThumbnailThreshold: getEnvInt("CLIP_THUMBNAIL_THRESHOLD", 10),
		OfficeConverterURL: getEnvStr("CLIP_OFFICE_CONVERTER_URL", ""),
	}, nil
}

// ParseOrigin accepts a single serialized HTTP origin, without credentials or a path.
func ParseOrigin(raw string) (*url.URL, error) {
	origin, err := url.Parse(raw)
	if err != nil || (origin.Scheme != "http" && origin.Scheme != "https") ||
		origin.Hostname() == "" || origin.User != nil || origin.Path != "" ||
		origin.RawQuery != "" || origin.ForceQuery || origin.Fragment != "" {
		return nil, fmt.Errorf("invalid HTTP origin")
	}
	return origin, nil
}

// MaxFileSizeBytes returns the max file size limit in bytes.
func (c *Config) MaxFileSizeBytes() int64 {
	return int64(c.MaxFileSizeMB) * 1024 * 1024
}

// ThumbnailThresholdBytes returns the thumbnail threshold in bytes.
func (c *Config) ThumbnailThresholdBytes() int64 {
	return int64(c.ThumbnailThreshold) * 1024 * 1024
}

// getEnvStr reads a string environment variable or returns a default.
func getEnvStr(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// getEnvInt reads an integer environment variable or returns a default.
func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}
