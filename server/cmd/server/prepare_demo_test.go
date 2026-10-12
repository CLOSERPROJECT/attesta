package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestAttestaSeedEnabledTruthy(t *testing.T) {
	t.Parallel()
	for _, v := range []string{"1", "true", "TRUE", "yes", "YES", "on", "ON"} {
		if !attestaSeedEnabled(v) {
			t.Fatalf("attestaSeedEnabled(%q) = false, want true", v)
		}
	}
	for _, v := range []string{"", "0", "false", "no", "off", "maybe"} {
		if attestaSeedEnabled(v) {
			t.Fatalf("attestaSeedEnabled(%q) = true, want false", v)
		}
	}
}

func TestPrepareDemoStackSeedDisabledNoop(t *testing.T) {
	var calls []string
	err := PrepareDemoStack(t.Context(), PrepareDeps{
		SeedEnabled: boolPtr(false),
		WaitAppwriteReady: func(context.Context, string, int) error {
			calls = append(calls, "wait")
			return nil
		},
		Bootstrap: func(context.Context, BootstrapAppwriteDeps) (BootstrapResult, error) {
			calls = append(calls, "bootstrap")
			return BootstrapResult{}, nil
		},
		Seed: func(context.Context, appwriteRuntimeCredentials) error {
			calls = append(calls, "seed")
			return nil
		},
	})
	if err != nil {
		t.Fatalf("PrepareDemoStack: %v", err)
	}
	if len(calls) != 0 {
		t.Fatalf("calls = %#v, want no-op", calls)
	}
}

func TestPrepareDemoStackOrderWaitBootstrapSeed(t *testing.T) {
	t.Setenv("APPWRITE_ENDPOINT", "http://appwrite/v1")

	var mu sync.Mutex
	var calls []string
	envPath := filepath.Join(t.TempDir(), "attesta-appwrite.env")

	err := PrepareDemoStack(t.Context(), PrepareDeps{
		SeedEnabled:         boolPtr(true),
		Endpoint:            "http://appwrite/v1",
		ReadyTimeoutSeconds: intPtr(30),
		EnvFilePath:         envPath,
		WaitAppwriteReady: func(_ context.Context, healthURL string, timeoutSec int) error {
			mu.Lock()
			defer mu.Unlock()
			calls = append(calls, "wait")
			if healthURL != "http://appwrite/v1/health" {
				t.Fatalf("healthURL = %q", healthURL)
			}
			if timeoutSec != 30 {
				t.Fatalf("timeoutSec = %d", timeoutSec)
			}
			return nil
		},
		Bootstrap: func(_ context.Context, deps BootstrapAppwriteDeps) (BootstrapResult, error) {
			mu.Lock()
			defer mu.Unlock()
			calls = append(calls, "bootstrap")
			if deps.EnvFilePath != envPath {
				t.Fatalf("EnvFilePath = %q", deps.EnvFilePath)
			}
			if err := os.WriteFile(envPath, []byte("APPWRITE_PROJECT_ID=attesta\nAPPWRITE_API_KEY=from-bootstrap\n"), 0o600); err != nil {
				t.Fatalf("write env: %v", err)
			}
			return BootstrapResult{ProjectID: "attesta", APIKey: "from-bootstrap", Created: true}, nil
		},
		Seed: func(_ context.Context, creds appwriteRuntimeCredentials) error {
			mu.Lock()
			defer mu.Unlock()
			calls = append(calls, "seed")
			if creds.Endpoint != "http://appwrite/v1" || creds.ProjectID != "attesta" || creds.APIKey != "from-bootstrap" {
				t.Fatalf("seed creds = %#v", creds)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatalf("PrepareDemoStack: %v", err)
	}
	if got := strings.Join(calls, ","); got != "wait,bootstrap,seed" {
		t.Fatalf("order = %q, want wait,bootstrap,seed", got)
	}
}

func TestParsePrepareArgsPolicy(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		args    []string
		policy  string
		rest    []string
		wantErr string
	}{
		{name: "default coolify", args: nil, policy: PreparePolicyCoolify},
		{name: "equals coolify", args: []string{"--policy=coolify"}, policy: PreparePolicyCoolify},
		{name: "equals worktree", args: []string{"--policy=worktree"}, policy: PreparePolicyWorktree},
		{name: "space worktree", args: []string{"--policy", "worktree"}, policy: PreparePolicyWorktree},
		{name: "unknown policy", args: []string{"--policy=staging"}, wantErr: "unknown policy"},
		{name: "missing value", args: []string{"--policy"}, wantErr: "--policy requires"},
		{name: "retired soft flag", args: []string{"--soft-bootstrap"}, wantErr: "unknown flag"},
		{name: "help", args: []string{"--help"}, wantErr: "usage:"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			policy, rest, err := parsePrepareArgs(tc.args)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parsePrepareArgs: %v", err)
			}
			if policy != tc.policy {
				t.Fatalf("policy = %q, want %q", policy, tc.policy)
			}
			if len(rest) != len(tc.rest) {
				t.Fatalf("rest = %#v, want %#v", rest, tc.rest)
			}
		})
	}
}

func TestPrepareDemoStackWorktreePolicySoftBootstrapContinues(t *testing.T) {
	t.Setenv("APPWRITE_ENDPOINT", "http://appwrite/v1")
	t.Setenv("APPWRITE_PROJECT_ID", "attesta")
	t.Setenv("APPWRITE_API_KEY", "existing-key")

	var calls []string
	err := PrepareDemoStack(t.Context(), PrepareDeps{
		SeedEnabled:         boolPtr(true),
		ReadyTimeoutSeconds: intPtr(30),
		Policy:              PreparePolicyWorktree,
		Endpoint:            "http://appwrite/v1",
		WaitAppwriteReady: func(context.Context, string, int) error {
			calls = append(calls, "wait")
			return nil
		},
		Bootstrap: func(context.Context, BootstrapAppwriteDeps) (BootstrapResult, error) {
			calls = append(calls, "bootstrap")
			return BootstrapResult{}, errors.New("console unavailable")
		},
		Seed: func(_ context.Context, creds appwriteRuntimeCredentials) error {
			calls = append(calls, "seed")
			if creds.Endpoint != "http://appwrite/v1" || creds.ProjectID != "attesta" || creds.APIKey != "existing-key" {
				t.Fatalf("seed creds after soft bootstrap = %#v", creds)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatalf("PrepareDemoStack: %v", err)
	}
	if got := strings.Join(calls, ","); got != "wait,bootstrap,seed" {
		t.Fatalf("order = %q, want wait,bootstrap,seed after soft bootstrap", got)
	}
}

func TestPrepareDemoStackCoolifyPolicyHardBootstrapFails(t *testing.T) {
	var calls []string
	err := PrepareDemoStack(t.Context(), PrepareDeps{
		SeedEnabled:         boolPtr(true),
		ReadyTimeoutSeconds: intPtr(30),
		Policy:              PreparePolicyCoolify,
		WaitAppwriteReady: func(context.Context, string, int) error {
			calls = append(calls, "wait")
			return nil
		},
		Bootstrap: func(context.Context, BootstrapAppwriteDeps) (BootstrapResult, error) {
			calls = append(calls, "bootstrap")
			return BootstrapResult{}, errors.New("console unavailable")
		},
		Seed: func(context.Context, appwriteRuntimeCredentials) error {
			calls = append(calls, "seed")
			return nil
		},
	})
	if err == nil {
		t.Fatal("PrepareDemoStack: expected error")
	}
	if !strings.Contains(err.Error(), "console unavailable") {
		t.Fatalf("error = %v", err)
	}
	if got := strings.Join(calls, ","); got != "wait,bootstrap" {
		t.Fatalf("calls = %q, want wait,bootstrap without seed", got)
	}
}

func TestPrepareDemoStackWorktreePolicySkipsMissingCredentials(t *testing.T) {
	t.Setenv("APPWRITE_ENDPOINT", "http://appwrite/v1")
	t.Setenv("APPWRITE_PROJECT_ID", "")
	t.Setenv("APPWRITE_API_KEY", "")

	var seeded bool
	err := PrepareDemoStack(t.Context(), PrepareDeps{
		SeedEnabled:         boolPtr(true),
		ReadyTimeoutSeconds: intPtr(0),
		Policy:              PreparePolicyWorktree,
		Endpoint:            "http://appwrite/v1",
		Bootstrap: func(context.Context, BootstrapAppwriteDeps) (BootstrapResult, error) {
			return BootstrapResult{}, errors.New("console unavailable")
		},
		Seed: func(context.Context, appwriteRuntimeCredentials) error {
			seeded = true
			return nil
		},
	})
	if err != nil {
		t.Fatalf("PrepareDemoStack: %v", err)
	}
	if seeded {
		t.Fatal("seed ran despite missing credentials under worktree policy")
	}
}

func TestPrepareDemoStackTimeoutZeroSkipsWait(t *testing.T) {
	t.Setenv("APPWRITE_ENDPOINT", "http://127.0.0.1:9/v1")

	var waited bool
	var seeded bool
	err := PrepareDemoStack(t.Context(), PrepareDeps{
		SeedEnabled:         boolPtr(true),
		ReadyTimeoutSeconds: intPtr(0),
		Endpoint:            "http://127.0.0.1:9/v1",
		WaitAppwriteReady: func(context.Context, string, int) error {
			waited = true
			return errors.New("wait should not run when timeout is 0")
		},
		Bootstrap: func(context.Context, BootstrapAppwriteDeps) (BootstrapResult, error) {
			return BootstrapResult{ProjectID: "attesta", APIKey: "k"}, nil
		},
		Seed: func(_ context.Context, creds appwriteRuntimeCredentials) error {
			if creds.ProjectID != "attesta" || creds.APIKey != "k" {
				t.Fatalf("seed creds = %#v", creds)
			}
			seeded = true
			return nil
		},
	})
	if err != nil {
		t.Fatalf("PrepareDemoStack: %v", err)
	}
	if waited {
		t.Fatal("WaitAppwriteReady was called despite timeout 0")
	}
	if !seeded {
		t.Fatal("seed did not run")
	}
}

func TestPrepareDemoStackPassesBootstrapCredsWithoutMutatingEnv(t *testing.T) {
	envPath := filepath.Join(t.TempDir(), "attesta-appwrite.env")
	t.Setenv("APPWRITE_API_KEY", "stale-key")
	t.Setenv("APPWRITE_PROJECT_ID", "attesta")
	t.Setenv("APPWRITE_ENDPOINT", "http://appwrite/v1")

	var seedCreds appwriteRuntimeCredentials
	err := PrepareDemoStack(t.Context(), PrepareDeps{
		SeedEnabled:         boolPtr(true),
		ReadyTimeoutSeconds: intPtr(0),
		EnvFilePath:         envPath,
		Endpoint:            "http://appwrite/v1",
		Bootstrap: func(context.Context, BootstrapAppwriteDeps) (BootstrapResult, error) {
			if err := os.WriteFile(envPath, []byte("APPWRITE_PROJECT_ID=attesta\nAPPWRITE_API_KEY=fresh-key\n"), 0o600); err != nil {
				return BootstrapResult{}, err
			}
			return BootstrapResult{ProjectID: "attesta", APIKey: "fresh-key", Created: true}, nil
		},
		Seed: func(_ context.Context, creds appwriteRuntimeCredentials) error {
			seedCreds = creds
			return nil
		},
	})
	if err != nil {
		t.Fatalf("PrepareDemoStack: %v", err)
	}
	if seedCreds.Endpoint != "http://appwrite/v1" || seedCreds.ProjectID != "attesta" || seedCreds.APIKey != "fresh-key" {
		t.Fatalf("seed creds = %#v", seedCreds)
	}
	if got := os.Getenv("APPWRITE_API_KEY"); got != "stale-key" {
		t.Fatalf("process env APPWRITE_API_KEY=%q after prepare, want stale-key (unmutated)", got)
	}
}

func TestWaitAppwriteReadyAccepts200And401(t *testing.T) {
	t.Parallel()
	for _, code := range []int{http.StatusOK, http.StatusUnauthorized} {
		code := code
		t.Run(http.StatusText(code), func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/health" {
					t.Fatalf("path = %q", r.URL.Path)
				}
				w.WriteHeader(code)
			}))
			t.Cleanup(srv.Close)
			err := waitAppwriteReady(t.Context(), srv.Client(), srv.URL+"/v1/health", 5)
			if err != nil {
				t.Fatalf("waitAppwriteReady: %v", err)
			}
		})
	}
}

func TestWaitAppwriteReadyTimesOut(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	err := waitAppwriteReady(ctx, srv.Client(), srv.URL+"/v1/health", 1)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestApplyAppwriteRuntimeEnvFile(t *testing.T) {
	t.Setenv("APPWRITE_API_KEY", "stale")
	t.Setenv("APPWRITE_PROJECT_ID", "stale-project")

	if err := applyAppwriteRuntimeEnvFile(filepath.Join(t.TempDir(), "missing.env")); err != nil {
		t.Fatalf("missing file: %v", err)
	}
	if got := os.Getenv("APPWRITE_API_KEY"); got != "stale" {
		t.Fatalf("missing file mutated APPWRITE_API_KEY=%q", got)
	}

	path := filepath.Join(t.TempDir(), "attesta-appwrite.env")
	content := "# comment\n\nAPPWRITE_PROJECT_ID=attesta\nAPPWRITE_API_KEY=fresh\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := applyAppwriteRuntimeEnvFile(path); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if got := os.Getenv("APPWRITE_API_KEY"); got != "fresh" {
		t.Fatalf("APPWRITE_API_KEY=%q, want fresh", got)
	}
	if got := os.Getenv("APPWRITE_PROJECT_ID"); got != "attesta" {
		t.Fatalf("APPWRITE_PROJECT_ID=%q, want attesta", got)
	}
}

func boolPtr(v bool) *bool { return &v }
func intPtr(v int) *int    { return &v }
