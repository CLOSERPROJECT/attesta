package living

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	registerTaskPath     = "intake/register"
	codiceFirTaskPath    = "intake/codiceFir"
	palletWeightTaskPath = "intake/palletWeight"
)

// CommitRequest is the Commit wire body (no run id).
type CommitRequest struct {
	TaskPath      string         `json:"taskPath"`
	JobInstanceID string         `json:"jobInstanceId"`
	Value         map[string]any `json:"value"`
}

type CommitResult struct {
	Org           string `json:"org"`
	JobInstanceID string `json:"jobInstanceId"`
	Domain        Domain `json:"domain"`
}

// CommitError maps to HTTP status in the handler.
type CommitError struct {
	Kind    string
	Message string
}

func (e *CommitError) Error() string {
	if e.Message == "" {
		return "living: " + e.Kind
	}
	return "living: " + e.Kind + ": " + e.Message
}

type Committer struct {
	Domains    DomainStore
	Configs    ConfigStore
	Authorizer Authorizer
	Now        func() time.Time
}

func (c *Committer) Commit(ctx context.Context, org string, op OperatorIdentity, req CommitRequest) (CommitResult, error) {
	org = strings.TrimSpace(org)
	if org == "" {
		return CommitResult{}, &CommitError{Kind: "forbidden", Message: "organization required"}
	}
	if c.Domains == nil || c.Configs == nil || c.Authorizer == nil {
		return CommitResult{}, fmt.Errorf("living: commit stores/authorizer required")
	}

	taskPath := strings.TrimSpace(req.TaskPath)
	ref, err := ParseTaskPath(taskPath)
	if err != nil {
		return CommitResult{}, &CommitError{Kind: "unknown-target", Message: err.Error()}
	}
	jobInstanceID := strings.TrimSpace(req.JobInstanceID)
	if jobInstanceID == "" {
		return CommitResult{}, &CommitError{Kind: "prep", Message: "jobInstanceId required"}
	}
	if req.Value == nil {
		return CommitResult{}, &CommitError{Kind: "prep", Message: "value required"}
	}

	version, err := c.Configs.ActiveVersion(ctx, org)
	if err != nil {
		return CommitResult{}, err
	}
	cfg, err := c.Configs.Get(ctx, org, version)
	if err != nil {
		return CommitResult{}, err
	}

	domain, err := c.Domains.Get(ctx, org)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			return CommitResult{}, err
		}
		domain = EmptyDomain()
	}

	allowed, err := c.Authorizer.AllowCommit(ctx, AuthorizeCommit{
		Operator:      op,
		Org:           org,
		TaskPath:      taskPath,
		JobID:         ref.JobID,
		TaskID:        ref.TaskID,
		JobInstanceID: jobInstanceID,
		RolesAllowed:  rolesAllowedForTask(cfg, ref.JobID, ref.TaskID),
		SequenceOk:    sequenceOkForCommit(cfg, domain, ref.JobID, ref.TaskID, jobInstanceID),
	})
	if err != nil {
		return CommitResult{}, err
	}
	if !allowed {
		return CommitResult{}, &CommitError{Kind: "unauthorized"}
	}

	now := c.Now
	if now == nil {
		now = time.Now
	}
	ts := now().UTC().Format(time.RFC3339Nano)

	var next Domain
	switch taskPath {
	case registerTaskPath:
		next, err = applyRegisterCommit(domain, op.UserID, jobInstanceID, req.Value, ts)
	case codiceFirTaskPath:
		next, err = applyCodiceFirCommit(domain, op.UserID, jobInstanceID, req.Value, ts)
	case palletWeightTaskPath:
		next, err = applyPalletWeightCommit(domain, op.UserID, jobInstanceID, req.Value, ts)
	default:
		return CommitResult{}, &CommitError{
			Kind:    "unknown-target",
			Message: fmt.Sprintf("unsupported task path %q", taskPath),
		}
	}
	if err != nil {
		var ce *CommitError
		if errors.As(err, &ce) {
			return CommitResult{}, ce
		}
		var me MutateError
		if errors.As(err, &me) {
			return CommitResult{}, &CommitError{Kind: "mutate", Message: me.Error()}
		}
		return CommitResult{}, err
	}

	if err := c.Domains.Put(ctx, org, next); err != nil {
		return CommitResult{}, err
	}

	return CommitResult{Org: org, JobInstanceID: jobInstanceID, Domain: next}, nil
}

func applyRegisterCommit(domain Domain, userID, jobInstanceID string, value map[string]any, ts string) (Domain, error) {
	palletCount, ok := intFromAny(value["palletCount"])
	if !ok || palletCount < 1 {
		return Domain{}, &CommitError{Kind: "prep", Message: "value.palletCount must be a positive integer"}
	}

	loadFacts := map[string]any{}
	for _, key := range []string{
		"client", "date", "wasteCode", "idoneitaRiutilizzo",
		"truckWeight", "truckTare", "palletCount", "notes",
	} {
		if v, exists := value[key]; exists {
			loadFacts[key] = v
		}
	}

	ops := make([]MutateOp, 0, 1+palletCount)
	ops = append(ops, MintOp{Entity: EntityLoad, ID: jobInstanceID, Set: loadFacts})
	for i := 1; i <= palletCount; i++ {
		palletID := fmt.Sprintf("%s.%d", jobInstanceID, i)
		ops = append(ops, MintOp{
			Entity: EntityPallet,
			ID:     palletID,
			Set:    map[string]any{"loadId": jobInstanceID},
		})
	}

	return Mutate(domain, MutateRequest{
		JobID:         "intake",
		TaskID:        "register",
		JobInstanceID: jobInstanceID,
		Facts:         cloneFacts(value),
		Ops:           ops,
		Timestamp:     ts,
		UserID:        userID,
		NewJobInstance: &JobInstance{
			JobID: "intake",
			ID:    jobInstanceID,
			Label: jobInstanceID,
		},
	})
}

func applyCodiceFirCommit(domain Domain, userID, jobInstanceID string, value map[string]any, ts string) (Domain, error) {
	codiceFir, ok := value["codiceFir"].(string)
	if !ok || strings.TrimSpace(codiceFir) == "" {
		return Domain{}, &CommitError{Kind: "prep", Message: "value.codiceFir must be a non-empty string"}
	}
	return Mutate(domain, MutateRequest{
		JobID:         "intake",
		TaskID:        "codiceFir",
		JobInstanceID: jobInstanceID,
		Facts:         cloneFacts(value),
		Ops: []MutateOp{
			PatchOp{
				Entity: EntityLoad,
				ID:     jobInstanceID,
				Set:    map[string]any{"codiceFir": codiceFir},
			},
		},
		Timestamp: ts,
		UserID:    userID,
	})
}

// Leftovers completeness all: one Commit must cover every eligible pallet
// (weight still unset for this Job instance). Partial maps are prep-rejected.
func applyPalletWeightCommit(domain Domain, userID, jobInstanceID string, value map[string]any, ts string) (Domain, error) {
	leftoverIDs := palletWeightLeftoverIDs(domain, jobInstanceID)
	weightsRaw, _ := value["weights"].(map[string]any)
	if weightsRaw == nil {
		weightsRaw = map[string]any{}
	}

	if len(leftoverIDs) == 0 {
		return Mutate(domain, MutateRequest{
			JobID:         "intake",
			TaskID:        "palletWeight",
			JobInstanceID: jobInstanceID,
			Facts:         map[string]any{},
			Ops:           nil,
			Timestamp:     ts,
			UserID:        userID,
		})
	}

	filled, err := filterFilledObjectWeights(weightsRaw)
	if err != nil {
		return Domain{}, &CommitError{Kind: "prep", Message: err.Error()}
	}
	if msg := validateWeighFacts(filled, leftoverIDs); msg != "" {
		return Domain{}, &CommitError{Kind: "prep", Message: msg}
	}
	if msg := enforceAllCompleteness(filled, leftoverIDs); msg != "" {
		return Domain{}, &CommitError{Kind: "prep", Message: msg}
	}

	ops := make([]MutateOp, 0, len(filled))
	for id, row := range filled {
		ops = append(ops, PatchOp{
			Entity: EntityPallet,
			ID:     id,
			Set: map[string]any{
				"weight":     row.Weight,
				"tareKind":   row.TareKind,
				"idoneita":   row.Idoneita,
				"panelCount": row.PanelCount,
			},
		})
	}
	return Mutate(domain, MutateRequest{
		JobID:         "intake",
		TaskID:        "palletWeight",
		JobInstanceID: jobInstanceID,
		Facts:         weighFactsAsMap(filled),
		Ops:           ops,
		Timestamp:     ts,
		UserID:        userID,
	})
}

func intFromAny(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	case float64:
		if n != float64(int(n)) {
			return 0, false
		}
		return int(n), true
	case float32:
		if n != float32(int(n)) {
			return 0, false
		}
		return int(n), true
	default:
		return 0, false
	}
}
