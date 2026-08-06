package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestSignAppToken(t *testing.T) {
	now := time.Unix(1_725_000_123, 0)
	token, err := signAppToken("app-123", "secret-456", now)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("token has %d parts", len(parts))
	}

	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatal(err)
	}
	var claims struct {
		IssuedAt  int64  `json:"iat"`
		ExpiresAt int64  `json:"exp"`
		AppID     string `json:"_appId"`
	}
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		t.Fatal(err)
	}
	wantIAT := now.Unix() / 1800 * 1800
	if claims.IssuedAt != wantIAT || claims.ExpiresAt != wantIAT+1980 || claims.AppID != "app-123" {
		t.Fatalf("unexpected claims: %+v", claims)
	}

	mac := hmac.New(sha256.New, []byte("secret-456"))
	_, _ = mac.Write([]byte(parts[0] + "." + parts[1]))
	wantSignature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if parts[2] != wantSignature {
		t.Fatal("invalid token signature")
	}
}

func TestSprintQueryRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v3/project/project-1/sprint/search" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		if r.URL.Query().Get("q") != "Sprint 8" || r.URL.Query().Get("pageSize") != "25" {
			t.Errorf("unexpected query: %s", r.URL.RawQuery)
		}
		assertHeaders(t, r, "")
		_, _ = io.WriteString(w, `{"result":[]}`)
	}))
	defer server.Close()

	cfg := testConfig(server.URL + "/api")
	query := url.Values{"q": {"Sprint 8"}, "pageSize": {"25"}}
	response, err := request(server.Client(), cfg, time.Unix(1_725_000_123, 0), http.MethodGet, "/v3/project/project-1/sprint/search", query, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if string(response) != `{"result":[]}` {
		t.Fatalf("unexpected response: %s", response)
	}
}

func TestProjectTaskQueryRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v3/project/project-1/task/query" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		if r.URL.Query().Get("tql") != "isDone = false" || r.URL.Query().Get("pageSize") != "50" {
			t.Errorf("unexpected query: %s", r.URL.RawQuery)
		}
		assertHeaders(t, r, "")
		_, _ = io.WriteString(w, `{"result":[]}`)
	}))
	defer server.Close()

	t.Setenv("TEAMBITION_APP_ID", "app-1")
	t.Setenv("TEAMBITION_APP_SECRET", "secret-1")
	t.Setenv("TEAMBITION_ORG_ID", "org-1")
	var stdout, stderr strings.Builder
	code := runProjectTaskQuery([]string{
		"--project-id", "project-1",
		"--tql", "isDone = false",
		"--page-size", "50",
		"--base-url", server.URL + "/api",
	}, &stdout, &stderr, server.Client(), func() time.Time {
		return time.Unix(1_725_000_123, 0)
	})
	if code != 0 {
		t.Fatalf("exit code %d, stderr: %s", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != `{"result":[]}` {
		t.Fatalf("unexpected response: %s", stdout.String())
	}
}

func TestProjectPriorityQueryRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v3/project/priority/list" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		assertHeaders(t, r, "")
		_, _ = io.WriteString(w, `{"result":[]}`)
	}))
	defer server.Close()

	t.Setenv("TEAMBITION_APP_ID", "app-1")
	t.Setenv("TEAMBITION_APP_SECRET", "secret-1")
	t.Setenv("TEAMBITION_ORG_ID", "org-1")
	var stdout, stderr strings.Builder
	code := runProjectPriorityQuery([]string{
		"--base-url", server.URL + "/api",
	}, &stdout, &stderr, server.Client(), func() time.Time {
		return time.Unix(1_725_000_123, 0)
	})
	if code != 0 {
		t.Fatalf("exit code %d, stderr: %s", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != `{"result":[]}` {
		t.Fatalf("unexpected response: %s", stdout.String())
	}
}

func TestMemberQueryRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v3/member/query" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		if r.URL.Query().Get("userIds") != "user-1,user-2" || r.URL.Query().Get("pageSize") != "50" {
			t.Errorf("unexpected query: %s", r.URL.RawQuery)
		}
		assertHeaders(t, r, "")
		_, _ = io.WriteString(w, `{"result":[]}`)
	}))
	defer server.Close()

	t.Setenv("TEAMBITION_APP_ID", "app-1")
	t.Setenv("TEAMBITION_APP_SECRET", "secret-1")
	t.Setenv("TEAMBITION_ORG_ID", "org-1")
	var stdout, stderr strings.Builder
	code := runMemberQuery([]string{
		"--user-id", "user-1,user-2",
		"--page-size", "50",
		"--base-url", server.URL + "/api",
	}, &stdout, &stderr, server.Client(), func() time.Time {
		return time.Unix(1_725_000_123, 0)
	})
	if code != 0 {
		t.Fatalf("exit code %d, stderr: %s", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != `{"result":[]}` {
		t.Fatalf("unexpected response: %s", stdout.String())
	}
}

func TestTaskSprintUpdateRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/v3/task/task-1/sprint" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		assertHeaders(t, r, "user-1")
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["sprintId"] != "sprint-8" {
			t.Fatalf("unexpected body: %#v", body)
		}
		_, _ = io.WriteString(w, `{"result":{"ok":true}}`)
	}))
	defer server.Close()

	t.Setenv("TEAMBITION_APP_ID", "app-1")
	t.Setenv("TEAMBITION_APP_SECRET", "secret-1")
	t.Setenv("TEAMBITION_ORG_ID", "org-1")
	t.Setenv("TEAMBITION_OPERATOR_ID", "user-1")
	var stdout, stderr strings.Builder
	code := runTaskUpdate([]string{
		"--task-id", "task-1",
		"--sprint-id", "sprint-8",
		"--base-url", server.URL + "/api",
	}, &stdout, &stderr, server.Client(), func() time.Time {
		return time.Unix(1_725_000_123, 0)
	})
	if code != 0 {
		t.Fatalf("exit code %d, stderr: %s", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != `{"result":{"ok":true}}` {
		t.Fatalf("unexpected response: %s", stdout.String())
	}
}

func TestGatewayErrorInSuccessfulHTTPResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"code":0,"errorCode":"10002","errorMessage":"Invalid parameter"}`)
	}))
	defer server.Close()

	cfg := testConfig(server.URL)
	_, err := request(server.Client(), cfg, time.Now(), http.MethodGet, "/v3/test", nil, nil, "")
	if err == nil || !strings.Contains(err.Error(), "10002") {
		t.Fatalf("expected gateway error, got %v", err)
	}
}

func assertHeaders(t *testing.T, r *http.Request, operatorID string) {
	t.Helper()
	if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
		t.Error("missing bearer token")
	}
	if r.Header.Get("X-Tenant-Id") != "org-1" || r.Header.Get("X-Tenant-Type") != "organization" {
		t.Errorf("unexpected tenant headers: %#v", r.Header)
	}
	if r.Header.Get("X-Operator-Id") != operatorID {
		t.Errorf("unexpected operator ID: %q", r.Header.Get("X-Operator-Id"))
	}
}

func testConfig(baseURL string) config {
	return config{
		appID:     "app-1",
		appSecret: "secret-1",
		orgID:     "org-1",
		baseURL:   baseURL,
	}
}
