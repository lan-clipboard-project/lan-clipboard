package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"lan-clipboard/config"
	"lan-clipboard/model"
)

func TestUploaderRoundTrip(t *testing.T) {
	env := newTestEnv(t, "", nil)
	req, err := http.NewRequest(http.MethodPost, env.srv.URL+"/api/clips", strings.NewReader(`{"content":"uploader test"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:132.0) Gecko/20100101 Firefox/132.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("create text: %d %s (%v)", resp.StatusCode, body, err)
	}
	created := mustUnmarshal[model.Clip](t, body)
	if created.Uploader != "Ubuntu Firefox" {
		t.Fatalf("uploader on create = %q", created.Uploader)
	}
	for _, path := range []string{"/api/clips", "/api/clips/search?q=uploader"} {
		status, body := env.request(t, http.MethodGet, path, nil, "")
		clips := mustUnmarshal[clipListResponse](t, body).Clips
		if status != http.StatusOK || len(clips) != 1 || clips[0].Uploader != created.Uploader {
			t.Fatalf("uploader missing on %s: %d %s", path, status, body)
		}
	}
}

func TestOfficePreviewCache(t *testing.T) {
	const original = "a local document"
	const pdf = "%PDF-1.7\ndocument preview"
	var calls atomic.Int32
	converter := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodPost || r.URL.Path != "/forms/libreoffice/convert" {
			t.Errorf("unexpected converter request %s %s", r.Method, r.URL.Path)
		}
		file, header, err := r.FormFile("files")
		if err != nil {
			t.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer file.Close()
		data, err := io.ReadAll(file)
		if err != nil || header.Filename != "文档 #1.docx" || string(data) != original {
			t.Errorf("unexpected converter upload: %q %q (%v)", header.Filename, data, err)
		}
		w.Header().Set("Content-Type", "application/pdf")
		io.WriteString(w, pdf)
	}))
	defer converter.Close()
	env := newTestEnv(t, "", func(cfg *config.Config) { cfg.OfficeConverterURL = converter.URL })
	status, body := env.uploadFile(t, "files", "文档 #1.docx", []byte(original))
	if status != http.StatusCreated {
		t.Fatalf("upload: %d %s", status, body)
	}
	clip := mustUnmarshal[clipListResponse](t, body).Clips[0]
	path := "/api/clips/" + clip.ID + "/preview"
	status, body = env.request(t, http.MethodGet, path, nil, "")
	if status != http.StatusOK || string(body) != pdf {
		t.Fatalf("preview: %d %q", status, body)
	}
	converter.Close()
	req, err := http.NewRequest(http.MethodGet, env.srv.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Range", "bytes=0-4")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, err = io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil || resp.StatusCode != http.StatusPartialContent || string(body) != "%PDF-" || calls.Load() != 1 {
		t.Fatalf("cached range: %d %q, converter calls %d (%v)", resp.StatusCode, body, calls.Load(), err)
	}
	status, body = env.request(t, http.MethodGet, clip.FileURL, nil, "")
	if status != http.StatusOK || string(body) != original {
		t.Fatalf("original download: %d %q", status, body)
	}
	status, _ = env.request(t, http.MethodDelete, "/api/clips/"+clip.ID, nil, "")
	statusAfterDelete, _ := env.request(t, http.MethodGet, path, nil, "")
	entries, err := os.ReadDir(env.cfg.DataDir + "/previews")
	if status != http.StatusOK || statusAfterDelete != http.StatusNotFound || (err != nil && !os.IsNotExist(err)) || len(entries) != 0 {
		t.Fatalf("preview cleanup: DELETE %d, GET %d, remaining files %v (%v)", status, statusAfterDelete, entries, err)
	}
}

func TestOfficePreviewDisabledKeepsOriginalAvailable(t *testing.T) {
	env := newTestEnv(t, "", nil)
	status, body := env.uploadFile(t, "files", "Slides.pptx", []byte("presentation"))
	if status != http.StatusCreated {
		t.Fatalf("upload: %d %s", status, body)
	}
	clip := mustUnmarshal[clipListResponse](t, body).Clips[0]
	status, _ = env.request(t, http.MethodGet, "/api/clips/"+clip.ID+"/preview", nil, "")
	if status != http.StatusServiceUnavailable {
		t.Fatalf("disabled preview: %d", status)
	}
	status, body = env.request(t, http.MethodGet, clip.FileURL, nil, "")
	if status != http.StatusOK || string(body) != "presentation" {
		t.Fatalf("download with preview disabled: %d %q", status, body)
	}
}

func TestHTMLPreviewUsesSandboxAndPreservesDownload(t *testing.T) {
	env := newTestEnv(t, "", nil)
	const html = `<!doctype html><title>Preview</title><h1>HTML file</h1>`
	for _, filename := range []string{"preview.HTML", "preview.htm"} {
		status, body := env.uploadFile(t, "files", filename, []byte(html))
		if status != http.StatusCreated {
			t.Fatalf("upload: %d %s", status, body)
		}
		clip := mustUnmarshal[clipListResponse](t, body).Clips[0]

		for query, want := range map[string]string{"": "attachment", "?preview=1": "inline"} {
			resp, err := http.Get(env.srv.URL + clip.FileURL + query)
			if err != nil {
				t.Fatal(err)
			}
			content, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil || resp.StatusCode != http.StatusOK || string(content) != html {
				t.Fatalf("%s%s: status %d body %q (%v)", filename, query, resp.StatusCode, content, err)
			}
			disposition := resp.Header.Get("Content-Disposition")
			if !strings.HasPrefix(disposition, want) || !strings.Contains(disposition, filename) {
				t.Fatalf("%s%s: unexpected disposition %q", filename, query, disposition)
			}
			if query != "" {
				policy := resp.Header.Get("Content-Security-Policy")
				if !strings.HasPrefix(policy, "sandbox;") || !strings.Contains(policy, "default-src 'none'") ||
					!strings.Contains(policy, "form-action 'none'") || resp.Header.Get("X-Content-Type-Options") != "nosniff" ||
					!strings.HasPrefix(resp.Header.Get("Content-Type"), "text/html") {
					t.Fatalf("missing protection on HTML preview: %v", resp.Header)
				}
			}
		}
	}
}

func TestMediaPreviewPreservesDownload(t *testing.T) {
	env := newTestEnv(t, "", nil)
	status, body := env.uploadFile(t, "files", "track.M4A", []byte("audio"))
	if status != http.StatusCreated {
		t.Fatalf("upload: %d %s", status, body)
	}
	clip := mustUnmarshal[clipListResponse](t, body).Clips[0]
	for query, disposition := range map[string]string{"": "attachment", "?preview=1": "inline"} {
		resp, err := http.Get(env.srv.URL + clip.FileURL + query)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK || !strings.HasPrefix(resp.Header.Get("Content-Disposition"), disposition) {
			t.Errorf("media %q: %d, Disposition %q", query, resp.StatusCode, resp.Header.Get("Content-Disposition"))
		}
		if query != "" && resp.Header.Get("Content-Type") != "audio/mp4" {
			t.Errorf("media MIME = %q", resp.Header.Get("Content-Type"))
		}
	}
}
