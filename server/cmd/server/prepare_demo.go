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

// Prepare host policies (ADR-0004). Empty Policy means coolify.
const (
	PreparePolicyCoolify  = "coolify"
	PreparePolicyWorktree = "worktree"
)

// PrepareDeps is the injectable seam for demo-stack prepare orchestration.
// Host adapters set Policy; shared glue lives here.
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

	// Policy selects soft vs hard failure. Empty or "coolify" hard-fails bootstrap
	// and missing credentials. "worktree" soft-skips both with warnings.
	Policy string

	HTTPClient *http.Client

	// Seed receives Appwrite open params by value. When nil, PrepareDemoStack
	// returns after bootstrap (env file may still be written for Server restart).
	Seed func(ctx context.Context, creds appwriteRuntimeCredentials) error

	// BootstrapDeps supplies bootstrap inputs; empty fields filled from env.
	BootstrapDeps BootstrapAppwriteDeps
}

// prepareHooks are unexported wait/bootstrap doubles for same-package tests.
// Production uses defaultPrepareHooks; Seed stays on PrepareDeps.
type prepareHooks struct {
	Wait      func(ctx context.Context, healthURL string, timeoutSec int) error
	Bootstrap func(ctx context.Context, deps BootstrapAppwriteDeps) (BootstrapResult, error)
}

func defaultPrepareHooks(client *http.Client) prepareHooks {
	if client == nil {
		client = http.DefaultClient
	}
	return prepareHooks{
		Wait: func(ctx context.Context, healthURL string, timeoutSec int) error {
			return waitAppwriteReady(ctx, client, healthURL, timeoutSec)
		},
		Bootstrap: BootstrapAppwriteProject,
	}
}

// PrepareDemoStack gates on ATTESTA_SEED, then wait → bootstrap → seed.
// Bootstrap may still write the on-disk runtime env file for Server startup;
// prepare→seed does not mutate process env or apply that file in-process.
// When seed is disabled it is a no-op success.
func PrepareDemoStack(ctx context.Context, deps PrepareDeps) error {
	return prepareDemoStack(ctx, deps, defaultPrepareHooks(deps.HTTPClient))
}

func prepareDemoStack(ctx context.Context, deps PrepareDeps, hooks prepareHooks) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !prepareSeedEnabled(deps) {
		return nil
	}

	deps = normalizePrepareDeps(deps)
	timeoutSec := prepareReadyTimeout(deps)

	if timeoutSec != 0 {
		waitFn := hooks.Wait
		if waitFn == nil {
			waitFn = defaultPrepareHooks(deps.HTTPClient).Wait
		}
		if err := waitFn(ctx, deps.HealthURL, timeoutSec); err != nil {
			return err
		}
	}

	soft, err := preparePolicyIsSoft(deps.Policy)
	if err != nil {
		return err
	}

	bootstrapFn := hooks.Bootstrap
	if bootstrapFn == nil {
		bootstrapFn = BootstrapAppwriteProject
	}
	bootstrapDeps := mergeBootstrapDeps(deps.BootstrapDeps, deps.EnvFilePath)
	result, bootstrapErr := bootstrapFn(ctx, bootstrapDeps)

	fromBootstrap := runtimeCredsFromBootstrap(deps.Endpoint, result)
	ambient := ambientAppwriteRuntimeCredentials()
	creds, usedAmbient, err := prepareCredentialsAfterBootstrap(soft, bootstrapErr, fromBootstrap, ambient)
	if err != nil {
		return err
	}
	if usedAmbient {
		log.Printf("warning: attesta bootstrap failed; continuing with existing APPWRITE_* credentials: %v", bootstrapErr)
	}

	skip, err := prepareSeedCredentialGate(soft, creds)
	if err != nil {
		return err
	}
	if skip {
		missing := missingAppwriteCredentials(creds)
		log.Printf("skipping attesta seed: prepare: missing %s (or run task seed later)", strings.Join(missing, ", "))
		return nil
	}

	if deps.Seed == nil {
		return nil
	}
	if err := deps.Seed(ctx, creds); err != nil {
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

func normalizePreparePolicy(policy string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(policy)) {
	case "", PreparePolicyCoolify:
		return PreparePolicyCoolify, nil
	case PreparePolicyWorktree:
		return PreparePolicyWorktree, nil
	default:
		return "", fmt.Errorf("prepare: unknown policy %q (want coolify or worktree)", policy)
	}
}

func preparePolicyIsSoft(policy string) (bool, error) {
	normalized, err := normalizePreparePolicy(policy)
	if err != nil {
		return false, err
	}
	return normalized == PreparePolicyWorktree, nil
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

// runtimeCredsFromBootstrap maps a successful BootstrapResult onto open params.
// Endpoint comes from prepare (bootstrap does not return it).
func runtimeCredsFromBootstrap(endpoint string, result BootstrapResult) appwriteRuntimeCredentials {
	return appwriteRuntimeCredentials{
		Endpoint:  endpoint,
		ProjectID: result.ProjectID,
		APIKey:    result.APIKey,
	}
}

// ambientAppwriteRuntimeCredentials reads current APPWRITE_* process env.
func ambientAppwriteRuntimeCredentials() appwriteRuntimeCredentials {
	return appwriteRuntimeCredentials{
		Endpoint:  os.Getenv("APPWRITE_ENDPOINT"),
		ProjectID: os.Getenv("APPWRITE_PROJECT_ID"),
		APIKey:    os.Getenv("APPWRITE_API_KEY"),
	}
}

// prepareCredentialsAfterBootstrap picks runtime creds after a bootstrap attempt.
// Soft policy falls back to ambient on bootstrap error; hard policy returns the error.
// usedAmbient is true when soft fallback was taken (caller may log).
func prepareCredentialsAfterBootstrap(
	soft bool,
	bootstrapErr error,
	fromBootstrap, ambient appwriteRuntimeCredentials,
) (creds appwriteRuntimeCredentials, usedAmbient bool, err error) {
	if bootstrapErr != nil {
		if !soft {
			return appwriteRuntimeCredentials{}, false, fmt.Errorf("prepare: bootstrap: %w", bootstrapErr)
		}
		return ambient, true, nil
	}
	return fromBootstrap, false, nil
}

// prepareSeedCredentialGate checks required open params under soft vs hard policy.
// Soft + missing → skip seed (no error). Hard + missing → error.
func prepareSeedCredentialGate(soft bool, creds appwriteRuntimeCredentials) (skip bool, err error) {
	missing := missingAppwriteCredentials(creds)
	if len(missing) == 0 {
		return false, nil
	}
	msg := "prepare: missing " + strings.Join(missing, ", ")
	if soft {
		return true, nil
	}
	return false, fmt.Errorf("%s", msg)
}

func missingAppwriteCredentials(creds appwriteRuntimeCredentials) []string {
	var missing []string
	if strings.TrimSpace(creds.Endpoint) == "" {
		missing = append(missing, "APPWRITE_ENDPOINT")
	}
	if strings.TrimSpace(creds.ProjectID) == "" {
		missing = append(missing, "APPWRITE_PROJECT_ID")
	}
	if strings.TrimSpace(creds.APIKey) == "" {
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
