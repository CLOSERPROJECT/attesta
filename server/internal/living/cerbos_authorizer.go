package living

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	livingResourceKind = "living"
	livingCommitAction = "commit"
)

// CerbosAuthorizer: product allow/deny only through Cerbos.
type CerbosAuthorizer struct {
	url    string
	client *http.Client
	now    func() time.Time
}

func NewCerbosAuthorizer(url string, client *http.Client, now func() time.Time) *CerbosAuthorizer {
	if client == nil {
		client = http.DefaultClient
	}
	if now == nil {
		now = time.Now
	}
	return &CerbosAuthorizer{url: url, client: client, now: now}
}

func (a *CerbosAuthorizer) AllowCommit(ctx context.Context, q AuthorizeCommit) (bool, error) {
	if a == nil {
		return false, fmt.Errorf("living: cerbos authorizer required")
	}
	principalID := strings.TrimSpace(q.Operator.UserID)
	if principalID == "" {
		principalID = "anonymous"
	}
	resourceID := strings.TrimSpace(q.TaskPath)
	if resourceID == "" {
		resourceID = q.JobID + "/" + q.TaskID
	}

	rolesAllowed := append([]string(nil), q.RolesAllowed...)
	if rolesAllowed == nil {
		rolesAllowed = []string{}
	}
	roleSlugs := append([]string(nil), q.Operator.Roles...)
	if roleSlugs == nil {
		roleSlugs = []string{}
	}

	return a.check(ctx,
		map[string]any{
			"id":    principalID,
			"roles": []string{"authenticated"},
			"attr": map[string]any{
				"orgSlug":   strings.TrimSpace(q.Operator.Org),
				"roleSlugs": roleSlugs,
			},
		},
		livingResourceKind,
		resourceID,
		map[string]any{
			"orgSlug":      strings.TrimSpace(q.Org),
			"taskPath":     strings.TrimSpace(q.TaskPath),
			"jobId":        strings.TrimSpace(q.JobID),
			"taskId":       strings.TrimSpace(q.TaskID),
			"rolesAllowed": rolesAllowed,
			"sequenceOk":   q.SequenceOk,
			"readyWhenOk":  q.ReadyWhenOk,
		},
		livingCommitAction,
	)
}

func (a *CerbosAuthorizer) check(ctx context.Context, principal map[string]any, resourceKind, resourceID string, resourceAttr map[string]any, action string) (bool, error) {
	request := map[string]any{
		"requestId": fmt.Sprintf("living-%d", a.now().UnixNano()),
		"principal": principal,
		"resource": map[string]any{
			"kind": resourceKind,
			"instances": map[string]any{
				resourceID: map[string]any{
					"attr": resourceAttr,
				},
			},
		},
		"actions": []string{action},
	}

	body, err := json.Marshal(request)
	if err != nil {
		return false, err
	}
	endpoint := strings.TrimSuffix(a.url, "/") + "/api/check"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("cerbos status %d", resp.StatusCode)
	}

	var result struct {
		ResourceInstances map[string]struct {
			Actions map[string]string `json:"actions"`
		} `json:"resourceInstances"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}
	if res, ok := result.ResourceInstances[resourceID]; ok {
		if effect, ok := res.Actions[action]; ok {
			return strings.EqualFold(effect, "EFFECT_ALLOW"), nil
		}
	}
	return false, nil
}
