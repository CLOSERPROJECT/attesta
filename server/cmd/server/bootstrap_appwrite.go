package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultAppwriteConsoleEmail     = "console@attesta.local"
	defaultAppwriteConsolePassword  = "password"
	defaultAppwriteBootstrapEnvFile = "/tmp/attesta-appwrite.env"
	defaultAppwriteConsoleTeamID    = "attesta"
	appwriteBootstrapKeyName        = "attesta-bootstrap"

	// Bound 429 retries to a few attempts / ~30–60s total wait.
	appwriteBootstrap429MaxAttempts = 6
	appwriteBootstrap429BaseDelay   = 500 * time.Millisecond
	appwriteBootstrap429MaxDelay    = 16 * time.Second
)

// appwriteBootstrapKeyScopes covers identity_appwrite admin + org-assets storage.
// Names must match Appwrite's project-key allowlist (no sessions.read in 1.8.x).
var appwriteBootstrapKeyScopes = []string{
	"users.read",
	"users.write",
	"teams.read",
	"teams.write",
	"sessions.write",
	"files.read",
	"files.write",
	"buckets.read",
	"buckets.write",
}

// BootstrapAppwriteDeps is the injectable seam for console project bootstrap.
type BootstrapAppwriteDeps struct {
	Endpoint        string
	ProjectID       string
	APIKey          string
	ConsoleEmail    string
	ConsolePassword string
	OrgAssetsBucket string
	ConsoleTeamID   string
	ProjectName     string
	EnvFilePath     string
	HTTPClient      *http.Client
	// Sleep backs off on HTTP 429. Nil uses a context-aware time.Timer sleep.
	Sleep func(ctx context.Context, d time.Duration) error
}

// BootstrapResult reports credentials after probe or console bootstrap.
type BootstrapResult struct {
	ProjectID string
	APIKey    string
	Skipped   bool
	Created   bool
}

// BootstrapAppwriteProject ensures the Attesta Appwrite project, API key, and
// org-assets bucket exist. When the current APPWRITE_API_KEY can list users,
// bootstrap is a no-op (aside from ensuring the bucket and writing EnvFilePath).
// Otherwise it uses the console project (X-Appwrite-Project: console) to create
// the console account/session/team/project/key, then the storage bucket.
//
// EnvFilePath (default /tmp/attesta-appwrite.env) always receives the runtime
// credentials so Coolify Preview can overwrite a stale injected API key.
func BootstrapAppwriteProject(ctx context.Context, deps BootstrapAppwriteDeps) (BootstrapResult, error) {
	if err := ctx.Err(); err != nil {
		return BootstrapResult{}, err
	}
	deps = normalizeBootstrapAppwriteDeps(deps)
	if deps.ProjectID == "" {
		return BootstrapResult{}, fmt.Errorf("APPWRITE_PROJECT_ID is required")
	}
	if deps.Endpoint == "" {
		return BootstrapResult{}, fmt.Errorf("APPWRITE_ENDPOINT is required")
	}

	client := deps.HTTPClient
	if client == nil {
		jar, err := cookiejar.New(nil)
		if err != nil {
			return BootstrapResult{}, fmt.Errorf("appwrite bootstrap cookie jar: %w", err)
		}
		client = &http.Client{Jar: jar}
	} else if client.Jar == nil {
		cloned := *client
		jar, err := cookiejar.New(nil)
		if err != nil {
			return BootstrapResult{}, fmt.Errorf("appwrite bootstrap cookie jar: %w", err)
		}
		cloned.Jar = jar
		client = &cloned
	}

	apiKey := deps.APIKey
	skipped := false
	created := false
	sleep := resolveBootstrapSleep(deps.Sleep)

	if apiKey != "" && probeAppwriteAdmin(ctx, client, sleep, deps.Endpoint, deps.ProjectID, apiKey) {
		skipped = true
	} else {
		secret, err := bootstrapAppwriteViaConsole(ctx, client, sleep, deps)
		if err != nil {
			return BootstrapResult{}, err
		}
		apiKey = secret
		created = true
	}

	if err := ensureAppwriteBucket(ctx, client, sleep, deps.Endpoint, deps.ProjectID, apiKey, deps.OrgAssetsBucket); err != nil {
		return BootstrapResult{}, err
	}
	if err := writeAppwriteBootstrapEnvFile(deps.EnvFilePath, deps.ProjectID, apiKey); err != nil {
		return BootstrapResult{}, err
	}
	return BootstrapResult{
		ProjectID: deps.ProjectID,
		APIKey:    apiKey,
		Skipped:   skipped,
		Created:   created,
	}, nil
}

func normalizeBootstrapAppwriteDeps(deps BootstrapAppwriteDeps) BootstrapAppwriteDeps {
	deps.Endpoint = strings.TrimRight(strings.TrimSpace(deps.Endpoint), "/")
	deps.ProjectID = strings.TrimSpace(deps.ProjectID)
	deps.APIKey = strings.TrimSpace(deps.APIKey)
	deps.ConsoleEmail = strings.TrimSpace(deps.ConsoleEmail)
	deps.ConsolePassword = strings.TrimSpace(deps.ConsolePassword)
	deps.OrgAssetsBucket = strings.TrimSpace(strings.ToLower(deps.OrgAssetsBucket))
	deps.ConsoleTeamID = strings.TrimSpace(deps.ConsoleTeamID)
	deps.ProjectName = strings.TrimSpace(deps.ProjectName)
	deps.EnvFilePath = strings.TrimSpace(deps.EnvFilePath)
	if deps.OrgAssetsBucket == "" {
		deps.OrgAssetsBucket = "org-assets"
	}
	if deps.ConsoleTeamID == "" {
		deps.ConsoleTeamID = defaultAppwriteConsoleTeamID
	}
	if deps.ProjectName == "" {
		deps.ProjectName = deps.ProjectID
	}
	if deps.EnvFilePath == "" {
		deps.EnvFilePath = defaultAppwriteBootstrapEnvFile
	}
	return deps
}

func resolveBootstrapSleep(sleep func(context.Context, time.Duration) error) func(context.Context, time.Duration) error {
	if sleep != nil {
		return sleep
	}
	return func(ctx context.Context, d time.Duration) error {
		if d <= 0 {
			return ctx.Err()
		}
		timer := time.NewTimer(d)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return nil
		}
	}
}

func bootstrapAppwriteDepsFromEnv() BootstrapAppwriteDeps {
	return BootstrapAppwriteDeps{
		Endpoint:        envOr("APPWRITE_ENDPOINT", "http://appwrite/v1"),
		ProjectID:       strings.TrimSpace(os.Getenv("APPWRITE_PROJECT_ID")),
		APIKey:          strings.TrimSpace(os.Getenv("APPWRITE_API_KEY")),
		ConsoleEmail:    envOr("APPWRITE_CONSOLE_EMAIL", defaultAppwriteConsoleEmail),
		ConsolePassword: envOr("APPWRITE_CONSOLE_PASSWORD", defaultAppwriteConsolePassword),
		OrgAssetsBucket: envOr("APPWRITE_ORG_ASSETS_BUCKET", "org-assets"),
		ConsoleTeamID:   envOr("APPWRITE_CONSOLE_TEAM_ID", defaultAppwriteConsoleTeamID),
		EnvFilePath:     envOr("ATTESTA_APPWRITE_ENV_FILE", defaultAppwriteBootstrapEnvFile),
	}
}

func probeAppwriteAdmin(ctx context.Context, client *http.Client, sleep func(context.Context, time.Duration) error, endpoint, projectID, apiKey string) bool {
	resp, err := appwriteJSONRequest(ctx, client, sleep, http.MethodGet, endpoint+"/users", projectID, apiKey, nil)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func bootstrapAppwriteViaConsole(ctx context.Context, client *http.Client, sleep func(context.Context, time.Duration) error, deps BootstrapAppwriteDeps) (string, error) {
	if deps.ConsoleEmail == "" || deps.ConsolePassword == "" {
		return "", fmt.Errorf("appwrite bootstrap: APPWRITE_CONSOLE_EMAIL and APPWRITE_CONSOLE_PASSWORD are required when admin probe fails")
	}
	if err := ensureConsoleLoggedIn(ctx, client, sleep, deps.Endpoint, deps.ConsoleEmail, deps.ConsolePassword); err != nil {
		return "", err
	}
	teamID, err := ensureConsoleTeam(ctx, client, sleep, deps.Endpoint, deps.ConsoleTeamID, deps.ProjectName)
	if err != nil {
		return "", err
	}
	if err := ensureConsoleProject(ctx, client, sleep, deps.Endpoint, deps.ProjectID, deps.ProjectName, teamID); err != nil {
		return "", err
	}
	secret, err := ensureConsoleProjectKey(ctx, client, sleep, deps.Endpoint, deps.ProjectID)
	if err != nil {
		return "", err
	}
	return secret, nil
}

// ensureConsoleLoggedIn prefers an existing console session (login-first) so
// restart loops do not hammer POST /account once the first root exists
// (Appwrite 501 user_console_count_exceeded / 429 storms on Coolify previews).
func ensureConsoleLoggedIn(ctx context.Context, client *http.Client, sleep func(context.Context, time.Duration) error, endpoint, email, password string) error {
	loggedIn, err := tryConsoleSession(ctx, client, sleep, endpoint, email, password)
	if err != nil {
		return err
	}
	if loggedIn {
		return nil
	}
	if err := ensureConsoleAccount(ctx, client, sleep, endpoint, email, password); err != nil {
		return err
	}
	loggedIn, err = tryConsoleSession(ctx, client, sleep, endpoint, email, password)
	if err != nil {
		return err
	}
	if !loggedIn {
		return fmt.Errorf("appwrite bootstrap console session: login failed after signup")
	}
	return nil
}

func ensureConsoleAccount(ctx context.Context, client *http.Client, sleep func(context.Context, time.Duration) error, endpoint, email, password string) error {
	body := map[string]any{
		"userId":   "unique()",
		"email":    email,
		"password": password,
		"name":     "Attesta Console",
	}
	resp, err := appwriteJSONRequest(ctx, client, sleep, http.MethodPost, endpoint+"/account", "console", "", body)
	if err != nil {
		return fmt.Errorf("appwrite bootstrap create console account: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return nil
	case resp.StatusCode == http.StatusConflict:
		// Email already registered — continue and retry login.
		return nil
	case resp.StatusCode == http.StatusNotImplemented:
		// Appwrite disables further console signups after the first root
		// (user_console_count_exceeded / _APP_CONSOLE_WHITELIST_ROOT). Continue
		// and let login validate the existing account.
		return nil
	default:
		return fmt.Errorf("appwrite bootstrap create console account: HTTP %d", resp.StatusCode)
	}
}

// tryConsoleSession posts email login. On success returns true. On 401 (or other
// auth-style failure) returns false so callers may attempt signup. Other errors
// fail closed.
func tryConsoleSession(ctx context.Context, client *http.Client, sleep func(context.Context, time.Duration) error, endpoint, email, password string) (bool, error) {
	body := map[string]any{
		"email":    email,
		"password": password,
	}
	resp, err := appwriteJSONRequest(ctx, client, sleep, http.MethodPost, endpoint+"/account/sessions/email", "console", "", body)
	if err != nil {
		return false, fmt.Errorf("appwrite bootstrap console session: %w", err)
	}
	defer resp.Body.Close()
	payload, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true, nil
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return false, nil
	}
	return false, fmt.Errorf("appwrite bootstrap console session: HTTP %d: %s", resp.StatusCode, truncateForErr(payload))
}

func ensureConsoleTeam(ctx context.Context, client *http.Client, sleep func(context.Context, time.Duration) error, endpoint, teamID, name string) (string, error) {
	resp, err := appwriteJSONRequest(ctx, client, sleep, http.MethodGet, endpoint+"/teams", "console", "", nil)
	if err != nil {
		return "", fmt.Errorf("appwrite bootstrap list console teams: %w", err)
	}
	defer resp.Body.Close()
	payload, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var listed struct {
			Teams []struct {
				ID   string `json:"$id"`
				Name string `json:"name"`
			} `json:"teams"`
		}
		if err := json.Unmarshal(payload, &listed); err == nil {
			for _, team := range listed.Teams {
				if team.ID == teamID {
					return team.ID, nil
				}
			}
			if len(listed.Teams) > 0 && strings.TrimSpace(listed.Teams[0].ID) != "" {
				return listed.Teams[0].ID, nil
			}
		}
	}

	createBody := map[string]any{
		"teamId": teamID,
		"name":   name,
	}
	createResp, err := appwriteJSONRequest(ctx, client, sleep, http.MethodPost, endpoint+"/teams", "console", "", createBody)
	if err != nil {
		return "", fmt.Errorf("appwrite bootstrap create console team: %w", err)
	}
	defer createResp.Body.Close()
	createPayload, _ := io.ReadAll(createResp.Body)
	if createResp.StatusCode == http.StatusConflict {
		return teamID, nil
	}
	if createResp.StatusCode < 200 || createResp.StatusCode >= 300 {
		return "", fmt.Errorf("appwrite bootstrap create console team: HTTP %d: %s", createResp.StatusCode, truncateForErr(createPayload))
	}
	var created struct {
		ID string `json:"$id"`
	}
	if err := json.Unmarshal(createPayload, &created); err == nil && strings.TrimSpace(created.ID) != "" {
		return created.ID, nil
	}
	return teamID, nil
}

func ensureConsoleProject(ctx context.Context, client *http.Client, sleep func(context.Context, time.Duration) error, endpoint, projectID, name, teamID string) error {
	getURL := endpoint + "/projects/" + url.PathEscape(projectID)
	resp, err := appwriteJSONRequest(ctx, client, sleep, http.MethodGet, getURL, "console", "", nil)
	if err != nil {
		return fmt.Errorf("appwrite bootstrap get project: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	if resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("appwrite bootstrap get project: HTTP %d", resp.StatusCode)
	}

	body := map[string]any{
		"projectId": projectID,
		"name":      name,
		"teamId":    teamID,
		"region":    "default",
	}
	createResp, err := appwriteJSONRequest(ctx, client, sleep, http.MethodPost, endpoint+"/projects", "console", "", body)
	if err != nil {
		return fmt.Errorf("appwrite bootstrap create project: %w", err)
	}
	defer createResp.Body.Close()
	payload, _ := io.ReadAll(createResp.Body)
	if createResp.StatusCode == http.StatusConflict {
		return nil
	}
	if createResp.StatusCode < 200 || createResp.StatusCode >= 300 {
		return fmt.Errorf("appwrite bootstrap create project: HTTP %d: %s", createResp.StatusCode, truncateForErr(payload))
	}
	return nil
}

// ensureConsoleProjectKey reuses or rotates the single named bootstrap key.
// Appwrite list/get responses usually omit secrets for existing keys, so when
// attesta-bootstrap exists without a secret we delete it and create once —
// avoiding unbounded key accumulation on long-lived preview volumes.
func ensureConsoleProjectKey(ctx context.Context, client *http.Client, sleep func(context.Context, time.Duration) error, endpoint, projectID string) (string, error) {
	keysURL := endpoint + "/projects/" + url.PathEscape(projectID) + "/keys"
	listResp, err := appwriteJSONRequest(ctx, client, sleep, http.MethodGet, keysURL, "console", "", nil)
	if err != nil {
		return "", fmt.Errorf("appwrite bootstrap list keys: %w", err)
	}
	defer listResp.Body.Close()
	listPayload, _ := io.ReadAll(listResp.Body)
	if listResp.StatusCode >= 200 && listResp.StatusCode < 300 {
		var listed struct {
			Keys []struct {
				ID     string `json:"$id"`
				Name   string `json:"name"`
				Secret string `json:"secret"`
			} `json:"keys"`
		}
		if err := json.Unmarshal(listPayload, &listed); err == nil {
			for _, key := range listed.Keys {
				if key.Name != appwriteBootstrapKeyName {
					continue
				}
				if secret := strings.TrimSpace(key.Secret); secret != "" {
					return secret, nil
				}
				if keyID := strings.TrimSpace(key.ID); keyID != "" {
					if err := deleteConsoleProjectKey(ctx, client, sleep, endpoint, projectID, keyID); err != nil {
						return "", err
					}
				}
				break
			}
		}
	}

	return createConsoleProjectKey(ctx, client, sleep, endpoint, projectID)
}

func deleteConsoleProjectKey(ctx context.Context, client *http.Client, sleep func(context.Context, time.Duration) error, endpoint, projectID, keyID string) error {
	delURL := endpoint + "/projects/" + url.PathEscape(projectID) + "/keys/" + url.PathEscape(keyID)
	resp, err := appwriteJSONRequest(ctx, client, sleep, http.MethodDelete, delURL, "console", "", nil)
	if err != nil {
		return fmt.Errorf("appwrite bootstrap delete key: %w", err)
	}
	defer resp.Body.Close()
	payload, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("appwrite bootstrap delete key: HTTP %d: %s", resp.StatusCode, truncateForErr(payload))
	}
	return nil
}

func createConsoleProjectKey(ctx context.Context, client *http.Client, sleep func(context.Context, time.Duration) error, endpoint, projectID string) (string, error) {
	body := map[string]any{
		"name":   appwriteBootstrapKeyName,
		"scopes": appwriteBootstrapKeyScopes,
	}
	keyURL := endpoint + "/projects/" + url.PathEscape(projectID) + "/keys"
	resp, err := appwriteJSONRequest(ctx, client, sleep, http.MethodPost, keyURL, "console", "", body)
	if err != nil {
		return "", fmt.Errorf("appwrite bootstrap create key: %w", err)
	}
	defer resp.Body.Close()
	payload, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("appwrite bootstrap create key: HTTP %d: %s", resp.StatusCode, truncateForErr(payload))
	}
	var created struct {
		Secret string `json:"secret"`
	}
	if err := json.Unmarshal(payload, &created); err != nil {
		return "", fmt.Errorf("appwrite bootstrap create key decode: %w", err)
	}
	secret := strings.TrimSpace(created.Secret)
	if secret == "" {
		return "", fmt.Errorf("appwrite bootstrap create key: empty secret")
	}
	return secret, nil
}

func ensureAppwriteBucket(ctx context.Context, client *http.Client, sleep func(context.Context, time.Duration) error, endpoint, projectID, apiKey, bucketID string) error {
	getURL := endpoint + "/storage/buckets/" + url.PathEscape(bucketID)
	resp, err := appwriteJSONRequest(ctx, client, sleep, http.MethodGet, getURL, projectID, apiKey, nil)
	if err != nil {
		return fmt.Errorf("appwrite bootstrap get bucket: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	if resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("appwrite bootstrap get bucket: HTTP %d", resp.StatusCode)
	}

	body := map[string]any{
		"bucketId":     bucketID,
		"name":         bucketID,
		"fileSecurity": false,
	}
	createResp, err := appwriteJSONRequest(ctx, client, sleep, http.MethodPost, endpoint+"/storage/buckets", projectID, apiKey, body)
	if err != nil {
		return fmt.Errorf("appwrite bootstrap create bucket: %w", err)
	}
	defer createResp.Body.Close()
	payload, _ := io.ReadAll(createResp.Body)
	if createResp.StatusCode == http.StatusConflict {
		return nil
	}
	if createResp.StatusCode < 200 || createResp.StatusCode >= 300 {
		return fmt.Errorf("appwrite bootstrap create bucket: HTTP %d: %s", createResp.StatusCode, truncateForErr(payload))
	}
	return nil
}

func writeAppwriteBootstrapEnvFile(path, projectID, apiKey string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("appwrite bootstrap env file path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("appwrite bootstrap env dir: %w", err)
	}
	content := "APPWRITE_PROJECT_ID=" + projectID + "\nAPPWRITE_API_KEY=" + apiKey + "\n"
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o600); err != nil {
		return fmt.Errorf("appwrite bootstrap write env: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("appwrite bootstrap publish env: %w", err)
	}
	return nil
}

func appwriteJSONRequest(ctx context.Context, client *http.Client, sleep func(context.Context, time.Duration) error, method, rawURL, projectID, apiKey string, body any) (*http.Response, error) {
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	delay := appwriteBootstrap429BaseDelay
	var lastStatus int
	for attempt := 1; attempt <= appwriteBootstrap429MaxAttempts; attempt++ {
		var reader io.Reader
		if payload != nil {
			reader = bytes.NewReader(payload)
		}
		req, err := http.NewRequestWithContext(ctx, method, rawURL, reader)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Appwrite-Project", projectID)
		if apiKey != "" {
			req.Header.Set("X-Appwrite-Key", apiKey)
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}
		lastStatus = resp.StatusCode
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		if attempt == appwriteBootstrap429MaxAttempts {
			break
		}
		if err := sleep(ctx, delay); err != nil {
			return nil, fmt.Errorf("appwrite bootstrap: backoff interrupted: %w", err)
		}
		if delay < appwriteBootstrap429MaxDelay {
			delay *= 2
			if delay > appwriteBootstrap429MaxDelay {
				delay = appwriteBootstrap429MaxDelay
			}
		}
	}
	return nil, fmt.Errorf("appwrite bootstrap: HTTP %d after %d attempts", lastStatus, appwriteBootstrap429MaxAttempts)
}

func truncateForErr(payload []byte) string {
	text := strings.TrimSpace(string(payload))
	if len(text) > 240 {
		return text[:240] + "…"
	}
	return text
}
