package storage

import (
	"path/filepath"
	"testing"
)

func TestNewSQLiteCreatesAccountScopedItems(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.db")
	store, err := NewSQLite(path)
	if err != nil {
		t.Fatalf("create database: %v", err)
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

	if _, err := store.DB().Exec(`INSERT INTO accounts (id) VALUES ('account-a'), ('account-b');`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DB().Exec(`INSERT INTO items (id, account_id, payload, modified_at, version) VALUES ('shared-id', 'account-a', '{}', 1, 1);`); err != nil {
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
