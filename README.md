# S3 Backup Service (Go)

## Overview
- Auto backup on file change (debounced)
- Zstd compression
- S3-compatible upload (AWS SDK v2)
- Metadata stored in pure-Go SQLite (modernc.org/sqlite)
- Recovery download + decompress
- Web dashboard to manage watch items, endpoints, and trigger backups/recovery

## Configuration
Default config file: [`config.json`](config.json)

```json
{
  "app": {
    "data_dir": "./data",
    "db_path": "./data/s3-backup.sqlite",
    "dashboard_bind": "127.0.0.1:8080"
  },
  "retention": {
    "keep_last": 10,
    "max_age_days": 30
  },
  "recovery": {
    "restore_root": "./restore",
    "overwrite": false
  },
  "s3_defaults": {
    "region": "us-east-1",
    "use_ssl": false,
    "path_style": true
  }
}
```

**Note:** Watch items and S3 endpoints are managed in the database via the dashboard API. Each watch item selects its endpoint and target path.

## Run
```bash
go run ./cmd/server -config ./config.json
```

Dashboard: `http://127.0.0.1:8080`

## API Summary
- `GET /api/status`
- `GET/POST/PUT/DELETE /api/endpoints`
- `GET/POST/PUT/DELETE /api/watch`
- `POST /api/backup/trigger?watch_id=ID`
- `GET /api/backups/list`
- `GET /api/backups/detail?id=ID`
- `POST /api/recovery/start?backup_id=ID`

## Project Layout
- [`cmd/server/main.go`](cmd/server/main.go)
- [`internal/config/config.go`](internal/config/config.go)
- [`internal/db/db.go`](internal/db/db.go)
- [`internal/db/models.go`](internal/db/models.go)
- [`internal/watch/service.go`](internal/watch/service.go)
- [`internal/backup/manager.go`](internal/backup/manager.go)
- [`internal/storage/s3.go`](internal/storage/s3.go)
- [`internal/storage/object_key.go`](internal/storage/object_key.go)
- [`internal/api/server.go`](internal/api/server.go)
- [`web/index.html`](web/index.html)
- [`web/embed.go`](web/embed.go)

## Validation Steps
1. Start server and open dashboard.
2. Add an S3 endpoint.
3. Add a watch item (path + endpoint + target path).
4. Modify a file under the watch path and confirm new backup in dashboard.
5. Trigger recovery for a backup ID and verify restored files under `recovery.restore_root`.

## Notes
- Debounce is set to 2s in [`internal/watch/service.go`](internal/watch/service.go:34).
- S3 object keys use prefix + endpoint name + target path + date + hash.
