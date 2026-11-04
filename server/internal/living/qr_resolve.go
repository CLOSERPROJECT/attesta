package living

import (
	"net/url"
	"strings"
)

type QrConfig struct {
	Rules []QrRule `json:"rules" yaml:"rules"`
}

type QrRule struct {
	ID       string            `json:"id" yaml:"id"`
	When     string            `json:"when,omitempty" yaml:"when,omitempty"`
	Match    QrMatch           `json:"match" yaml:"match"`
	Eligible string            `json:"eligible,omitempty" yaml:"eligible,omitempty"`
	Seed     map[string]string `json:"seed,omitempty" yaml:"seed,omitempty"`
	Messages QrMessages        `json:"messages,omitempty" yaml:"messages,omitempty"`
}

type QrMatch struct {
	JobID      string `json:"jobId" yaml:"jobId"`
	GroupID    string `json:"groupId,omitempty" yaml:"groupId,omitempty"`
	TaskID     string `json:"taskId" yaml:"taskId"`
	AllowSpawn bool   `json:"allowSpawn,omitempty" yaml:"allowSpawn,omitempty"`
}

type QrMessages struct {
	OnZero       string `json:"onZero,omitempty" yaml:"onZero,omitempty"`
	OnMany       string `json:"onMany,omitempty" yaml:"onMany,omitempty"`
	OnIneligible string `json:"onIneligible,omitempty" yaml:"onIneligible,omitempty"`
}

type QrNavigate struct {
	JobID           string `json:"jobId"`
	JobInstanceID   string `json:"jobInstanceId"`
	GroupID         string `json:"groupId,omitempty"`
	GroupInstanceID string `json:"groupInstanceId,omitempty"`
	TaskID          string `json:"taskId"`
}

type QrResolveError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type QrResolveResult struct {
	Navigate *QrNavigate     `json:"navigate,omitempty"`
	Seed     map[string]any  `json:"seed,omitempty"`
	Error    *QrResolveError `json:"error,omitempty"`
}

// DecodeScanEntityID extracts entityId from a cold-link URL/query or treats raw as bare id.
// Fixed convention — no Config parse. URL-shaped input without entityId yields "".
func DecodeScanEntityID(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if u, err := url.Parse(raw); err == nil {
		if id := strings.TrimSpace(u.Query().Get("entityId")); id != "" {
			return id
		}
		// Absolute or query-only URL without entityId is not a bare id.
		if u.Scheme != "" || strings.HasPrefix(raw, "?") {
			return ""
		}
	}
	if i := strings.Index(raw, "entityId="); i >= 0 {
		rest := raw[i+len("entityId="):]
		if end := strings.IndexAny(rest, "&#"); end >= 0 {
			rest = rest[:end]
		}
		if id, err := url.QueryUnescape(rest); err == nil {
			return strings.TrimSpace(id)
		}
		return strings.TrimSpace(rest)
	}
	return raw
}

// ResolveQR runs Org Config qr.rules against the viewer Worklist.
// First when-matching rule owns the outcome (eligible / zero / one / many).
func ResolveQR(cfg OrgConfig, domain Domain, op OperatorIdentity, raw string) QrResolveResult {
	if cfg.QR == nil || len(cfg.QR.Rules) == 0 {
		return qrErr("no-rule", "No QR rule configured")
	}
	entityID := DecodeScanEntityID(raw)
	bindings := domainBindings(domain, "", cfg.Catalogs)
	bindings["scan"] = map[string]any{"entityId": entityID}
	wl := ProjectWorklist(cfg, domain, op)

	for _, rule := range cfg.QR.Rules {
		if strings.TrimSpace(rule.When) != "" {
			v, err := evaluateExpr(rule.When, map[string]any{}, bindings)
			if err != nil || !truthy(v) {
				continue
			}
		}
		if strings.TrimSpace(rule.Eligible) != "" {
			v, err := evaluateExpr(rule.Eligible, map[string]any{}, bindings)
			if err != nil || !truthy(v) {
				return qrErr("ineligible", msgOr(rule.Messages.OnIneligible, "Ineligible"))
			}
		}
		cands := matchQrCandidates(wl, rule.Match)
		switch len(cands) {
		case 0:
			return qrErr("zero", msgOr(rule.Messages.OnZero, "No destination"))
		case 1:
			seed, err := evalQrSeed(rule.Seed, bindings)
			if err != nil {
				return qrErr("seed", "QR seed not evaluable")
			}
			nav := cands[0]
			return QrResolveResult{Navigate: &nav, Seed: seed}
		default:
			return qrErr("many", msgOr(rule.Messages.OnMany, "Multiple destinations"))
		}
	}
	return qrErr("no-rule", "No QR rule for this code")
}

func qrErr(code, message string) QrResolveResult {
	return QrResolveResult{Error: &QrResolveError{Code: code, Message: message}}
}

func msgOr(authored, fallback string) string {
	if strings.TrimSpace(authored) != "" {
		return authored
	}
	return fallback
}

func matchQrCandidates(wl Worklist, match QrMatch) []QrNavigate {
	jobID := strings.TrimSpace(match.JobID)
	taskID := strings.TrimSpace(match.TaskID)
	groupID := strings.TrimSpace(match.GroupID)
	if jobID == "" || taskID == "" {
		return nil
	}
	var out []QrNavigate
	for _, job := range wl.Jobs {
		if job.JobID != jobID {
			continue
		}
		for _, inst := range job.Instances {
			if groupID != "" {
				g, ok := inst.Groups[groupID]
				if !ok {
					continue
				}
				for _, gi := range g.Instances {
					if gi.Tasks[taskID] == TaskStatusActionable {
						out = append(out, QrNavigate{
							JobID:           jobID,
							JobInstanceID:   inst.ID,
							GroupID:         groupID,
							GroupInstanceID: gi.ID,
							TaskID:          taskID,
						})
					}
				}
				if match.AllowSpawn && g.CanSpawn {
					out = append(out, QrNavigate{
						JobID:           jobID,
						JobInstanceID:   inst.ID,
						GroupID:         groupID,
						GroupInstanceID: ProvisionalGroupInstanceID,
						TaskID:          taskID,
					})
				}
				continue
			}
			if inst.Tasks[taskID] == TaskStatusActionable {
				out = append(out, QrNavigate{
					JobID:         jobID,
					JobInstanceID: inst.ID,
					TaskID:        taskID,
				})
			}
		}
	}
	return out
}

func evalQrSeed(seed map[string]string, bindings ExprBindings) (map[string]any, error) {
	if len(seed) == 0 {
		return map[string]any{}, nil
	}
	out := make(map[string]any, len(seed))
	for key, expr := range seed {
		v, err := evaluateExpr(expr, map[string]any{}, bindings)
		if err != nil {
			return nil, err
		}
		out[key] = v
	}
	return out, nil
}
