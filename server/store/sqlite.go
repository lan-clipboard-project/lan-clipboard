package store

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"lan-clipboard/model"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStore manages all database operations for clips.
type SQLiteStore struct {
	db      *sql.DB
	dataDir string
}

// NewSQLiteStore creates a new store, initialises the database and tables.
func NewSQLiteStore(dataDir string) (*SQLiteStore, error) {
	// Ensure the data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "clipboard.db")
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Verify the connection is alive
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	store := &SQLiteStore{db: db, dataDir: dataDir}
	if err := store.migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("SQLite store initialized at", dbPath)
	return store, nil
}

// migrate creates tables, indexes, FTS virtual table, and sync triggers.
func (s *SQLiteStore) migrate() error {
	migrations := []string{
		// Main clips table
		`CREATE TABLE IF NOT EXISTS clips (
			id         TEXT PRIMARY KEY,
			type       TEXT NOT NULL,
			content    TEXT NOT NULL DEFAULT '',
			file_name  TEXT NOT NULL DEFAULT '',
			file_path  TEXT NOT NULL DEFAULT '',
			file_size  INTEGER NOT NULL DEFAULT 0,
			mime_type  TEXT NOT NULL DEFAULT '',
			uploader   TEXT NOT NULL DEFAULT '',
			has_thumb  INTEGER NOT NULL DEFAULT 0,
			pinned     INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT (datetime('now'))
		)`,

		// Indexes for common queries
		`CREATE INDEX IF NOT EXISTS idx_clips_created_at ON clips(created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_clips_pinned ON clips(pinned)`,

		// FTS5 virtual table for full-text search on content and file_name
		`CREATE VIRTUAL TABLE IF NOT EXISTS clips_fts USING fts5(
			content,
			file_name,
			content='clips',
			content_rowid='rowid'
		)`,

		// Trigger: keep FTS in sync on INSERT
		`CREATE TRIGGER IF NOT EXISTS clips_ai AFTER INSERT ON clips BEGIN
			INSERT INTO clips_fts(rowid, content, file_name)
			VALUES (new.rowid, new.content, new.file_name);
		END`,

		// Trigger: keep FTS in sync on DELETE
		`CREATE TRIGGER IF NOT EXISTS clips_ad AFTER DELETE ON clips BEGIN
			INSERT INTO clips_fts(clips_fts, rowid, content, file_name)
			VALUES ('delete', old.rowid, old.content, old.file_name);
		END`,

		// Trigger: keep FTS in sync on UPDATE
		`CREATE TRIGGER IF NOT EXISTS clips_au AFTER UPDATE ON clips BEGIN
			INSERT INTO clips_fts(clips_fts, rowid, content, file_name)
			VALUES ('delete', old.rowid, old.content, old.file_name);
			INSERT INTO clips_fts(rowid, content, file_name)
			VALUES (new.rowid, new.content, new.file_name);
		END`,
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, m := range migrations {
		if _, err := tx.Exec(m); err != nil {
			return fmt.Errorf("migration failed: %s — %w", m, err)
		}
	}

	var hasUploader bool
	if err := tx.QueryRow(`SELECT EXISTS (SELECT 1 FROM pragma_table_info('clips') WHERE name = 'uploader')`).Scan(&hasUploader); err != nil {
		return fmt.Errorf("check clips schema: %w", err)
	}
	if !hasUploader {
		if _, err := tx.Exec(`ALTER TABLE clips ADD COLUMN uploader TEXT NOT NULL DEFAULT ''`); err != nil {
			return fmt.Errorf("add clip uploader: %w", err)
		}
	}
	return tx.Commit()
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// CreateClip inserts a new clip into the database.
func (s *SQLiteStore) CreateClip(clip *model.Clip) error {
	// Store timestamps in UTC with a fixed format so string comparison in
	// cursor queries and retention queries is chronologically correct.
	if clip.CreatedAt.IsZero() {
		clip.CreatedAt = time.Now().UTC()
	} else {
		clip.CreatedAt = clip.CreatedAt.UTC()
	}

	_, err := s.db.Exec(
		`INSERT INTO clips (id, type, content, file_name, file_path, file_size, mime_type, has_thumb, pinned, created_at, uploader)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		clip.ID, clip.Type, clip.Content, clip.FileName, clip.FilePath,
		clip.FileSize, clip.MimeType, boolToInt(clip.HasThumb), boolToInt(clip.Pinned), clip.CreatedAt, clip.Uploader,
	)
	if err != nil {
		return fmt.Errorf("failed to create clip: %w", err)
	}
	return nil
}

// GetClip retrieves a single clip by its ID.
func (s *SQLiteStore) GetClip(id string) (*model.Clip, error) {
	clip := &model.Clip{}
	err := s.db.QueryRow(
		`SELECT id, type, content, file_name, file_path, file_size, mime_type, has_thumb, pinned, created_at, uploader
		 FROM clips WHERE id = ?`, id,
	).Scan(
		&clip.ID, &clip.Type, &clip.Content, &clip.FileName, &clip.FilePath,
		&clip.FileSize, &clip.MimeType, &clip.HasThumb, &clip.Pinned, &clip.CreatedAt, &clip.Uploader,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get clip: %w", err)
	}
	return clip, nil
}

// ListClips returns a page of non-pinned clips ordered by created_at DESC.
// Uses cursor-based pagination: cursor is the created_at timestamp of the last
// item of the previous page (nil for the first page).
// Returns the clips, a hasMore flag, and an error.
func (s *SQLiteStore) ListClips(cursor *time.Time, limit int, clipType string) ([]model.Clip, bool, error) {
	return s.listClips(cursor, limit, clipType, false)
}

// ListNewerClips returns the nearest newer page, in descending timeline order.
func (s *SQLiteStore) ListNewerClips(after time.Time, limit int, clipType string) ([]model.Clip, bool, error) {
	return s.listClips(&after, limit, clipType, true)
}

func (s *SQLiteStore) listClips(cursor *time.Time, limit int, clipType string, newer bool) ([]model.Clip, bool, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	// Fetch one extra to determine if there are more results
	fetchLimit := limit + 1

	var args []interface{}
	query := `SELECT id, type, content, file_name, file_path, file_size, mime_type, has_thumb, pinned, created_at, uploader
			  FROM clips WHERE pinned = 0`

	if clipType != "" {
		query += ` AND type = ?`
		args = append(args, clipType)
	}

	if cursor != nil {
		if newer {
			query += ` AND created_at > ?`
		} else {
			query += ` AND created_at < ?`
		}
		args = append(args, cursor.UTC())
	}

	if newer {
		query += ` ORDER BY created_at ASC LIMIT ?`
	} else {
		query += ` ORDER BY created_at DESC LIMIT ?`
	}
	args = append(args, fetchLimit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, false, fmt.Errorf("failed to list clips: %w", err)
	}
	defer rows.Close()

	clips := make([]model.Clip, 0, limit)
	for rows.Next() {
		var clip model.Clip
		if err := rows.Scan(
			&clip.ID, &clip.Type, &clip.Content, &clip.FileName, &clip.FilePath,
			&clip.FileSize, &clip.MimeType, &clip.HasThumb, &clip.Pinned, &clip.CreatedAt, &clip.Uploader,
		); err != nil {
			return nil, false, fmt.Errorf("failed to scan clip: %w", err)
		}
		clips = append(clips, clip)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("row iteration error: %w", err)
	}

	hasMore := len(clips) > limit
	if hasMore {
		clips = clips[:limit]
	}
	if newer {
		slices.Reverse(clips)
	}
	return clips, hasMore, nil
}

// ListClipDates reads only timestamps so the date index can be built without loading clip bodies.
func (s *SQLiteStore) ListClipDates(ctx context.Context, location *time.Location, clipType string) ([]string, error) {
	query := `SELECT created_at FROM clips WHERE pinned = 0`
	var args []any
	if clipType != "" {
		query += ` AND type = ?`
		args = append(args, clipType)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seen := make(map[string]struct{})
	for rows.Next() {
		var createdAt time.Time
		if err := rows.Scan(&createdAt); err != nil {
			return nil, err
		}
		seen[createdAt.In(location).Format(time.DateOnly)] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	dates := make([]string, 0, len(seen))
	for date := range seen {
		dates = append(dates, date)
	}
	slices.Sort(dates)
	slices.Reverse(dates)
	return dates, nil
}

// ListPinnedClips returns all pinned clips ordered by created_at DESC.
func (s *SQLiteStore) ListPinnedClips() ([]model.Clip, error) {
	rows, err := s.db.Query(
		`SELECT id, type, content, file_name, file_path, file_size, mime_type, has_thumb, pinned, created_at, uploader
		 FROM clips WHERE pinned = 1 ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list pinned clips: %w", err)
	}
	defer rows.Close()

	clips := make([]model.Clip, 0)
	for rows.Next() {
		var clip model.Clip
		if err := rows.Scan(
			&clip.ID, &clip.Type, &clip.Content, &clip.FileName, &clip.FilePath,
			&clip.FileSize, &clip.MimeType, &clip.HasThumb, &clip.Pinned, &clip.CreatedAt, &clip.Uploader,
		); err != nil {
			return nil, fmt.Errorf("failed to scan pinned clip: %w", err)
		}
		clips = append(clips, clip)
	}
	return clips, rows.Err()
}

// UpdateClip updates the pinned status of a clip.
func (s *SQLiteStore) UpdateClip(id string, pinned bool) error {
	result, err := s.db.Exec(`UPDATE clips SET pinned = ? WHERE id = ?`, boolToInt(pinned), id)
	if err != nil {
		return fmt.Errorf("failed to update clip: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("clip not found")
	}
	return nil
}

// DeleteClip removes a clip by ID and returns it for file cleanup.
func (s *SQLiteStore) DeleteClip(id string) (*model.Clip, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op after commit

	clip := &model.Clip{}
	err = tx.QueryRow(
		`SELECT id, type, content, file_name, file_path, file_size, mime_type, has_thumb, pinned, created_at, uploader
		 FROM clips WHERE id = ?`, id,
	).Scan(
		&clip.ID, &clip.Type, &clip.Content, &clip.FileName, &clip.FilePath,
		&clip.FileSize, &clip.MimeType, &clip.HasThumb, &clip.Pinned, &clip.CreatedAt, &clip.Uploader,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get clip: %w", err)
	}

	if _, err := tx.Exec(`DELETE FROM clips WHERE id = ?`, id); err != nil {
		return nil, fmt.Errorf("failed to delete clip: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit delete: %w", err)
	}
	return clip, nil
}

// DeleteAllClips removes all clips and returns them for file cleanup.
func (s *SQLiteStore) DeleteAllClips() ([]model.Clip, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op after commit

	rows, err := tx.Query(
		`SELECT id, type, content, file_name, file_path, file_size, mime_type, has_thumb, pinned, created_at, uploader FROM clips`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list clips for deletion: %w", err)
	}

	var clips []model.Clip
	for rows.Next() {
		var clip model.Clip
		if err := rows.Scan(
			&clip.ID, &clip.Type, &clip.Content, &clip.FileName, &clip.FilePath,
			&clip.FileSize, &clip.MimeType, &clip.HasThumb, &clip.Pinned, &clip.CreatedAt, &clip.Uploader,
		); err != nil {
			rows.Close()
			return nil, fmt.Errorf("failed to scan clip: %w", err)
		}
		clips = append(clips, clip)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	if _, err := tx.Exec(`DELETE FROM clips`); err != nil {
		return nil, fmt.Errorf("failed to delete all clips: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit delete all: %w", err)
	}
	return clips, nil
}

// SearchClips searches clip text content and file names.
//
// FTS5's default unicode61 tokenizer treats a whole run of CJK characters as a
// single token and only matches complete Latin words. That is surprising for a
// clipboard: users expect substring search ("实际", "artic", "article_doc").
// We therefore combine FTS5 matches (token-aware, diacritic-insensitive) with a
// LIKE substring fallback where every whitespace-separated term must appear in
// the content or in the file name.
func (s *SQLiteStore) SearchClips(query string, clipType string, limit int) ([]model.Clip, error) {
	terms := strings.Fields(query)
	if len(terms) == 0 {
		return []model.Clip{}, nil
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	var args []interface{}

	// FTS5 branch: quoted terms joined by spaces implement AND semantics and
	// never fail on user-supplied FTS syntax (everything is quoted).
	where := `c.rowid IN (SELECT rowid FROM clips_fts WHERE clips_fts MATCH ?)`
	args = append(args, sanitizeFTSQuery(query))

	// LIKE branch: literal substring match for every term, so partial words
	// ("artic" → "article") and unsegmented CJK text ("实际") are found too.
	termClauses := make([]string, 0, len(terms))
	for _, term := range terms {
		pattern := "%" + escapeLikePattern(term) + "%"
		termClauses = append(termClauses,
			`(c.content LIKE ? ESCAPE '\' OR c.file_name LIKE ? ESCAPE '\')`)
		args = append(args, pattern, pattern)
	}
	where = `(` + where + ` OR (` + strings.Join(termClauses, " AND ") + `))`

	sqlQuery := `SELECT c.id, c.type, c.content, c.file_name, c.file_path, c.file_size,
	               c.mime_type, c.has_thumb, c.pinned, c.created_at, c.uploader
	        FROM clips c
	        WHERE ` + where

	if clipType != "" {
		sqlQuery += ` AND c.type = ?`
		args = append(args, clipType)
	}

	sqlQuery += ` ORDER BY c.created_at DESC, c.id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search clips: %w", err)
	}
	defer rows.Close()

	clips := make([]model.Clip, 0, limit)
	for rows.Next() {
		var clip model.Clip
		if err := rows.Scan(
			&clip.ID, &clip.Type, &clip.Content, &clip.FileName, &clip.FilePath,
			&clip.FileSize, &clip.MimeType, &clip.HasThumb, &clip.Pinned, &clip.CreatedAt, &clip.Uploader,
		); err != nil {
			return nil, fmt.Errorf("failed to scan search result: %w", err)
		}
		clips = append(clips, clip)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}
	return clips, nil
}

// DeleteExpiredClips removes non-pinned clips older than retentionDays and returns them for file cleanup.
func (s *SQLiteStore) DeleteExpiredClips(retentionDays int) ([]model.Clip, error) {
	cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op after commit

	rows, err := tx.Query(
		`SELECT id, type, content, file_name, file_path, file_size, mime_type, has_thumb, pinned, created_at, uploader
		 FROM clips WHERE pinned = 0 AND created_at < ?`, cutoff,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query expired clips: %w", err)
	}

	var clips []model.Clip
	for rows.Next() {
		var clip model.Clip
		if err := rows.Scan(
			&clip.ID, &clip.Type, &clip.Content, &clip.FileName, &clip.FilePath,
			&clip.FileSize, &clip.MimeType, &clip.HasThumb, &clip.Pinned, &clip.CreatedAt, &clip.Uploader,
		); err != nil {
			rows.Close()
			return nil, fmt.Errorf("failed to scan expired clip: %w", err)
		}
		clips = append(clips, clip)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	if len(clips) > 0 {
		if _, err := tx.Exec(
			`DELETE FROM clips WHERE pinned = 0 AND created_at < ?`, cutoff,
		); err != nil {
			return nil, fmt.Errorf("failed to delete expired clips: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit expired cleanup: %w", err)
	}
	return clips, nil
}

// GetStorageStats returns the total number of clips and total file size.
func (s *SQLiteStore) GetStorageStats() (totalClips int64, totalSize int64, err error) {
	err = s.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(file_size), 0) FROM clips`).Scan(&totalClips, &totalSize)
	if err != nil {
		err = fmt.Errorf("failed to get storage stats: %w", err)
	}
	return
}

// boolToInt converts a boolean to 0 or 1 for SQLite storage.
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// sanitizeFTSQuery wraps each word in double quotes to prevent FTS5 syntax errors.
func sanitizeFTSQuery(query string) string {
	words := strings.Fields(query)
	if len(words) == 0 {
		return "\"\""
	}
	var parts []string
	for _, w := range words {
		// Escape any internal double quotes
		escaped := strings.ReplaceAll(w, "\"", "\"\"")
		parts = append(parts, "\""+escaped+"\"")
	}
	return strings.Join(parts, " ")
}

// escapeLikePattern escapes SQL LIKE wildcards so user input is matched literally.
func escapeLikePattern(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 4)
	for _, r := range s {
		if r == '\\' || r == '%' || r == '_' {
			b.WriteRune('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}
