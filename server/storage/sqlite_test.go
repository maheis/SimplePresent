package storage

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestNewSQLiteMigratesItemsToAccountScopedPrimaryKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	statements := []string{
		`CREATE TABLE accounts (id TEXT PRIMARY KEY, created_at INTEGER, max_devices INTEGER DEFAULT 5, max_items INTEGER DEFAULT 10000, max_bytes INTEGER DEFAULT 10485760, pairing_public_key TEXT DEFAULT '', pin_hash TEXT DEFAULT '', last_active_at INTEGER DEFAULT 0, archived INTEGER DEFAULT 0, archived_at INTEGER DEFAULT 0);`,
		`CREATE TABLE devices (id TEXT PRIMARY KEY, account_id TEXT, name TEXT, created_at INTEGER, revoked INTEGER DEFAULT 0, token_version INTEGER DEFAULT 1, FOREIGN KEY(account_id) REFERENCES accounts(id));`,
		`CREATE TABLE items (id TEXT PRIMARY KEY, account_id TEXT, payload TEXT, modified_at INTEGER, tombstone INTEGER DEFAULT 0, origin_device_id TEXT, version INTEGER, FOREIGN KEY(account_id) REFERENCES accounts(id));`,
		`CREATE TABLE system_state (key TEXT PRIMARY KEY, value TEXT NOT NULL);`,
		`INSERT INTO accounts (id) VALUES ('account-a');`,
		`INSERT INTO items (id, account_id, payload, modified_at, version) VALUES ('shared-id', 'account-a', '{}', 1, 1);`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			db.Close()
			t.Fatalf("prepare legacy database: %v", err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	store, err := NewSQLite(path)
	if err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	defer store.Close()

	primaryKeys := map[string]int{}
	rows, err := store.DB().Query(`PRAGMA table_info(items);`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue interface{}
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		primaryKeys[name] = primaryKey
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
	if primaryKeys["account_id"] == 0 || primaryKeys["id"] == 0 {
		t.Fatalf("expected composite primary key, got %#v", primaryKeys)
	}

	if _, err := store.DB().Exec(`INSERT INTO accounts (id) VALUES ('account-b');`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DB().Exec(`INSERT INTO items (id, account_id, payload, modified_at, version) VALUES ('shared-id', 'account-b', '{}', 2, 1);`); err != nil {
		t.Fatalf("same item id in another account must be allowed: %v", err)
	}
	var count int
	if err := store.DB().QueryRow(`SELECT COUNT(*) FROM items WHERE id = 'shared-id'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("expected both account-scoped items, got %d", count)
	}
}
