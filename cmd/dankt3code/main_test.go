package main

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

var fixedTime = time.Date(2026, 9, 13, 12, 30, 0, 0, time.UTC)

func TestSnapshotFromSyntheticFixture(t *testing.T) {
	fixture := readFixture(t, "shell-states.json")
	var sawToken atomic.Bool
	var snapshotReturned atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/.well-known/t3/environment":
			if request.Header.Get("Authorization") != "" {
				t.Error("descriptor request included bearer token")
			}
			writeDescriptor(response, supportedServerVersion)
		case "/api/orchestration/shell":
			sawToken.Store(request.Header.Get("Authorization") == "Bearer synthetic-token")
			response.Header().Set("Content-Type", "application/json")
			_, _ = response.Write(fixture)
			snapshotReturned.Store(true)
		default:
			http.NotFound(response, request)
		}
	}))
	defer server.Close()

	result, exitCode := run(snapshotArgs(t, server.URL, "synthetic-token"), func() time.Time {
		if !snapshotReturned.Load() {
			t.Error("observation timestamp was taken before collection completed")
		}
		return fixedTime
	})
	if exitCode != 0 || result.State != "connected" || result.Error != nil {
		t.Fatalf("unexpected result: exit=%d result=%+v", exitCode, result)
	}
	if !sawToken.Load() {
		t.Fatal("snapshot request did not contain bearer token")
	}
	if result.ObservedAt != "2026-09-13T12:30:00Z" || result.Environment == nil || result.Environment.ID != "environment-synthetic" {
		t.Fatalf("unexpected observation or environment: %+v", result)
	}
	if result.Snapshot == nil || result.Snapshot.Sequence != 42 || len(result.Snapshot.Projects) != 1 || len(result.Snapshot.Threads) != 6 {
		t.Fatalf("unexpected minimized snapshot: %+v", result.Snapshot)
	}
	if got := result.Snapshot.Projects[0]; got.ID != "project-synthetic" || got.Title != "Synthetic project" {
		t.Fatalf("unexpected project: %+v", got)
	}
	first := result.Snapshot.Threads[0]
	if first.ProviderInstanceID != "codex" || first.Session == nil || first.Session.Status != "running" || first.Session.ProviderName != "codex" || first.LatestTurn == nil || first.LatestTurn.State != "running" || !first.HasPendingUserInput || first.BackgroundLiveness == nil || *first.BackgroundLiveness != "working" {
		t.Fatalf("unexpected first thread: %+v", first)
	}
	if result.Snapshot.Threads[3].Session == nil || result.Snapshot.Threads[3].Session.Status != "error" || result.Snapshot.Threads[3].LatestTurn.State != "error" {
		t.Fatalf("error state was not retained: %+v", result.Snapshot.Threads[3])
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"workspaceRoot", "worktreePath", "lastError", "scripts", "planProgress", "Synthetic provider failure", "/synthetic/workspace"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("output contains private or unnecessary field %q: %s", forbidden, encoded)
		}
	}
}

func TestEmptySnapshotIsConnected(t *testing.T) {
	server := fixtureServer(t, supportedServerVersion, readFixture(t, "shell-empty.json"), http.StatusOK)
	result, exitCode := run(snapshotArgs(t, server.URL, "synthetic-token"), func() time.Time { return fixedTime })
	if exitCode != 0 || result.State != "connected" || result.Snapshot == nil || result.Snapshot.Projects == nil || result.Snapshot.Threads == nil {
		t.Fatalf("empty snapshot was not a connected observation: exit=%d result=%+v", exitCode, result)
	}
}

func TestUnsupportedVersionStopsBeforeAuthenticatedFetch(t *testing.T) {
	var shellRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/.well-known/t3/environment" {
			writeDescriptor(response, "0.0.41")
			return
		}
		shellRequests.Add(1)
		http.Error(response, "should not be reached", http.StatusInternalServerError)
	}))
	defer server.Close()

	result, exitCode := run(snapshotArgs(t, server.URL, "synthetic-token"), func() time.Time { return fixedTime })
	assertError(t, result, exitCode, "incompatible_server")
	if result.Environment == nil || result.Environment.ServerVersion != "0.0.41" || shellRequests.Load() != 0 {
		t.Fatalf("unsupported server handling was not fail-closed: %+v requests=%d", result, shellRequests.Load())
	}
}

func TestMalformedDescriptor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		_, _ = response.Write([]byte(`{"environmentId":"","label":"Synthetic environment","serverVersion":"0.0.40"}`))
	}))
	defer server.Close()

	result, exitCode := run(snapshotArgs(t, server.URL, "synthetic-token"), func() time.Time { return fixedTime })
	assertError(t, result, exitCode, "malformed_descriptor")
	if result.Environment != nil {
		t.Fatalf("malformed environment identity was emitted: %+v", result.Environment)
	}
}

func TestAuthenticationAndAuthorizationErrors(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		code   string
	}{
		{name: "unauthorized", status: http.StatusUnauthorized, code: "authentication_failed"},
		{name: "forbidden", status: http.StatusForbidden, code: "authorization_failed"},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := fixtureServer(t, supportedServerVersion, []byte(`{"private":"server detail"}`), test.status)
			result, exitCode := run(snapshotArgs(t, server.URL, "synthetic-token"), func() time.Time { return fixedTime })
			assertError(t, result, exitCode, test.code)
			encoded, _ := json.Marshal(result)
			if strings.Contains(string(encoded), "server detail") {
				t.Fatalf("raw server error leaked: %s", encoded)
			}
		})
	}
}

func TestMalformedSnapshotCannotLookIdle(t *testing.T) {
	for _, body := range []string{
		`{"snapshotSequence":0,"updatedAt":"2026-09-13T12:00:00Z"}`,
		`{"snapshotSequence":0,"projects":[],"threads":[],"updatedAt":"not-a-date"}`,
		`{"snapshotSequence":0,"projects":[],"threads":[{"id":"thread","projectId":"missing","title":"Title","createdAt":"2026-09-13T12:00:00Z","updatedAt":"2026-09-13T12:00:00Z","latestUserMessageAt":null,"latestTurn":null,"session":null,"hasPendingApprovals":false,"hasPendingUserInput":false,"hasActionableProposedPlan":false}],"updatedAt":"2026-09-13T12:00:00Z"}`,
	} {
		server := fixtureServer(t, supportedServerVersion, []byte(body), http.StatusOK)
		result, exitCode := run(snapshotArgs(t, server.URL, "synthetic-token"), func() time.Time { return fixedTime })
		server.Close()
		assertError(t, result, exitCode, "malformed_snapshot")
		if result.Snapshot != nil {
			t.Fatalf("malformed response produced a snapshot: %+v", result.Snapshot)
		}
	}
}

func TestUnknownStatusesArePreservedAsUnknown(t *testing.T) {
	body := strings.ReplaceAll(string(readFixture(t, "shell-states.json")), `"status": "running"`, `"status": "future-session-state"`)
	body = strings.ReplaceAll(body, `"state": "running"`, `"state": "future-turn-state"`)
	body = strings.Replace(body, `"backgroundLiveness": "working"`, `"backgroundLiveness": "future-background-state"`, 1)
	server := fixtureServer(t, supportedServerVersion, []byte(body), http.StatusOK)
	result, exitCode := run(snapshotArgs(t, server.URL, "synthetic-token"), func() time.Time { return fixedTime })
	if exitCode != 0 || result.Snapshot == nil {
		t.Fatalf("unknown statuses should remain decodable: %+v", result)
	}
	first := result.Snapshot.Threads[0]
	if first.Session.Status != "unknown" || first.LatestTurn.State != "unknown" || first.BackgroundLiveness == nil || *first.BackgroundLiveness != "unknown" {
		t.Fatalf("unknown statuses were not normalized: %+v", first)
	}
}

func TestNullSessionProviderUsesUnknown(t *testing.T) {
	body := strings.Replace(string(readFixture(t, "shell-states.json")), `"providerName": "codex"`, `"providerName": null`, 1)
	server := fixtureServer(t, supportedServerVersion, []byte(body), http.StatusOK)
	result, exitCode := run(snapshotArgs(t, server.URL, "synthetic-token"), func() time.Time { return fixedTime })
	if exitCode != 0 || result.Snapshot == nil || result.Snapshot.Threads[0].Session == nil {
		t.Fatalf("nullable provider name should remain decodable: %+v", result)
	}
	if result.Snapshot.Threads[0].Session.ProviderName != "unknown" {
		t.Fatalf("null provider name was not normalized: %+v", result.Snapshot.Threads[0].Session)
	}
}

func TestOversizedResponse(t *testing.T) {
	server := fixtureServer(t, supportedServerVersion, []byte(strings.Repeat("x", maxResponseBytes+1)), http.StatusOK)
	result, exitCode := run(snapshotArgs(t, server.URL, "synthetic-token"), func() time.Time { return fixedTime })
	assertError(t, result, exitCode, "response_too_large")
}

func TestUnavailableServer(t *testing.T) {
	listener, err := netListenLoopback()
	if err != nil {
		t.Fatal(err)
	}
	endpoint := "http://" + listener.Addr().String()
	listener.Close()
	result, exitCode := run(snapshotArgs(t, endpoint, "synthetic-token"), func() time.Time { return fixedTime })
	assertError(t, result, exitCode, "server_unavailable")
}

func TestEndpointValidation(t *testing.T) {
	valid := []string{"http://localhost:3000", "http://127.0.0.1:3000/", "https://[::1]:443"}
	for _, endpoint := range valid {
		if _, err := validateEndpoint(endpoint); err != nil {
			t.Errorf("expected valid endpoint %q: %v", endpoint, err)
		}
	}
	invalid := []string{
		"http://example.com:3000",
		"http://127.0.0.1:3000/private",
		"http://user:pass@127.0.0.1:3000",
		"http://127.0.0.1:3000?token=secret",
		"http://127.0.0.1:3000?",
		"http://127.0.0.1:3000#fragment",
		"ftp://127.0.0.1:3000",
		"http://127.0.0.1:invalid",
		"http://127.0.0.1:http",
		"http://127.0.0.1:",
		"http://127.0.0.1:65536",
	}
	for _, endpoint := range invalid {
		if _, err := validateEndpoint(endpoint); err == nil {
			t.Errorf("expected invalid endpoint %q", endpoint)
		}
	}
}

func TestTokenFileRequirements(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "token")
	if err := os.WriteFile(path, []byte("synthetic-token\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if token, err := readToken(path); err != nil || token != "synthetic-token" {
		t.Fatalf("valid token was rejected: token=%q err=%v", token, err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readToken(path); err == nil {
		t.Fatal("group/world-readable token was accepted")
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}

	symlink := filepath.Join(directory, "token-link")
	if err := os.Symlink(path, symlink); err != nil {
		t.Fatal(err)
	}
	if _, err := readToken(symlink); err == nil {
		t.Fatal("symlink token was accepted")
	}
}

func TestAuthenticatedRedirectDoesNotForwardToken(t *testing.T) {
	var redirectedRequests atomic.Int32
	destination := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		redirectedRequests.Add(1)
		response.WriteHeader(http.StatusOK)
	}))
	defer destination.Close()
	source := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/.well-known/t3/environment" {
			writeDescriptor(response, supportedServerVersion)
			return
		}
		http.Redirect(response, request, destination.URL, http.StatusTemporaryRedirect)
	}))
	defer source.Close()
	result, exitCode := run(snapshotArgs(t, source.URL, "synthetic-token"), func() time.Time { return fixedTime })
	assertError(t, result, exitCode, "unexpected_response")
	if redirectedRequests.Load() != 0 {
		t.Fatal("collector followed an authenticated redirect")
	}
}

func TestDescriptorErrorsDoNotBlameBearer(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			if request.URL.Path != "/.well-known/t3/environment" || request.Header.Get("Authorization") != "" {
				t.Error("descriptor failure should stop collection before bearer use")
			}
			response.WriteHeader(status)
		}))
		result, exitCode := run(snapshotArgs(t, server.URL, "synthetic-token"), func() time.Time { return fixedTime })
		server.Close()
		assertError(t, result, exitCode, "unexpected_response")
	}
}

func TestInvalidUTF8DoesNotRewriteIdentity(t *testing.T) {
	for _, location := range []string{"descriptor", "snapshot"} {
		t.Run(location, func(t *testing.T) {
			body := strings.Replace(string(readFixture(t, "shell-states.json")), "thread-running-input", "thread-\xff", 1)
			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				if request.URL.Path == "/.well-known/t3/environment" {
					if location == "descriptor" {
						_, _ = response.Write([]byte("{\"environmentId\":\"\xff\",\"label\":\"Synthetic\",\"serverVersion\":\"0.0.40\"}"))
					} else {
						writeDescriptor(response, supportedServerVersion)
					}
					return
				}
				_, _ = response.Write([]byte(body))
			}))
			defer server.Close()
			result, exitCode := run(snapshotArgs(t, server.URL, "synthetic-token"), func() time.Time { return fixedTime })
			assertError(t, result, exitCode, "malformed_"+location)
			if result.Snapshot != nil {
				t.Fatal("malformed identity produced a usable snapshot")
			}
		})
	}
}

func TestArgumentErrorsAreSanitized(t *testing.T) {
	secretPath := "/private/token/path"
	result, exitCode := run([]string{"snapshot", "--token-file", secretPath}, func() time.Time { return fixedTime })
	assertError(t, result, exitCode, "invalid_arguments")
	encoded, _ := json.Marshal(result)
	if strings.Contains(string(encoded), secretPath) {
		t.Fatalf("argument value leaked: %s", encoded)
	}
}

func fixtureServer(t *testing.T, version string, shell []byte, shellStatus int) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/.well-known/t3/environment":
			writeDescriptor(response, version)
		case "/api/orchestration/shell":
			response.WriteHeader(shellStatus)
			_, _ = response.Write(shell)
		default:
			http.NotFound(response, request)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func writeDescriptor(response http.ResponseWriter, version string) {
	response.Header().Set("Content-Type", "application/json")
	_, _ = response.Write([]byte(`{"environmentId":"environment-synthetic","label":"Synthetic environment","serverVersion":"` + version + `","platform":{"os":"linux","arch":"x64"},"capabilities":{}}`))
}

func snapshotArgs(t *testing.T, endpoint, token string) []string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(path, []byte(token+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return []string{"snapshot", "--endpoint", endpoint, "--token-file", path}
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "tests", "fixtures", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func assertError(t *testing.T, result output, exitCode int, code string) {
	t.Helper()
	if exitCode == 0 || result.State != "error" || result.Error == nil || result.Error.Code != code || result.Snapshot != nil {
		t.Fatalf("expected %q error, exit=%d result=%+v", code, exitCode, result)
	}
}

func netListenLoopback() (net.Listener, error) {
	return net.Listen("tcp", "127.0.0.1:0")
}
