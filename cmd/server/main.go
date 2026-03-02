package main

import (
	"context"
	"flag"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"s3-backup/internal/api"
	"s3-backup/internal/autostart"
	"s3-backup/internal/backup"
	"s3-backup/internal/config"
	"s3-backup/internal/db"
	"s3-backup/internal/storage"
	"s3-backup/internal/watch"
	"s3-backup/web"
)

const defaultConfigJSON = `{
  "app": {
    "data_dir": "data",
    "db_path": "data/s3-backup.sqlite",
    "dashboard_bind": "127.0.0.1:8080"
  },
  "retention": {
    "keep_last": 10,
    "max_age_days": 30
  },
  "recovery": {
    "restore_root": "restore",
    "overwrite": false
  },
  "s3_defaults": {
    "region": "us-east-1",
    "use_ssl": false,
    "path_style": true
  }
}
`

func main() {
	configLong := flag.String("config", "", "path to config.json")
	configShort := flag.String("c", "", "path to config.json")
	installLong := flag.Bool("install", false, "install binary and enable autostart")
	installShort := flag.Bool("i", false, "install binary and enable autostart")
	uninstallLong := flag.Bool("uninstall", false, "disable autostart and remove installed binary")
	uninstallShort := flag.Bool("u", false, "disable autostart and remove installed binary")
	flag.Parse()

	configPath, provided, err := resolveConfigPath(*configLong, *configShort)
	if err != nil {
		log.Fatalf("resolve config path: %v", err)
	}
	if !provided {
		if err := ensureDefaultConfig(configPath); err != nil {
			log.Fatalf("ensure default config: %v", err)
		}
	}

	if *installLong || *installShort {
		if err := installSelf(configPath); err != nil {
			log.Fatalf("install: %v", err)
		}
		log.Print("install complete")
		return
	}

	if *uninstallLong || *uninstallShort {
		if err := uninstallSelf(); err != nil {
			log.Fatalf("uninstall: %v", err)
		}
		log.Print("uninstall complete")
		return
	}

	cfg, err := config.Load(configPath)
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

func resolveConfigPath(configLong, configShort string) (string, bool, error) {
	if configLong != "" {
		return configLong, true, nil
	}
	if configShort != "" {
		return configShort, true, nil
	}
	path, err := defaultConfigPath()
	return path, false, err
}

func defaultConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "s3-backup", "config.json"), nil
}

func ensureDefaultConfig(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	if err := os.WriteFile(path, []byte(defaultConfigJSON), 0o644); err != nil {
		return err
	}

	log.Printf("created default config at %s", path)
	return nil
}

func installSelf(configPath string) error {
	srcPath, err := os.Executable()
	if err != nil {
		return err
	}
	srcPath, err = filepath.EvalSymlinks(srcPath)
	if err != nil {
		return err
	}

	dstPath, err := installPath()
	if err != nil {
		return err
	}

	if err := copyFile(srcPath, dstPath); err != nil {
		return err
	}

	if err := autostart.Enable("s3-backup", "S3 Backup Service", []string{dstPath, "--config", configPath}); err != nil {
		return err
	}

	log.Printf("installed to %s and enabled autostart", dstPath)
	return nil
}

func uninstallSelf() error {
	dstPath, err := installPath()
	if err != nil {
		return err
	}

	if err := autostart.Disable("s3-backup", "S3 Backup Service", []string{dstPath}); err != nil {
		return err
	}

	if err := os.Remove(dstPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	log.Printf("removed %s and disabled autostart", dstPath)
	return nil
}

func installPath() (string, error) {
	switch runtime.GOOS {
	case "windows":
		configDir, err := os.UserConfigDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(configDir, "s3-backup", "s3-backup.exe"), nil
	case "darwin":
		configDir, err := os.UserConfigDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(configDir, "s3-backup", "s3-backup"), nil
	default:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".local", "bin", "s3-backup"), nil
	}
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
