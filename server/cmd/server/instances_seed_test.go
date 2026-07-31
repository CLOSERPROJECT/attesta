package main

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestSeedStreamLeftEmptyEveryFifth(t *testing.T) {
	for _, tc := range []struct {
		index int
		want  bool
	}{
		{0, false},
		{1, false},
		{2, false},
		{3, false},
		{4, true},
		{5, false},
		{9, true},
		{-1, false},
	} {
		if got := seedStreamLeftEmpty(tc.index); got != tc.want {
			t.Fatalf("seedStreamLeftEmpty(%d) = %v, want %v", tc.index, got, tc.want)
		}
	}
}

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

func testSeedWorkflowConfig(dpp bool, withFile bool) RuntimeConfig {
	subs := []WorkflowSub{
		{SubstepID: "1.1", Order: 1, Title: "Text", InputKey: "note", InputType: "string"},
		{SubstepID: "1.2", Order: 2, Title: "Batch", InputKey: "batchId", InputType: "formata",
			Schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"batchId": map[string]interface{}{"type": "string"},
				},
			},
		},
	}
	if withFile {
		subs = append(subs, WorkflowSub{SubstepID: "1.3", Order: 3, Title: "Doc", InputKey: "doc", InputType: "file"})
	}
	cfg := RuntimeConfig{
		Workflow: WorkflowDef{Name: "Seed WF", Steps: []WorkflowStep{{StepID: "1", Order: 1, Substep: subs}}},
	}
	if dpp {
		cfg.DPP = DPPConfig{Enabled: true, GTIN: "09506000134352", LotInputKey: "batchId", SerialStrategy: "process_id_hex"}
	}
	return cfg
}

func TestSeedWorkflowInstancesReplaceAndMix(t *testing.T) {
	store := NewMemoryStore()
	cfg := testSeedWorkflowConfig(true, false)
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	n, err := seedWorkflowInstances(t.Context(), store, "wf-seed", cfg, now)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if n != 8 {
		t.Fatalf("inserted = %d, want 8", n)
	}
	store.SeedProcess(Process{WorkflowKey: "wf-seed", Status: processStatusActive, CreatedAt: now})
	n, err = seedWorkflowInstances(t.Context(), store, "wf-seed", cfg, now)
	if err != nil {
		t.Fatalf("re-seed: %v", err)
	}
	list, err := store.ListRecentProcessesByWorkflow(t.Context(), "wf-seed", 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 8 {
		t.Fatalf("after replace len = %d, want 8", len(list))
	}
	var active, done, term, withDPP int
	for _, p := range list {
		switch p.Status {
		case processStatusActive:
			active++
		case processStatusDone:
			done++
			if p.DPP != nil {
				withDPP++
			}
		case processStatusTerminated:
			term++
			if p.Termination == nil || p.Termination.Reason == "" {
				t.Fatalf("terminated missing reason: %+v", p)
			}
		}
	}
	if active != 3 || done != 3 || term != 2 {
		t.Fatalf("status counts active=%d done=%d term=%d", active, done, term)
	}
	if withDPP != 2 {
		t.Fatalf("withDPP = %d, want 2", withDPP)
	}
}

func TestSeedWorkflowInstancesFileAttachmentRoundTrip(t *testing.T) {
	store := NewMemoryStore()
	cfg := testSeedWorkflowConfig(false, true)
	now := time.Now().UTC()
	if _, err := seedWorkflowInstances(t.Context(), store, "wf-file", cfg, now); err != nil {
		t.Fatalf("seed: %v", err)
	}
	list, err := store.ListRecentProcessesByWorkflow(t.Context(), "wf-file", 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var found bool
	for _, p := range list {
		if p.Status != processStatusDone {
			continue
		}
		progress := normalizeProgressKeys(p.Progress)
		step, ok := progress["1.3"]
		if !ok || step.State != "done" || step.Data == nil {
			continue
		}
		id, _ := step.Data["attachmentId"].(string)
		if id == "" {
			t.Fatalf("missing attachmentId in %#v", step.Data)
		}
		oid, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			t.Fatalf("attachment id: %v", err)
		}
		if _, err := store.LoadAttachmentByID(t.Context(), oid); err != nil {
			t.Fatalf("LoadAttachmentByID: %v", err)
		}
		found = true
		break
	}
	if !found {
		t.Fatal("expected a done process with file progress")
	}
}
