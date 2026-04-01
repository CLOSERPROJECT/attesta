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
	CanDeleteStream(ctx context.Context, user *AccountUser, workflowKey string, createdByUserID string, hasProcesses bool) (bool, error)
	CanAccess(ctx context.Context, user *AccountUser, resourceKind, resourceID string, resourceAttr map[string]interface{}, action string) (bool, error)
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

func (a *CerbosAuthorizer) checkResourceAction(ctx context.Context, principal map[string]interface{}, resourceKind, resourceID string, resourceAttr map[string]interface{}, action string) (bool, error) {
	request := map[string]interface{}{
		"requestId": fmt.Sprintf("req-%d", a.now().UnixNano()),
		"principal": principal,
		"resource": map[string]interface{}{
			"kind": resourceKind,
			"instances": map[string]interface{}{
				resourceID: map[string]interface{}{
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

func principalForAccountUser(user *AccountUser) (string, []string, map[string]interface{}) {
	if user == nil {
		return "", nil, nil
	}

	roles := []string{"authenticated"}
	if user.IsPlatformAdmin {
		roles = append(roles, "platform_admin")
	}
	if userIsOrgAdmin(user) {
		roles = append(roles, "org_admin")
	}

	principalID := formataStreamUserID(user)
	if principalID == "" {
		principalID = accountActorID(user)
	}

	attr := map[string]interface{}{
		"userId":          strings.TrimSpace(principalID),
		"identityUserId":  strings.TrimSpace(user.IdentityUserID),
		"orgSlug":         strings.TrimSpace(user.OrgSlug),
		"roleSlugs":       append([]string(nil), user.RoleSlugs...),
		"isPlatformAdmin": user.IsPlatformAdmin,
		"isOrgAdmin":      userIsOrgAdmin(user),
		"status":          strings.TrimSpace(user.Status),
	}
	return principalID, roles, attr
}

func (a *CerbosAuthorizer) CanComplete(ctx context.Context, actor Actor, processID string, workflowKey string, sub WorkflowSub, stepOrder int, stepOrgSlug string, sequenceOK bool) (bool, error) {
	rolesAllowed := append([]string(nil), sub.Roles...)
	if len(rolesAllowed) == 0 && strings.TrimSpace(sub.Role) != "" {
		rolesAllowed = []string{strings.TrimSpace(sub.Role)}
	}
	if len(actor.RoleSlugs) == 0 && strings.TrimSpace(actor.Role) != "" {
		actor.RoleSlugs = []string{strings.TrimSpace(actor.Role)}
	}
	return a.checkResourceAction(ctx,
		map[string]interface{}{
			"id":    actor.ID,
			"roles": []string{"authenticated"},
			"attr": map[string]interface{}{
				"orgSlug":     strings.TrimSpace(actor.OrgSlug),
				"roleSlugs":   actor.RoleSlugs,
				"activeRole":  strings.TrimSpace(actor.Role),
				"workflowKey": strings.TrimSpace(actor.WorkflowKey),
			},
		},
		"substep",
		sub.SubstepID,
		map[string]interface{}{
			"orgSlug":      strings.TrimSpace(stepOrgSlug),
			"rolesAllowed": rolesAllowed,
			"stepOrder":    stepOrder,
			"substepOrder": sub.Order,
			"substepId":    sub.SubstepID,
			"processId":    processID,
			"workflowKey":  strings.TrimSpace(workflowKey),
			"sequenceOk":   sequenceOK,
		},
		"complete",
	)
}

func (a *CerbosAuthorizer) CanDeleteStream(ctx context.Context, user *AccountUser, workflowKey string, createdByUserID string, hasProcesses bool) (bool, error) {
	if user == nil {
		return false, nil
	}
	principalID, roles, attr := principalForAccountUser(user)

	return a.checkResourceAction(ctx,
		map[string]interface{}{
			"id":    principalID,
			"roles": roles,
			"attr": mergeStringMap(attr, map[string]interface{}{
				"workflowKey": strings.TrimSpace(workflowKey),
			}),
		},
		"stream",
		strings.TrimSpace(workflowKey),
		map[string]interface{}{
			"workflowKey":     strings.TrimSpace(workflowKey),
			"createdByUserId": strings.TrimSpace(createdByUserID),
			"hasProcesses":    hasProcesses,
		},
		"delete",
	)
}

func (a *CerbosAuthorizer) CanAccess(ctx context.Context, user *AccountUser, resourceKind, resourceID string, resourceAttr map[string]interface{}, action string) (bool, error) {
	if user == nil {
		return false, nil
	}
	principalID, roles, attr := principalForAccountUser(user)
	return a.checkResourceAction(ctx,
		map[string]interface{}{
			"id":    principalID,
			"roles": roles,
			"attr":  attr,
		},
		strings.TrimSpace(resourceKind),
		strings.TrimSpace(resourceID),
		resourceAttr,
		strings.TrimSpace(action),
	)
}

func mergeStringMap(base map[string]interface{}, extra map[string]interface{}) map[string]interface{} {
	if len(base) == 0 && len(extra) == 0 {
		return map[string]interface{}{}
	}
	merged := make(map[string]interface{}, len(base)+len(extra))
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range extra {
		merged[key] = value
	}
	return merged
}
