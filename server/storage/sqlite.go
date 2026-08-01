package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func NewSQLite(path string) (*Store, error) {
	if err := ensureDBPath(path); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	// Keep SQLite access serialized. This also ensures connection-local PRAGMAs
	// such as foreign_keys apply consistently to every operation.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	s := &Store{db: db}
	if err := s.initSchema(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func ensureDBPath(path string) error {
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create db directory: %w", err)
		}
	}

	if _, err := os.Stat(path); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("stat db file: %w", err)
		}

		f, createErr := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if createErr != nil {
			if !os.IsExist(createErr) {
				return fmt.Errorf("create empty db file: %w", createErr)
			}
		} else {
			_ = f.Close()
		}
	}

	return nil
}

func (s *Store) initSchema() error {
	if _, err := s.db.Exec(`PRAGMA journal_mode = WAL;`); err != nil {
		return fmt.Errorf("set wal mode: %w", err)
	}
	if _, err := s.db.Exec(`PRAGMA busy_timeout = 5000;`); err != nil {
		return fmt.Errorf("set busy timeout: %w", err)
	}
	if _, err := s.db.Exec(`PRAGMA synchronous = NORMAL;`); err != nil {
		return fmt.Errorf("set synchronous mode: %w", err)
	}
	if _, err := s.db.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		return fmt.Errorf("enable foreign keys: %w", err)
	}

	stmts := []string{
		`CREATE TABLE IF NOT EXISTS accounts (id TEXT PRIMARY KEY, created_at INTEGER, max_devices INTEGER DEFAULT 5, max_items INTEGER DEFAULT 10000, max_bytes INTEGER DEFAULT 10485760, pairing_public_key TEXT DEFAULT '', pin_hash TEXT DEFAULT '', last_active_at INTEGER DEFAULT 0, archived INTEGER DEFAULT 0, archived_at INTEGER DEFAULT 0);`,
		`CREATE TABLE IF NOT EXISTS devices (id TEXT PRIMARY KEY, account_id TEXT NOT NULL, name TEXT, created_at INTEGER, revoked INTEGER DEFAULT 0, token_version INTEGER DEFAULT 1, FOREIGN KEY(account_id) REFERENCES accounts(id) ON DELETE CASCADE);`,
		`CREATE TABLE IF NOT EXISTS items (id TEXT NOT NULL, account_id TEXT NOT NULL, payload TEXT, modified_at INTEGER, tombstone INTEGER DEFAULT 0, origin_device_id TEXT, version INTEGER, PRIMARY KEY(account_id, id), FOREIGN KEY(account_id) REFERENCES accounts(id) ON DELETE CASCADE);`,
		`CREATE TABLE IF NOT EXISTS system_state (key TEXT PRIMARY KEY, value TEXT NOT NULL);`,
		`CREATE INDEX IF NOT EXISTS idx_devices_account_active ON devices(account_id, revoked);`,
		`CREATE INDEX IF NOT EXISTS idx_items_account_modified ON items(account_id, modified_at);`,
		`CREATE INDEX IF NOT EXISTS idx_accounts_archived_last_active ON accounts(archived, last_active_at);`,
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	for _, st := range stmts {
		if _, err := tx.Exec(st); err != nil {
			tx.Rollback()
			return fmt.Errorf("init schema: %w", err)
		}
	}
	return tx.Commit()
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) DB() *sql.DB {
	return s.db
}
