package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Authorizer interface {
	CanComplete(ctx context.Context, actor Actor, processID string, workflowKey string, sub WorkflowSub, stepOrder int, stepOrgSlug string, sequenceOK bool) (bool, error)
}

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

func (a *CerbosAuthorizer) CanComplete(ctx context.Context, actor Actor, processID string, workflowKey string, sub WorkflowSub, stepOrder int, stepOrgSlug string, sequenceOK bool) (bool, error) {
	rolesAllowed := append([]string(nil), sub.Roles...)
	if len(rolesAllowed) == 0 && strings.TrimSpace(sub.Role) != "" {
		rolesAllowed = []string{strings.TrimSpace(sub.Role)}
	}
	if len(actor.RoleSlugs) == 0 && strings.TrimSpace(actor.Role) != "" {
		actor.RoleSlugs = []string{strings.TrimSpace(actor.Role)}
	}
	request := map[string]interface{}{
		"requestId": fmt.Sprintf("req-%d", a.now().UnixNano()),
		"principal": map[string]interface{}{
			"id":    actor.UserID,
			"roles": []string{"authenticated"},
			"attr": map[string]interface{}{
				"orgSlug":     strings.TrimSpace(actor.OrgSlug),
				"roleSlugs":   actor.RoleSlugs,
				"activeRole":  strings.TrimSpace(actor.Role),
				"workflowKey": strings.TrimSpace(actor.WorkflowKey),
			},
		},
		"resource": map[string]interface{}{
			"kind": "substep",
			"instances": map[string]interface{}{
				sub.SubstepID: map[string]interface{}{
					"attr": map[string]interface{}{
						"orgSlug":      strings.TrimSpace(stepOrgSlug),
						"rolesAllowed": rolesAllowed,
						"stepOrder":    stepOrder,
						"substepOrder": sub.Order,
						"substepId":    sub.SubstepID,
						"processId":    processID,
						"workflowKey":  strings.TrimSpace(workflowKey),
						"sequenceOk":   sequenceOK,
					},
				},
			},
		},
		"actions": []string{"complete"},
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
	if res, ok := result.ResourceInstances[sub.SubstepID]; ok {
		if effect, ok := res.Actions["complete"]; ok {
			return strings.EqualFold(effect, "EFFECT_ALLOW"), nil
		}
	}
	return false, nil
}
