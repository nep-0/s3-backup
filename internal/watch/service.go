package watch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"s3-backup/internal/backup"
	"s3-backup/internal/db"
	"s3-backup/internal/storage"
)

type Service struct {
	DB         *db.DB
	Backup     *backup.Manager
	Debounce   time.Duration
	mu         sync.Mutex
	watchers   map[int64]*itemWatcher
	cancelFunc context.CancelFunc
}

type itemWatcher struct {
	watchID   int64
	path      string
	watchFile string
	watcher   *fsnotify.Watcher
	stopCh    chan struct{}
	pending   *time.Timer
}

func NewService(database *db.DB, manager *backup.Manager, debounce time.Duration) *Service {
	return &Service{
		DB:       database,
		Backup:   manager,
		Debounce: debounce,
		watchers: make(map[int64]*itemWatcher),
	}
}

func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.cancelFunc != nil {
		s.mu.Unlock()
		return nil
	}
	ctx, cancel := context.WithCancel(ctx)
	s.cancelFunc = cancel
	s.mu.Unlock()

	return s.Refresh(ctx)
}

func (s *Service) Stop() {
	s.mu.Lock()
	cancel := s.cancelFunc
	s.cancelFunc = nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, iw := range s.watchers {
		close(iw.stopCh)
		_ = iw.watcher.Close()
	}
	s.watchers = make(map[int64]*itemWatcher)
}

func (s *Service) Refresh(ctx context.Context) error {
	items, err := s.DB.ListWatchItems(ctx)
	if err != nil {
		return err
	}
	active := map[int64]db.WatchItem{}
	for _, item := range items {
		if !item.Enabled {
			continue
		}
		active[item.ID] = item
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for id, iw := range s.watchers {
		if _, ok := active[id]; !ok {
			close(iw.stopCh)
			_ = iw.watcher.Close()
			delete(s.watchers, id)
		}
	}
	for id, item := range active {
		if _, ok := s.watchers[id]; ok {
			continue
		}
		if err := s.startWatch(ctx, item); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) startWatch(ctx context.Context, item db.WatchItem) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}
	watchPath := item.Path
	var watchFile string
	info, statErr := os.Stat(item.Path)
	if statErr == nil && !info.IsDir() {
		watchFile = filepath.Clean(item.Path)
		watchPath = filepath.Dir(item.Path)
	}
	if err := watcher.Add(watchPath); err != nil {
		_ = watcher.Close()
		return fmt.Errorf("watch path %s: %w", watchPath, err)
	}
	itemWatcher := &itemWatcher{
		watchID:   item.ID,
		path:      watchPath,
		watchFile: watchFile,
		watcher:   watcher,
		stopCh:    make(chan struct{}),
	}
	s.watchers[item.ID] = itemWatcher

	go s.runWatcher(ctx, itemWatcher)
	return nil
}

func (s *Service) runWatcher(ctx context.Context, iw *itemWatcher) {
	for {
		select {
		case <-iw.stopCh:
			return
		case <-ctx.Done():
			return
		case event, ok := <-iw.watcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Rename) == 0 {
				continue
			}
			if iw.watchFile != "" && filepath.Clean(event.Name) != iw.watchFile {
				continue
			}
			iw.pending = s.scheduleBackup(ctx, iw)
		case err, ok := <-iw.watcher.Errors:
			if !ok {
				return
			}
			_ = err
		}
	}
}

func (s *Service) scheduleBackup(ctx context.Context, iw *itemWatcher) *time.Timer {
	if iw.pending != nil {
		iw.pending.Stop()
	}
	return time.AfterFunc(s.Debounce, func() {
		watch, err := s.DB.GetWatchItem(ctx, iw.watchID)
		if err != nil {
			return
		}
		endpoint, err := s.DB.GetEndpoint(ctx, watch.EndpointID)
		if err != nil {
			return
		}
		client, err := storage.NewClient(storage.S3Config{
			Endpoint:  endpoint.Endpoint,
			AccessKey: endpoint.AccessKey,
			SecretKey: endpoint.SecretKey,
			Bucket:    endpoint.Bucket,
			Region:    endpoint.Region,
			UseSSL:    endpoint.UseSSL,
			PathStyle: endpoint.PathStyle,
		})
		if err != nil {
			return
		}
		base := filepath.Base(watch.Path)
		s.Backup.S3Client = client
		s.Backup.ObjectKeys.Prefix = endpoint.Prefix
		s.Backup.ObjectKeys.EndpointName = endpoint.Name
		s.Backup.ObjectKeys.TargetPath = filepath.ToSlash(filepath.Join(watch.TargetPath, base))
		_, _ = s.Backup.BackupWatchItem(ctx, watch, endpoint)
	})
}
