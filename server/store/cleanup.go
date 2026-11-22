package store

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"lan-clipboard/config"
	"lan-clipboard/model"
)

// StartCleanupScheduler runs a background goroutine that triggers daily at 3:00 AM
// to delete expired (non-pinned) clips and their associated files.
// onDeleted (optional) is invoked with the removed clips so the caller can
// notify connected clients.
func StartCleanupScheduler(store *SQLiteStore, cfg *config.Config, onDeleted func([]model.Clip)) {
	go func() {
		for {
			now := time.Now()
			// Calculate the next 3:00 AM
			next := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, now.Location())
			if !next.After(now) {
				// Already past 3 AM today, schedule for tomorrow
				next = next.Add(24 * time.Hour)
			}
			delay := next.Sub(now)
			log.Printf("Cleanup scheduler: next run in %s (at %s)", delay.Round(time.Minute), next.Format(time.RFC3339))

			timer := time.NewTimer(delay)
			<-timer.C

			runCleanup(store, cfg, onDeleted)
		}
	}()
}

// runCleanup deletes expired clips and removes their files from disk.
func runCleanup(store *SQLiteStore, cfg *config.Config, onDeleted func([]model.Clip)) {
	log.Printf("Cleanup: starting cleanup of clips older than %d days", cfg.RetentionDays)

	clips, err := store.DeleteExpiredClips(cfg.RetentionDays)
	if err != nil {
		log.Printf("Cleanup: error deleting expired clips: %v", err)
		return
	}

	if len(clips) == 0 {
		log.Println("Cleanup: no expired clips found")
		return
	}

	filesDeleted := 0
	for _, clip := range clips {
		if clip.FilePath != "" {
			if err := os.Remove(clip.FilePath); err != nil && !os.IsNotExist(err) {
				log.Printf("Cleanup: failed to delete file %s: %v", clip.FilePath, err)
			} else {
				filesDeleted++
			}

			// Also delete thumbnail if it exists
			if clip.HasThumb {
				thumbPath := buildThumbPath(clip.FilePath)
				if err := os.Remove(thumbPath); err != nil && !os.IsNotExist(err) {
					log.Printf("Cleanup: failed to delete thumbnail %s: %v", thumbPath, err)
				}
			}
			if cachePath := clip.OfficePreviewPath(cfg.DataDir); cachePath != "" {
				if err := os.Remove(cachePath); err != nil && !os.IsNotExist(err) {
					log.Printf("Cleanup: failed to delete office preview %s: %v", cachePath, err)
				}
			}
		}
	}

	log.Printf("Cleanup: deleted %d expired clips, removed %d files", len(clips), filesDeleted)

	if onDeleted != nil {
		onDeleted(clips)
	}
}

// buildThumbPath derives the thumbnail path from an original file path.
// e.g., /data/uploads/2024/01/15/uuid_photo.jpg → /data/uploads/2024/01/15/uuid_photo_thumb.jpg
func buildThumbPath(filePath string) string {
	dir := filepath.Dir(filePath)
	ext := filepath.Ext(filePath)
	base := filePath[:len(filePath)-len(ext)]
	return filepath.Join(dir, filepath.Base(base)+"_thumb.jpg")
}
