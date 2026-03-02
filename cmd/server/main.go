package main

import (
	"context"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"s3-backup/internal/api"
	"s3-backup/internal/backup"
	"s3-backup/internal/config"
	"s3-backup/internal/db"
	"s3-backup/internal/storage"
	"s3-backup/internal/watch"
	"s3-backup/web"
)

func main() {
	configPath := flag.String("config", "./config.json", "path to config.json")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if err := config.EnsureDataDirs(cfg); err != nil {
		log.Fatalf("init data dirs: %v", err)
	}

	database, err := db.Open(cfg.App.DBPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer database.Close()

	manager := &backup.Manager{
		DB:         database,
		ObjectKeys: storage.ObjectKeyBuilder{},
		Retention: backup.RetentionPolicy{
			KeepLast:   cfg.Retention.KeepLast,
			MaxAgeDays: cfg.Retention.MaxAgeDays,
		},
		RecoveryDir: cfg.Recovery.RestoreRoot,
		Overwrite:   cfg.Recovery.Overwrite,
	}

	watchService := watch.NewService(database, manager, 2*time.Second)
	if err := watchService.Start(context.Background()); err != nil {
		log.Fatalf("start watcher: %v", err)
	}

	apiServer := &api.Server{
		DB:     database,
		Watch:  watchService,
		Backup: manager,
	}

	mux := http.NewServeMux()

	// Single Page App handler
	subFS, err := fs.Sub(web.Assets, ".")
	if err != nil {
		log.Fatal(err)
	}

	fsHandler := http.FileServer(http.FS(subFS))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path[1:]
		if path == "" {
			path = "index.html"
		}

		f, err := subFS.Open(path)
		if err != nil {
			if os.IsNotExist(err) {
				// File not found, serve index.html for SPA
				r.URL.Path = "/"
				fsHandler.ServeHTTP(w, r)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		f.Close()
		fsHandler.ServeHTTP(w, r)
	})

	mux.Handle("/api/", apiServer.Routes())

	log.Printf("dashboard on http://%s", cfg.App.DashboardBind)
	if err := http.ListenAndServe(cfg.App.DashboardBind, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
