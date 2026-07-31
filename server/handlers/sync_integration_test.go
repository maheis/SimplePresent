package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/simplepresent/server/storage"
)

func newSyncTestServer(t *testing.T) *Server {
	t.Helper()
	store, err := storage.NewSQLite(filepath.Join(t.TempDir(), "sync.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if _, err := store.DB().Exec(`INSERT INTO accounts (id, created_at, max_devices, max_items, max_bytes) VALUES ('account-a', 1, 5, 100, 1048576)`); err != nil {
		t.Fatal(err)
	}
	return &Server{
		DB:            store.DB(),
		DefaultQuotas: Quotas{MaxDevices: 5, MaxItems: 100, MaxBytesPerAccount: 1048576},
	}
}

func pushAsDevice(t *testing.T, server *Server, deviceID, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/push", strings.NewReader(body))
	request = request.WithContext(context.WithValue(request.Context(), authContextKey, AuthInfo{
		AccountID: "account-a",
		DeviceID:  deviceID,
	}))
	response := httptest.NewRecorder()
	server.Push(response, request)
	return response
}

func pullAsDevice(t *testing.T, server *Server, deviceID string) []map[string]interface{} {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/pull?since=0", nil)
	request = request.WithContext(context.WithValue(request.Context(), authContextKey, AuthInfo{
		AccountID: "account-a",
		DeviceID:  deviceID,
	}))
	response := httptest.NewRecorder()
	server.Pull(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("pull failed: %d %s", response.Code, response.Body.String())
	}
	var body struct {
		Items []map[string]interface{} `json:"items"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body.Items
}

func TestTwoDevicesPushPullAndConflict(t *testing.T) {
	server := newSyncTestServer(t)

	response := pushAsDevice(t, server, "device-a", `{
		"items":[{
			"id":"task:one",
			"payload":{"value":"from-a"},
			"modified_at":100,
			"origin_device_id":"spoofed-device",
			"version":1
		}]
	}`)
	if response.Code != http.StatusOK {
		t.Fatalf("device A push failed: %d %s", response.Code, response.Body.String())
	}

	response = pushAsDevice(t, server, "device-b", `{
		"items":[{
			"id":"task:two",
			"payload":{"value":"from-b"},
			"modified_at":110,
			"version":1
		}]
	}`)
	if response.Code != http.StatusOK {
		t.Fatalf("device B push failed: %d %s", response.Code, response.Body.String())
	}

	items := pullAsDevice(t, server, "device-a")
	if len(items) != 2 {
		t.Fatalf("expected both devices' items, got %d", len(items))
	}
	origins := map[string]string{}
	for _, item := range items {
		origins[item["id"].(string)] = item["origin_device_id"].(string)
	}
	if origins["task:one"] != "device-a" || origins["task:two"] != "device-b" {
		t.Fatalf("server did not bind origins to authenticated devices: %#v", origins)
	}

	response = pushAsDevice(t, server, "device-b", `{
		"items":[{
			"id":"task:one",
			"payload":{"value":"newest"},
			"modified_at":200,
			"version":2
		}]
	}`)
	if response.Code != http.StatusOK {
		t.Fatalf("newer device B push failed: %d %s", response.Code, response.Body.String())
	}

	response = pushAsDevice(t, server, "device-a", `{
		"items":[{
			"id":"task:one",
			"payload":{"value":"stale"},
			"modified_at":150,
			"version":2
		}]
	}`)
	if response.Code != http.StatusConflict {
		t.Fatalf("expected stale device A push to conflict, got %d %s", response.Code, response.Body.String())
	}

	var payload string
	if err := server.DB.QueryRow(`SELECT payload FROM items WHERE account_id = 'account-a' AND id = 'task:one'`).Scan(&payload); err != nil {
		t.Fatal(err)
	}
	if payload != `{"value":"newest"}` {
		t.Fatalf("conflict overwrote newest server state: %s", payload)
	}
}

func TestPushConflictRollsBackEntireSnapshot(t *testing.T) {
	server := newSyncTestServer(t)
	response := pushAsDevice(t, server, "device-b", `{
		"items":[{
			"id":"task:existing",
			"payload":{"value":"server"},
			"modified_at":200,
			"version":2
		}]
	}`)
	if response.Code != http.StatusOK {
		t.Fatalf("initial push failed: %d %s", response.Code, response.Body.String())
	}

	response = pushAsDevice(t, server, "device-a", `{
		"items":[
			{
				"id":"task:new",
				"payload":{"value":"must-rollback"},
				"modified_at":150,
				"version":1
			},
			{
				"id":"task:existing",
				"payload":{"value":"stale"},
				"modified_at":150,
				"version":1
			}
		]
	}`)
	if response.Code != http.StatusConflict {
		t.Fatalf("expected snapshot conflict, got %d %s", response.Code, response.Body.String())
	}

	var count int
	if err := server.DB.QueryRow(`SELECT COUNT(*) FROM items WHERE account_id = 'account-a' AND id = 'task:new'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("conflicting snapshot was partially committed")
	}
}

func TestConcurrentReorderKeepsNewestSnapshot(t *testing.T) {
	server := newSyncTestServer(t)
	response := pushAsDevice(t, server, "device-a", `{
		"items":[
			{"id":"task:one","payload":{"position":0},"modified_at":100,"version":1},
			{"id":"task:two","payload":{"position":1},"modified_at":100,"version":1}
		]
	}`)
	if response.Code != http.StatusOK {
		t.Fatalf("initial order push failed: %d %s", response.Code, response.Body.String())
	}

	response = pushAsDevice(t, server, "device-b", `{
		"items":[
			{"id":"task:one","payload":{"position":1},"modified_at":200,"version":2},
			{"id":"task:two","payload":{"position":0},"modified_at":200,"version":2}
		]
	}`)
	if response.Code != http.StatusOK {
		t.Fatalf("newer reorder push failed: %d %s", response.Code, response.Body.String())
	}

	response = pushAsDevice(t, server, "device-a", `{
		"items":[
			{"id":"task:one","payload":{"position":0},"modified_at":150,"version":2},
			{"id":"task:two","payload":{"position":1},"modified_at":150,"version":2}
		]
	}`)
	if response.Code != http.StatusConflict {
		t.Fatalf("expected stale reorder conflict, got %d %s", response.Code, response.Body.String())
	}

	items := pullAsDevice(t, server, "device-a")
	positions := map[string]int{}
	for _, item := range items {
		payload := item["payload"].(map[string]interface{})
		positions[item["id"].(string)] = int(payload["position"].(float64))
	}
	if positions["task:one"] != 1 || positions["task:two"] != 0 {
		t.Fatalf("stale reorder changed newest positions: %#v", positions)
	}
}
