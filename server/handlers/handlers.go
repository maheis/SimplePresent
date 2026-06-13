package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type Server struct {
	DB *sql.DB
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func (s *Server) Register(w http.ResponseWriter, r *http.Request) {
	id := uuid.New().String()
	now := time.Now().Unix()
	_, err := s.DB.Exec("INSERT INTO accounts (id, created_at) VALUES (?, ?)", id, now)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"account_id": id})
}

func (s *Server) Pair(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AccountID string `json:"account_id"`
		Name      string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	id := uuid.New().String()
	now := time.Now().Unix()
	_, err := s.DB.Exec("INSERT INTO devices (id, account_id, name, created_at) VALUES (?, ?, ?, ?)", id, req.AccountID, req.Name, now)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"device_id": id})
}

func (s *Server) Pull(w http.ResponseWriter, r *http.Request) {
	account := r.URL.Query().Get("account_id")
	since := r.URL.Query().Get("since")
	var sinceInt int64
	if since != "" {
		if parsed, err := json.Number(since).Int64(); err == nil {
			sinceInt = parsed
		}
	}
	rows, err := s.DB.Query("SELECT id, payload, modified_at, tombstone, origin_device_id, version FROM items WHERE account_id = ? AND modified_at > ?", account, sinceInt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var id, payload, origin string
		var modified int64
		var tomb int
		var version int
		if err := rows.Scan(&id, &payload, &modified, &tomb, &origin, &version); err != nil {
			continue
		}
		var payloadObj interface{}
		json.Unmarshal([]byte(payload), &payloadObj)
		out = append(out, map[string]interface{}{"id": id, "payload": payloadObj, "modified_at": modified, "tombstone": tomb, "origin_device_id": origin, "version": version})
	}
	writeJSON(w, map[string]interface{}{"items": out})
}

func (s *Server) Push(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AccountID string `json:"account_id"`
		Items     []struct {
			ID             string      `json:"id"`
			Payload        interface{} `json:"payload"`
			ModifiedAt     int64       `json:"modified_at"`
			Tombstone      bool        `json:"tombstone"`
			OriginDeviceID string      `json:"origin_device_id"`
			Version        int         `json:"version"`
		} `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	tx, err := s.DB.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	stmt, _ := tx.Prepare("INSERT OR REPLACE INTO items (id, account_id, payload, modified_at, tombstone, origin_device_id, version) VALUES (?, ?, ?, ?, ?, ?, ?)")
	defer stmt.Close()
	for _, it := range req.Items {
		b, _ := json.Marshal(it.Payload)
		tomb := 0
		if it.Tombstone {
			tomb = 1
		}
		if _, err := stmt.Exec(it.ID, req.AccountID, string(b), it.ModifiedAt, tomb, it.OriginDeviceID, it.Version); err != nil {
			tx.Rollback()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	tx.Commit()
	writeJSON(w, map[string]string{"status": "ok"})
}
