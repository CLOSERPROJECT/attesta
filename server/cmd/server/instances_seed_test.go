package main

import (
	"testing"
	"time"
)

func TestBuildSeedInstancePlansMix(t *testing.T) {
	cfg := RuntimeConfig{
		Workflow: WorkflowDef{
			Name: "Demo",
			Steps: []WorkflowStep{{
				StepID: "1", Order: 1, Title: "S1",
				Substep: []WorkflowSub{
					{SubstepID: "1.1", Order: 1, Title: "A", InputKey: "a", InputType: "string"},
					{SubstepID: "1.2", Order: 2, Title: "B", InputKey: "b", InputType: "string"},
					{SubstepID: "1.3", Order: 3, Title: "C", InputKey: "c", InputType: "string"},
					{SubstepID: "1.4", Order: 4, Title: "D", InputKey: "d", InputType: "string"},
					{SubstepID: "1.5", Order: 5, Title: "E", InputKey: "e", InputType: "string"},
				},
			}},
		},
		DPP: DPPConfig{Enabled: true, GTIN: "09506000134352", LotDefault: "LOT", SerialStrategy: "process_id_hex"},
	}
	plans := buildSeedInstancePlans(cfg, time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC))
	if len(plans) != 8 {
		t.Fatalf("len(plans) = %d, want 8", len(plans))
	}
	var fresh, mid, doneDPP, doneNo, term int
	for _, p := range plans {
		switch p.Kind {
		case seedKindFreshActive:
			fresh++
			if p.Status != processStatusActive || p.DoneSubstepCount != 0 {
				t.Fatalf("fresh plan = %+v", p)
			}
		case seedKindMidActive:
			mid++
			if p.Status != processStatusActive || p.DoneSubstepCount != 2 { // 40% of 5 → 2
				t.Fatalf("mid plan = %+v", p)
			}
		case seedKindDoneWithDPP:
			doneDPP++
			if p.Status != processStatusDone || !p.WantDPP || p.DoneSubstepCount != 5 {
				t.Fatalf("done+dpp = %+v", p)
			}
		case seedKindDoneNoDPP:
			doneNo++
			if p.Status != processStatusDone || p.WantDPP || p.DoneSubstepCount != 5 {
				t.Fatalf("done-no-dpp = %+v", p)
			}
		case seedKindTerminated:
			term++
			if p.Status != processStatusTerminated || p.TerminationReason == "" {
				t.Fatalf("terminated = %+v", p)
			}
		default:
			t.Fatalf("unknown kind %q", p.Kind)
		}
	}
	if fresh != 1 || mid != 2 || doneDPP != 2 || doneNo != 1 || term != 2 {
		t.Fatalf("counts fresh=%d mid=%d doneDPP=%d doneNo=%d term=%d", fresh, mid, doneDPP, doneNo, term)
	}
}

func TestBuildSeedInstancePlansNoDPPWhenDisabled(t *testing.T) {
	cfg := RuntimeConfig{
		Workflow: WorkflowDef{Name: "X", Steps: []WorkflowStep{{
			StepID: "1", Order: 1,
			Substep: []WorkflowSub{{SubstepID: "1.1", Order: 1, InputKey: "a", InputType: "string"}},
		}}},
		DPP: DPPConfig{Enabled: false},
	}
	plans := buildSeedInstancePlans(cfg, time.Now().UTC())
	for _, p := range plans {
		if p.Kind == seedKindDoneWithDPP || p.WantDPP {
			t.Fatalf("DPP disabled but plan wants DPP: %+v", p)
		}
	}
	var done int
	for _, p := range plans {
		if p.Status == processStatusDone {
			done++
		}
	}
	if done != 3 {
		t.Fatalf("done count = %d, want 3", done)
	}
}

func TestSynthesizeSeedProgressEncodedKeysAndPrefix(t *testing.T) {
	def := WorkflowDef{Steps: []WorkflowStep{{
		StepID: "1", Order: 1,
		Substep: []WorkflowSub{
			{SubstepID: "1.1", Order: 1, InputKey: "a", InputType: "string"},
			{SubstepID: "1.2", Order: 2, InputKey: "b", InputType: "string"},
			{SubstepID: "1.3", Order: 3, InputKey: "c", InputType: "string"},
		},
	}}}
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	actor := Actor{ID: "seed", Role: "seed"}
	progress := synthesizeSeedProgress(def, RuntimeConfig{}, 2, now, actor)
	if _, ok := progress["1_1"]; !ok {
		t.Fatalf("expected encoded key 1_1, got %#v", progress)
	}
	if _, ok := progress["1.1"]; ok {
		t.Fatal("must not store dotted key 1.1")
	}
	if len(progress) != 2 {
		t.Fatalf("len(progress) = %d, want 2", len(progress))
	}
	if progress["1_2"].State != "done" || progress["1_2"].DoneBy == nil {
		t.Fatalf("1_2 = %+v", progress["1_2"])
	}
}
