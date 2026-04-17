package main

import (
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// dppSerialFromStrategy returns a deterministic serial using the configured strategy.
func dppSerialFromStrategy(strategy string, processID primitive.ObjectID) (string, error) {
	normalized, err := normalizeDPPSerialStrategy(strategy)
	if err != nil {
		return "", err
	}
	switch normalized {
	case "process_id_hex":
		return processID.Hex(), nil
	default:
		return "", fmt.Errorf("unsupported dpp.serialStrategy %q (allowed: process_id_hex)", strategy)
	}
}

func buildProcessDPP(def WorkflowDef, cfg DPPConfig, process *Process, generatedAt time.Time) (ProcessDPP, error) {
	if process == nil {
		return ProcessDPP{}, errors.New("missing process")
	}
	if !cfg.Enabled {
		return ProcessDPP{}, errors.New("dpp is disabled")
	}
	if strings.TrimSpace(cfg.GTIN) == "" {
		return ProcessDPP{}, errors.New("missing dpp.gtin")
	}

	lot := dppFirstStringValue(def, process, cfg.LotInputKey)
	if lot == "" {
		lot = cfg.LotDefault
	}
	serial := ""
	if cfg.SerialInputKey != "" {
		serial = dppFirstStringValue(def, process, cfg.SerialInputKey)
	}
	if serial == "" {
		derivedSerial, err := dppSerialFromStrategy(cfg.SerialStrategy, process.ID)
		if err != nil {
			return ProcessDPP{}, err
		}
		serial = derivedSerial
	}
	if lot == "" {
		return ProcessDPP{}, errors.New("missing dpp lot value")
	}
	if serial == "" {
		return ProcessDPP{}, errors.New("missing dpp serial value")
	}
	return ProcessDPP{
		GTIN:        cfg.GTIN,
		Lot:         lot,
		Serial:      serial,
		GeneratedAt: generatedAt,
	}, nil
}

func dppFirstStringValue(def WorkflowDef, process *Process, key string) string {
	trimKey := strings.TrimSpace(key)
	if process == nil || trimKey == "" {
		return ""
	}
	for _, substep := range orderedSubsteps(def) {
		entry, ok := process.Progress[substep.SubstepID]
		if !ok || entry.State != "done" || entry.Data == nil {
			continue
		}
		raw, ok := entry.Data[trimKey]
		if !ok {
			continue
		}
		value, ok := raw.(string)
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func parseDigitalLinkPath(path string) (string, string, string, error) {
	trimmed := strings.Trim(strings.TrimSpace(path), "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 6 || parts[0] != "01" || parts[2] != "10" || parts[4] != "21" {
		return "", "", "", errors.New("invalid digital link path")
	}
	gtinRaw, err := url.PathUnescape(parts[1])
	if err != nil {
		return "", "", "", err
	}
	gtin, err := normalizeGTIN(gtinRaw)
	if err != nil {
		return "", "", "", err
	}
	lot, err := url.PathUnescape(parts[3])
	if err != nil {
		return "", "", "", err
	}
	serial, err := url.PathUnescape(parts[5])
	if err != nil {
		return "", "", "", err
	}
	lot = strings.TrimSpace(lot)
	serial = strings.TrimSpace(serial)
	if lot == "" || serial == "" {
		return "", "", "", errors.New("missing lot or serial")
	}
	return gtin, lot, serial, nil
}

func digitalLinkURL(gtin, lot, serial string) string {
	return "/01/" + url.PathEscape(strings.TrimSpace(gtin)) +
		"/10/" + url.PathEscape(strings.TrimSpace(lot)) +
		"/21/" + url.PathEscape(strings.TrimSpace(serial))
}

func gs1ElementString(gtin, lot, serial string) string {
	trimmedGTIN := strings.TrimSpace(gtin)
	trimmedLot := strings.TrimSpace(lot)
	trimmedSerial := strings.TrimSpace(serial)
	if trimmedGTIN == "" || trimmedLot == "" || trimmedSerial == "" {
		return ""
	}
	return fmt.Sprintf("(01)%s(10)%s(21)%s", trimmedGTIN, trimmedLot, trimmedSerial)
}

func buildDPPTraceabilityView(def WorkflowDef, process *Process, workflowKey string, roleMeta map[string]RoleMeta, orgNames map[string]string) []DPPTraceabilityStep {
	steps := make([]DPPTraceabilityStep, 0, len(def.Steps))
	if process == nil {
		return steps
	}
	availableMap := computeAvailability(def, process)
	for _, step := range sortedSteps(def) {
		stepView := DPPTraceabilityStep{
			StepID:           step.StepID,
			Title:            step.Title,
			OrganizationName: organizationDisplayName(step.OrganizationSlug, orgNames),
			DetailsDialogID:  "dpp-step-dialog-" + dppDialogIDFragment(step.StepID),
		}
		allDone := len(step.Substep) > 0
		var latestDoneAt time.Time
		for _, sub := range sortedSubsteps(step) {
			allowedRoles := substepRoles(sub)
			primaryRole := strings.TrimSpace(sub.Role)
			if primaryRole == "" && len(allowedRoles) > 0 {
				primaryRole = strings.TrimSpace(allowedRoles[0])
			}
			meta := roleMetaFor(primaryRole, roleMeta)
			roleLabel := strings.TrimSpace(meta.Label)
			if roleLabel == "" {
				roleLabel = primaryRole
			}
			roleBadges := make([]DPPTraceabilityRoleBadge, 0, len(allowedRoles))
			for _, role := range allowedRoles {
				roleStyle := roleMetaFor(role, roleMeta)
				roleBadges = append(roleBadges, DPPTraceabilityRoleBadge{
					ID:     role,
					Label:  roleStyle.Label,
					Color:  cssValue(roleStyle.Color, "var(--role-fallback)"),
					Border: cssValue(roleStyle.Border, "var(--border)"),
				})
			}
			subView := DPPTraceabilitySubstep{
				SubstepID:  sub.SubstepID,
				Title:      sub.Title,
				Role:       roleLabel,
				RoleBadges: roleBadges,
				RoleColor:  cssValue(meta.Color, "var(--role-fallback)"),
				RoleBorder: cssValue(meta.Border, "var(--border)"),
				Status:     "locked",
			}
			progress, done := process.Progress[sub.SubstepID]
			if done && progress.State == "done" {
				subView.Status = "done"
				if progress.DoneAt != nil {
					subView.DoneAt = progress.DoneAt.UTC().Format(time.RFC3339)
					subView.DoneAtHuman = humanReadableTraceabilityTime(*progress.DoneAt)
					if progress.DoneAt.After(latestDoneAt) {
						latestDoneAt = *progress.DoneAt
					}
				}
				if progress.DoneBy != nil {
					subView.DoneBy = progress.DoneBy.ID
					doneRole := strings.TrimSpace(progress.DoneBy.Role)
					if doneRole != "" {
						selectedMeta := roleMetaFor(doneRole, roleMeta)
						subView.Role = selectedMeta.Label
						subView.RoleBadges = []DPPTraceabilityRoleBadge{
							{
								ID:     doneRole,
								Label:  selectedMeta.Label,
								Color:  cssValue(selectedMeta.Color, "var(--role-fallback)"),
								Border: cssValue(selectedMeta.Border, "var(--border)"),
							},
						}
						subView.RoleColor = cssValue(selectedMeta.Color, "var(--role-fallback)")
						subView.RoleBorder = cssValue(selectedMeta.Border, "var(--border)")
					}
				}
				subView.Digest = digestPayload(progress.Data)
				subView.Values = dppTraceValues(sub, progress.Data)
				subView.Attachments = buildActionAttachments(workflowKey, process, progress.Data)
			} else if availableMap[sub.SubstepID] {
				subView.Status = "available"
				allDone = false
			} else {
				allDone = false
			}
			stepView.Substeps = append(stepView.Substeps, subView)
		}
		if allDone && !latestDoneAt.IsZero() {
			stepView.CompletedAt = latestDoneAt.UTC().Format(time.RFC3339)
			stepView.CompletedAtHuman = humanReadableTraceabilityTime(latestDoneAt)
		}
		steps = append(steps, stepView)
	}
	return steps
}

func humanReadableTraceabilityTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format("2 Jan 2006 at 15:04 MST")
}

func dppDialogIDFragment(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "step"
	}
	var builder strings.Builder
	builder.Grow(len(value))
	lastDash := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
			lastDash = false
		case r >= 'A' && r <= 'Z':
			builder.WriteRune(r + ('a' - 'A'))
			lastDash = false
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				builder.WriteByte('-')
				lastDash = true
			}
		}
	}
	out := strings.Trim(builder.String(), "-")
	if out == "" {
		return "step"
	}
	return out
}

func dppTraceValues(sub WorkflowSub, data map[string]interface{}) []DPPTraceabilityValue {
	if len(data) == 0 {
		return nil
	}

	flattened := make([]ActionKV, 0)
	if strings.EqualFold(strings.TrimSpace(sub.InputType), "formata") {
		if raw, ok := data[sub.InputKey]; ok {
			flattened = append(flattened, flattenDisplayValues("", raw)...)
		}
	} else if raw, ok := data[sub.InputKey]; ok && !isAttachmentMetaValue(raw) {
		flattened = append(flattened, flattenDisplayValues(sub.InputKey, raw)...)
	}
	if len(flattened) == 0 {
		keys := make([]string, 0, len(data))
		for key := range data {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			raw := data[key]
			if isAttachmentMetaValue(raw) {
				continue
			}
			flattened = append(flattened, flattenDisplayValues(key, raw)...)
		}
	}

	values := make([]DPPTraceabilityValue, 0, len(flattened))
	for _, item := range flattened {
		if strings.TrimSpace(item.Value) == "" {
			continue
		}
		values = append(values, DPPTraceabilityValue{Key: item.Key, Value: item.Value})
	}
	return values
}
