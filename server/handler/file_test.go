package handler

import (
	"path/filepath"
	"strings"
	"testing"

	"lan-clipboard/config"
)

func TestSanitizeFileName(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "photo.jpg", want: "photo.jpg"},
		{in: "../../etc/passwd.txt", want: "passwd.txt"},
		{in: `C:\Users\me\file.txt`, want: "file.txt"},
		{in: "a\x00b.txt", want: "ab.txt"},
		{in: "   ", want: "file"},
		{in: "", want: "file"},
		{in: ".", want: "file"},
		{in: "..", want: "file"},
	}

	for _, tc := range tests {
		if got := sanitizeFileName(tc.in); got != tc.want {
			t.Errorf("sanitizeFileName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}

	longName := strings.Repeat("a", 300) + ".txt"
	got := sanitizeFileName(longName)
	if len(got) > 200 {
		t.Errorf("sanitizeFileName did not shorten long name: len=%d", len(got))
	}
	if !strings.HasSuffix(got, ".txt") {
		t.Errorf("sanitizeFileName dropped the extension: %q", got)
	}
}

func TestResolveUploadPath(t *testing.T) {
	dataDir := t.TempDir()
	h := &FileHandler{cfg: &config.Config{DataDir: dataDir}}
	uploadRoot := filepath.Join(dataDir, "uploads")

	valid := []string{
		"/2024/01/15/uuid_photo.jpg",
		"2024/01/15/uuid_photo.jpg",
		"/2024/01/15/..hidden..name.txt",
	}
	for _, reqPath := range valid {
		got, err := h.resolveUploadPath(reqPath)
		if err != nil {
			t.Fatalf("resolveUploadPath(%q) returned error: %v", reqPath, err)
		}
		if !strings.HasPrefix(got, uploadRoot+string(filepath.Separator)) {
			t.Fatalf("resolveUploadPath(%q) = %q, want a path under %q", reqPath, got, uploadRoot)
		}
	}

	invalid := []string{
		"/../etc/passwd",
		"/a/../../etc/passwd",
		`/a\..\..\etc\passwd`,
		"/..",
	}
	for _, reqPath := range invalid {
		if _, err := h.resolveUploadPath(reqPath); err == nil {
			t.Fatalf("resolveUploadPath(%q) should have failed", reqPath)
		}
	}
}
