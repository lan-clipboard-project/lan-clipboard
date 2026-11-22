// Runtime acceptance probe for an isolated test server, never a production instance.
// Run from server/: go run ../scripts/check-runtime.go BASE_URL create|verify STATE_FILE
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const fileBody = "persistent upload from container acceptance test\n"

type clip struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	FileURL string `json:"fileUrl"`
	Pinned  bool   `json:"pinned"`
}

type state struct {
	Text clip `json:"text"`
	File clip `json:"file"`
}

var base string
var client = &http.Client{Timeout: 10 * time.Second}

func require(ok bool, format string, args ...any) {
	if !ok {
		log.Fatalf(format, args...)
	}
}

func request(method, path string, body io.Reader, contentType string, want int) []byte {
	req, err := http.NewRequest(method, base+path, body)
	require(err == nil, "new request: %v", err)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := client.Do(req)
	require(err == nil, "%s %s: %v", method, path, err)
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	require(err == nil, "read response: %v", err)
	require(resp.StatusCode == want, "%s %s: got %d, want %d: %s", method, path, resp.StatusCode, want, data)
	return data
}

func decode[T any](data []byte) T {
	var value T
	err := json.Unmarshal(data, &value)
	require(err == nil, "decode JSON: %v", err)
	return value
}

func main() {
	require(len(os.Args) == 4, "usage: check-runtime BASE_URL create|verify STATE_FILE")
	base = strings.TrimRight(os.Args[1], "/")
	phase, stateFile := os.Args[2], os.Args[3]
	require(phase == "create" || phase == "verify", "unknown phase: %s", phase)

	// Verify the actual built SPA, its JS/CSS bundles and the client route fallback.
	index := request("GET", "/", nil, "", 200)
	require(bytes.Contains(index, []byte(`id="root"`)), "SPA shell missing")
	assets := regexp.MustCompile(`(?:src|href)="(/assets/[^"?]+\.(?:js|css))"`).FindAllSubmatch(index, -1)
	require(len(assets) >= 2, "expected built JS and CSS assets")
	for _, asset := range assets {
		request("GET", string(asset[1]), nil, "", 200)
	}
	fallback := request("GET", "/test/client/route", nil, "", 200)
	require(bytes.Equal(index, fallback), "SPA fallback differs from index")
	request("GET", "/api/no-such-route", nil, "", 404)

	var saved state
	if phase == "create" {
		stats := decode[struct{ TotalClips int }](request("GET", "/api/stats", nil, "", 200))
		require(stats.TotalClips == 0, "create requires an empty test database")

		conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(base, "http")+"/ws", nil)
		require(err == nil, "WebSocket handshake: %v", err)
		defer conn.Close()
		conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		// A pong proves ReadPump is running, which happens after hub registration.
		ready := make(chan struct{})
		conn.SetPongHandler(func(string) error { close(ready); return nil })
		event := make(chan []byte, 1)
		go func() {
			_, data, err := conn.ReadMessage()
			if err == nil {
				event <- data
			}
		}()
		err = conn.WriteControl(websocket.PingMessage, []byte("ready"), time.Now().Add(time.Second))
		require(err == nil, "WebSocket ping: %v", err)
		select {
		case <-ready:
		case <-time.After(5 * time.Second):
			log.Fatal("WebSocket pong timeout")
		}

		saved.Text = decode[clip](request("POST", "/api/clips", strings.NewReader(`{"content":"café container persistence"}`), "application/json", 201))
		select {
		case data := <-event:
			msg := decode[struct {
				Type string
				Data clip
			}](data)
			require(msg.Type == "clip_created" && msg.Data.ID == saved.Text.ID, "incorrect WebSocket broadcast: %s", data)
		case <-time.After(5 * time.Second):
			log.Fatal("WebSocket broadcast timeout")
		}
		request("PATCH", "/api/clips/"+saved.Text.ID, strings.NewReader(`{"pinned":true}`), "application/json", 200)

		var body bytes.Buffer
		form := multipart.NewWriter(&body)
		part, err := form.CreateFormFile("file", "persistent.txt")
		require(err == nil, "multipart: %v", err)
		_, err = io.WriteString(part, fileBody)
		require(err == nil, "multipart write: %v", err)
		require(form.Close() == nil, "multipart close")
		upload := decode[struct{ Clips []clip }](request("POST", "/api/clips/upload", &body, form.FormDataContentType(), 201))
		require(len(upload.Clips) == 1, "expected one uploaded file")
		saved.File = upload.Clips[0]
		data, err := json.Marshal(saved)
		require(err == nil, "encode state: %v", err)
		require(os.WriteFile(stateFile, data, 0600) == nil, "save test state")
	} else {
		data, err := os.ReadFile(stateFile)
		require(err == nil, "read state: %v", err)
		saved = decode[state](data)
	}

	text := decode[clip](request("GET", "/api/clips/"+saved.Text.ID, nil, "", 200))
	require(text.Content == "café container persistence" && text.Pinned, "text/pin did not persist: %+v", text)
	// LIKE cannot match cafe to café: this proves the FTS5 branch works.
	search := decode[struct{ Clips []clip }](request("GET", "/api/clips/search?q="+url.QueryEscape("cafe"), nil, "", 200))
	require(len(search.Clips) == 1 && search.Clips[0].ID == saved.Text.ID, "FTS5 accent-insensitive search failed")
	request("GET", "/api/clips/"+saved.File.ID, nil, "", 200)
	download := request("GET", saved.File.FileURL, nil, "", 200)
	require(string(download) == fileBody, "uploaded file did not persist")
	stats := decode[struct{ TotalClips int }](request("GET", "/api/stats", nil, "", 200))
	require(stats.TotalClips == 2, "expected two persisted clips: %+v", stats)

	if phase == "verify" {
		request("DELETE", "/api/clips/"+saved.Text.ID, nil, "", 200)
		request("DELETE", "/api/clips/"+saved.File.ID, nil, "", 200)
		request("GET", saved.File.FileURL, nil, "", 404)
	}
	fmt.Printf("PASS runtime %s: SPA/assets, API, FTS5, text/pin, upload/download%s\n", phase, map[bool]string{true: ", WebSocket"}[phase == "create"])
}
