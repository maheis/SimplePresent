package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestSecureOnlyRejectsForwardedHTTPSFromUntrustedPeer(t *testing.T) {
	server := &Server{RequireTLS: true, TrustProxyHeaders: true}
	handler := server.SecureOnly(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "http://example.test/pull", nil)
	request.RemoteAddr = "203.0.113.10:1234"
	request.Header.Set("X-Forwarded-Proto", "https")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUpgradeRequired {
		t.Fatalf("expected untrusted proxy header to be rejected, got %d", response.Code)
	}

	request = httptest.NewRequest(http.MethodGet, "http://example.test/pull", nil)
	request.RemoteAddr = "127.0.0.1:1234"
	request.Header.Set("X-Forwarded-Proto", "https")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("expected loopback proxy header to be accepted, got %d", response.Code)
	}
}

func TestClientIPUsesRightmostUntrustedForwardedAddress(t *testing.T) {
	server := &Server{TrustProxyHeaders: true}
	request := httptest.NewRequest(http.MethodGet, "http://example.test", nil)
	request.RemoteAddr = "127.0.0.1:1234"
	request.Header.Set("X-Forwarded-For", "198.51.100.40, 203.0.113.25")
	if got := server.clientIP(request); got != "203.0.113.25" {
		t.Fatalf("expected nearest untrusted client address, got %q", got)
	}
}

func TestPushRejectsStaleRevision(t *testing.T) {
	db, err := sql.Open("sqlite", "file:push-conflict?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	statements := []string{
		`CREATE TABLE accounts (id TEXT PRIMARY KEY, max_items INTEGER, max_bytes INTEGER);`,
		`CREATE TABLE items (id TEXT NOT NULL, account_id TEXT NOT NULL, payload TEXT, modified_at INTEGER, tombstone INTEGER, origin_device_id TEXT, version INTEGER, PRIMARY KEY(account_id, id));`,
		`INSERT INTO accounts (id, max_items, max_bytes) VALUES ('account-a', 100, 1048576);`,
		`INSERT INTO items (id, account_id, payload, modified_at, tombstone, origin_device_id, version) VALUES ('task-1', 'account-a', '{"value":"new"}', 200, 0, 'device-b', 2);`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}

	server := &Server{DB: db, DefaultQuotas: Quotas{MaxItems: 100}}
	body := `{"items":[{"id":"task-1","payload":{"value":"old"},"modified_at":100,"version":1}]}`
	request := httptest.NewRequest(http.MethodPost, "/push", strings.NewReader(body))
	request = request.WithContext(context.WithValue(request.Context(), authContextKey, AuthInfo{
		AccountID: "account-a",
		DeviceID:  "device-a",
	}))
	response := httptest.NewRecorder()
	server.Push(response, request)
	if response.Code != http.StatusConflict {
		t.Fatalf("expected stale push conflict, got %d: %s", response.Code, response.Body.String())
	}

	var version int
	var payload string
	if err := db.QueryRow(`SELECT version, payload FROM items WHERE account_id = 'account-a' AND id = 'task-1'`).Scan(&version, &payload); err != nil {
		t.Fatal(err)
	}
	if version != 2 || payload != `{"value":"new"}` {
		t.Fatalf("stale push changed stored item: version=%d payload=%s", version, payload)
	}
}
