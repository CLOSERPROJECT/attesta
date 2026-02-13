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

func buildDPPTraceabilityView(def WorkflowDef, process *Process, workflowKey string) []DPPTraceabilityStep {
	steps := make([]DPPTraceabilityStep, 0, len(def.Steps))
	if process == nil {
		return steps
	}
	availableMap := computeAvailability(def, process)
	for _, step := range sortedSteps(def) {
		stepView := DPPTraceabilityStep{
			StepID: step.StepID,
			Title:  step.Title,
		}
		for _, sub := range sortedSubsteps(step) {
			subView := DPPTraceabilitySubstep{
				SubstepID: sub.SubstepID,
				Title:     sub.Title,
				Role:      sub.Role,
				Status:    "locked",
			}
			progress, done := process.Progress[sub.SubstepID]
			if done && progress.State == "done" {
				subView.Status = "done"
				if progress.DoneAt != nil {
					subView.DoneAt = progress.DoneAt.Format(time.RFC3339)
				}
				if progress.DoneBy != nil {
					subView.DoneBy = progress.DoneBy.UserID
				}
				subView.Digest = digestPayload(progress.Data)
				subView.Values = dppTraceValues(progress.Data)
				if attachment, ok := readAttachmentPayload(progress.Data, sub.InputKey); ok {
					subView.FileName = attachment.Filename
					subView.FileSHA256 = attachment.SHA256
					subView.FileURL = fmt.Sprintf("%s/process/%s/substep/%s/file", workflowPath(workflowKey), process.ID.Hex(), sub.SubstepID)
				}
			} else if availableMap[sub.SubstepID] {
				subView.Status = "available"
			}
			stepView.Substeps = append(stepView.Substeps, subView)
		}
		steps = append(steps, stepView)
	}
	return steps
}

func dppTraceValues(data map[string]interface{}) []DPPTraceabilityValue {
	if len(data) == 0 {
		return nil
	}
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	values := make([]DPPTraceabilityValue, 0, len(keys))
	for _, key := range keys {
		raw := data[key]
		if nested, ok := raw.(map[string]interface{}); ok {
			if _, hasAttachment := nested["attachmentId"]; hasAttachment {
				continue
			}
		}
		text := strings.TrimSpace(fmt.Sprintf("%v", raw))
		if text == "" {
			continue
		}
		values = append(values, DPPTraceabilityValue{Key: key, Value: text})
	}
	return values
}
