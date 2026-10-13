package main

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func feragameDraftAndSealDef() WorkflowDef {
	return WorkflowDef{
		Name:               "Feragame production day",
		CompletionStrategy: CompletionStrategyDraftAndSeal,
		Steps: []WorkflowStep{{
			StepID:           "1",
			Title:            "Production day",
			Order:            1,
			OrganizationSlug: "feragame",
			Substep: []WorkflowSub{
				{
					SubstepID: "1.1",
					Title:     "Batch loads",
					Order:     1,
					Role:      "registrar",
					InputKey:  "batches",
					InputType: "formata",
					Schema:    feragameArraySchema("batches", []string{"loadId", "weightKg"}),
				},
				{
					SubstepID: "1.2",
					Title:     "UdC registrations",
					Order:     2,
					Role:      "registrar",
					InputKey:  "udc",
					InputType: "formata",
					Schema:    feragameArraySchema("udc", []string{"station", "weightKg"}),
				},
			},
		}},
	}
}

func feragameArraySchema(arrayKey string, required []string) map[string]interface{} {
	props := map[string]interface{}{}
	for _, field := range required {
		props[field] = map[string]interface{}{"type": "string"}
	}
	requiredAny := make([]interface{}, len(required))
	for i, field := range required {
		requiredAny[i] = field
	}
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			arrayKey: map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type":       "object",
					"properties": props,
					"required":   requiredAny,
				},
			},
		},
	}
}

func seedFeragameProcess(t *testing.T, store *MemoryStore, def WorkflowDef) *Process {
	t.Helper()
	processID := primitive.NewObjectID()
	progress := map[string]ProcessStep{}
	for _, sub := range orderedSubsteps(def) {
		progress[encodeProgressKey(sub.SubstepID)] = ProcessStep{State: "pending"}
	}
	store.SeedProcess(Process{
		ID:       processID,
		Status:   processStatusActive,
		Progress: progress,
	})
	process, err := store.LoadProcessByID(context.Background(), processID)
	if err != nil {
		t.Fatalf("LoadProcessByID: %v", err)
	}
	process.Progress = normalizeProgressKeys(process.Progress)
	return process
}

func TestDraftAndSealParallelAvailability(t *testing.T) {
	def := feragameDraftAndSealDef()
	process := &Process{
		Status: processStatusActive,
		Progress: map[string]ProcessStep{
			"1.1": {State: "pending"},
			"1.2": {State: "pending"},
		},
	}

	availability := substepAvailability(def, process)
	if !availability["1.1"] || !availability["1.2"] {
		t.Fatalf("expected both kind Substeps available, got %#v", availability)
	}
	if !substepSequenceOK(def, process, "1.1") || !substepSequenceOK(def, process, "1.2") {
		t.Fatal("expected both kind Substeps sequence-OK under draft-and-seal")
	}

	// Classic remains sequential on the same shape without strategy flag.
	classic := def
	classic.CompletionStrategy = CompletionStrategyClassic
	classicAvail := substepAvailability(classic, process)
	if !classicAvail["1.1"] || classicAvail["1.2"] {
		t.Fatalf("classic should unlock only first Substep, got %#v", classicAvail)
	}
}

func TestDraftAndSealCrossStepLock(t *testing.T) {
	def := feragameDraftAndSealDef()
	def.Steps = append(def.Steps, WorkflowStep{
		StepID:           "2",
		Title:            "Downstream org",
		Order:            2,
		OrganizationSlug: "buyer",
		Substep: []WorkflowSub{{
			SubstepID: "2.1",
			Title:     "Receive",
			Order:     1,
			Role:      "clerk",
			InputKey:  "receipt",
			InputType: "formata",
			Schema:    map[string]interface{}{"type": "object"},
		}},
	})
	process := &Process{
		Status: processStatusActive,
		Progress: map[string]ProcessStep{
			"1.1": {State: "draft"},
			"1.2": {State: "pending"},
			"2.1": {State: "pending"},
		},
	}

	availability := substepAvailability(def, process)
	if !availability["1.1"] || !availability["1.2"] {
		t.Fatalf("expected Feragame kinds available, got %#v", availability)
	}
	if availability["2.1"] {
		t.Fatal("expected Step 2 locked until Feragame Step is sealed/done")
	}
	if substepSequenceOK(def, process, "2.1") {
		t.Fatal("expected 2.1 sequence blocked while prior Step incomplete")
	}

	process.Progress["1.1"] = ProcessStep{State: "done"}
	process.Progress["1.2"] = ProcessStep{State: "done"}
	availability = substepAvailability(def, process)
	if availability["1.1"] || availability["1.2"] {
		t.Fatalf("sealed kinds should not be available, got %#v", availability)
	}
	if !availability["2.1"] || !substepSequenceOK(def, process, "2.1") {
		t.Fatalf("expected Step 2 unlocked after prior Step done, got %#v", availability)
	}
}

func TestDraftAndSealSaveDraftPersistsWithoutNotarization(t *testing.T) {
	fixedNow := time.Date(2026, 10, 9, 8, 0, 0, 0, time.UTC)
	def := feragameDraftAndSealDef()
	cfg := RuntimeConfig{Workflow: def}
	store := NewMemoryStore()
	svc := &ProcessService{store: store, now: func() time.Time { return fixedNow }}
	process := seedFeragameProcess(t, store, def)

	batchPayload := map[string]interface{}{
		"batches": []interface{}{
			map[string]interface{}{"loadId": "BL-1", "weightKg": "1200"},
		},
	}
	updated, err := svc.SaveDraft(context.Background(), SaveDraftCmd{
		Process:     process,
		WorkflowKey: "feragame",
		SubstepID:   "1.1",
		Substep:     def.Steps[0].Substep[0],
		Actor:       Actor{ID: "registrar-1", Role: "registrar", RoleSlugs: []string{"registrar"}, OrgSlug: "feragame"},
		Payload:     batchPayload,
		Config:      cfg,
		Now:         fixedNow,
	})
	if err != nil {
		t.Fatalf("SaveDraft batches: %v", err)
	}
	step := updated.Progress["1.1"]
	if step.State != progressStateDraft {
		t.Fatalf("1.1 state = %q, want %q", step.State, progressStateDraft)
	}
	if step.DoneAt != nil || step.DoneBy != nil {
		t.Fatalf("draft must not set DoneAt/DoneBy, got DoneAt=%v DoneBy=%v", step.DoneAt, step.DoneBy)
	}
	if !reflect.DeepEqual(step.Data, batchPayload) {
		t.Fatalf("1.1 data = %#v, want %#v", step.Data, batchPayload)
	}
	if len(store.Notarizations()) != 0 {
		t.Fatalf("draft must not notarize, got %d notarizations", len(store.Notarizations()))
	}

	udcPayload := map[string]interface{}{
		"udc": []interface{}{
			map[string]interface{}{"station": "S-3", "weightKg": "18.5"},
		},
	}
	updated, err = svc.SaveDraft(context.Background(), SaveDraftCmd{
		Process:     updated,
		WorkflowKey: "feragame",
		SubstepID:   "1.2",
		Substep:     def.Steps[0].Substep[1],
		Actor:       Actor{ID: "registrar-1", Role: "registrar", RoleSlugs: []string{"registrar"}, OrgSlug: "feragame"},
		Payload:     udcPayload,
		Config:      cfg,
		Now:         fixedNow,
	})
	if err != nil {
		t.Fatalf("SaveDraft udc: %v", err)
	}

	// Interleave: append another batch row, then another UdC row; both remain editable drafts.
	batchPayload = map[string]interface{}{
		"batches": []interface{}{
			map[string]interface{}{"loadId": "BL-1", "weightKg": "1200"},
			map[string]interface{}{"loadId": "BL-2", "weightKg": "800"},
		},
	}
	updated, err = svc.SaveDraft(context.Background(), SaveDraftCmd{
		Process:     updated,
		WorkflowKey: "feragame",
		SubstepID:   "1.1",
		Substep:     def.Steps[0].Substep[0],
		Actor:       Actor{ID: "registrar-1", Role: "registrar"},
		Payload:     batchPayload,
		Config:      cfg,
	})
	if err != nil {
		t.Fatalf("SaveDraft interleaved batches: %v", err)
	}
	udcPayload = map[string]interface{}{
		"udc": []interface{}{
			map[string]interface{}{"station": "S-3", "weightKg": "18.5"},
			map[string]interface{}{"station": "S-1", "weightKg": "22"},
		},
	}
	updated, err = svc.SaveDraft(context.Background(), SaveDraftCmd{
		Process:     updated,
		WorkflowKey: "feragame",
		SubstepID:   "1.2",
		Substep:     def.Steps[0].Substep[1],
		Actor:       Actor{ID: "registrar-1", Role: "registrar"},
		Payload:     udcPayload,
		Config:      cfg,
	})
	if err != nil {
		t.Fatalf("SaveDraft interleaved udc: %v", err)
	}

	reloaded, err := store.LoadProcessByID(context.Background(), updated.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	reloaded.Progress = normalizeProgressKeys(reloaded.Progress)
	if reloaded.Progress["1.1"].State != progressStateDraft || reloaded.Progress["1.2"].State != progressStateDraft {
		t.Fatalf("expected both kinds draft after reload, got %#v", reloaded.Progress)
	}
	if got := len(reloaded.Progress["1.1"].Data["batches"].([]interface{})); got != 2 {
		t.Fatalf("batches rows = %d, want 2", got)
	}
	if got := len(reloaded.Progress["1.2"].Data["udc"].([]interface{})); got != 2 {
		t.Fatalf("udc rows = %d, want 2", got)
	}
	if len(store.Notarizations()) != 0 {
		t.Fatalf("interleaved drafts must not notarize, got %d", len(store.Notarizations()))
	}

	availability := substepAvailability(def, reloaded)
	if !availability["1.1"] || !availability["1.2"] {
		t.Fatalf("drafted kinds must stay available, got %#v", availability)
	}
}

func TestDraftAndSealSaveDraftValidation(t *testing.T) {
	def := feragameDraftAndSealDef()
	cfg := RuntimeConfig{Workflow: def}
	store := NewMemoryStore()
	svc := &ProcessService{store: store, now: time.Now}
	process := seedFeragameProcess(t, store, def)

	_, err := svc.SaveDraft(context.Background(), SaveDraftCmd{
		Process: process, WorkflowKey: "feragame", SubstepID: "1.1",
		Substep: def.Steps[0].Substep[0], Actor: Actor{ID: "u1"},
		Payload: map[string]interface{}{
			"batches[][loadId]": "BL-1",
			"batches[][weightKg]": "10",
		},
		Config: cfg,
	})
	if !errors.Is(err, ErrDraftValidation) {
		t.Fatalf("flat FormData keys error = %v, want ErrDraftValidation", err)
	}

	_, err = svc.SaveDraft(context.Background(), SaveDraftCmd{
		Process: process, WorkflowKey: "feragame", SubstepID: "1.1",
		Substep: def.Steps[0].Substep[0], Actor: Actor{ID: "u1"},
		Payload: map[string]interface{}{
			"batches": []interface{}{
				map[string]interface{}{"loadId": "BL-1"}, // missing weightKg
			},
		},
		Config: cfg,
	})
	if !errors.Is(err, ErrDraftValidation) {
		t.Fatalf("SaveDraft error = %v, want ErrDraftValidation", err)
	}
	if len(store.Notarizations()) != 0 {
		t.Fatal("invalid draft must not notarize")
	}
	stored, _ := store.LoadProcessByID(context.Background(), process.ID)
	if step := stored.Progress[encodeProgressKey("1.1")]; step.State != "pending" || step.Data != nil {
		t.Fatalf("invalid draft must not persist, got %#v", step)
	}

	// Empty array is valid (no minimum count).
	updated, err := svc.SaveDraft(context.Background(), SaveDraftCmd{
		Process: process, WorkflowKey: "feragame", SubstepID: "1.1",
		Substep: def.Steps[0].Substep[0], Actor: Actor{ID: "u1"},
		Payload: map[string]interface{}{"batches": []interface{}{}},
		Config:  cfg,
	})
	if err != nil {
		t.Fatalf("empty array SaveDraft: %v", err)
	}
	if updated.Progress["1.1"].State != progressStateDraft {
		t.Fatalf("empty draft state = %q, want %q", updated.Progress["1.1"].State, progressStateDraft)
	}
}

func TestDraftAndSealCompleteSubstepRejected(t *testing.T) {
	def := feragameDraftAndSealDef()
	cfg := RuntimeConfig{Workflow: def}
	store := NewMemoryStore()
	svc := &ProcessService{store: store, now: time.Now}
	process := seedFeragameProcess(t, store, def)

	_, err := svc.CompleteSubstep(context.Background(), CompleteSubstepCmd{
		Process: process, WorkflowKey: "feragame", SubstepID: "1.1",
		Substep: def.Steps[0].Substep[0], Actor: Actor{ID: "u1", Role: "registrar"},
		Payload: map[string]interface{}{"batches": []interface{}{}}, Config: cfg,
	})
	if !errors.Is(err, ErrCompletionUnsupported) {
		t.Fatalf("CompleteSubstep error = %v, want ErrCompletionUnsupported", err)
	}
	if len(store.Notarizations()) != 0 {
		t.Fatal("rejected complete must not notarize")
	}
}

func TestDraftAndSealSealStepFreezesNotarizesAndCompletes(t *testing.T) {
	fixedNow := time.Date(2026, 10, 9, 18, 30, 0, 0, time.UTC)
	def := feragameDraftAndSealDef()
	cfg := RuntimeConfig{Workflow: def}
	store := NewMemoryStore()
	svc := &ProcessService{store: store, now: func() time.Time { return fixedNow }}
	process := seedFeragameProcess(t, store, def)

	batchPayload := map[string]interface{}{
		"batches": []interface{}{
			map[string]interface{}{"loadId": "BL-1", "weightKg": "1200"},
		},
	}
	udcPayload := map[string]interface{}{
		"udc": []interface{}{
			map[string]interface{}{"station": "S-3", "weightKg": "18.5"},
		},
	}
	registrar := Actor{ID: "registrar-1", Role: "registrar", RoleSlugs: []string{"registrar"}, OrgSlug: "feragame"}
	updated, err := svc.SaveDraft(context.Background(), SaveDraftCmd{
		Process: process, WorkflowKey: "feragame", SubstepID: "1.1",
		Substep: def.Steps[0].Substep[0], Actor: registrar, Payload: batchPayload, Config: cfg,
	})
	if err != nil {
		t.Fatalf("SaveDraft batches: %v", err)
	}
	updated, err = svc.SaveDraft(context.Background(), SaveDraftCmd{
		Process: updated, WorkflowKey: "feragame", SubstepID: "1.2",
		Substep: def.Steps[0].Substep[1], Actor: registrar, Payload: udcPayload, Config: cfg,
	})
	if err != nil {
		t.Fatalf("SaveDraft udc: %v", err)
	}

	supervisor := Actor{
		ID: "supervisor-1", Role: "org-admin", RoleSlugs: []string{"org-admin"}, OrgSlug: "feragame",
	}
	_, err = svc.SealStep(context.Background(), SealStepCmd{
		Process: updated, WorkflowKey: "feragame", StepID: "1",
		Actor: supervisor, Config: cfg, Confirm: false, Now: fixedNow,
	})
	if !errors.Is(err, ErrSealNotConfirmed) {
		t.Fatalf("SealStep without confirm error = %v, want ErrSealNotConfirmed", err)
	}

	_, err = svc.SealStep(context.Background(), SealStepCmd{
		Process: updated, WorkflowKey: "feragame", StepID: "1",
		Actor: registrar, Config: cfg, Confirm: true, Now: fixedNow,
	})
	if !errors.Is(err, ErrSealUnauthorized) {
		t.Fatalf("SealStep registrar error = %v, want ErrSealUnauthorized", err)
	}

	wrongOrg := Actor{ID: "other-admin", Role: "org-admin", RoleSlugs: []string{"org-admin"}, OrgSlug: "other-org"}
	_, err = svc.SealStep(context.Background(), SealStepCmd{
		Process: updated, WorkflowKey: "feragame", StepID: "1",
		Actor: wrongOrg, Config: cfg, Confirm: true, Now: fixedNow,
	})
	if !errors.Is(err, ErrSealUnauthorized) {
		t.Fatalf("SealStep wrong-org admin error = %v, want ErrSealUnauthorized", err)
	}

	sealed, err := svc.SealStep(context.Background(), SealStepCmd{
		Process: updated, WorkflowKey: "feragame", StepID: "1",
		Actor: supervisor, Config: cfg, Confirm: true, Now: fixedNow,
	})
	if err != nil {
		t.Fatalf("SealStep: %v", err)
	}
	if sealed.Status != processStatusDone {
		t.Fatalf("sealed instance status = %q, want %q", sealed.Status, processStatusDone)
	}
	for _, id := range []string{"1.1", "1.2"} {
		step := sealed.Progress[id]
		if step.State != "done" {
			t.Fatalf("%s state = %q, want done", id, step.State)
		}
		if step.DoneAt == nil || !step.DoneAt.Equal(fixedNow) {
			t.Fatalf("%s DoneAt = %v, want %v", id, step.DoneAt, fixedNow)
		}
		if step.DoneBy == nil || step.DoneBy.ID != supervisor.ID {
			t.Fatalf("%s DoneBy = %#v, want supervisor", id, step.DoneBy)
		}
	}
	if !reflect.DeepEqual(sealed.Progress["1.1"].Data, batchPayload) {
		t.Fatalf("1.1 sealed data = %#v", sealed.Progress["1.1"].Data)
	}
	if !reflect.DeepEqual(sealed.Progress["1.2"].Data, udcPayload) {
		t.Fatalf("1.2 sealed data = %#v", sealed.Progress["1.2"].Data)
	}

	notaries := store.Notarizations()
	if len(notaries) != 2 {
		t.Fatalf("expected 2 notarizations (one per kind leaf), got %d", len(notaries))
	}
	bySubstep := map[string]Notarization{}
	for _, n := range notaries {
		bySubstep[n.SubstepID] = n
	}
	for _, id := range []string{"1.1", "1.2"} {
		n, ok := bySubstep[id]
		if !ok {
			t.Fatalf("missing notarization for %s", id)
		}
		if !n.CreatedAt.Equal(fixedNow) {
			t.Fatalf("%s notarization CreatedAt = %v, want seal time %v", id, n.CreatedAt, fixedNow)
		}
		if n.FakeNotary.Digest != digestPayload(sealed.Progress[id].Data) {
			t.Fatalf("%s digest mismatch: %q vs payload digest", id, n.FakeNotary.Digest)
		}
		if n.Actor.ID != supervisor.ID {
			t.Fatalf("%s notarization actor = %#v", id, n.Actor)
		}
	}

	export := buildNotarizedExport(def, sealed)
	if len(export.Steps) != 1 || len(export.Steps[0].Substeps) != 2 {
		t.Fatalf("notarized export structure = %#v", export.Steps)
	}
	if export.Steps[0].StepID != "1" {
		t.Fatalf("export StepID = %q, want 1", export.Steps[0].StepID)
	}
	for i, id := range []string{"1.1", "1.2"} {
		entry := export.Steps[0].Substeps[i]
		if entry.SubstepID != id || entry.Status != "done" {
			t.Fatalf("export leaf %d = %#v", i, entry)
		}
		if entry.Digest == "" || entry.Payload == nil {
			t.Fatalf("export leaf %s missing digest/payload: %#v", id, entry)
		}
	}

	_, err = svc.SaveDraft(context.Background(), SaveDraftCmd{
		Process: sealed, WorkflowKey: "feragame", SubstepID: "1.1",
		Substep: def.Steps[0].Substep[0], Actor: registrar,
		Payload: map[string]interface{}{
			"batches": []interface{}{
				map[string]interface{}{"loadId": "BL-X", "weightKg": "1"},
			},
		},
		Config: cfg,
	})
	if err == nil {
		t.Fatal("expected post-seal SaveDraft to be rejected")
	}

	_, err = svc.SealStep(context.Background(), SealStepCmd{
		Process: sealed, WorkflowKey: "feragame", StepID: "1",
		Actor: supervisor, Config: cfg, Confirm: true, Now: fixedNow,
	})
	if !errors.Is(err, ErrSealAlreadySealed) {
		t.Fatalf("second SealStep error = %v, want ErrSealAlreadySealed", err)
	}
	if len(store.Notarizations()) != 2 {
		t.Fatalf("second seal must not add notarizations, got %d", len(store.Notarizations()))
	}
}

func TestDraftAndSealSealStepEmptyKinds(t *testing.T) {
	fixedNow := time.Date(2026, 10, 9, 19, 0, 0, 0, time.UTC)
	def := feragameDraftAndSealDef()
	cfg := RuntimeConfig{Workflow: def}
	store := NewMemoryStore()
	svc := &ProcessService{store: store, now: func() time.Time { return fixedNow }}
	process := seedFeragameProcess(t, store, def)

	// One kind drafted empty; the other never drafted — both freeze.
	updated, err := svc.SaveDraft(context.Background(), SaveDraftCmd{
		Process: process, WorkflowKey: "feragame", SubstepID: "1.1",
		Substep: def.Steps[0].Substep[0],
		Actor:   Actor{ID: "u1", Role: "registrar", RoleSlugs: []string{"registrar"}, OrgSlug: "feragame"},
		Payload: map[string]interface{}{"batches": []interface{}{}},
		Config:  cfg,
	})
	if err != nil {
		t.Fatalf("empty draft: %v", err)
	}

	sealed, err := svc.SealStep(context.Background(), SealStepCmd{
		Process: updated, WorkflowKey: "feragame", StepID: "1",
		Actor:  Actor{ID: "admin", Role: "org-admin", RoleSlugs: []string{"org-admin"}, OrgSlug: "feragame"},
		Config: cfg, Confirm: true, Now: fixedNow,
	})
	if err != nil {
		t.Fatalf("SealStep empty kinds: %v", err)
	}
	if sealed.Status != processStatusDone {
		t.Fatalf("status = %q, want done", sealed.Status)
	}
	if sealed.Progress["1.1"].State != "done" || sealed.Progress["1.2"].State != "done" {
		t.Fatalf("expected both kinds done, got %#v", sealed.Progress)
	}
	if got, ok := sealed.Progress["1.1"].Data["batches"].([]interface{}); !ok || len(got) != 0 {
		t.Fatalf("1.1 empty batches = %#v", sealed.Progress["1.1"].Data)
	}
	if got, ok := sealed.Progress["1.2"].Data["udc"].([]interface{}); !ok || len(got) != 0 {
		t.Fatalf("1.2 never-drafted kind should freeze empty udc array, got %#v", sealed.Progress["1.2"].Data)
	}
	notaries := store.Notarizations()
	if len(notaries) != 2 {
		t.Fatalf("expected 2 notarizations for empty-kind seal, got %d", len(notaries))
	}
	bySubstep := map[string]Notarization{}
	for _, n := range notaries {
		bySubstep[n.SubstepID] = n
	}
	wantEmptyUDC := map[string]interface{}{"udc": []interface{}{}}
	if bySubstep["1.2"].FakeNotary.Digest != digestPayload(wantEmptyUDC) {
		t.Fatalf("1.2 notarization digest should cover empty udc array payload, got payload %#v digest %q",
			bySubstep["1.2"].Payload, bySubstep["1.2"].FakeNotary.Digest)
	}
}

func TestDraftAndSealRejectsDraftOnDoneSubstep(t *testing.T) {
	def := feragameDraftAndSealDef()
	cfg := RuntimeConfig{Workflow: def}
	store := NewMemoryStore()
	svc := &ProcessService{store: store, now: time.Now}
	process := seedFeragameProcess(t, store, def)
	process.Progress["1.1"] = ProcessStep{State: "done", Data: map[string]interface{}{"batches": []interface{}{}}}

	_, err := svc.SaveDraft(context.Background(), SaveDraftCmd{
		Process: process, WorkflowKey: "feragame", SubstepID: "1.1",
		Substep: def.Steps[0].Substep[0], Actor: Actor{ID: "u1"},
		Payload: map[string]interface{}{"batches": []interface{}{}}, Config: cfg,
	})
	if err == nil {
		t.Fatal("expected error drafting a sealed/done Substep")
	}
}

func TestDraftAndSealSealStepRejectsPartialLeaves(t *testing.T) {
	fixedNow := time.Date(2026, 10, 9, 20, 0, 0, 0, time.UTC)
	def := feragameDraftAndSealDef()
	cfg := RuntimeConfig{Workflow: def}
	store := NewMemoryStore()
	svc := &ProcessService{store: store, now: func() time.Time { return fixedNow }}
	process := seedFeragameProcess(t, store, def)

	// Simulate a torn seal: one leaf already done+notarized, peer still drafting.
	admin := Actor{ID: "admin", Role: "org-admin", RoleSlugs: []string{"org-admin"}, OrgSlug: "feragame"}
	process.Progress["1.1"] = ProcessStep{
		State: "done", DoneAt: &fixedNow, DoneBy: &admin,
		Data: map[string]interface{}{"batches": []interface{}{}},
	}
	process.Progress["1.2"] = ProcessStep{
		State: progressStateDraft,
		Data: map[string]interface{}{
			"udc": []interface{}{
				map[string]interface{}{"station": "S-1", "weightKg": "1"},
			},
		},
	}
	if err := store.UpdateProcessProgress(context.Background(), process.ID, "feragame", "1.1", process.Progress["1.1"]); err != nil {
		t.Fatalf("seed 1.1: %v", err)
	}
	if err := store.UpdateProcessProgress(context.Background(), process.ID, "feragame", "1.2", process.Progress["1.2"]); err != nil {
		t.Fatalf("seed 1.2: %v", err)
	}
	if err := store.InsertNotarization(context.Background(), Notarization{
		ProcessID: process.ID, SubstepID: "1.1", Payload: process.Progress["1.1"].Data,
		Actor: admin, CreatedAt: fixedNow,
		FakeNotary: FakeNotary{Method: "sha256", Digest: digestPayload(process.Progress["1.1"].Data)},
	}); err != nil {
		t.Fatalf("seed notarization: %v", err)
	}
	process, err := store.LoadProcessByID(context.Background(), process.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	process.Progress = normalizeProgressKeys(process.Progress)

	_, err = svc.SealStep(context.Background(), SealStepCmd{
		Process: process, WorkflowKey: "feragame", StepID: "1",
		Actor: admin, Config: cfg, Confirm: true, Now: fixedNow,
	})
	if !errors.Is(err, ErrSealAlreadySealed) {
		t.Fatalf("partial Step SealStep error = %v, want ErrSealAlreadySealed", err)
	}
	reloaded, err := store.LoadProcessByID(context.Background(), process.ID)
	if err != nil {
		t.Fatalf("LoadProcessByID: %v", err)
	}
	reloaded.Progress = normalizeProgressKeys(reloaded.Progress)
	if reloaded.Progress["1.2"].State == "done" {
		t.Fatal("Seal must not finish the remainder of a torn Step")
	}
	if len(store.Notarizations()) != 1 {
		t.Fatalf("partial Seal must not add peer notarization, got %d", len(store.Notarizations()))
	}
}

func TestDraftAndSealSealStepAtomicOrFailClosed(t *testing.T) {
	fixedNow := time.Date(2026, 10, 9, 21, 0, 0, 0, time.UTC)
	def := feragameDraftAndSealDef()
	cfg := RuntimeConfig{Workflow: def}
	base := NewMemoryStore()
	// failOnCall 1: Seal writes progress as one batch; a failed batch leaves no leaf done.
	store := &failAfterProgressStore{MemoryStore: base, failOnCall: 1}
	svc := &ProcessService{store: store, now: func() time.Time { return fixedNow }}
	process := seedFeragameProcess(t, base, def)

	admin := Actor{ID: "admin", Role: "org-admin", RoleSlugs: []string{"org-admin"}, OrgSlug: "feragame"}
	_, err := svc.SealStep(context.Background(), SealStepCmd{
		Process: process, WorkflowKey: "feragame", StepID: "1",
		Actor: admin, Config: cfg, Confirm: true, Now: fixedNow,
	})
	if err == nil {
		t.Fatal("expected SealStep to fail when progress persistence fails mid-seal")
	}

	reloaded, err := base.LoadProcessByID(context.Background(), process.ID)
	if err != nil {
		t.Fatalf("LoadProcessByID: %v", err)
	}
	reloaded.Progress = normalizeProgressKeys(reloaded.Progress)

	doneCount := 0
	for _, id := range []string{"1.1", "1.2"} {
		if reloaded.Progress[id].State == "done" {
			doneCount++
		}
	}
	// Prefer true all-or-nothing (0 done). Fail-closed (some done but not sealable as remainder) is also OK.
	if doneCount == 0 {
		if len(base.Notarizations()) != 0 {
			t.Fatalf("rolled-back seal must not leave notarizations, got %d", len(base.Notarizations()))
		}
		return
	}
	if doneCount == len([]string{"1.1", "1.2"}) {
		t.Fatal("mid-seal failure must not leave a fully sealed Step without returning success")
	}

	process.Progress = reloaded.Progress
	store.failOnCall = 0 // allow retry to exercise Seal policy, not the injected fault
	_, err = svc.SealStep(context.Background(), SealStepCmd{
		Process: process, WorkflowKey: "feragame", StepID: "1",
		Actor: admin, Config: cfg, Confirm: true, Now: fixedNow,
	})
	if !errors.Is(err, ErrSealAlreadySealed) {
		t.Fatalf("retry after torn seal error = %v, want ErrSealAlreadySealed (not seal-the-remainder)", err)
	}
	after, err := base.LoadProcessByID(context.Background(), process.ID)
	if err != nil {
		t.Fatalf("reload after retry: %v", err)
	}
	after.Progress = normalizeProgressKeys(after.Progress)
	stillOpen := 0
	for _, id := range []string{"1.1", "1.2"} {
		if after.Progress[id].State != "done" {
			stillOpen++
		}
	}
	if stillOpen == 0 {
		t.Fatal("fail-closed retry must not complete the remainder of a torn seal")
	}
}

// failAfterProgressStore fails UpdateProcessProgress on the Nth call (1-based).
// When Seal uses UpdateProcessProgressBatch, that single call is counted here too.
type failAfterProgressStore struct {
	*MemoryStore
	failOnCall int
	calls      int
}

func (s *failAfterProgressStore) UpdateProcessProgress(ctx context.Context, id primitive.ObjectID, workflowKey, substepID string, progress ProcessStep) error {
	s.calls++
	if s.failOnCall > 0 && s.calls >= s.failOnCall {
		return errors.New("injected progress failure")
	}
	return s.MemoryStore.UpdateProcessProgress(ctx, id, workflowKey, substepID, progress)
}

func (s *failAfterProgressStore) UpdateProcessProgressBatch(ctx context.Context, id primitive.ObjectID, workflowKey string, progressBySubstepID map[string]ProcessStep) error {
	s.calls++
	if s.failOnCall > 0 && s.calls >= s.failOnCall {
		return errors.New("injected progress failure")
	}
	return s.MemoryStore.UpdateProcessProgressBatch(ctx, id, workflowKey, progressBySubstepID)
}
