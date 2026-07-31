package main

import (
	"strings"
	"time"
)

type seedInstanceKind string

const (
	seedKindFreshActive seedInstanceKind = "fresh_active"
	seedKindMidActive   seedInstanceKind = "mid_active"
	seedKindDoneWithDPP seedInstanceKind = "done_with_dpp"
	seedKindDoneNoDPP   seedInstanceKind = "done_no_dpp"
	seedKindTerminated  seedInstanceKind = "terminated"
	seedLotValue                         = "SEED-LOT-001"
)

type seedInstancePlan struct {
	Kind              seedInstanceKind
	Name              string
	Status            string
	DoneSubstepCount  int
	TerminationReason string
	WantDPP           bool
	CreatedOffset     time.Duration
}

func seedDoneSubstepCount(total int, fraction float64) int {
	if total <= 0 {
		return 0
	}
	n := int(float64(total) * fraction)
	if n < 1 && total >= 1 && fraction > 0 {
		n = 1
	}
	if n > total {
		n = total
	}
	return n
}

func buildSeedInstancePlans(cfg RuntimeConfig, now time.Time) []seedInstancePlan {
	_ = now
	total := len(orderedSubsteps(cfg.Workflow))
	midDone := seedDoneSubstepCount(total, 0.4)
	termDone := seedDoneSubstepCount(total, 0.5)
	name := strings.TrimSpace(cfg.Workflow.Name)
	if name == "" {
		name = "Stream"
	}
	dppOn := cfg.DPP.Enabled
	plans := []seedInstancePlan{
		{Kind: seedKindFreshActive, Name: name + " — just started", Status: processStatusActive, DoneSubstepCount: 0, CreatedOffset: -7 * time.Minute},
		{Kind: seedKindMidActive, Name: name + " — mid run A", Status: processStatusActive, DoneSubstepCount: midDone, CreatedOffset: -6 * time.Minute},
		{Kind: seedKindMidActive, Name: name + " — mid run B", Status: processStatusActive, DoneSubstepCount: midDone, CreatedOffset: -5 * time.Minute},
	}
	if dppOn {
		plans = append(plans,
			seedInstancePlan{Kind: seedKindDoneWithDPP, Name: name + " — completed (passport)", Status: processStatusDone, DoneSubstepCount: total, WantDPP: true, CreatedOffset: -4 * time.Minute},
			seedInstancePlan{Kind: seedKindDoneWithDPP, Name: name + " — completed (passport 2)", Status: processStatusDone, DoneSubstepCount: total, WantDPP: true, CreatedOffset: -3 * time.Minute},
			seedInstancePlan{Kind: seedKindDoneNoDPP, Name: name + " — completed", Status: processStatusDone, DoneSubstepCount: total, WantDPP: false, CreatedOffset: -2 * time.Minute},
		)
	} else {
		plans = append(plans,
			seedInstancePlan{Kind: seedKindDoneNoDPP, Name: name + " — completed A", Status: processStatusDone, DoneSubstepCount: total, CreatedOffset: -4 * time.Minute},
			seedInstancePlan{Kind: seedKindDoneNoDPP, Name: name + " — completed B", Status: processStatusDone, DoneSubstepCount: total, CreatedOffset: -3 * time.Minute},
			seedInstancePlan{Kind: seedKindDoneNoDPP, Name: name + " — completed C", Status: processStatusDone, DoneSubstepCount: total, CreatedOffset: -2 * time.Minute},
		)
	}
	plans = append(plans,
		seedInstancePlan{Kind: seedKindTerminated, Name: name + " — stopped", Status: processStatusTerminated, DoneSubstepCount: termDone, TerminationReason: "Seeded termination: operator stopped run", CreatedOffset: -time.Minute},
		seedInstancePlan{Kind: seedKindTerminated, Name: name + " — abandoned", Status: processStatusTerminated, DoneSubstepCount: termDone, TerminationReason: "Seeded termination: abandoned incomplete run", CreatedOffset: -30 * time.Second},
	)
	return plans
}

func synthesizeSeedProgress(def WorkflowDef, cfg RuntimeConfig, doneCount int, now time.Time, actor Actor) map[string]ProcessStep {
	subs := orderedSubsteps(def)
	if doneCount > len(subs) {
		doneCount = len(subs)
	}
	if doneCount < 0 {
		doneCount = 0
	}
	progress := make(map[string]ProcessStep, doneCount)
	doneAt := now
	actorCopy := actor
	for i := 0; i < doneCount; i++ {
		sub := subs[i]
		key := encodeProgressKey(sub.SubstepID)
		progress[key] = ProcessStep{
			State:  "done",
			DoneAt: &doneAt,
			DoneBy: &actorCopy,
			Data:   synthesizeSeedStepData(sub, cfg),
		}
	}
	return progress
}

func synthesizeSeedStepData(sub WorkflowSub, cfg RuntimeConfig) map[string]interface{} {
	inputType := normalizeInputTypeForCheck(sub.InputType)
	if inputType == "file" {
		return nil
	}
	if cfg.DPP.Enabled {
		lotKey := strings.TrimSpace(cfg.DPP.LotInputKey)
		if lotKey == "" {
			lotKey = "batchId"
		}
		if seedSubstepSuppliesLot(sub, lotKey) {
			return map[string]interface{}{lotKey: seedLotValue}
		}
	}
	switch inputType {
	case "formata", "object":
		key := strings.TrimSpace(sub.InputKey)
		if key == "" {
			key = "value"
		}
		return map[string]interface{}{
			key: map[string]interface{}{
				"seed": true,
				"note": "Seeded placeholder",
			},
		}
	case "number":
		key := seedScalarDataKey(sub)
		return map[string]interface{}{key: float64(1)}
	default:
		key := seedScalarDataKey(sub)
		return map[string]interface{}{key: "seed"}
	}
}

func seedScalarDataKey(sub WorkflowSub) string {
	if key := strings.TrimSpace(sub.InputKey); key != "" {
		return key
	}
	return "value"
}

func seedSubstepSuppliesLot(sub WorkflowSub, lotKey string) bool {
	lotKey = strings.TrimSpace(lotKey)
	if lotKey == "" {
		return false
	}
	if strings.TrimSpace(sub.InputKey) == lotKey {
		return true
	}
	if sub.Schema == nil {
		return false
	}
	props, ok := sub.Schema["properties"].(map[string]interface{})
	if !ok {
		return false
	}
	_, ok = props[lotKey]
	return ok
}
