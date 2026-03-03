package backup

import "s3-backup/internal/db"

type TaskProgress struct {
	Type           string `json:"type"`
	Stage          string `json:"stage"`
	BackupID       int64  `json:"backup_id"`
	CurrentFile    string `json:"current_file"`
	TotalFiles     int64  `json:"total_files"`
	CompletedFiles int64  `json:"completed_files"`
	BytesTotal     int64  `json:"bytes_total"`
	BytesDone      int64  `json:"bytes_done"`
	UpdatedAt      string `json:"updated_at"`
}

func (m *Manager) GetProgress() *TaskProgress {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.progress == nil {
		return nil
	}
	copy := *m.progress
	return &copy
}

func (m *Manager) setProgress(progress *TaskProgress) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.progress = progress
}

func (m *Manager) updateProgress(update func(p *TaskProgress)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.progress == nil {
		return
	}
	update(m.progress)
	m.progress.UpdatedAt = db.NowUTC()
}

func (m *Manager) clearProgress() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.progress = nil
}
