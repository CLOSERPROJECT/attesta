package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"
)

type TeamResolver interface {
	TeamIDsForUser(ctx context.Context, userID string) ([]string, error)
}

type NoopTeamResolver struct{}

func (NoopTeamResolver) TeamIDsForUser(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}

func teamResolverEnabled(resolver TeamResolver) bool {
	switch resolver.(type) {
	case nil, NoopTeamResolver, *NoopTeamResolver:
		return false
	default:
		return true
	}
}

type AppwriteTeamResolver struct {
	endpoint  string
	projectID string
	apiKey    string
	client    *http.Client
}

type AppwriteSyncOptions struct {
	DefaultPassword string
	MembershipURL   string
}

type AppwriteSyncReport struct {
	TeamsCreated       int
	UsersCreated       int
	MembershipsCreated int
}

func NewAppwriteTeamResolver(endpoint, projectID, apiKey string, client *http.Client) *AppwriteTeamResolver {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	base := strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if base != "" && !strings.HasSuffix(base, "/v1") {
		base += "/v1"
	}
	return &AppwriteTeamResolver{
		endpoint:  base,
		projectID: strings.TrimSpace(projectID),
		apiKey:    strings.TrimSpace(apiKey),
		client:    client,
	}
}

func NewTeamResolverFromEnv(client *http.Client) TeamResolver {
	endpoint := strings.TrimSpace(os.Getenv("APPWRITE_ENDPOINT"))
	projectID := strings.TrimSpace(os.Getenv("APPWRITE_PROJECT_ID"))
	apiKey := strings.TrimSpace(os.Getenv("APPWRITE_API_KEY"))
	if endpoint == "" || projectID == "" || apiKey == "" {
		return NoopTeamResolver{}
	}
	return NewAppwriteTeamResolver(endpoint, projectID, apiKey, client)
}

func (a *AppwriteTeamResolver) TeamIDsForUser(ctx context.Context, userID string) ([]string, error) {
	trimmedUser := strings.TrimSpace(userID)
	if trimmedUser == "" {
		return nil, nil
	}
	endpoint := strings.TrimSpace(a.endpoint)
	if endpoint == "" {
		return nil, nil
	}

	reqURL := fmt.Sprintf("%s/users/%s/memberships?limit=100", endpoint, url.PathEscape(trimmedUser))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Appwrite-Project", a.projectID)
	req.Header.Set("X-Appwrite-Key", a.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return []string{}, nil
	}
	if resp.StatusCode != http.StatusOK {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("appwrite memberships status %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}

	var decoded struct {
		Memberships []struct {
			TeamID string `json:"teamId"`
		} `json:"memberships"`
		Documents []struct {
			TeamID string `json:"teamId"`
		} `json:"documents"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}

	teamIDs := make([]string, 0, len(decoded.Memberships)+len(decoded.Documents))
	for _, membership := range decoded.Memberships {
		teamIDs = append(teamIDs, membership.TeamID)
	}
	for _, membership := range decoded.Documents {
		teamIDs = append(teamIDs, membership.TeamID)
	}
	return normalizeIDs(teamIDs), nil
}

func (a *AppwriteTeamResolver) SyncRuntimeConfig(ctx context.Context, cfg RuntimeConfig, options AppwriteSyncOptions) (AppwriteSyncReport, error) {
	report := AppwriteSyncReport{}
	defaultPassword := strings.TrimSpace(options.DefaultPassword)
	if defaultPassword == "" {
		return report, errors.New("default password is empty")
	}

	teamNameByID := collectTeamNames(cfg)
	teamIDs := make([]string, 0, len(teamNameByID))
	for teamID := range teamNameByID {
		teamIDs = append(teamIDs, teamID)
	}
	sort.Strings(teamIDs)
	for _, teamID := range teamIDs {
		created, err := a.createTeam(ctx, teamID, teamNameByID[teamID])
		if err != nil {
			return report, err
		}
		if created {
			report.TeamsCreated++
		}
	}

	departmentTeam := departmentTeamMap(cfg)
	membershipURL := strings.TrimSpace(options.MembershipURL)
	if membershipURL == "" {
		membershipURL = "http://localhost:3030"
	}
	for _, user := range cfg.Users {
		email := strings.TrimSpace(user.Email)
		if email == "" {
			continue
		}
		createdUser, err := a.createUser(ctx, user, defaultPassword)
		if err != nil {
			return report, err
		}
		if createdUser {
			report.UsersCreated++
		}

		desiredTeams := desiredTeamIDsForUser(user, departmentTeam)
		for _, teamID := range desiredTeams {
			createdMembership, err := a.createMembership(ctx, teamID, user, membershipURL)
			if err != nil {
				return report, err
			}
			if createdMembership {
				report.MembershipsCreated++
			}
		}
	}
	return report, nil
}

func collectTeamNames(cfg RuntimeConfig) map[string]string {
	teamNameByID := make(map[string]string)
	for _, department := range cfg.Departments {
		teamID := strings.TrimSpace(department.AppwriteTeamID)
		if teamID == "" {
			continue
		}
		name := strings.TrimSpace(department.Name)
		if name == "" {
			name = teamID
		}
		teamNameByID[teamID] = name
	}
	for _, step := range cfg.Workflow.Steps {
		for _, substep := range step.Substep {
			for _, teamID := range normalizeIDs(substep.AppwriteTeamIDs) {
				if _, exists := teamNameByID[teamID]; !exists {
					teamNameByID[teamID] = teamID
				}
			}
		}
	}
	for _, user := range cfg.Users {
		for _, teamID := range normalizeIDs(user.AppwriteTeamIDs) {
			if _, exists := teamNameByID[teamID]; !exists {
				teamNameByID[teamID] = teamID
			}
		}
	}
	return teamNameByID
}

func departmentTeamMap(cfg RuntimeConfig) map[string]string {
	teams := make(map[string]string)
	for _, department := range cfg.Departments {
		departmentID := strings.TrimSpace(department.ID)
		teamID := strings.TrimSpace(department.AppwriteTeamID)
		if departmentID == "" || teamID == "" {
			continue
		}
		teams[departmentID] = teamID
	}
	return teams
}

func desiredTeamIDsForUser(user User, departmentTeams map[string]string) []string {
	ids := append([]string(nil), user.AppwriteTeamIDs...)
	if mapped := strings.TrimSpace(departmentTeams[strings.TrimSpace(user.DepartmentID)]); mapped != "" {
		ids = append(ids, mapped)
	}
	return normalizeIDs(ids)
}

func (a *AppwriteTeamResolver) createTeam(ctx context.Context, teamID, name string) (bool, error) {
	payload := map[string]interface{}{
		"teamId": strings.TrimSpace(teamID),
		"name":   strings.TrimSpace(name),
	}
	resp, err := a.doJSON(ctx, http.MethodPost, "/teams", payload)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusCreated, http.StatusOK:
		return true, nil
	case http.StatusConflict:
		return false, nil
	default:
		return false, appwriteStatusError("create team", resp)
	}
}

func (a *AppwriteTeamResolver) createUser(ctx context.Context, user User, defaultPassword string) (bool, error) {
	payload := map[string]interface{}{
		"userId":   strings.TrimSpace(user.ID),
		"email":    strings.TrimSpace(user.Email),
		"password": strings.TrimSpace(defaultPassword),
		"name":     strings.TrimSpace(user.Name),
	}
	resp, err := a.doJSON(ctx, http.MethodPost, "/users", payload)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusCreated, http.StatusOK:
		return true, nil
	case http.StatusConflict:
		return false, nil
	default:
		return false, appwriteStatusError("create user", resp)
	}
}

func (a *AppwriteTeamResolver) createMembership(ctx context.Context, teamID string, user User, membershipURL string) (bool, error) {
	payload := map[string]interface{}{
		"email":  strings.TrimSpace(user.Email),
		"userId": strings.TrimSpace(user.ID),
		"roles":  normalizeIDs([]string{user.DepartmentID}),
		"url":    strings.TrimSpace(membershipURL),
		"name":   strings.TrimSpace(user.Name),
	}
	path := fmt.Sprintf("/teams/%s/memberships", url.PathEscape(strings.TrimSpace(teamID)))
	resp, err := a.doJSON(ctx, http.MethodPost, path, payload)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusCreated, http.StatusOK:
		return true, nil
	case http.StatusConflict:
		return false, nil
	default:
		return false, appwriteStatusError("create membership", resp)
	}
}

func (a *AppwriteTeamResolver) doJSON(ctx context.Context, method, path string, payload interface{}) (*http.Response, error) {
	endpoint := strings.TrimRight(strings.TrimSpace(a.endpoint), "/")
	if endpoint == "" {
		return nil, errors.New("appwrite endpoint is empty")
	}
	requestPath := path
	if !strings.HasPrefix(requestPath, "/") {
		requestPath = "/" + requestPath
	}
	body := bytes.NewReader(nil)
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint+requestPath, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Appwrite-Project", a.projectID)
	req.Header.Set("X-Appwrite-Key", a.apiKey)
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return a.client.Do(req)
}

func appwriteStatusError(action string, resp *http.Response) error {
	payload, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return fmt.Errorf("%s status %d: %s", action, resp.StatusCode, strings.TrimSpace(string(payload)))
}

func normalizeIDs(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	sort.Strings(normalized)
	if normalized == nil {
		return []string{}
	}
	return normalized
}
