package store_test

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"lan-clipboard/model"
	"lan-clipboard/store"
)

func TestUploaderMigrationPreservesExistingClips(t *testing.T) {
	dir := t.TempDir()
	legacy, err := sql.Open("sqlite3", filepath.Join(dir, "clipboard.db"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = legacy.Exec(`CREATE TABLE clips (
		id TEXT PRIMARY KEY, type TEXT NOT NULL, content TEXT NOT NULL DEFAULT '',
		file_name TEXT NOT NULL DEFAULT '', file_path TEXT NOT NULL DEFAULT '',
		file_size INTEGER NOT NULL DEFAULT 0, mime_type TEXT NOT NULL DEFAULT '',
		has_thumb INTEGER NOT NULL DEFAULT 0, pinned INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL DEFAULT (datetime('now'))
	);
	INSERT INTO clips (id, type, content) VALUES ('old', 'text', 'old clip');`)
	legacy.Close()
	if err != nil {
		t.Fatal(err)
	}

	s, err := store.NewSQLiteStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	legacyClip, err := s.GetClip("old")
	if err != nil || legacyClip == nil || legacyClip.Content != "old clip" || legacyClip.Uploader != "" {
		t.Fatalf("existing clip after migration: %+v (%v)", legacyClip, err)
	}
	if err := s.CreateClip(&model.Clip{ID: "new", Type: model.TypeText, Uploader: "macOS Chrome"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = store.NewSQLiteStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	clip, err := s.GetClip("new")
	if err != nil || clip == nil || clip.Uploader != "macOS Chrome" {
		t.Fatalf("new clip after reopening: %+v (%v)", clip, err)
	}
}

// newTestStore creates an isolated SQLite store backed by a temp directory.
func newTestStore(t *testing.T) *store.SQLiteStore {
	t.Helper()
	s, err := store.NewSQLiteStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Errorf("closing store: %v", err)
		}
	})
	return s
}

// seedTextClip inserts a text clip with a deterministic ID and timestamp.
func seedTextClip(t *testing.T, s *store.SQLiteStore, id, content string, createdAt time.Time) {
	t.Helper()
	err := s.CreateClip(&model.Clip{
		ID:        id,
		Type:      model.TypeText,
		Content:   content,
		CreatedAt: createdAt,
	})
	if err != nil {
		t.Fatalf("CreateClip(%s): %v", id, err)
	}
}

func TestCreateAndGetClip(t *testing.T) {
	s := newTestStore(t)
	now := time.Now().UTC().Truncate(time.Second)
	seedTextClip(t, s, "clip-1", "hello world", now)

	clip, err := s.GetClip("clip-1")
	if err != nil {
		t.Fatalf("GetClip: %v", err)
	}
	if clip == nil {
		t.Fatal("GetClip returned nil for an existing clip")
	}
	if clip.Content != "hello world" || clip.Type != model.TypeText {
		t.Fatalf("unexpected clip: %+v", clip)
	}
	if !clip.CreatedAt.Equal(now) {
		t.Fatalf("CreatedAt = %v, want %v", clip.CreatedAt, now)
	}
	if !clip.CreatedAt.Equal(clip.CreatedAt.UTC()) {
		t.Fatalf("CreatedAt must be stored in UTC, got %v", clip.CreatedAt)
	}

	missing, err := s.GetClip("does-not-exist")
	if err != nil {
		t.Fatalf("GetClip(missing): %v", err)
	}
	if missing != nil {
		t.Fatalf("GetClip(missing) = %+v, want nil", missing)
	}
}

func TestListClipsPagination(t *testing.T) {
	s := newTestStore(t)

	const total = 25
	base := time.Now().UTC().Add(-time.Hour)
	for i := 0; i < total; i++ {
		seedTextClip(t, s,
			fmt.Sprintf("clip-%02d", i),
			fmt.Sprintf("content %d", i),
			base.Add(time.Duration(i)*time.Second),
		)
	}

	var (
		seen     []string
		cursor   *time.Time
		pageSize = 10
	)
	for page := 0; page < 3; page++ {
		clips, hasMore, err := s.ListClips(cursor, pageSize, "")
		if err != nil {
			t.Fatalf("ListClips page %d: %v", page, err)
		}

		wantCount := pageSize
		if page == 2 {
			wantCount = total - 2*pageSize
		}
		if len(clips) != wantCount {
			t.Fatalf("page %d returned %d clips, want %d", page, len(clips), wantCount)
		}
		if wantMore := page < 2; hasMore != wantMore {
			t.Fatalf("page %d hasMore = %v, want %v", page, hasMore, wantMore)
		}

		for _, clip := range clips {
			seen = append(seen, clip.ID)
		}
		last := clips[len(clips)-1].CreatedAt
		cursor = &last
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
	// Ordered newest first.
	if seen[0] != "clip-24" || seen[len(seen)-1] != "clip-00" {
		t.Fatalf("unexpected order: first=%s last=%s", seen[0], seen[len(seen)-1])
	}

	// Empty page past the end.
	cursor = &base
	clips, hasMore, err := s.ListClips(cursor, pageSize, "")
	if err != nil {
		t.Fatalf("ListClips past end: %v", err)
	}
	if len(clips) != 0 || hasMore {
		t.Fatalf("past end: got %d clips, hasMore=%v", len(clips), hasMore)
	}
}

func TestListClipsFiltersAndPinned(t *testing.T) {
	s := newTestStore(t)
	now := time.Now().UTC()

	seedTextClip(t, s, "text-1", "text clip", now)
	if err := s.CreateClip(&model.Clip{
		ID: "file-1", Type: model.TypeFile, FileName: "doc.pdf",
		FileSize: 123, CreatedAt: now.Add(time.Second),
	}); err != nil {
		t.Fatalf("CreateClip(file-1): %v", err)
	}

	texts, hasMore, err := s.ListClips(nil, 10, model.TypeText)
	if err != nil {
		t.Fatalf("ListClips(text): %v", err)
	}
	if len(texts) != 1 || texts[0].ID != "text-1" || hasMore {
		t.Fatalf("unexpected text filter result: %+v hasMore=%v", texts, hasMore)
	}

	// Pinned clips are excluded from the main list and returned separately.
	if err := s.UpdateClip("text-1", true); err != nil {
		t.Fatalf("UpdateClip: %v", err)
	}
	clips, _, err := s.ListClips(nil, 10, "")
	if err != nil {
		t.Fatalf("ListClips: %v", err)
	}
	if len(clips) != 1 || clips[0].ID != "file-1" {
		t.Fatalf("pinned clip leaked into main list: %+v", clips)
	}
	pinned, err := s.ListPinnedClips()
	if err != nil {
		t.Fatalf("ListPinnedClips: %v", err)
	}
	if len(pinned) != 1 || pinned[0].ID != "text-1" || !pinned[0].Pinned {
		t.Fatalf("unexpected pinned clips: %+v", pinned)
	}
}

func TestUpdateAndDeleteClip(t *testing.T) {
	s := newTestStore(t)
	now := time.Now().UTC()
	seedTextClip(t, s, "clip-1", "hello", now)

	if err := s.UpdateClip("missing", true); err == nil {
		t.Fatal("UpdateClip on missing clip should fail")
	}

	deleted, err := s.DeleteClip("clip-1")
	if err != nil {
		t.Fatalf("DeleteClip: %v", err)
	}
	if deleted == nil || deleted.ID != "clip-1" {
		t.Fatalf("DeleteClip returned %+v", deleted)
	}
	if remaining, err := s.GetClip("clip-1"); err != nil || remaining != nil {
		t.Fatalf("clip still present after delete: %+v, err=%v", remaining, err)
	}
	if deletedAgain, err := s.DeleteClip("clip-1"); err != nil || deletedAgain != nil {
		t.Fatalf("second DeleteClip = %+v, err=%v; want nil, nil", deletedAgain, err)
	}
}

func TestSearchClips(t *testing.T) {
	s := newTestStore(t)
	now := time.Now().UTC()

	seedTextClip(t, s, "text-1", "The quick brown fox jumps over the lazy dog", now)
	seedTextClip(t, s, "text-2", "unrelated content about go programming", now.Add(time.Second))
	if err := s.CreateClip(&model.Clip{
		ID: "file-1", Type: model.TypeFile, FileName: "quarterly-report.pdf",
		FileSize: 10, CreatedAt: now.Add(2 * time.Second),
	}); err != nil {
		t.Fatalf("CreateClip(file-1): %v", err)
	}

	tests := []struct {
		name     string
		query    string
		clipType string
		wantIDs  []string
	}{
		{name: "content match", query: "brown fox", wantIDs: []string{"text-1"}},
		{name: "file name match", query: "quarterly", wantIDs: []string{"file-1"}},
		{name: "type filter", query: "quick", clipType: model.TypeText, wantIDs: []string{"text-1"}},
		{name: "type filter excludes", query: "quick", clipType: model.TypeFile, wantIDs: nil},
		{name: "no match", query: "zebra", wantIDs: nil},
		{name: "quotes are safe", query: `un"related`, wantIDs: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			clips, err := s.SearchClips(tc.query, tc.clipType, 50)
			if err != nil {
				t.Fatalf("SearchClips(%q): %v", tc.query, err)
			}
			if len(clips) != len(tc.wantIDs) {
				t.Fatalf("SearchClips(%q) returned %d clips, want %d", tc.query, len(clips), len(tc.wantIDs))
			}
			for i, wantID := range tc.wantIDs {
				if clips[i].ID != wantID {
					t.Fatalf("SearchClips(%q)[%d] = %s, want %s", tc.query, i, clips[i].ID, wantID)
				}
			}
		})
	}
}

func TestSearchClipsSubstringAndCJK(t *testing.T) {
	s := newTestStore(t)
	now := time.Now().UTC()

	// Mirrors the type of content that exposed the old whole-token search bug.
	seedTextClip(t, s, "text-cjk",
		"节点 6 会进入实际开发编码。 核心工作是改造现有“大纲 → 分段写作 → CTA”，"+
			"让模型直接生成结构化正文，最终由 Python 组装成权威文章文件 article_doc.json。",
		now)
	seedTextClip(t, s, "text-pct", "progress 100% done, 9_of_10 steps", now.Add(time.Second))

	tests := []struct {
		name     string
		query    string
		clipType string
		wantIDs  []string
	}{
		{name: "CJK substring inside unsegmented run", query: "实际", wantIDs: []string{"text-cjk"}},
		{name: "another CJK substring", query: "核心", wantIDs: []string{"text-cjk"}},
		{name: "standalone CJK token", query: "节点", wantIDs: []string{"text-cjk"}},
		{name: "English word prefix", query: "artic", wantIDs: []string{"text-cjk"}},
		{name: "English word", query: "article", wantIDs: []string{"text-cjk"}},
		{name: "digit", query: "6", wantIDs: []string{"text-cjk"}},
		{name: "CJK multi-term AND", query: "实际 核心", wantIDs: []string{"text-cjk"}},
		{name: "CJK with type filter", query: "实际", clipType: model.TypeText, wantIDs: []string{"text-cjk"}},
		{name: "CJK with mismatching type filter", query: "实际", clipType: model.TypeFile, wantIDs: nil},
		{name: "literal percent", query: "100%", wantIDs: []string{"text-pct"}},
		{name: "percent wildcard is escaped", query: "%", wantIDs: []string{"text-pct"}},
		{name: "underscore is escaped", query: "9_of_10", wantIDs: []string{"text-pct"}},
		{name: "no match", query: "不存在的词", wantIDs: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			clips, err := s.SearchClips(tc.query, tc.clipType, 50)
			if err != nil {
				t.Fatalf("SearchClips(%q): %v", tc.query, err)
			}
			if len(clips) != len(tc.wantIDs) {
				t.Fatalf("SearchClips(%q) returned %d clips, want %d", tc.query, len(clips), len(tc.wantIDs))
			}
			for i, wantID := range tc.wantIDs {
				if clips[i].ID != wantID {
					t.Fatalf("SearchClips(%q)[%d] = %s, want %s", tc.query, i, clips[i].ID, wantID)
				}
			}
		})
	}

	// An empty query returns an empty (non-nil) slice, never nil.
	empty, err := s.SearchClips("   ", "", 50)
	if err != nil {
		t.Fatalf("SearchClips(blank): %v", err)
	}
	if empty == nil || len(empty) != 0 {
		t.Fatalf("SearchClips(blank) = %#v, want empty slice", empty)
	}
}

func TestDeleteExpiredClips(t *testing.T) {
	s := newTestStore(t)
	now := time.Now().UTC()

	seedTextClip(t, s, "old", "expired", now.AddDate(0, 0, -40))
	seedTextClip(t, s, "fresh", "recent", now)
	if err := s.CreateClip(&model.Clip{
		ID: "old-pinned", Type: model.TypeText, Content: "keep me",
		Pinned: true, CreatedAt: now.AddDate(0, 0, -40),
	}); err != nil {
		t.Fatalf("CreateClip(old-pinned): %v", err)
	}

	deleted, err := s.DeleteExpiredClips(30)
	if err != nil {
		t.Fatalf("DeleteExpiredClips: %v", err)
	}
	if len(deleted) != 1 || deleted[0].ID != "old" {
		t.Fatalf("DeleteExpiredClips = %+v, want [old]", deleted)
	}

	for _, id := range []string{"fresh", "old-pinned"} {
		clip, err := s.GetClip(id)
		if err != nil || clip == nil {
			t.Fatalf("clip %s should survive cleanup (err=%v)", id, err)
		}
	}
	if clip, _ := s.GetClip("old"); clip != nil {
		t.Fatal("expired clip was not deleted")
	}
}

func TestDeleteAllClipsAndStats(t *testing.T) {
	s := newTestStore(t)
	now := time.Now().UTC()

	seedTextClip(t, s, "text-1", "one", now)
	seedTextClip(t, s, "text-2", "two", now.Add(time.Second))
	if err := s.CreateClip(&model.Clip{
		ID: "file-1", Type: model.TypeFile, FileName: "a.bin", FileSize: 500,
		CreatedAt: now.Add(2 * time.Second),
	}); err != nil {
		t.Fatalf("CreateClip(file-1): %v", err)
	}

	count, size, err := s.GetStorageStats()
	if err != nil {
		t.Fatalf("GetStorageStats: %v", err)
	}
	if count != 3 || size != 500 {
		t.Fatalf("GetStorageStats = (%d, %d), want (3, 500)", count, size)
	}

	deleted, err := s.DeleteAllClips()
	if err != nil {
		t.Fatalf("DeleteAllClips: %v", err)
	}
	if len(deleted) != 3 {
		t.Fatalf("DeleteAllClips returned %d clips, want 3", len(deleted))
	}
	count, size, err = s.GetStorageStats()
	if err != nil {
		t.Fatalf("GetStorageStats after delete: %v", err)
	}
	if count != 0 || size != 0 {
		t.Fatalf("stats after delete = (%d, %d), want (0, 0)", count, size)
	}
}
