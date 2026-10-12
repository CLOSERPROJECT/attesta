package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestBootstrapAppwriteProjectNoopWhenAdminProbeSucceeds(t *testing.T) {
	var seenPaths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPaths = append(seenPaths, r.Method+" "+r.URL.Path)
		if r.Header.Get("X-Appwrite-Project") != "attesta" {
			t.Fatalf("probe project = %q", r.Header.Get("X-Appwrite-Project"))
		}
		if r.Header.Get("X-Appwrite-Key") != "working-key" {
			t.Fatalf("probe key = %q", r.Header.Get("X-Appwrite-Key"))
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/users":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"total":0,"users":[]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/storage/buckets/org-assets":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"$id":"org-assets","name":"org-assets"}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	envPath := filepath.Join(t.TempDir(), "attesta-appwrite.env")
	result, err := BootstrapAppwriteProject(t.Context(), BootstrapAppwriteDeps{
		Endpoint:        srv.URL + "/v1",
		ProjectID:       "attesta",
		APIKey:          "working-key",
		OrgAssetsBucket: "org-assets",
		EnvFilePath:     envPath,
		HTTPClient:      srv.Client(),
	})
	if err != nil {
		t.Fatalf("BootstrapAppwriteProject: %v", err)
	}
	if !result.Skipped {
		t.Fatalf("result = %#v, want Skipped", result)
	}
	if result.ProjectID != "attesta" || result.APIKey != "working-key" {
		t.Fatalf("result credentials = %#v", result)
	}
	assertEnvFile(t, envPath, "attesta", "working-key")
	if len(seenPaths) < 1 || seenPaths[0] != "GET /v1/users" {
		t.Fatalf("seenPaths = %#v, want probe first", seenPaths)
	}
}

func TestBootstrapAppwriteProjectCreatesProjectKeyAndBucketWhenProbeFails(t *testing.T) {
	var accountCreates int
	var loginAttempts int
	var keyCreates int
	var keyBody map[string]any
	var projectBody map[string]any
	var bucketBody map[string]any
	var consoleTeamCreates int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		project := r.Header.Get("X-Appwrite-Project")
		key := r.Header.Get("X-Appwrite-Key")
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/users" && project == "attesta":
			// Stale Coolify key / empty project.
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"message":"invalid key","code":401}`))

		case r.Method == http.MethodPost && r.URL.Path == "/v1/account" && project == "console":
			accountCreates++
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"$id":"console-user","email":"admin@appwrite.local"}`))

		case r.Method == http.MethodPost && r.URL.Path == "/v1/account/sessions/email" && project == "console":
			loginAttempts++
			http.SetCookie(w, &http.Cookie{Name: "a_session_console", Value: "console-session", Path: "/"})
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"$id":"sess-1","userId":"console-user","secret":"console-session"}`))

		case r.Method == http.MethodGet && r.URL.Path == "/v1/teams" && project == "console":
			if cookie, err := r.Cookie("a_session_console"); err != nil || cookie.Value != "console-session" {
				t.Fatalf("console session cookie missing: %v", err)
			}
			_, _ = w.Write([]byte(`{"total":0,"teams":[]}`))

		case r.Method == http.MethodPost && r.URL.Path == "/v1/teams" && project == "console":
			consoleTeamCreates++
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode team: %v", err)
			}
			if body["teamId"] != "attesta" || body["name"] == "" {
				t.Fatalf("team body = %#v", body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"$id":"attesta","name":"Attesta"}`))

		case r.Method == http.MethodGet && r.URL.Path == "/v1/projects/attesta" && project == "console":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"Project not found","code":404}`))

		case r.Method == http.MethodPost && r.URL.Path == "/v1/projects" && project == "console":
			if err := json.NewDecoder(r.Body).Decode(&projectBody); err != nil {
				t.Fatalf("decode project: %v", err)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"$id":"attesta","name":"Attesta","teamId":"attesta"}`))

		case r.Method == http.MethodGet && r.URL.Path == "/v1/projects/attesta/keys" && project == "console":
			_, _ = w.Write([]byte(`{"total":0,"keys":[]}`))

		case r.Method == http.MethodPost && r.URL.Path == "/v1/projects/attesta/keys" && project == "console":
			keyCreates++
			if err := json.NewDecoder(r.Body).Decode(&keyBody); err != nil {
				t.Fatalf("decode key: %v", err)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"$id":"key-1","secret":"standard_bootstrapped_key","name":"attesta-bootstrap"}`))

		case r.Method == http.MethodGet && r.URL.Path == "/v1/storage/buckets/org-assets" && project == "attesta":
			if key != "standard_bootstrapped_key" {
				t.Fatalf("bucket probe key = %q", key)
			}
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"Bucket not found","code":404}`))

		case r.Method == http.MethodPost && r.URL.Path == "/v1/storage/buckets" && project == "attesta":
			if key != "standard_bootstrapped_key" {
				t.Fatalf("bucket create key = %q", key)
			}
			if err := json.NewDecoder(r.Body).Decode(&bucketBody); err != nil {
				t.Fatalf("decode bucket: %v", err)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"$id":"org-assets","name":"org-assets"}`))

		default:
			body, _ := io.ReadAll(r.Body)
			t.Fatalf("unexpected request: %s %s project=%q key=%q body=%s", r.Method, r.URL.Path, project, key, body)
		}
	}))
	defer srv.Close()

	envPath := filepath.Join(t.TempDir(), "attesta-appwrite.env")
	result, err := BootstrapAppwriteProject(t.Context(), BootstrapAppwriteDeps{
		Endpoint:        srv.URL + "/v1",
		ProjectID:       "attesta",
		APIKey:          "stale-coolify-key",
		ConsoleEmail:    "admin@appwrite.local",
		ConsolePassword: "admin@appwrite.local",
		OrgAssetsBucket: "org-assets",
		ConsoleTeamID:   "attesta",
		EnvFilePath:     envPath,
		HTTPClient:      srv.Client(),
	})
	if err != nil {
		t.Fatalf("BootstrapAppwriteProject: %v", err)
	}
	if result.Skipped || !result.Created {
		t.Fatalf("result = %#v, want Created", result)
	}
	if result.APIKey != "standard_bootstrapped_key" || result.ProjectID != "attesta" {
		t.Fatalf("result = %#v", result)
	}
	if accountCreates != 0 || loginAttempts != 1 || consoleTeamCreates != 1 || keyCreates != 1 {
		t.Fatalf("accountCreates=%d loginAttempts=%d teamCreates=%d keyCreates=%d", accountCreates, loginAttempts, consoleTeamCreates, keyCreates)
	}
	if projectBody["projectId"] != "attesta" || projectBody["teamId"] != "attesta" {
		t.Fatalf("project body = %#v", projectBody)
	}
	scopes, _ := keyBody["scopes"].([]any)
	if keyBody["name"] != "attesta-bootstrap" || len(scopes) == 0 {
		t.Fatalf("key body = %#v", keyBody)
	}
	for _, want := range []string{"users.read", "users.write", "teams.read", "teams.write", "sessions.write", "files.read", "files.write", "buckets.read", "buckets.write"} {
		if !scopeListContains(scopes, want) {
			t.Fatalf("scopes missing %q in %#v", want, scopes)
		}
	}
	if scopeListContains(scopes, "sessions.read") {
		t.Fatalf("scopes must not include invalid sessions.read: %#v", scopes)
	}
	if bucketBody["bucketId"] != "org-assets" {
		t.Fatalf("bucket body = %#v", bucketBody)
	}
	assertEnvFile(t, envPath, "attesta", "standard_bootstrapped_key")
}

func TestBootstrapAppwriteProjectLoginFirstSkipsSignup(t *testing.T) {
	var accountCreates int
	var loginAttempts int

	srv := newBootstrapConsoleStub(t, bootstrapConsoleStubOpts{
		onAccount: func(w http.ResponseWriter, _ *http.Request) {
			accountCreates++
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"$id":"u"}`))
		},
		onLogin: func(w http.ResponseWriter, _ *http.Request) {
			loginAttempts++
			http.SetCookie(w, &http.Cookie{Name: "a_session_console", Value: "sess", Path: "/"})
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"$id":"s","secret":"sess"}`))
		},
		keysJSON: `{"total":0,"keys":[]}`,
		onKeyCreate: func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"$id":"k","secret":"login-first-key","name":"attesta-bootstrap"}`))
		},
	})
	defer srv.Close()

	envPath := filepath.Join(t.TempDir(), "env")
	result, err := BootstrapAppwriteProject(t.Context(), BootstrapAppwriteDeps{
		Endpoint:        srv.URL + "/v1",
		ProjectID:       "attesta",
		ConsoleEmail:    "admin@appwrite.local",
		ConsolePassword: "admin@appwrite.local",
		OrgAssetsBucket: "org-assets",
		ConsoleTeamID:   "attesta",
		EnvFilePath:     envPath,
		HTTPClient:      srv.Client(),
	})
	if err != nil {
		t.Fatalf("BootstrapAppwriteProject: %v", err)
	}
	if accountCreates != 0 {
		t.Fatalf("signup called %d times; want 0 when login succeeds", accountCreates)
	}
	if loginAttempts != 1 {
		t.Fatalf("loginAttempts=%d, want 1", loginAttempts)
	}
	if result.APIKey != "login-first-key" {
		t.Fatalf("result = %#v", result)
	}
}

func TestBootstrapAppwriteProjectLogin401Signup501ThenLoginSucceeds(t *testing.T) {
	var accountCreates int
	var loginAttempts int
	var signupStatus int

	srv := newBootstrapConsoleStub(t, bootstrapConsoleStubOpts{
		onAccount: func(w http.ResponseWriter, _ *http.Request) {
			accountCreates++
			signupStatus = http.StatusNotImplemented
			w.WriteHeader(http.StatusNotImplemented)
			_, _ = w.Write([]byte(`{"message":"Sign up to the console is restricted.","code":501,"type":"user_console_count_exceeded"}`))
		},
		onLogin: func(w http.ResponseWriter, _ *http.Request) {
			loginAttempts++
			if loginAttempts == 1 {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"message":"Invalid credentials","code":401}`))
				return
			}
			http.SetCookie(w, &http.Cookie{Name: "a_session_console", Value: "sess", Path: "/"})
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"$id":"s","secret":"sess"}`))
		},
		keysJSON: `{"total":0,"keys":[]}`,
		onKeyCreate: func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"$id":"k","secret":"after-501-key","name":"attesta-bootstrap"}`))
		},
	})
	defer srv.Close()

	envPath := filepath.Join(t.TempDir(), "env")
	result, err := BootstrapAppwriteProject(t.Context(), BootstrapAppwriteDeps{
		Endpoint:        srv.URL + "/v1",
		ProjectID:       "attesta",
		ConsoleEmail:    "admin@appwrite.local",
		ConsolePassword: "admin@appwrite.local",
		OrgAssetsBucket: "org-assets",
		ConsoleTeamID:   "attesta",
		EnvFilePath:     envPath,
		HTTPClient:      srv.Client(),
	})
	if err != nil {
		t.Fatalf("BootstrapAppwriteProject: %v", err)
	}
	if accountCreates != 1 || signupStatus != http.StatusNotImplemented {
		t.Fatalf("accountCreates=%d signupStatus=%d", accountCreates, signupStatus)
	}
	if loginAttempts != 2 {
		t.Fatalf("loginAttempts=%d, want 2 (fail then retry)", loginAttempts)
	}
	if result.APIKey != "after-501-key" || !result.Created {
		t.Fatalf("result = %#v", result)
	}
	assertEnvFile(t, envPath, "attesta", "after-501-key")
}

func TestBootstrapAppwriteProjectIgnoresExistingConsoleAccount(t *testing.T) {
	t.Run("conflict", func(t *testing.T) {
		testBootstrapIgnoresConsoleSignupStatus(t, http.StatusConflict, `{"message":"user already exists","code":409}`)
	})
	t.Run("console_signup_restricted", func(t *testing.T) {
		// Appwrite 1.8 returns 501 user_console_count_exceeded after the first root.
		testBootstrapIgnoresConsoleSignupStatus(t, http.StatusNotImplemented, `{"message":"Sign up to the console is restricted.","code":501,"type":"user_console_count_exceeded"}`)
	})
}

func testBootstrapIgnoresConsoleSignupStatus(t *testing.T, signupStatus int, signupBody string) {
	t.Helper()
	var loginAttempts int
	srv := newBootstrapConsoleStub(t, bootstrapConsoleStubOpts{
		onAccount: func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(signupStatus)
			_, _ = w.Write([]byte(signupBody))
		},
		onLogin: func(w http.ResponseWriter, _ *http.Request) {
			loginAttempts++
			if loginAttempts == 1 {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"code":401}`))
				return
			}
			http.SetCookie(w, &http.Cookie{Name: "a_session_console", Value: "sess", Path: "/"})
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"$id":"s","secret":"sess"}`))
		},
		keysJSON: `{"total":0,"keys":[]}`,
		onKeyCreate: func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"$id":"k","secret":"new-key"}`))
		},
	})
	defer srv.Close()

	envPath := filepath.Join(t.TempDir(), "env")
	result, err := BootstrapAppwriteProject(t.Context(), BootstrapAppwriteDeps{
		Endpoint:        srv.URL + "/v1",
		ProjectID:       "attesta",
		ConsoleEmail:    "admin@appwrite.local",
		ConsolePassword: "admin@appwrite.local",
		OrgAssetsBucket: "org-assets",
		ConsoleTeamID:   "attesta",
		EnvFilePath:     envPath,
		HTTPClient:      srv.Client(),
	})
	if err != nil {
		t.Fatalf("BootstrapAppwriteProject: %v", err)
	}
	if result.APIKey != "new-key" || !result.Created {
		t.Fatalf("result = %#v", result)
	}
	assertEnvFile(t, envPath, "attesta", "new-key")
}

func TestBootstrapAppwriteProjectReusesExistingBootstrapKeySecret(t *testing.T) {
	var keyCreates int
	var keyDeletes int

	srv := newBootstrapConsoleStub(t, bootstrapConsoleStubOpts{
		onLogin: func(w http.ResponseWriter, _ *http.Request) {
			http.SetCookie(w, &http.Cookie{Name: "a_session_console", Value: "sess", Path: "/"})
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"$id":"s","secret":"sess"}`))
		},
		keysJSON: `{"total":1,"keys":[{"$id":"key-existing","name":"attesta-bootstrap","secret":"reused-secret"}]}`,
		onKeyCreate: func(w http.ResponseWriter, _ *http.Request) {
			keyCreates++
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"$id":"k","secret":"should-not-create"}`))
		},
		onKeyDelete: func(w http.ResponseWriter, _ *http.Request) {
			keyDeletes++
			w.WriteHeader(http.StatusNoContent)
		},
	})
	defer srv.Close()

	envPath := filepath.Join(t.TempDir(), "env")
	result, err := BootstrapAppwriteProject(t.Context(), BootstrapAppwriteDeps{
		Endpoint:        srv.URL + "/v1",
		ProjectID:       "attesta",
		ConsoleEmail:    "admin@appwrite.local",
		ConsolePassword: "admin@appwrite.local",
		OrgAssetsBucket: "org-assets",
		ConsoleTeamID:   "attesta",
		EnvFilePath:     envPath,
		HTTPClient:      srv.Client(),
	})
	if err != nil {
		t.Fatalf("BootstrapAppwriteProject: %v", err)
	}
	if keyCreates != 0 || keyDeletes != 0 {
		t.Fatalf("keyCreates=%d keyDeletes=%d; want reuse without create/delete", keyCreates, keyDeletes)
	}
	if result.APIKey != "reused-secret" {
		t.Fatalf("result = %#v", result)
	}
	assertEnvFile(t, envPath, "attesta", "reused-secret")
}

func TestBootstrapAppwriteProjectRotatesBootstrapKeyWhenSecretMissing(t *testing.T) {
	var keyCreates int
	var keyDeletes int
	var deletedPath string

	srv := newBootstrapConsoleStub(t, bootstrapConsoleStubOpts{
		onLogin: func(w http.ResponseWriter, _ *http.Request) {
			http.SetCookie(w, &http.Cookie{Name: "a_session_console", Value: "sess", Path: "/"})
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"$id":"s","secret":"sess"}`))
		},
		// Typical Appwrite list response: named key present, secret omitted.
		keysJSON: `{"total":1,"keys":[{"$id":"key-old","name":"attesta-bootstrap"}]}`,
		onKeyCreate: func(w http.ResponseWriter, r *http.Request) {
			keyCreates++
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode key: %v", err)
			}
			if body["name"] != appwriteBootstrapKeyName {
				t.Fatalf("key body = %#v", body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"$id":"key-new","secret":"rotated-secret","name":"attesta-bootstrap"}`))
		},
		onKeyDelete: func(w http.ResponseWriter, r *http.Request) {
			keyDeletes++
			deletedPath = r.URL.Path
			w.WriteHeader(http.StatusNoContent)
		},
	})
	defer srv.Close()

	envPath := filepath.Join(t.TempDir(), "env")
	result, err := BootstrapAppwriteProject(t.Context(), BootstrapAppwriteDeps{
		Endpoint:        srv.URL + "/v1",
		ProjectID:       "attesta",
		ConsoleEmail:    "admin@appwrite.local",
		ConsolePassword: "admin@appwrite.local",
		OrgAssetsBucket: "org-assets",
		ConsoleTeamID:   "attesta",
		EnvFilePath:     envPath,
		HTTPClient:      srv.Client(),
	})
	if err != nil {
		t.Fatalf("BootstrapAppwriteProject: %v", err)
	}
	if keyDeletes != 1 || keyCreates != 1 {
		t.Fatalf("keyDeletes=%d keyCreates=%d; want delete+create once", keyDeletes, keyCreates)
	}
	if deletedPath != "/v1/projects/attesta/keys/key-old" {
		t.Fatalf("deletedPath = %q", deletedPath)
	}
	if result.APIKey != "rotated-secret" {
		t.Fatalf("result = %#v", result)
	}
	assertEnvFile(t, envPath, "attesta", "rotated-secret")
}

func TestBootstrapAppwriteProjectRetries429OnLoginAndKey(t *testing.T) {
	var loginAttempts int
	var keyListAttempts int
	var keyCreateAttempts int
	var sleeps []time.Duration
	var mu sync.Mutex

	srv := newBootstrapConsoleStub(t, bootstrapConsoleStubOpts{
		onLogin: func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			loginAttempts++
			n := loginAttempts
			mu.Unlock()
			if n == 1 {
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"message":"Rate limit","code":429}`))
				return
			}
			http.SetCookie(w, &http.Cookie{Name: "a_session_console", Value: "sess", Path: "/"})
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"$id":"s","secret":"sess"}`))
		},
		onKeysList: func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			keyListAttempts++
			n := keyListAttempts
			mu.Unlock()
			if n == 1 {
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"code":429}`))
				return
			}
			_, _ = w.Write([]byte(`{"total":0,"keys":[]}`))
		},
		onKeyCreate: func(w http.ResponseWriter, _ *http.Request) {
			mu.Lock()
			keyCreateAttempts++
			n := keyCreateAttempts
			mu.Unlock()
			if n == 1 {
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"code":429}`))
				return
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"$id":"k","secret":"after-429-key","name":"attesta-bootstrap"}`))
		},
	})
	defer srv.Close()

	envPath := filepath.Join(t.TempDir(), "env")
	result, err := BootstrapAppwriteProject(t.Context(), BootstrapAppwriteDeps{
		Endpoint:        srv.URL + "/v1",
		ProjectID:       "attesta",
		ConsoleEmail:    "admin@appwrite.local",
		ConsolePassword: "admin@appwrite.local",
		OrgAssetsBucket: "org-assets",
		ConsoleTeamID:   "attesta",
		EnvFilePath:     envPath,
		HTTPClient:      srv.Client(),
		Sleep: func(_ context.Context, d time.Duration) error {
			mu.Lock()
			sleeps = append(sleeps, d)
			mu.Unlock()
			return nil
		},
	})
	if err != nil {
		t.Fatalf("BootstrapAppwriteProject: %v", err)
	}
	if loginAttempts != 2 || keyListAttempts != 2 || keyCreateAttempts != 2 {
		t.Fatalf("login=%d keyList=%d keyCreate=%d; want 2 each (429 then success)", loginAttempts, keyListAttempts, keyCreateAttempts)
	}
	if len(sleeps) != 3 {
		t.Fatalf("sleeps = %#v, want 3 backoff sleeps", sleeps)
	}
	for _, d := range sleeps {
		if d != appwriteBootstrap429BaseDelay {
			t.Fatalf("sleep delay = %v, want base %v", d, appwriteBootstrap429BaseDelay)
		}
	}
	if result.APIKey != "after-429-key" {
		t.Fatalf("result = %#v", result)
	}
	assertEnvFile(t, envPath, "attesta", "after-429-key")
}

func TestBootstrapAppwriteProjectRequiresProjectID(t *testing.T) {
	_, err := BootstrapAppwriteProject(t.Context(), BootstrapAppwriteDeps{
		Endpoint: "http://example.invalid/v1",
	})
	if err == nil || !strings.Contains(err.Error(), "APPWRITE_PROJECT_ID") {
		t.Fatalf("error = %v, want APPWRITE_PROJECT_ID", err)
	}
}

func TestBootstrapDepsFromEnvDefaults(t *testing.T) {
	root := t.TempDir()
	t.Setenv("ATTESTA_ROOT_DIR", root)
	t.Setenv("APPWRITE_ENDPOINT", "http://appwrite/v1")
	t.Setenv("APPWRITE_PROJECT_ID", "attesta")
	t.Setenv("APPWRITE_API_KEY", "")
	t.Setenv("APPWRITE_CONSOLE_EMAIL", "")
	t.Setenv("APPWRITE_CONSOLE_PASSWORD", "")
	t.Setenv("APPWRITE_ORG_ASSETS_BUCKET", "")

	deps := bootstrapAppwriteDepsFromEnv()
	if deps.ConsoleEmail != defaultAppwriteConsoleEmail {
		t.Fatalf("email = %q", deps.ConsoleEmail)
	}
	if deps.ConsolePassword != defaultAppwriteConsolePassword {
		t.Fatalf("password = %q", deps.ConsolePassword)
	}
	if deps.OrgAssetsBucket != "org-assets" {
		t.Fatalf("bucket = %q", deps.OrgAssetsBucket)
	}
	wantEnv := filepath.Join(root, ".env.appwrite")
	if deps.EnvFilePath != wantEnv {
		t.Fatalf("env file = %q, want %q", deps.EnvFilePath, wantEnv)
	}
}

type bootstrapConsoleStubOpts struct {
	onAccount   func(http.ResponseWriter, *http.Request)
	onLogin     func(http.ResponseWriter, *http.Request)
	onKeysList  func(http.ResponseWriter, *http.Request)
	onKeyCreate func(http.ResponseWriter, *http.Request)
	onKeyDelete func(http.ResponseWriter, *http.Request)
	keysJSON    string
}

func newBootstrapConsoleStub(t *testing.T, opts bootstrapConsoleStubOpts) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		project := r.Header.Get("X-Appwrite-Project")
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/users":
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"code":401}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/account" && project == "console":
			if opts.onAccount == nil {
				t.Fatalf("unexpected signup: %s %s", r.Method, r.URL.Path)
			}
			opts.onAccount(w, r)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/account/sessions/email" && project == "console":
			if opts.onLogin == nil {
				t.Fatalf("missing onLogin")
			}
			opts.onLogin(w, r)
		case r.Method == http.MethodGet && r.URL.Path == "/v1/teams":
			_, _ = w.Write([]byte(`{"total":1,"teams":[{"$id":"attesta","name":"Attesta"}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/projects/attesta":
			_, _ = w.Write([]byte(`{"$id":"attesta","name":"Attesta","teamId":"attesta"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/projects/attesta/keys" && project == "console":
			if opts.onKeysList != nil {
				opts.onKeysList(w, r)
				return
			}
			keys := opts.keysJSON
			if keys == "" {
				keys = `{"total":0,"keys":[]}`
			}
			_, _ = w.Write([]byte(keys))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/projects/attesta/keys" && project == "console":
			if opts.onKeyCreate == nil {
				t.Fatalf("missing onKeyCreate")
			}
			opts.onKeyCreate(w, r)
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/v1/projects/attesta/keys/") && project == "console":
			if opts.onKeyDelete == nil {
				t.Fatalf("unexpected key delete: %s", r.URL.Path)
			}
			opts.onKeyDelete(w, r)
		case r.Method == http.MethodGet && r.URL.Path == "/v1/storage/buckets/org-assets":
			_, _ = w.Write([]byte(`{"$id":"org-assets"}`))
		default:
			t.Fatalf("unexpected: %s %s", r.Method, r.URL.Path)
		}
	}))
}

func TestTruncateForErr(t *testing.T) {
	t.Parallel()
	if got := truncateForErr(nil); got != "" {
		t.Fatalf("nil = %q", got)
	}
	if got := truncateForErr([]byte("  hi  ")); got != "hi" {
		t.Fatalf("trim = %q", got)
	}
	long := strings.Repeat("a", 300)
	got := truncateForErr([]byte(long))
	if !strings.HasSuffix(got, "…") || !strings.HasPrefix(got, long[:240]) {
		t.Fatalf("truncated = %q", got)
	}
	if len(got) != 240+len("…") {
		t.Fatalf("truncated byte len=%d, want %d", len(got), 240+len("…"))
	}
}

func assertEnvFile(t *testing.T, path, projectID, apiKey string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read env file: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "APPWRITE_PROJECT_ID="+projectID+"\n") {
		t.Fatalf("env file missing project: %s", text)
	}
	if !strings.Contains(text, "APPWRITE_API_KEY="+apiKey+"\n") {
		t.Fatalf("env file missing key: %s", text)
	}
}

func scopeListContains(scopes []any, want string) bool {
	for _, scope := range scopes {
		if s, ok := scope.(string); ok && s == want {
			return true
		}
	}
	return false
}
