package main

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"lan-clipboard/config"

	"github.com/gorilla/websocket"
)

func TestClipboardRequestBoundary(t *testing.T) {
	env := newTestEnv(t, "", nil)
	for _, test := range []struct {
		name, host, origin, fetchSite string
		want                          int
	}{
		{name: "command line", want: http.StatusOK},
		{name: "same origin", origin: env.srv.URL, fetchSite: "same-origin", want: http.StatusOK},
		{name: "LAN IP", host: "192.168.1.20:18080", origin: "http://192.168.1.20:18080", want: http.StatusOK},
		{name: "unrelated origin", origin: "https://other.example.test", want: http.StatusForbidden},
		{name: "opaque origin", origin: "null", want: http.StatusForbidden},
		{name: "cross-site request without origin", fetchSite: "cross-site", want: http.StatusForbidden},
		{name: "unconfigured hostname", host: "other.example.test", origin: "http://other.example.test", want: http.StatusForbidden},
	} {
		t.Run(test.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, env.srv.URL+"/api/stats", nil)
			if err != nil {
				t.Fatal(err)
			}
			if test.host != "" {
				req.Host = test.host
			}
			if test.origin != "" {
				req.Header.Set("Origin", test.origin)
			}
			if test.fetchSite != "" {
				req.Header.Set("Sec-Fetch-Site", test.fetchSite)
			}
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()
			if res.StatusCode != test.want {
				t.Fatalf("status = %d, want %d", res.StatusCode, test.want)
			}
			if res.Header.Get("Access-Control-Allow-Origin") != "" {
				t.Fatal("response granted cross-origin access")
			}
		})
	}
}

func TestWebSocketRejectsOtherBrowserOrigins(t *testing.T) {
	env := newTestEnv(t, "", nil)
	endpoint := "ws" + strings.TrimPrefix(env.srv.URL, "http") + "/ws"
	for _, test := range []struct {
		origin string
		want   int
	}{
		{origin: env.srv.URL, want: http.StatusSwitchingProtocols},
		{origin: "https://other.example.test", want: http.StatusForbidden},
	} {
		conn, res, err := websocket.DefaultDialer.Dial(endpoint, http.Header{"Origin": {test.origin}})
		if conn != nil {
			conn.Close()
		}
		if res == nil || res.StatusCode != test.want {
			t.Fatalf("origin %q: response = %v, error = %v, want status %d", test.origin, res, err, test.want)
		}
		if res.Body != nil {
			res.Body.Close()
		}
	}
}

func TestPublicOriginBehindReverseProxy(t *testing.T) {
	env := newTestEnv(t, "", func(cfg *config.Config) {
		cfg.PublicOrigin = &url.URL{Scheme: "https", Host: "clip.example.test"}
	})
	for _, test := range []struct {
		origin string
		want   int
	}{
		{"https://clip.example.test", http.StatusOK},
		{"http://clip.example.test", http.StatusForbidden},
		{"https://other.example.test", http.StatusForbidden},
	} {
		req, err := http.NewRequest(http.MethodGet, env.srv.URL+"/api/stats", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Host = "clip.example.test"
		req.Header.Set("Origin", test.origin)
		req.Header.Set("X-Forwarded-Proto", "http")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != test.want {
			t.Fatalf("origin %q: status = %d, want %d", test.origin, res.StatusCode, test.want)
		}
	}

	endpoint := "ws" + strings.TrimPrefix(env.srv.URL, "http") + "/ws"
	conn, res, err := websocket.DefaultDialer.Dial(endpoint, http.Header{
		"Host":   {"clip.example.test:443"},
		"Origin": {"https://clip.example.test"},
	})
	if err != nil {
		t.Fatalf("WebSocket through proxy: %v (response %v)", err, res)
	}
	conn.Close()
}
