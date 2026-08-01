package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/simplepresent/server/handlers"
	"github.com/simplepresent/server/storage"
)

type httpAuthResult struct {
	AccountID string `json:"account_id"`
	DeviceID  string `json:"device_id"`
	Token     string `json:"token"`
}

type httpPullResult struct {
	Items []struct {
		ID      string                 `json:"id"`
		Payload map[string]interface{} `json:"payload"`
	} `json:"items"`
}

func newHTTPIntegrationServer(t *testing.T) *httptest.Server {
	t.Helper()
	store, err := storage.NewSQLite(filepath.Join(t.TempDir(), "http-sync.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	server := &handlers.Server{
		DB:              store.DB(),
		JWTSecret:       []byte("http-integration-test-secret"),
		DefaultQuotas:   handlers.Quotas{MaxDevices: 5, MaxItems: 100, MaxBytesPerAccount: 1048576},
		IPLimiters:      handlers.NewIPLimiters(6000, 1000),
		AccountLimiters: handlers.NewAccountLimiters(6000, 1000),
	}
	httpServer := httptest.NewServer(newRouter(server))
	t.Cleanup(httpServer.Close)
	return httpServer
}

func doJSONRequest(t *testing.T, client *http.Client, method, url, token string, body interface{}, out interface{}) (int, string) {
	t.Helper()
	var requestBody io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		requestBody = bytes.NewReader(encoded)
	}
	request, err := http.NewRequest(method, url, requestBody)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if out != nil && response.StatusCode >= 200 && response.StatusCode < 300 {
		if err := json.Unmarshal(responseBody, out); err != nil {
			t.Fatalf("decode %s %s response: %v; body=%s", method, url, err, responseBody)
		}
	}
	return response.StatusCode, string(responseBody)
}

func registerHTTPAccount(t *testing.T, server *httptest.Server, name string, publicKey ed25519.PublicKey) httpAuthResult {
	t.Helper()
	var result httpAuthResult
	status, body := doJSONRequest(t, server.Client(), http.MethodPost, server.URL+"/register", "", map[string]interface{}{
		"name":               name,
		"pairing_public_key": base64.StdEncoding.EncodeToString(publicKey),
		"pin":                "1234",
	}, &result)
	if status != http.StatusOK {
		t.Fatalf("register %s failed: %d %s", name, status, body)
	}
	return result
}

func pairHTTPDevice(t *testing.T, server *httptest.Server, accountID, name string, privateKey ed25519.PrivateKey) httpAuthResult {
	t.Helper()
	var challenge struct {
		ChallengeID string `json:"challenge_id"`
	}
	status, body := doJSONRequest(t, server.Client(), http.MethodPost, server.URL+"/pair/challenge", "", map[string]string{
		"account_id": accountID,
	}, &challenge)
	if status != http.StatusOK {
		t.Fatalf("pair challenge failed: %d %s", status, body)
	}

	message := []byte(fmt.Sprintf("simplepresent-pair|%s|%s|%s", accountID, challenge.ChallengeID, name))
	signature := ed25519.Sign(privateKey, message)
	var result httpAuthResult
	status, body = doJSONRequest(t, server.Client(), http.MethodPost, server.URL+"/pair", "", map[string]string{
		"account_id":   accountID,
		"name":         name,
		"challenge_id": challenge.ChallengeID,
		"signature":    base64.StdEncoding.EncodeToString(signature),
		"pin":          "1234",
	}, &result)
	if status != http.StatusOK {
		t.Fatalf("pair device failed: %d %s", status, body)
	}
	result.AccountID = accountID
	return result
}

func pushHTTPItem(t *testing.T, server *httptest.Server, token, value string) {
	t.Helper()
	status, body := doJSONRequest(t, server.Client(), http.MethodPost, server.URL+"/push", token, map[string]interface{}{
		"items": []map[string]interface{}{{
			"id":          "task:shared-id",
			"payload":     map[string]string{"value": value},
			"modified_at": 100,
			"version":     1,
		}},
	}, nil)
	if status != http.StatusOK {
		t.Fatalf("push %s failed: %d %s", value, status, body)
	}
}

func pullHTTPItems(t *testing.T, server *httptest.Server, token string) (int, httpPullResult, string) {
	t.Helper()
	var result httpPullResult
	status, body := doJSONRequest(t, server.Client(), http.MethodGet, server.URL+"/pull?since=0", token, nil, &result)
	return status, result, body
}

func TestHTTPRegisterPairSyncIsolationAndRevocation(t *testing.T) {
	server := newHTTPIntegrationServer(t)
	publicA, privateA, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	publicB, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	accountA := registerHTTPAccount(t, server, "account-a-owner", publicA)
	accountB := registerHTTPAccount(t, server, "account-b-owner", publicB)
	deviceA2 := pairHTTPDevice(t, server, accountA.AccountID, "account-a-second", privateA)

	pushHTTPItem(t, server, deviceA2.Token, "account-a-value")
	pushHTTPItem(t, server, accountB.Token, "account-b-value")

	status, itemsA, body := pullHTTPItems(t, server, accountA.Token)
	if status != http.StatusOK {
		t.Fatalf("account A pull failed: %d %s", status, body)
	}
	if len(itemsA.Items) != 1 || itemsA.Items[0].ID != "task:shared-id" || itemsA.Items[0].Payload["value"] != "account-a-value" {
		t.Fatalf("account A received non-isolated data: %#v", itemsA.Items)
	}

	status, itemsB, body := pullHTTPItems(t, server, accountB.Token)
	if status != http.StatusOK {
		t.Fatalf("account B pull failed: %d %s", status, body)
	}
	if len(itemsB.Items) != 1 || itemsB.Items[0].ID != "task:shared-id" || itemsB.Items[0].Payload["value"] != "account-b-value" {
		t.Fatalf("account B received non-isolated data: %#v", itemsB.Items)
	}

	status, body = doJSONRequest(t, server.Client(), http.MethodPost, server.URL+"/devices/"+deviceA2.DeviceID+"/revoke", accountA.Token, nil, nil)
	if status != http.StatusOK {
		t.Fatalf("device revoke failed: %d %s", status, body)
	}
	status, _, body = pullHTTPItems(t, server, deviceA2.Token)
	if status != http.StatusUnauthorized {
		t.Fatalf("revoked token remained usable: %d %s", status, body)
	}
}

func TestHTTPRouterEnforcesSecurityBoundaries(t *testing.T) {
	server := newHTTPIntegrationServer(t)
	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	account := registerHTTPAccount(t, server, "security-owner", publicKey)

	status, _, body := pullHTTPItems(t, server, "")
	if status != http.StatusUnauthorized {
		t.Fatalf("pull without token was not rejected: %d %s", status, body)
	}

	status, body = doJSONRequest(t, server.Client(), http.MethodPost, server.URL+"/push", account.Token, map[string]interface{}{
		"account_id": "another-account",
		"items": []map[string]interface{}{{
			"id":          "task:forbidden",
			"payload":     map[string]string{"value": "forbidden"},
			"modified_at": 100,
			"version":     1,
		}},
	}, nil)
	if status != http.StatusForbidden {
		t.Fatalf("account mismatch was not rejected: %d %s", status, body)
	}

	oversizedBody := fmt.Sprintf(
		`{"name":"oversized","pairing_public_key":"%s","pin":"1234","padding":"%s"}`,
		base64.StdEncoding.EncodeToString(publicKey),
		strings.Repeat("x", smallRequestBodyLimit),
	)
	request, err := http.NewRequest(http.MethodPost, server.URL+"/register", strings.NewReader(oversizedBody))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode < 400 {
		t.Fatalf("oversized register body was accepted: %d", response.StatusCode)
	}

	secureServer := &handlers.Server{RequireTLS: true, TrustProxyHeaders: true}
	proxyRequest := httptest.NewRequest(http.MethodPost, "http://example.test/register", strings.NewReader(`{}`))
	proxyRequest.RemoteAddr = "203.0.113.10:1234"
	proxyRequest.Header.Set("X-Forwarded-Proto", "https")
	proxyResponse := httptest.NewRecorder()
	newRouter(secureServer).ServeHTTP(proxyResponse, proxyRequest)
	if proxyResponse.Code != http.StatusUpgradeRequired {
		t.Fatalf("untrusted forwarded HTTPS was accepted: %d", proxyResponse.Code)
	}
}
