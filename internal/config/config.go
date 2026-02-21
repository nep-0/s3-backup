package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type AppConfig struct {
	DataDir       string `json:"data_dir"`
	DBPath        string `json:"db_path"`
	DashboardBind string `json:"dashboard_bind"`
}

type RetentionConfig struct {
	KeepLast   int `json:"keep_last"`
	MaxAgeDays int `json:"max_age_days"`
}

type RecoveryConfig struct {
	RestoreRoot string `json:"restore_root"`
	Overwrite   bool   `json:"overwrite"`
}

type S3Defaults struct {
	Region    string `json:"region"`
	UseSSL    bool   `json:"use_ssl"`
	PathStyle bool   `json:"path_style"`
}

type Config struct {
	App        AppConfig       `json:"app"`
	Retention  RetentionConfig `json:"retention"`
	Recovery   RecoveryConfig  `json:"recovery"`
	S3Defaults S3Defaults      `json:"s3_defaults"`
}

func Load(path string) (Config, error) {
	var cfg Config
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("read config: %w", err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func (c *Config) Validate() error {
	if c.App.DataDir == "" {
		return fmt.Errorf("app.data_dir required")
	}
	if c.App.DBPath == "" {
		return fmt.Errorf("app.db_path required")
	}
	if c.App.DashboardBind == "" {
		return fmt.Errorf("app.dashboard_bind required")
	}
	if c.Retention.KeepLast < 0 {
		return fmt.Errorf("retention.keep_last must be >= 0")
	}
	if c.Retention.MaxAgeDays < 0 {
		return fmt.Errorf("retention.max_age_days must be >= 0")
	}
	if c.Recovery.RestoreRoot == "" {
		return fmt.Errorf("recovery.restore_root required")
	}
	if c.S3Defaults.Region == "" {
		return fmt.Errorf("s3_defaults.region required")
	}
	return nil
}

func EnsureDataDirs(cfg Config) error {
	if err := os.MkdirAll(cfg.App.DataDir, 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	return os.MkdirAll(filepath.Dir(cfg.App.DBPath), 0o755)
}
