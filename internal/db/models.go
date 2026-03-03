package db

import (
	"context"
	"encoding/json"
	"fmt"
)

type Endpoint struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Endpoint  string `json:"endpoint"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	Bucket    string `json:"bucket"`
	Prefix    string `json:"prefix"`
	Region    string `json:"region"`
	UseSSL    bool   `json:"use_ssl"`
	PathStyle bool   `json:"path_style"`
}

type WatchItem struct {
	ID         int64    `json:"id"`
	Path       string   `json:"path"`
	EndpointID int64    `json:"endpoint_id"`
	TargetPath string   `json:"target_path"`
	Excludes   []string `json:"excludes"`
	Enabled    bool     `json:"enabled"`
}

type Backup struct {
	ID          int64  `json:"id"`
	WatchItemID int64  `json:"watch_item_id"`
	EndpointID  int64  `json:"endpoint_id"`
	StartedAt   string `json:"started_at"`
	CompletedAt string `json:"completed_at"`
	Status      string `json:"status"`
	TotalFiles  int64  `json:"total_files"`
	TotalBytes  int64  `json:"total_bytes"`
	Error       string `json:"error"`
}

type BackupSummary struct {
	ID          int64  `json:"id"`
	WatchItemID int64  `json:"watch_item_id"`
	EndpointID  int64  `json:"endpoint_id"`
	WatchPath   string `json:"watch_path"`
	TargetPath  string `json:"target_path"`
	StartedAt   string `json:"started_at"`
	CompletedAt string `json:"completed_at"`
	Status      string `json:"status"`
	TotalFiles  int64  `json:"total_files"`
	TotalBytes  int64  `json:"total_bytes"`
	Error       string `json:"error"`
}

type StorageTotals struct {
	OriginalBytes   int64 `json:"original_bytes"`
	CompressedBytes int64 `json:"compressed_bytes"`
}

type StorageTrendPoint struct {
	BackupID        int64  `json:"backup_id"`
	Timestamp       string `json:"timestamp"`
	OriginalBytes   int64  `json:"original_bytes"`
	CompressedBytes int64  `json:"compressed_bytes"`
}

type BackupFile struct {
	ID        int64  `json:"id"`
	BackupID  int64  `json:"backup_id"`
	Path      string `json:"path"`
	Size      int64  `json:"size"`
	ModTime   string `json:"mod_time"`
	Hash      string `json:"hash"`
	ObjectKey string `json:"object_key"`
	ZstdSize  int64  `json:"zstd_size"`
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func intToBool(value int) bool {
	return value != 0
}

func (d *DB) ListEndpoints(ctx context.Context) ([]Endpoint, error) {
	rows, err := d.QueryContext(ctx, `SELECT id, name, endpoint, access_key, secret_key, bucket, prefix, region, use_ssl, path_style FROM endpoints ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list endpoints: %w", err)
	}
	defer rows.Close()

	var items []Endpoint
	for rows.Next() {
		var item Endpoint
		var useSSL, pathStyle int
		if err := rows.Scan(&item.ID, &item.Name, &item.Endpoint, &item.AccessKey, &item.SecretKey, &item.Bucket, &item.Prefix, &item.Region, &useSSL, &pathStyle); err != nil {
			return nil, fmt.Errorf("scan endpoint: %w", err)
		}
		item.UseSSL = intToBool(useSSL)
		item.PathStyle = intToBool(pathStyle)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (d *DB) CreateEndpoint(ctx context.Context, item Endpoint) (int64, error) {
	res, err := d.ExecContext(ctx, `
		INSERT INTO endpoints (name, endpoint, access_key, secret_key, bucket, prefix, region, use_ssl, path_style)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, item.Name, item.Endpoint, item.AccessKey, item.SecretKey, item.Bucket, item.Prefix, item.Region, boolToInt(item.UseSSL), boolToInt(item.PathStyle))
	if err != nil {
		return 0, fmt.Errorf("create endpoint: %w", err)
	}
	return res.LastInsertId()
}

func (d *DB) UpdateEndpoint(ctx context.Context, item Endpoint) error {
	_, err := d.ExecContext(ctx, `
		UPDATE endpoints SET name=?, endpoint=?, access_key=?, secret_key=?, bucket=?, prefix=?, region=?, use_ssl=?, path_style=?
		WHERE id=?
	`, item.Name, item.Endpoint, item.AccessKey, item.SecretKey, item.Bucket, item.Prefix, item.Region, boolToInt(item.UseSSL), boolToInt(item.PathStyle), item.ID)
	if err != nil {
		return fmt.Errorf("update endpoint: %w", err)
	}
	return nil
}

func (d *DB) DeleteEndpoint(ctx context.Context, id int64) error {
	_, err := d.ExecContext(ctx, `DELETE FROM endpoints WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("delete endpoint: %w", err)
	}
	return nil
}

func (d *DB) ListWatchItems(ctx context.Context) ([]WatchItem, error) {
	rows, err := d.QueryContext(ctx, `SELECT id, path, endpoint_id, target_path, excludes, enabled FROM watch_items ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list watch items: %w", err)
	}
	defer rows.Close()

	var items []WatchItem
	for rows.Next() {
		var item WatchItem
		var excludesRaw string
		var enabled int
		if err := rows.Scan(&item.ID, &item.Path, &item.EndpointID, &item.TargetPath, &excludesRaw, &enabled); err != nil {
			return nil, fmt.Errorf("scan watch item: %w", err)
		}
		if excludesRaw != "" {
			if err := json.Unmarshal([]byte(excludesRaw), &item.Excludes); err != nil {
				return nil, fmt.Errorf("decode excludes: %w", err)
			}
		}
		item.Enabled = intToBool(enabled)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (d *DB) CreateWatchItem(ctx context.Context, item WatchItem) (int64, error) {
	excludesRaw, err := json.Marshal(item.Excludes)
	if err != nil {
		return 0, fmt.Errorf("encode excludes: %w", err)
	}
	res, err := d.ExecContext(ctx, `
		INSERT INTO watch_items (path, endpoint_id, target_path, excludes, enabled)
		VALUES (?, ?, ?, ?, ?)
	`, item.Path, item.EndpointID, item.TargetPath, string(excludesRaw), boolToInt(item.Enabled))
	if err != nil {
		return 0, fmt.Errorf("create watch item: %w", err)
	}
	return res.LastInsertId()
}

func (d *DB) UpdateWatchItem(ctx context.Context, item WatchItem) error {
	excludesRaw, err := json.Marshal(item.Excludes)
	if err != nil {
		return fmt.Errorf("encode excludes: %w", err)
	}
	_, err = d.ExecContext(ctx, `
		UPDATE watch_items SET path=?, endpoint_id=?, target_path=?, excludes=?, enabled=?
		WHERE id=?
	`, item.Path, item.EndpointID, item.TargetPath, string(excludesRaw), boolToInt(item.Enabled), item.ID)
	if err != nil {
		return fmt.Errorf("update watch item: %w", err)
	}
	return nil
}

func (d *DB) DeleteWatchItem(ctx context.Context, id int64) error {
	_, err := d.ExecContext(ctx, `DELETE FROM watch_items WHERE id=?`, id)
	if err != nil {
		return fmt.Errorf("delete watch item: %w", err)
	}
	return nil
}

func (d *DB) CreateBackup(ctx context.Context, item Backup) (int64, error) {
	res, err := d.ExecContext(ctx, `
		INSERT INTO backups (watch_item_id, endpoint_id, started_at, completed_at, status, total_files, total_bytes, error)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, item.WatchItemID, item.EndpointID, item.StartedAt, item.CompletedAt, item.Status, item.TotalFiles, item.TotalBytes, item.Error)
	if err != nil {
		return 0, fmt.Errorf("create backup: %w", err)
	}
	return res.LastInsertId()
}

func (d *DB) UpdateBackup(ctx context.Context, item Backup) error {
	_, err := d.ExecContext(ctx, `
		UPDATE backups SET completed_at=?, status=?, total_files=?, total_bytes=?, error=?
		WHERE id=?
	`, item.CompletedAt, item.Status, item.TotalFiles, item.TotalBytes, item.Error, item.ID)
	if err != nil {
		return fmt.Errorf("update backup: %w", err)
	}
	return nil
}

func (d *DB) ListBackups(ctx context.Context, limit int) ([]Backup, error) {
	query := `SELECT id, watch_item_id, endpoint_id, started_at, completed_at, status, total_files, total_bytes, error FROM backups ORDER BY id DESC`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := d.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list backups: %w", err)
	}
	defer rows.Close()

	var items []Backup
	for rows.Next() {
		var item Backup
		if err := rows.Scan(&item.ID, &item.WatchItemID, &item.EndpointID, &item.StartedAt, &item.CompletedAt, &item.Status, &item.TotalFiles, &item.TotalBytes, &item.Error); err != nil {
			return nil, fmt.Errorf("scan backup: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (d *DB) ListBackupsWithWatch(ctx context.Context, limit int) ([]BackupSummary, error) {
	query := `
		SELECT b.id, b.watch_item_id, b.endpoint_id, w.path, w.target_path,
			b.started_at, b.completed_at, b.status, b.total_files, b.total_bytes, b.error
		FROM backups b
		JOIN watch_items w ON w.id = b.watch_item_id
		ORDER BY b.id DESC`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := d.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list backups with watch: %w", err)
	}
	defer rows.Close()

	var items []BackupSummary
	for rows.Next() {
		var item BackupSummary
		if err := rows.Scan(
			&item.ID, &item.WatchItemID, &item.EndpointID, &item.WatchPath, &item.TargetPath,
			&item.StartedAt, &item.CompletedAt, &item.Status, &item.TotalFiles, &item.TotalBytes, &item.Error,
		); err != nil {
			return nil, fmt.Errorf("scan backup summary: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (d *DB) ListBackupFiles(ctx context.Context, backupID int64) ([]BackupFile, error) {
	rows, err := d.QueryContext(ctx, `SELECT id, backup_id, path, size, mod_time, hash, object_key, zstd_size FROM backup_files WHERE backup_id=? ORDER BY id`, backupID)
	if err != nil {
		return nil, fmt.Errorf("list backup files: %w", err)
	}
	defer rows.Close()

	var items []BackupFile
	for rows.Next() {
		var item BackupFile
		if err := rows.Scan(&item.ID, &item.BackupID, &item.Path, &item.Size, &item.ModTime, &item.Hash, &item.ObjectKey, &item.ZstdSize); err != nil {
			return nil, fmt.Errorf("scan backup file: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (d *DB) GetStorageTotals(ctx context.Context) (StorageTotals, error) {
	var totals StorageTotals
	row := d.QueryRowContext(ctx, `SELECT COALESCE(SUM(size), 0), COALESCE(SUM(zstd_size), 0) FROM backup_files`)
	if err := row.Scan(&totals.OriginalBytes, &totals.CompressedBytes); err != nil {
		return totals, fmt.Errorf("storage totals: %w", err)
	}
	return totals, nil
}

func (d *DB) ListStorageTrend(ctx context.Context, limit int) ([]StorageTrendPoint, error) {
	query := `
		SELECT b.id, b.completed_at, b.started_at,
			COALESCE(SUM(f.size), 0), COALESCE(SUM(f.zstd_size), 0)
		FROM backups b
		LEFT JOIN backup_files f ON f.backup_id = b.id
		GROUP BY b.id
		ORDER BY b.id DESC`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := d.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list storage trend: %w", err)
	}
	defer rows.Close()

	var points []StorageTrendPoint
	for rows.Next() {
		var point StorageTrendPoint
		var completedAt string
		var startedAt string
		if err := rows.Scan(&point.BackupID, &completedAt, &startedAt, &point.OriginalBytes, &point.CompressedBytes); err != nil {
			return nil, fmt.Errorf("scan storage trend: %w", err)
		}
		if completedAt != "" {
			point.Timestamp = completedAt
		} else {
			point.Timestamp = startedAt
		}
		points = append(points, point)
	}
	return points, rows.Err()
}

func (d *DB) CreateBackupFile(ctx context.Context, item BackupFile) (int64, error) {
	res, err := d.ExecContext(ctx, `
		INSERT INTO backup_files (backup_id, path, size, mod_time, hash, object_key, zstd_size)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, item.BackupID, item.Path, item.Size, item.ModTime, item.Hash, item.ObjectKey, item.ZstdSize)
	if err != nil {
		return 0, fmt.Errorf("create backup file: %w", err)
	}
	return res.LastInsertId()
}

func (d *DB) GetEndpoint(ctx context.Context, id int64) (Endpoint, error) {
	var item Endpoint
	var useSSL, pathStyle int
	row := d.QueryRowContext(ctx, `SELECT id, name, endpoint, access_key, secret_key, bucket, prefix, region, use_ssl, path_style FROM endpoints WHERE id=?`, id)
	if err := row.Scan(&item.ID, &item.Name, &item.Endpoint, &item.AccessKey, &item.SecretKey, &item.Bucket, &item.Prefix, &item.Region, &useSSL, &pathStyle); err != nil {
		return item, fmt.Errorf("get endpoint: %w", err)
	}
	item.UseSSL = intToBool(useSSL)
	item.PathStyle = intToBool(pathStyle)
	return item, nil
}

func (d *DB) GetWatchItem(ctx context.Context, id int64) (WatchItem, error) {
	var item WatchItem
	var excludesRaw string
	var enabled int
	row := d.QueryRowContext(ctx, `SELECT id, path, endpoint_id, target_path, excludes, enabled FROM watch_items WHERE id=?`, id)
	if err := row.Scan(&item.ID, &item.Path, &item.EndpointID, &item.TargetPath, &excludesRaw, &enabled); err != nil {
		return item, fmt.Errorf("get watch item: %w", err)
	}
	if excludesRaw != "" {
		if err := json.Unmarshal([]byte(excludesRaw), &item.Excludes); err != nil {
			return item, fmt.Errorf("decode excludes: %w", err)
		}
	}
	item.Enabled = intToBool(enabled)
	return item, nil
}

func (d *DB) GetBackup(ctx context.Context, id int64) (Backup, error) {
	var item Backup
	row := d.QueryRowContext(ctx, `SELECT id, watch_item_id, endpoint_id, started_at, completed_at, status, total_files, total_bytes, error FROM backups WHERE id=?`, id)
	if err := row.Scan(&item.ID, &item.WatchItemID, &item.EndpointID, &item.StartedAt, &item.CompletedAt, &item.Status, &item.TotalFiles, &item.TotalBytes, &item.Error); err != nil {
		return item, fmt.Errorf("get backup: %w", err)
	}
	return item, nil
}

func (d *DB) DeleteBackupFilesByBackup(ctx context.Context, backupID int64) error {
	_, err := d.ExecContext(ctx, `DELETE FROM backup_files WHERE backup_id=?`, backupID)
	if err != nil {
		return fmt.Errorf("delete backup files: %w", err)
	}
	return nil
}

func (d *DB) DeleteBackup(ctx context.Context, backupID int64) error {
	_, err := d.ExecContext(ctx, `DELETE FROM backups WHERE id=?`, backupID)
	if err != nil {
		return fmt.Errorf("delete backup: %w", err)
	}
	return nil
}

func (d *DB) GetLatestBackupsForWatch(ctx context.Context, watchID int64, limit int) ([]Backup, error) {
	query := `SELECT id, watch_item_id, endpoint_id, started_at, completed_at, status, total_files, total_bytes, error FROM backups WHERE watch_item_id=? ORDER BY id DESC`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := d.QueryContext(ctx, query, watchID)
	if err != nil {
		return nil, fmt.Errorf("list watch backups: %w", err)
	}
	defer rows.Close()

	var items []Backup
	for rows.Next() {
		var item Backup
		if err := rows.Scan(&item.ID, &item.WatchItemID, &item.EndpointID, &item.StartedAt, &item.CompletedAt, &item.Status, &item.TotalFiles, &item.TotalBytes, &item.Error); err != nil {
			return nil, fmt.Errorf("scan watch backup: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
