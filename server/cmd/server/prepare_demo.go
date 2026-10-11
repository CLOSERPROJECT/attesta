package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultAppwriteReadyTimeoutSeconds = 120

// PrepareDeps is the injectable seam for demo-stack prepare orchestration.
// Host adapters choose SoftBootstrap / SoftCredentialSkip; shared glue lives here.
type PrepareDeps struct {
	// SeedEnabled overrides ATTESTA_SEED when non-nil.
	SeedEnabled *bool

	// Endpoint is the Appwrite API base (…/v1). HealthURL defaults to Endpoint+"/health".
	Endpoint  string
	HealthURL string

	// ReadyTimeoutSeconds polls /health; nil reads ATTESTA_APPWRITE_READY_TIMEOUT_SECONDS
	// (default 120). Zero skips the readiness probe.
	ReadyTimeoutSeconds *int

	EnvFilePath string

	// SoftBootstrap continues with a warning when bootstrap fails (worktree).
	// Coolify leaves this false so prepare hard-fails.
	SoftBootstrap bool
	// SoftCredentialSkip skips seed with a warning when APPWRITE_* credentials
	// are missing after bootstrap (worktree). Coolify leaves this false.
	SoftCredentialSkip bool

	HTTPClient *http.Client

	// WaitAppwriteReady overrides the default health wait (tests).
	WaitAppwriteReady func(ctx context.Context, healthURL string, timeoutSec int) error

	// Bootstrap overrides BootstrapAppwriteProject (tests).
	Bootstrap func(ctx context.Context, deps BootstrapAppwriteDeps) (BootstrapResult, error)

	// Seed runs after env is applied. When nil, PrepareDemoStack returns
	// after bootstrap/env so the CLI can open a seed session with fresh env.
	Seed func(ctx context.Context) error

	// BootstrapDeps supplies bootstrap inputs; empty fields filled from env.
	BootstrapDeps BootstrapAppwriteDeps
}

// PrepareDemoStack gates on ATTESTA_SEED, then wait → bootstrap → apply env → seed.
// When seed is disabled it is a no-op success.
func PrepareDemoStack(ctx context.Context, deps PrepareDeps) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !prepareSeedEnabled(deps) {
		return nil
	}

	deps = normalizePrepareDeps(deps)
	timeoutSec := prepareReadyTimeout(deps)

	if timeoutSec != 0 {
		waitFn := deps.WaitAppwriteReady
		if waitFn == nil {
			client := deps.HTTPClient
			if client == nil {
				client = http.DefaultClient
			}
			waitFn = func(ctx context.Context, healthURL string, timeoutSec int) error {
				return waitAppwriteReady(ctx, client, healthURL, timeoutSec)
			}
		}
		if err := waitFn(ctx, deps.HealthURL, timeoutSec); err != nil {
			return err
		}
	}

	bootstrapFn := deps.Bootstrap
	if bootstrapFn == nil {
		bootstrapFn = BootstrapAppwriteProject
	}
	bootstrapDeps := mergeBootstrapDeps(deps.BootstrapDeps, deps.EnvFilePath)
	result, err := bootstrapFn(ctx, bootstrapDeps)
	if err != nil {
		if deps.SoftBootstrap {
			log.Printf("warning: attesta bootstrap-appwrite failed; continuing with existing APPWRITE_* credentials: %v", err)
		} else {
			return fmt.Errorf("prepare: bootstrap-appwrite: %w", err)
		}
	} else {
		_ = os.Setenv("APPWRITE_PROJECT_ID", result.ProjectID)
		_ = os.Setenv("APPWRITE_API_KEY", result.APIKey)
	}

	if err := applyAppwriteRuntimeEnvFile(deps.EnvFilePath); err != nil {
		return fmt.Errorf("prepare: apply env file: %w", err)
	}

	if missing := missingAppwriteCredentials(); len(missing) > 0 {
		msg := "prepare: missing " + strings.Join(missing, ", ")
		if deps.SoftCredentialSkip {
			log.Printf("skipping attesta seed: %s (or run task seed later)", msg)
			return nil
		}
		return fmt.Errorf("%s", msg)
	}

	if deps.Seed == nil {
		return nil
	}
	if err := deps.Seed(ctx); err != nil {
		return fmt.Errorf("prepare: seed: %w", err)
	}
	return nil
}

func prepareSeedEnabled(deps PrepareDeps) bool {
	if deps.SeedEnabled != nil {
		return *deps.SeedEnabled
	}
	return attestaSeedEnabled(os.Getenv("ATTESTA_SEED"))
}

func attestaSeedEnabled(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func normalizePrepareDeps(deps PrepareDeps) PrepareDeps {
	deps.Endpoint = strings.TrimRight(strings.TrimSpace(deps.Endpoint), "/")
	deps.HealthURL = strings.TrimSpace(deps.HealthURL)
	deps.EnvFilePath = strings.TrimSpace(deps.EnvFilePath)
	if deps.Endpoint == "" {
		deps.Endpoint = strings.TrimRight(strings.TrimSpace(envOr("APPWRITE_ENDPOINT", "http://appwrite/v1")), "/")
	}
	if deps.HealthURL == "" {
		deps.HealthURL = deps.Endpoint + "/health"
	}
	if deps.EnvFilePath == "" {
		deps.EnvFilePath = envOr("ATTESTA_APPWRITE_ENV_FILE", defaultAppwriteBootstrapEnvFile)
	}
	return deps
}

func prepareReadyTimeout(deps PrepareDeps) int {
	if deps.ReadyTimeoutSeconds != nil {
		return *deps.ReadyTimeoutSeconds
	}
	raw := strings.TrimSpace(os.Getenv("ATTESTA_APPWRITE_READY_TIMEOUT_SECONDS"))
	if raw == "" {
		return defaultAppwriteReadyTimeoutSeconds
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return defaultAppwriteReadyTimeoutSeconds
	}
	return n
}

func mergeBootstrapDeps(deps BootstrapAppwriteDeps, envFilePath string) BootstrapAppwriteDeps {
	fromEnv := bootstrapAppwriteDepsFromEnv()
	if strings.TrimSpace(deps.Endpoint) == "" {
		deps.Endpoint = fromEnv.Endpoint
	}
	if strings.TrimSpace(deps.ProjectID) == "" {
		deps.ProjectID = fromEnv.ProjectID
	}
	if strings.TrimSpace(deps.APIKey) == "" {
		deps.APIKey = fromEnv.APIKey
	}
	if strings.TrimSpace(deps.ConsoleEmail) == "" {
		deps.ConsoleEmail = fromEnv.ConsoleEmail
	}
	if strings.TrimSpace(deps.ConsolePassword) == "" {
		deps.ConsolePassword = fromEnv.ConsolePassword
	}
	if strings.TrimSpace(deps.OrgAssetsBucket) == "" {
		deps.OrgAssetsBucket = fromEnv.OrgAssetsBucket
	}
	if strings.TrimSpace(deps.ConsoleTeamID) == "" {
		deps.ConsoleTeamID = fromEnv.ConsoleTeamID
	}
	if strings.TrimSpace(deps.EnvFilePath) == "" {
		deps.EnvFilePath = envFilePath
	}
	if strings.TrimSpace(deps.EnvFilePath) == "" {
		deps.EnvFilePath = fromEnv.EnvFilePath
	}
	return deps
}

func missingAppwriteCredentials() []string {
	var missing []string
	if strings.TrimSpace(os.Getenv("APPWRITE_ENDPOINT")) == "" {
		missing = append(missing, "APPWRITE_ENDPOINT")
	}
	if strings.TrimSpace(os.Getenv("APPWRITE_PROJECT_ID")) == "" {
		missing = append(missing, "APPWRITE_PROJECT_ID")
	}
	if strings.TrimSpace(os.Getenv("APPWRITE_API_KEY")) == "" {
		missing = append(missing, "APPWRITE_API_KEY")
	}
	return missing
}

// applyAppwriteRuntimeEnvFile loads KEY=VALUE pairs from path into the process env.
// Missing files are ignored (bootstrap soft-skip may leave no file).
func applyAppwriteRuntimeEnvFile(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}
	return scanner.Err()
}

// waitAppwriteReady polls healthURL until HTTP 200 or 401, or timeoutSec elapses.
// timeoutSec 0 is a no-op success (callers should skip invoking this).
func waitAppwriteReady(ctx context.Context, client *http.Client, healthURL string, timeoutSec int) error {
	healthURL = strings.TrimSpace(healthURL)
	if healthURL == "" {
		return fmt.Errorf("prepare: wait for Appwrite requires a health URL")
	}
	if timeoutSec < 0 {
		return fmt.Errorf("prepare: ATTESTA_APPWRITE_READY_TIMEOUT_SECONDS must be a non-negative integer")
	}
	if timeoutSec == 0 {
		return nil
	}
	if client == nil {
		client = http.DefaultClient
	}

	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	log.Printf("waiting for Appwrite at %s", healthURL)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("prepare: Appwrite did not become ready at %s within %ds", healthURL, timeoutSec)
		}
		code, err := probeAppwriteHealth(ctx, client, healthURL)
		if err == nil && (code == http.StatusOK || code == http.StatusUnauthorized) {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

func probeAppwriteHealth(ctx context.Context, client *http.Client, healthURL string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		return 0, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode, nil
}
