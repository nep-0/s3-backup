package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"strings"
	"time"
)

type ObjectKeyBuilder struct {
	Prefix       string
	EndpointName string
	TargetPath   string
}

func (b ObjectKeyBuilder) Build(relPath string, t time.Time) string {
	cleanRel := strings.TrimLeft(relPath, "./\\")
	date := t.UTC()
	hash := sha256.Sum256([]byte(cleanRel))
	fileName := fmt.Sprintf("%s_%d.zst", hex.EncodeToString(hash[:8]), date.Unix())
	parts := []string{
		strings.Trim(b.Prefix, "/"),
		strings.Trim(b.EndpointName, "/"),
		strings.Trim(b.TargetPath, "/"),
		fmt.Sprintf("%04d", int(date.Year())),
		fmt.Sprintf("%02d", int(date.Month())),
		fmt.Sprintf("%02d", int(date.Day())),
		fileName,
	}
	var cleaned []string
	for _, p := range parts {
		if p == "" {
			continue
		}
		cleaned = append(cleaned, p)
	}
	return path.Join(cleaned...)
}
