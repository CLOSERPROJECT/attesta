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
		lookupKeys := []string{trimKey}
		if entry.Description == nil {
			lookupKeys = legacyDPPDataLookupKeys(substep, trimKey)
		}
		for _, dataKey := range lookupKeys {
			raw, ok := entry.Data[dataKey]
			if !ok {
				continue
			}
			value := dppStringValue(raw, trimKey)
			if value != "" {
				return value
			}
		}
	}
	return ""
}

func dppStringValue(raw interface{}, key string) string {
	switch typed := raw.(type) {
	case string:
		return strings.TrimSpace(typed)
	case map[string]interface{}:
		if nested, ok := typed[strings.TrimSpace(key)]; ok {
			return dppStringValue(nested, "")
		}
	case primitive.M:
		return dppStringValue(map[string]interface{}(typed), key)
	}
	return ""
}

// legacyDPPDataLookupKeys supports completed steps stored before ProcessStep.Description
// marked the current payload shape.
func legacyDPPDataLookupKeys(sub WorkflowSub, key string) []string {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return nil
	}
	keys := []string{trimmed}
	if trimmed == strings.TrimSpace(sub.InputKey) || trimmed == substepDataKey(sub) {
		for _, dataKey := range substepDataKeys(sub) {
			found := false
			for _, existing := range keys {
				if existing == dataKey {
					found = true
					break
				}
			}
			if !found {
				keys = append(keys, dataKey)
			}
		}
	}
	return keys
}

func parseDigitalLinkPath(path string) (string, string, string, error) {
	trimmed := strings.Trim(strings.TrimSpace(path), "/")
	parts := strings.Split(trimmed, "/")
	return parseDigitalLinkParts(parts)
}

func parseDigitalLinkAttachmentPath(path string) (string, string, string, string, bool, error) {
	trimmed := strings.Trim(strings.TrimSpace(path), "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 9 || parts[6] != "attachment" || parts[8] != "file" {
		return "", "", "", "", false, nil
	}
	gtin, lot, serial, err := parseDigitalLinkParts(parts[:6])
	if err != nil {
		return "", "", "", "", true, err
	}
	attachmentID, err := url.PathUnescape(parts[7])
	if err != nil {
		return "", "", "", "", true, err
	}
	attachmentID = strings.TrimSpace(attachmentID)
	if attachmentID == "" {
		return "", "", "", "", true, errors.New("missing attachment id")
	}
	return gtin, lot, serial, attachmentID, true, nil
}

func parseDigitalLinkParts(parts []string) (string, string, string, error) {
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

func buildDPPTraceabilityView(def WorkflowDef, process *Process, workflowKey string, roleIndex map[roleMetaKey]RoleMeta, cfgRoles []WorkflowRole, orgNames map[string]string) []DPPTraceabilityStep {
	steps := make([]DPPTraceabilityStep, 0, len(def.Steps))
	if process == nil {
		return steps
	}
	substepOrgs := substepOrganizationMap(def)
	availableMap := computeAvailability(def, process)
	terminated := process.Termination != nil
	terminationSubstepID := ""
	terminationReason := ""
	if terminated {
		terminationSubstepID = strings.TrimSpace(process.Termination.SubstepID)
		terminationReason = strings.TrimSpace(process.Termination.Reason)
	}
	pastTermination := false
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
			meta := roleMetaForOrg(substepOrgs[sub.SubstepID], primaryRole, roleIndex, cfgRoles)
			roleLabel := strings.TrimSpace(meta.Label)
			if roleLabel == "" {
				roleLabel = primaryRole
			}
			roleBadges := make([]DPPTraceabilityRoleBadge, 0, len(allowedRoles))
			for _, role := range allowedRoles {
				roleStyle := roleMetaForOrg(substepOrgs[sub.SubstepID], role, roleIndex, cfgRoles)
				roleBadges = append(roleBadges, DPPTraceabilityRoleBadge{
					ID:      role,
					Label:   roleStyle.Label,
					Palette: roleStyle.Palette,
				})
			}
			subView := DPPTraceabilitySubstep{
				SubstepID:  sub.SubstepID,
				Title:      sub.Title,
				Role:       roleLabel,
				RoleBadges: roleBadges,
				Palette:    meta.Palette,
				Status:     "locked",
			}
			progress, done := process.Progress[sub.SubstepID]
			if done && progress.State == "done" {
				subView.Status = "done"
				if override, ok := process.Overrides[sub.SubstepID]; ok && strings.TrimSpace(override.SubstepID) != "" {
					subView.Reason = "Completed with local form adaptation."
					subView.DetailMessage = subView.Reason
					if strings.TrimSpace(override.Reason) != "" {
						subView.DetailMessage += " Reason: " + strings.TrimSpace(override.Reason)
					}
				}
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
						selectedMeta := roleMetaForOrg(substepOrgs[sub.SubstepID], doneRole, roleIndex, cfgRoles)
						subView.Role = selectedMeta.Label
						subView.RoleBadges = []DPPTraceabilityRoleBadge{
							{
								ID:      doneRole,
								Label:   selectedMeta.Label,
								Palette: selectedMeta.Palette,
							},
						}
						subView.Palette = selectedMeta.Palette
					}
				}
				subView.Digest = digestPayload(progress.Data)
				subView.Description = processStepDescription(progress, sub)
				subView.Values = dppTraceValues(sub, progress)
				subView.Attachments = buildActionAttachments(workflowKey, process, progress.Data)
			} else if terminated && strings.TrimSpace(sub.SubstepID) == terminationSubstepID {
				subView.Status = processStatusTerminated
				subView.Reason = "Stream ended early"
				subView.DetailMessage = terminationReason
				if subView.DetailMessage == "" {
					subView.DetailMessage = "No reason provided."
				}
				allDone = false
			} else if terminated && (pastTermination || terminationSubstepID == "") {
				subView.Status = "skipped"
				subView.Reason = "Stream ended early"
				subView.DetailMessage = "Step not completed because the stream was ended before this."
				allDone = false
			} else if availableMap[sub.SubstepID] {
				subView.Status = "available"
				allDone = false
			} else {
				allDone = false
			}
			stepView.Substeps = append(stepView.Substeps, subView)
			if terminated && strings.TrimSpace(sub.SubstepID) == terminationSubstepID {
				pastTermination = true
			}
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

func dppTraceValues(sub WorkflowSub, progress ProcessStep) []DPPTraceabilityValue {
	data := progress.Data
	if len(data) == 0 {
		return nil
	}

	flattened := make([]ActionKV, 0)
	if raw, ok := processStepDataValue(progress, sub); ok {
		flattened = append(flattened, flattenDisplayValues("", raw)...)
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
