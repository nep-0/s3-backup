package api

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"

	"s3-backup/internal/backup"
	"s3-backup/internal/db"
	"s3-backup/internal/storage"
	"s3-backup/internal/watch"
)

type Server struct {
	DB     *db.DB
	Watch  *watch.Service
	Backup *backup.Manager
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/endpoints", s.handleEndpoints)
	mux.HandleFunc("/api/watch", s.handleWatch)
	mux.HandleFunc("/api/backup/trigger", s.handleBackupTrigger)
	mux.HandleFunc("/api/backups/list", s.handleBackupsList)
	mux.HandleFunc("/api/backups/detail", s.handleBackupDetail)
	mux.HandleFunc("/api/recovery/start", s.handleRecoveryStart)
	return mux
}

func (s *Server) writeJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	watchItems, _ := s.DB.ListWatchItems(r.Context())
	endpoints, _ := s.DB.ListEndpoints(r.Context())
	backups, _ := s.DB.ListBackups(r.Context(), 10)
	payload := map[string]any{
		"watch_items": watchItems,
		"endpoints":   endpoints,
		"backups":     backups,
	}
	s.writeJSON(w, http.StatusOK, payload)
}

func (s *Server) handleEndpoints(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := s.DB.ListEndpoints(r.Context())
		if err != nil {
			s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		s.writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		var item db.Endpoint
		if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
			s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		id, err := s.DB.CreateEndpoint(r.Context(), item)
		if err != nil {
			s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		item.ID = id
		s.writeJSON(w, http.StatusOK, item)
	case http.MethodPut:
		var item db.Endpoint
		if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
			s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if err := s.DB.UpdateEndpoint(r.Context(), item); err != nil {
			s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		s.writeJSON(w, http.StatusOK, item)
	case http.MethodDelete:
		idStr := r.URL.Query().Get("id")
		id, _ := strconv.ParseInt(idStr, 10, 64)
		if id == 0 {
			s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id required"})
			return
		}
		if err := s.DB.DeleteEndpoint(r.Context(), id); err != nil {
			s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		s.writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleWatch(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := s.DB.ListWatchItems(r.Context())
		if err != nil {
			s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		s.writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		var item db.WatchItem
		if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
			s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		id, err := s.DB.CreateWatchItem(r.Context(), item)
		if err != nil {
			s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		item.ID = id
		_ = s.Watch.Refresh(r.Context())
		s.writeJSON(w, http.StatusOK, item)
	case http.MethodPut:
		var item db.WatchItem
		if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
			s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if err := s.DB.UpdateWatchItem(r.Context(), item); err != nil {
			s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		_ = s.Watch.Refresh(r.Context())
		s.writeJSON(w, http.StatusOK, item)
	case http.MethodDelete:
		idStr := r.URL.Query().Get("id")
		id, _ := strconv.ParseInt(idStr, 10, 64)
		if id == 0 {
			s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id required"})
			return
		}
		if err := s.DB.DeleteWatchItem(r.Context(), id); err != nil {
			s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		_ = s.Watch.Refresh(r.Context())
		s.writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleBackupTrigger(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	idStr := r.URL.Query().Get("watch_id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if id == 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "watch_id required"})
		return
	}
	watchItem, err := s.DB.GetWatchItem(r.Context(), id)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	endpoint, err := s.DB.GetEndpoint(r.Context(), watchItem.EndpointID)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
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
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.Backup.S3Client = client
	s.Backup.ObjectKeys.Prefix = endpoint.Prefix
	s.Backup.ObjectKeys.EndpointName = endpoint.Name
	base := filepath.Base(watchItem.Path)
	s.Backup.ObjectKeys.TargetPath = filepath.ToSlash(filepath.Join(watchItem.TargetPath, base))
	_, err = s.Backup.BackupWatchItem(r.Context(), watchItem, endpoint)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "triggered"})
}

func (s *Server) handleBackupsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	items, err := s.DB.ListBackupsWithWatch(r.Context(), 50)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleBackupDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	idStr := r.URL.Query().Get("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if id == 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id required"})
		return
	}
	items, err := s.DB.ListBackupFiles(r.Context(), id)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleRecoveryStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	idStr := r.URL.Query().Get("backup_id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if id == 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "backup_id required"})
		return
	}
	backupItem, err := s.DB.GetBackup(r.Context(), id)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	endpoint, err := s.DB.GetEndpoint(r.Context(), backupItem.EndpointID)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
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
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.Backup.S3Client = client
	s.Backup.ObjectKeys.Prefix = endpoint.Prefix
	if err := s.Backup.RecoverBackup(r.Context(), id); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "recovered"})
}
