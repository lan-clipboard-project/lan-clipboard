package main

import (
	"net/http"
	"net/url"
	"slices"
	"testing"
	"time"

	"lan-clipboard/model"
)

func TestTimelineDatesUseViewerTimeZone(t *testing.T) {
	env := newTestEnv(t, "", nil)
	for _, clip := range []struct {
		id, kind, timestamp string
		pinned              bool
	}{
		{"previous-day", model.TypeText, "2026-11-01T03:55:00Z", false},
		{"after-midnight", model.TypeText, "2026-11-01T04:05:00Z", false},
		{"after-clock-change", model.TypeFile, "2026-11-01T07:05:00Z", false},
		{"pinned", model.TypeText, "2026-10-20T12:00:00Z", true},
	} {
		createdAt, err := time.Parse(time.RFC3339, clip.timestamp)
		if err != nil {
			t.Fatal(err)
		}
		if err := env.db.CreateClip(&model.Clip{
			ID: clip.id, Type: clip.kind, CreatedAt: createdAt, Pinned: clip.pinned,
		}); err != nil {
			t.Fatal(err)
		}
	}

	for _, test := range []struct {
		query string
		want  []string
	}{
		{"timezone=America%2FNew_York", []string{"2026-11-01", "2026-10-31"}},
		{"timezone=UTC", []string{"2026-11-01"}},
		{"timezone=America%2FNew_York&type=file", []string{"2026-11-01"}},
	} {
		status, body := env.request(t, http.MethodGet, "/api/clips/dates?"+test.query, nil, "")
		if status != http.StatusOK {
			t.Fatalf("%s: status %d, body %s", test.query, status, body)
		}
		dates := mustUnmarshal[struct {
			Dates []string `json:"dates"`
		}](t, body).Dates
		if !slices.Equal(dates, test.want) {
			t.Fatalf("%s: dates %v, want %v", test.query, dates, test.want)
		}
	}
}

func TestTimelineCanPageTowardsNewerClips(t *testing.T) {
	env := newTestEnv(t, "", nil)
	start := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	for i, id := range []string{"old", "first", "second", "third"} {
		if err := env.db.CreateClip(&model.Clip{
			ID: id, Type: model.TypeText, CreatedAt: start.Add(time.Duration(i) * time.Minute),
		}); err != nil {
			t.Fatal(err)
		}
	}

	before := url.QueryEscape(start.Add(time.Second).Format(time.RFC3339Nano))
	status, body := env.request(t, http.MethodGet, "/api/clips?limit=2&cursor="+before, nil, "")
	if page := mustUnmarshal[clipListResponse](t, body); status != http.StatusOK ||
		len(page.Clips) != 1 || page.Clips[0].ID != "old" {
		t.Fatalf("seek: status %d, body %s", status, body)
	}

	after := url.QueryEscape(start.Format(time.RFC3339Nano))
	status, body = env.request(t, http.MethodGet, "/api/clips?limit=2&after="+after, nil, "")
	page := mustUnmarshal[clipListResponse](t, body)
	if status != http.StatusOK || len(page.Clips) != 2 || page.Clips[0].ID != "second" ||
		page.Clips[1].ID != "first" || !page.HasMore {
		t.Fatalf("newer page: status %d, body %s", status, body)
	}
	status, body = env.request(t, http.MethodGet, "/api/clips?limit=2&after="+url.QueryEscape(page.NextCursor), nil, "")
	if page := mustUnmarshal[clipListResponse](t, body); status != http.StatusOK ||
		len(page.Clips) != 1 || page.Clips[0].ID != "third" || page.HasMore {
		t.Fatalf("latest page: status %d, body %s", status, body)
	}
}
