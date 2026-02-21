package backup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"

	"s3-backup/internal/db"
	"s3-backup/internal/storage"
)

type Manager struct {
	DB          *db.DB
	ObjectKeys  storage.ObjectKeyBuilder
	S3Client    *storage.Client
	Retention   RetentionPolicy
	RecoveryDir string
	Overwrite   bool
}

type RetentionPolicy struct {
	KeepLast   int
	MaxAgeDays int
}

func (m *Manager) BackupWatchItem(ctx context.Context, watch db.WatchItem, endpoint db.Endpoint) (int64, error) {
	start := db.NowUTC()
	backupID, err := m.DB.CreateBackup(ctx, db.Backup{
		WatchItemID: watch.ID,
		EndpointID:  endpoint.ID,
		StartedAt:   start,
		Status:      "running",
		TotalFiles:  0,
		TotalBytes:  0,
	})
	if err != nil {
		return 0, err
	}

	var totalFiles int64
	var totalBytes int64
	status := "success"
	var lastErr error

	rootInfo, statErr := os.Stat(watch.Path)
	if statErr != nil {
		err = statErr
	} else if !rootInfo.IsDir() {
		rel := filepath.ToSlash(filepath.Base(watch.Path))
		if !shouldExclude(rel, watch.Excludes) {
			err = m.backupFile(ctx, backupID, watch, rel, watch.Path, rootInfo, &totalFiles, &totalBytes)
		}
	} else {
		err = filepath.WalkDir(watch.Path, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(watch.Path, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if shouldExclude(rel, watch.Excludes) {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return err
			}
			return m.backupFile(ctx, backupID, watch, rel, path, info, &totalFiles, &totalBytes)
		})
	}
	if err != nil {
		status = "failed"
		lastErr = err
	}

	if updateErr := m.DB.UpdateBackup(ctx, db.Backup{
		ID:          backupID,
		CompletedAt: db.NowUTC(),
		Status:      status,
		TotalFiles:  totalFiles,
		TotalBytes:  totalBytes,
		Error:       errString(lastErr),
	}); updateErr != nil {
		return backupID, updateErr
	}
	if lastErr != nil {
		return backupID, lastErr
	}
	if err := m.ApplyRetention(ctx, watch.ID); err != nil {
		return backupID, err
	}
	return backupID, nil
}

func (m *Manager) ApplyRetention(ctx context.Context, watchID int64) error {
	if m.Retention.KeepLast <= 0 && m.Retention.MaxAgeDays <= 0 {
		return nil
	}
	backups, err := m.DB.GetLatestBackupsForWatch(ctx, watchID, 0)
	if err != nil {
		return err
	}
	var toDelete []db.Backup
	if m.Retention.KeepLast > 0 && len(backups) > m.Retention.KeepLast {
		toDelete = append(toDelete, backups[m.Retention.KeepLast:]...)
	}
	if m.Retention.MaxAgeDays > 0 {
		cutoff := time.Now().AddDate(0, 0, -m.Retention.MaxAgeDays)
		for _, b := range backups {
			if b.CompletedAt == "" {
				continue
			}
			parsed, err := time.Parse(time.RFC3339Nano, b.CompletedAt)
			if err != nil {
				continue
			}
			if parsed.Before(cutoff) {
				toDelete = append(toDelete, b)
			}
		}
	}
	seen := map[int64]struct{}{}
	for _, b := range toDelete {
		if _, ok := seen[b.ID]; ok {
			continue
		}
		seen[b.ID] = struct{}{}
		files, err := m.DB.ListBackupFiles(ctx, b.ID)
		if err != nil {
			return err
		}
		for _, f := range files {
			if err := m.S3Client.RemoveObject(ctx, f.ObjectKey); err != nil {
				return err
			}
		}
		if err := m.DB.DeleteBackupFilesByBackup(ctx, b.ID); err != nil {
			return err
		}
		if err := m.DB.DeleteBackup(ctx, b.ID); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) RecoverBackup(ctx context.Context, backupID int64) error {
	files, err := m.DB.ListBackupFiles(ctx, backupID)
	if err != nil {
		return err
	}
	for _, file := range files {
		data, err := m.S3Client.GetObject(ctx, file.ObjectKey)
		if err != nil {
			return err
		}
		decoded, err := decompressZstd(data)
		if err != nil {
			return err
		}
		target := filepath.Join(m.RecoveryDir, filepath.FromSlash(file.Path))
		if !m.Overwrite {
			if _, err := os.Stat(target); err == nil {
				continue
			}
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, decoded, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func compressZstd(data []byte) ([]byte, error) {
	enc, err := zstd.NewWriter(nil)
	if err != nil {
		return nil, err
	}
	defer enc.Close()
	return enc.EncodeAll(data, nil), nil
}

func decompressZstd(data []byte) ([]byte, error) {
	dec, err := zstd.NewReader(nil)
	if err != nil {
		return nil, err
	}
	defer dec.Close()
	return dec.DecodeAll(data, nil)
}

func shouldExclude(rel string, excludes []string) bool {
	for _, ex := range excludes {
		if ex == "" {
			continue
		}
		if strings.HasPrefix(rel, ex) {
			return true
		}
	}
	return false
}

func (m *Manager) backupFile(ctx context.Context, backupID int64, watch db.WatchItem, relPath string, fullPath string, info os.FileInfo, totalFiles *int64, totalBytes *int64) error {
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(data)
	compressed, err := compressZstd(data)
	if err != nil {
		return err
	}
	key := m.ObjectKeys.Build(relPath, time.Now())
	metadata := map[string]string{
		"original_path": relPath,
		"mod_time":      info.ModTime().UTC().Format(time.RFC3339Nano),
		"size":          fmt.Sprintf("%d", len(data)),
		"hash":          hex.EncodeToString(hash[:]),
		"zstd_size":     fmt.Sprintf("%d", len(compressed)),
		"watch_item_id": fmt.Sprintf("%d", watch.ID),
	}
	if err := m.S3Client.PutObject(ctx, key, compressed, metadata); err != nil {
		return err
	}
	_, err = m.DB.CreateBackupFile(ctx, db.BackupFile{
		BackupID:  backupID,
		Path:      relPath,
		Size:      info.Size(),
		ModTime:   info.ModTime().UTC().Format(time.RFC3339Nano),
		Hash:      hex.EncodeToString(hash[:]),
		ObjectKey: key,
		ZstdSize:  int64(len(compressed)),
	})
	if err != nil {
		return err
	}
	*totalFiles += 1
	*totalBytes += info.Size()
	return nil
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
