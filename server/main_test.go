package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"lan-clipboard/config"
	"lan-clipboard/model"
	"lan-clipboard/store"
	"lan-clipboard/ws"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// testEnv is a fully wired test server with its own temp data directory.
type testEnv struct {
	srv *httptest.Server
	cfg *config.Config
	db  *store.SQLiteStore
}

func newTestEnv(t *testing.T, staticDir string, mutate func(*config.Config)) *testEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		DataDir:            t.TempDir(),
		StaticDir:          staticDir,
		MaxFileSizeMB:      10,
		MaxTextSize:        1024,
		RetentionDays:      30,
		ThumbnailThreshold: 10,
	}
	if mutate != nil {
		mutate(cfg)
	}

	db, err := store.NewSQLiteStore(cfg.DataDir)
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	hub := ws.NewHub()
	go hub.Run()

	srv := httptest.NewServer(newRouter(cfg, db, hub))
	t.Cleanup(srv.Close)

	return &testEnv{srv: srv, cfg: cfg, db: db}
}

// request performs an HTTP request against the test server and returns
// the status code plus the raw response body.
func (e *testEnv) request(t *testing.T, method, path string, body io.Reader, contentType string) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, e.srv.URL+path, body)
	if err != nil {
		t.Fatalf("NewRequest(%s %s): %v", method, path, err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading response body: %v", err)
	}
	return resp.StatusCode, data
}

func mustUnmarshal[T any](t *testing.T, data []byte) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("invalid JSON response %q: %v", string(data), err)
	}
	return out
}

func (e *testEnv) createTextClip(t *testing.T, content string) model.Clip {
	t.Helper()
	payload, err := json.Marshal(map[string]string{"content": content})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	status, data := e.request(t, http.MethodPost, "/api/clips", bytes.NewReader(payload), "application/json")
	if status != http.StatusCreated {
		t.Fatalf("create clip: status %d, body %s", status, data)
	}
	return mustUnmarshal[model.Clip](t, data)
}

func (e *testEnv) uploadFile(t *testing.T, fieldName, filename string, content []byte) (int, []byte) {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile(fieldName, filename)
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("writing form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("closing multipart writer: %v", err)
	}
	return e.request(t, http.MethodPost, "/api/clips/upload", &buf, writer.FormDataContentType())
}

type clipListResponse struct {
	Clips      []model.Clip `json:"clips"`
	HasMore    bool         `json:"hasMore"`
	NextCursor string       `json:"nextCursor"`
}

func TestClipLifecycleAPI(t *testing.T) {
	env := newTestEnv(t, "", nil)

	// Create
	created := env.createTextClip(t, "hello API")
	if created.ID == "" {
		t.Fatal("created clip has no ID")
	}
	if created.Type != model.TypeText || created.Content != "hello API" {
		t.Fatalf("unexpected created clip: %+v", created)
	}
	if created.CreatedAt.IsZero() {
		t.Fatal("created clip has zero CreatedAt")
	}

	// List
	status, data := env.request(t, http.MethodGet, "/api/clips", nil, "")
	if status != http.StatusOK {
		t.Fatalf("list clips: status %d, body %s", status, data)
	}
	list := mustUnmarshal[clipListResponse](t, data)
	if len(list.Clips) != 1 || list.Clips[0].ID != created.ID {
		t.Fatalf("unexpected clip list: %+v", list)
	}
	if list.HasMore || list.NextCursor != "" {
		t.Fatalf("unexpected pagination flags: %+v", list)
	}

	// Get single
	status, data = env.request(t, http.MethodGet, "/api/clips/"+created.ID, nil, "")
	if status != http.StatusOK {
		t.Fatalf("get clip: status %d", status)
	}
	if got := mustUnmarshal[model.Clip](t, data); got.ID != created.ID {
		t.Fatalf("get clip returned %+v", got)
	}

	// Pin
	status, data = env.request(t, http.MethodPatch, "/api/clips/"+created.ID,
		strings.NewReader(`{"pinned":true}`), "application/json")
	if status != http.StatusOK {
		t.Fatalf("pin clip: status %d, body %s", status, data)
	}
	if got := mustUnmarshal[model.Clip](t, data); !got.Pinned {
		t.Fatalf("pin clip returned %+v", got)
	}

	// Pinned clips live outside the main list.
	status, data = env.request(t, http.MethodGet, "/api/clips", nil, "")
	if status != http.StatusOK || len(mustUnmarshal[clipListResponse](t, data).Clips) != 0 {
		t.Fatalf("main list should be empty after pinning: status %d, body %s", status, data)
	}
	status, data = env.request(t, http.MethodGet, "/api/clips/pinned", nil, "")
	if status != http.StatusOK {
		t.Fatalf("list pinned: status %d", status)
	}
	pinned := mustUnmarshal[clipListResponse](t, data)
	if len(pinned.Clips) != 1 || pinned.Clips[0].ID != created.ID {
		t.Fatalf("unexpected pinned list: %+v", pinned)
	}

	// Search: exact word, partial word and a query without results.
	status, data = env.request(t, http.MethodGet, "/api/clips/search?q=hello", nil, "")
	if status != http.StatusOK || len(mustUnmarshal[clipListResponse](t, data).Clips) != 1 {
		t.Fatalf("search: status %d, body %s", status, data)
	}
	status, data = env.request(t, http.MethodGet, "/api/clips/search?q=ell", nil, "")
	if status != http.StatusOK || len(mustUnmarshal[clipListResponse](t, data).Clips) != 1 {
		t.Fatalf("substring search: status %d, body %s", status, data)
	}
	status, data = env.request(t, http.MethodGet, "/api/clips/search?q=no-such-clip", nil, "")
	if status != http.StatusOK {
		t.Fatalf("empty search: status %d, body %s", status, data)
	}
	if noHits := mustUnmarshal[clipListResponse](t, data); noHits.Clips == nil || len(noHits.Clips) != 0 {
		t.Fatalf("empty search should return an empty JSON array, got %s", data)
	}

	// Stats
	status, data = env.request(t, http.MethodGet, "/api/stats", nil, "")
	if status != http.StatusOK {
		t.Fatalf("stats: status %d", status)
	}
	stats := mustUnmarshal[struct {
		TotalClips int64 `json:"totalClips"`
		TotalSize  int64 `json:"totalSize"`
	}](t, data)
	if stats.TotalClips != 1 {
		t.Fatalf("stats = %+v, want totalClips 1", stats)
	}

	// Delete
	status, _ = env.request(t, http.MethodDelete, "/api/clips/"+created.ID, nil, "")
	if status != http.StatusOK {
		t.Fatalf("delete clip: status %d", status)
	}
	status, _ = env.request(t, http.MethodGet, "/api/clips/"+created.ID, nil, "")
	if status != http.StatusNotFound {
		t.Fatalf("get deleted clip: status %d, want 404", status)
	}
	status, _ = env.request(t, http.MethodGet, "/api/clips/pinned", nil, "")
	if status != http.StatusOK {
		t.Fatalf("list pinned after delete: status %d", status)
	}
	if len(mustUnmarshal[clipListResponse](t, data).Clips) != 0 {
		t.Fatalf("pinned list should be empty after delete: %s", data)
	}
}

func TestAPIValidationErrors(t *testing.T) {
	env := newTestEnv(t, "", nil)

	tests := []struct {
		name   string
		method string
		path   string
		body   string
		want   int
	}{
		{name: "invalid json", method: http.MethodPost, path: "/api/clips", body: "{", want: http.StatusBadRequest},
		{name: "empty content", method: http.MethodPost, path: "/api/clips", body: `{"content":""}`, want: http.StatusBadRequest},
		{name: "text too large", method: http.MethodPost, path: "/api/clips", body: `{"content":"` + strings.Repeat("a", 2000) + `"}`, want: http.StatusBadRequest},
		{name: "missing pinned field", method: http.MethodPatch, path: "/api/clips/unknown", body: `{}`, want: http.StatusBadRequest},
		{name: "update missing clip", method: http.MethodPatch, path: "/api/clips/unknown", body: `{"pinned":true}`, want: http.StatusNotFound},
		{name: "delete missing clip", method: http.MethodDelete, path: "/api/clips/unknown", want: http.StatusNotFound},
		{name: "search requires q", method: http.MethodGet, path: "/api/clips/search", want: http.StatusBadRequest},
		{name: "search blank q", method: http.MethodGet, path: "/api/clips/search?q=%20%20", want: http.StatusBadRequest},
		{name: "invalid type filter", method: http.MethodGet, path: "/api/clips?type=bogus", want: http.StatusBadRequest},
		{name: "invalid cursor", method: http.MethodGet, path: "/api/clips?cursor=not-a-date", want: http.StatusBadRequest},
		{name: "upload without file", method: http.MethodPost, path: "/api/clips/upload", want: http.StatusBadRequest},
		{name: "missing static route", method: http.MethodGet, path: "/api/does-not-exist", want: http.StatusNotFound},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var body io.Reader
			contentType := ""
			if tc.body != "" {
				body = strings.NewReader(tc.body)
				contentType = "application/json"
			}
			status, data := env.request(t, tc.method, tc.path, body, contentType)
			if status != tc.want {
				t.Fatalf("%s %s: status %d, want %d (body: %s)", tc.method, tc.path, status, tc.want, data)
			}
		})
	}
}

func TestCursorPaginationAPI(t *testing.T) {
	env := newTestEnv(t, "", nil)

	const total = 25
	base := time.Now().UTC().Add(-time.Hour)
	for i := 0; i < total; i++ {
		err := env.db.CreateClip(&model.Clip{
			ID:        fmt.Sprintf("clip-%02d", i),
			Type:      model.TypeText,
			Content:   fmt.Sprintf("content %d", i),
			CreatedAt: base.Add(time.Duration(i) * time.Second),
		})
		if err != nil {
			t.Fatalf("CreateClip(%d): %v", i, err)
		}
	}

	var (
		cursor   string
		pageSize = 10
		seen     []string
	)
	for page := 0; page < 3; page++ {
		path := fmt.Sprintf("/api/clips?limit=%d", pageSize)
		if cursor != "" {
			path += "&cursor=" + url.QueryEscape(cursor)
		}
		status, data := env.request(t, http.MethodGet, path, nil, "")
		if status != http.StatusOK {
			t.Fatalf("page %d: status %d, body %s", page, status, data)
		}
		list := mustUnmarshal[clipListResponse](t, data)

		wantCount := pageSize
		if page == 2 {
			wantCount = total - 2*pageSize
		}
		if len(list.Clips) != wantCount {
			t.Fatalf("page %d returned %d clips, want %d", page, len(list.Clips), wantCount)
		}
		if wantMore := page < 2; list.HasMore != wantMore {
			t.Fatalf("page %d hasMore = %v, want %v", page, list.HasMore, wantMore)
		}
		for _, clip := range list.Clips {
			seen = append(seen, clip.ID)
		}
		cursor = list.NextCursor
		if wantMore := page < 2; (cursor != "") != wantMore {
			t.Fatalf("page %d nextCursor = %q", page, cursor)
		}
	}

	if len(seen) != total {
		t.Fatalf("pagination returned %d clips, want %d", len(seen), total)
	}
	unique := make(map[string]bool, total)
	for _, id := range seen {
		if unique[id] {
			t.Fatalf("duplicate clip %s across pages", id)
		}
		unique[id] = true
	}
}

func TestUploadServeAndDelete(t *testing.T) {
	// Threshold 0 forces thumbnail generation for every non-empty image.
	env := newTestEnv(t, "", func(cfg *config.Config) { cfg.ThumbnailThreshold = 0 })

	// Build a small valid PNG.
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for x := 0; x < 4; x++ {
		for y := 0; y < 4; y++ {
			img.Set(x, y, color.RGBA{R: 200, G: 30, B: 30, A: 255})
		}
	}
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		t.Fatalf("png.Encode: %v", err)
	}
	pngBytes := pngBuf.Bytes()

	status, data := env.uploadFile(t, "file", "photo.png", pngBytes)
	if status != http.StatusCreated {
		t.Fatalf("upload image: status %d, body %s", status, data)
	}
	uploaded := mustUnmarshal[struct {
		Clips []model.Clip `json:"clips"`
	}](t, data)
	if len(uploaded.Clips) != 1 {
		t.Fatalf("upload returned %d clips, want 1: %s", len(uploaded.Clips), data)
	}
	imageClip := uploaded.Clips[0]
	if imageClip.Type != model.TypeImage {
		t.Fatalf("uploaded PNG type = %q, want image", imageClip.Type)
	}
	if imageClip.FileName != "photo.png" || imageClip.FileSize != int64(len(pngBytes)) {
		t.Fatalf("unexpected uploaded clip: %+v", imageClip)
	}
	if imageClip.FileURL == "" {
		t.Fatal("uploaded clip is missing fileUrl")
	}
	if !imageClip.HasThumb || imageClip.ThumbURL == "" {
		t.Fatalf("uploaded image should have a thumbnail: %+v", imageClip)
	}

	// Original file is served inline with the original bytes.
	status, data = env.request(t, http.MethodGet, imageClip.FileURL, nil, "")
	if status != http.StatusOK {
		t.Fatalf("serve file: status %d, body %s", status, data)
	}
	if !bytes.Equal(data, pngBytes) {
		t.Fatalf("served file bytes differ from uploaded bytes (got %d, want %d)", len(data), len(pngBytes))
	}

	// Thumbnail is served as JPEG.
	status, data = env.request(t, http.MethodGet, imageClip.ThumbURL, nil, "")
	if status != http.StatusOK || len(data) == 0 {
		t.Fatalf("serve thumbnail: status %d, bytes %d", status, len(data))
	}

	// Non-image files are offered as downloads.
	status, data = env.uploadFile(t, "file", "notes.txt", []byte("hello attachment"))
	if status != http.StatusCreated {
		t.Fatalf("upload text file: status %d, body %s", status, data)
	}
	fileClips := mustUnmarshal[struct {
		Clips []model.Clip `json:"clips"`
	}](t, data)
	if len(fileClips.Clips) != 1 || fileClips.Clips[0].Type != model.TypeFile {
		t.Fatalf("unexpected text file upload: %s", data)
	}
	fileClip := fileClips.Clips[0]
	req, err := http.NewRequest(http.MethodGet, env.srv.URL+fileClip.FileURL, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET uploaded file: %v", err)
	}
	resp.Body.Close()
	if disposition := resp.Header.Get("Content-Disposition"); !strings.HasPrefix(disposition, "attachment") {
		t.Fatalf("Content-Disposition = %q, want attachment", disposition)
	}
	if disposition := resp.Header.Get("Content-Disposition"); !strings.Contains(disposition, "notes.txt") {
		t.Fatalf("Content-Disposition = %q, want original filename", disposition)
	}

	// Deleting a clip also deletes its files from disk.
	status, _ = env.request(t, http.MethodDelete, "/api/clips/"+imageClip.ID, nil, "")
	if status != http.StatusOK {
		t.Fatalf("delete image clip: status %d", status)
	}
	status, _ = env.request(t, http.MethodGet, imageClip.FileURL, nil, "")
	if status != http.StatusNotFound {
		t.Fatalf("serve deleted file: status %d, want 404", status)
	}
	status, _ = env.request(t, http.MethodGet, imageClip.ThumbURL, nil, "")
	if status != http.StatusNotFound {
		t.Fatalf("serve deleted thumbnail: status %d, want 404", status)
	}
}

func TestUploadTooLarge(t *testing.T) {
	env := newTestEnv(t, "", func(cfg *config.Config) { cfg.MaxFileSizeMB = 0 })

	status, data := env.uploadFile(t, "file", "big.bin", bytes.Repeat([]byte("x"), 1024))
	if status != http.StatusBadRequest {
		t.Fatalf("oversized upload: status %d, want 400 (body: %s)", status, data)
	}
}

func TestDeleteAllClipsAPI(t *testing.T) {
	env := newTestEnv(t, "", nil)
	env.createTextClip(t, "one")
	env.createTextClip(t, "two")

	status, _ := env.request(t, http.MethodDelete, "/api/clips", nil, "")
	if status != http.StatusOK {
		t.Fatalf("delete all: status %d", status)
	}
	status, data := env.request(t, http.MethodGet, "/api/clips", nil, "")
	if status != http.StatusOK {
		t.Fatalf("list after clear: status %d", status)
	}
	if len(mustUnmarshal[clipListResponse](t, data).Clips) != 0 {
		t.Fatalf("clips remain after clear: %s", data)
	}
}

func TestStaticSPAFallback(t *testing.T) {
	staticDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(staticDir, "index.html"), []byte("<html><body>SPA SHELL</body></html>"), 0o644); err != nil {
		t.Fatalf("write index.html: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(staticDir, "assets"), 0o755); err != nil {
		t.Fatalf("mkdir assets: %v", err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "assets", "app.js"), []byte("console.log('spa');"), 0o644); err != nil {
		t.Fatalf("write app.js: %v", err)
	}

	env := newTestEnv(t, staticDir, nil)

	for _, path := range []string{"/", "/some/client/route"} {
		status, data := env.request(t, http.MethodGet, path, nil, "")
		if status != http.StatusOK || !strings.Contains(string(data), "SPA SHELL") {
			t.Fatalf("GET %s: status %d, body %s", path, status, data)
		}
	}

	status, data := env.request(t, http.MethodGet, "/assets/app.js", nil, "")
	if status != http.StatusOK || !strings.Contains(string(data), "console.log") {
		t.Fatalf("GET asset: status %d, body %s", status, data)
	}

	// The SPA fallback must never swallow API 404s.
	status, data = env.request(t, http.MethodGet, "/api/nope", nil, "")
	if status != http.StatusNotFound || !strings.Contains(string(data), "not found") {
		t.Fatalf("GET /api/nope: status %d, body %s", status, data)
	}

	// Uploaded files are served from the data dir, not the static dir.
	status, data = env.request(t, http.MethodGet, "/api/files/2024/01/01/missing.png", nil, "")
	if status != http.StatusNotFound {
		t.Fatalf("GET missing upload: status %d, body %s", status, data)
	}
}

func TestWebSocketBroadcastsPerFrame(t *testing.T) {
	env := newTestEnv(t, "", nil)

	wsURL := "ws" + strings.TrimPrefix(env.srv.URL, "http") + "/ws"
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		status := 0
		if resp != nil {
			status = resp.StatusCode
		}
		t.Fatalf("dial %s: %v (status %d)", wsURL, err, status)
	}
	defer conn.Close()

	// Give the hub a moment to register the client before broadcasting.
	time.Sleep(100 * time.Millisecond)

	readMessage := func() (string, model.Clip) {
		t.Helper()
		if err := conn.SetReadDeadline(time.Now().Add(3 * time.Second)); err != nil {
			t.Fatalf("SetReadDeadline: %v", err)
		}
		_, payload, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("ReadMessage: %v", err)
		}
		// Every frame must be a standalone JSON document.
		msg := mustUnmarshal[struct {
			Type string     `json:"type"`
			Data model.Clip `json:"data"`
		}](t, payload)
		return msg.Type, msg.Data
	}

	created := env.createTextClip(t, "ws message")
	msgType, data := readMessage()
	if msgType != "clip_created" {
		t.Fatalf("first message type = %q, want clip_created", msgType)
	}
	if data.ID != created.ID {
		t.Fatalf("broadcast clip ID = %q, want %q", data.ID, created.ID)
	}

	// A burst of events must arrive as individually parseable frames.
	const burst = 5
	createdIDs := make(map[string]bool, burst)
	for i := 0; i < burst; i++ {
		clip := env.createTextClip(t, fmt.Sprintf("burst %d", i))
		createdIDs[clip.ID] = true
	}
	for i := 0; i < burst; i++ {
		msgType, data := readMessage()
		if msgType != "clip_created" {
			t.Fatalf("burst message %d type = %q, want clip_created", i, msgType)
		}
		if !createdIDs[data.ID] {
			t.Fatalf("burst message %d contains unexpected clip %q", i, data.ID)
		}
	}
}
