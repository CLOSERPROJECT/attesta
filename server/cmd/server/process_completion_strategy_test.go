package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"gopkg.in/yaml.v3"
)

func TestNormalizeCompletionStrategy(t *testing.T) {
	t.Run("empty defaults to classic", func(t *testing.T) {
		def := WorkflowDef{}
		if err := normalizeCompletionStrategy(&def); err != nil {
			t.Fatalf("normalizeCompletionStrategy: %v", err)
		}
		if def.CompletionStrategy != CompletionStrategyClassic {
			t.Fatalf("CompletionStrategy = %q, want %q", def.CompletionStrategy, CompletionStrategyClassic)
		}
	})

	t.Run("classic preserved", func(t *testing.T) {
		def := WorkflowDef{CompletionStrategy: "Classic"}
		if err := normalizeCompletionStrategy(&def); err != nil {
			t.Fatalf("normalizeCompletionStrategy: %v", err)
		}
		if def.CompletionStrategy != CompletionStrategyClassic {
			t.Fatalf("CompletionStrategy = %q, want %q", def.CompletionStrategy, CompletionStrategyClassic)
		}
	})

	t.Run("draft-and-seal parseable", func(t *testing.T) {
		def := WorkflowDef{CompletionStrategy: " draft-and-seal "}
		if err := normalizeCompletionStrategy(&def); err != nil {
			t.Fatalf("normalizeCompletionStrategy: %v", err)
		}
		if def.CompletionStrategy != CompletionStrategyDraftAndSeal {
			t.Fatalf("CompletionStrategy = %q, want %q", def.CompletionStrategy, CompletionStrategyDraftAndSeal)
		}
	})

	t.Run("unknown rejected", func(t *testing.T) {
		def := WorkflowDef{CompletionStrategy: "parallel-events"}
		err := normalizeCompletionStrategy(&def)
		if err == nil {
			t.Fatal("expected error for unknown strategy")
		}
		if !strings.Contains(err.Error(), "unsupported completionStrategy") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestWorkflowCompletionStrategySelection(t *testing.T) {
	cases := []struct {
		name string
		def  WorkflowDef
		want string
	}{
		{name: "unset", def: WorkflowDef{}, want: CompletionStrategyClassic},
		{name: "classic", def: WorkflowDef{CompletionStrategy: CompletionStrategyClassic}, want: CompletionStrategyClassic},
		{name: "draft-and-seal", def: WorkflowDef{CompletionStrategy: CompletionStrategyDraftAndSeal}, want: CompletionStrategyDraftAndSeal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := workflowCompletionStrategy(tc.def); got != tc.want {
				t.Fatalf("workflowCompletionStrategy = %q, want %q", got, tc.want)
			}
			svc := &ProcessService{store: NewMemoryStore()}
			strategy := svc.completionStrategy(tc.def)
			if strategy.Name() != tc.want {
				t.Fatalf("strategy.Name() = %q, want %q", strategy.Name(), tc.want)
			}
		})
	}
}

func TestClassicStrategyCompleteUnchanged(t *testing.T) {
	fixedNow := time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC)
	def := WorkflowDef{
		CompletionStrategy: CompletionStrategyClassic,
		Steps: []WorkflowStep{{
			StepID: "1",
			Substep: []WorkflowSub{
				{SubstepID: "1.1", Order: 1, Role: "dep1", InputKey: "value", InputType: "formata"},
			},
		}},
	}
	cfg := RuntimeConfig{Workflow: def}
	store := NewMemoryStore()
	svc := &ProcessService{store: store, now: func() time.Time { return fixedNow }}

	processID := primitive.NewObjectID()
	store.SeedProcess(Process{
		ID:       processID,
		Status:   "active",
		Progress: map[string]ProcessStep{"1_1": {State: "pending"}},
	})
	process, err := store.LoadProcessByID(context.Background(), processID)
	if err != nil {
		t.Fatalf("LoadProcessByID: %v", err)
	}
	process.Progress = normalizeProgressKeys(process.Progress)

	updated, err := svc.CompleteSubstep(context.Background(), CompleteSubstepCmd{
		Process:     process,
		WorkflowKey: "workflow",
		SubstepID:   "1.1",
		Substep:     def.Steps[0].Substep[0],
		Actor:       Actor{ID: "user-1", Role: "dep1", RoleSlugs: []string{"dep1"}},
		Payload:     map[string]interface{}{"status": "ok"},
		Config:      cfg,
		Now:         fixedNow,
	})
	if err != nil {
		t.Fatalf("CompleteSubstep: %v", err)
	}
	step, ok := updated.Progress["1.1"]
	if !ok || step.State != "done" {
		t.Fatalf("expected substep 1.1 done, got %#v", updated.Progress)
	}
	if len(store.Notarizations()) != 1 {
		t.Fatalf("expected 1 notarization, got %d", len(store.Notarizations()))
	}
}

func TestClassicStrategyAvailabilitySequential(t *testing.T) {
	def := testRuntimeConfig().Workflow
	def.CompletionStrategy = CompletionStrategyClassic

	availability := substepAvailability(def, processWithDone())
	for _, sub := range orderedSubsteps(def) {
		want := sub.SubstepID == "1.1"
		if got := availability[sub.SubstepID]; got != want {
			t.Fatalf("availability[%s]=%t, want %t", sub.SubstepID, got, want)
		}
	}
	if !substepSequenceOK(def, processWithDone(), "1.1") {
		t.Fatal("expected first substep sequence-OK under classic")
	}
	if substepSequenceOK(def, processWithDone(), "2.1") {
		t.Fatal("expected 2.1 blocked under classic when predecessors pending")
	}
}

func TestDraftAndSealCompleteAndSealGates(t *testing.T) {
	def := WorkflowDef{
		CompletionStrategy: CompletionStrategyDraftAndSeal,
		Steps: []WorkflowStep{{
			StepID: "1",
			Substep: []WorkflowSub{
				{SubstepID: "1.1", Order: 1, InputKey: "batches"},
				{SubstepID: "1.2", Order: 2, InputKey: "udc"},
			},
		}},
	}
	cfg := RuntimeConfig{Workflow: def}
	svc := &ProcessService{store: NewMemoryStore(), now: time.Now}
	process := &Process{ID: primitive.NewObjectID(), Progress: map[string]ProcessStep{"1.1": {State: "pending"}}}

	_, err := svc.SealStep(context.Background(), SealStepCmd{
		Process: process, WorkflowKey: "feragame", StepID: "1",
		Actor:  Actor{ID: "registrar", Role: "registrar", RoleSlugs: []string{"registrar"}, OrgSlug: "feragame"},
		Config: cfg, Confirm: true,
	})
	if !errors.Is(err, ErrSealUnauthorized) {
		t.Fatalf("SealStep without Org admin error = %v, want ErrSealUnauthorized", err)
	}

	_, err = svc.CompleteSubstep(context.Background(), CompleteSubstepCmd{
		Process: process, WorkflowKey: "feragame", SubstepID: "1.1",
		Substep: def.Steps[0].Substep[0], Actor: Actor{ID: "u1"},
		Payload: map[string]interface{}{"rows": []interface{}{}}, Config: cfg,
	})
	if !errors.Is(err, ErrCompletionUnsupported) {
		t.Fatalf("CompleteSubstep error = %v, want ErrCompletionUnsupported", err)
	}

	availability := substepAvailability(def, process)
	if !availability["1.1"] || !availability["1.2"] {
		t.Fatalf("draft-and-seal should expose parallel availability, got %#v", availability)
	}
	if !substepSequenceOK(def, process, "1.1") || !substepSequenceOK(def, process, "1.2") {
		t.Fatal("draft-and-seal should allow in-Step sequence for both kinds")
	}
}

func TestClassicSaveDraftAndSealUnsupported(t *testing.T) {
	svc := &ProcessService{store: NewMemoryStore()}
	cfg := RuntimeConfig{Workflow: WorkflowDef{CompletionStrategy: CompletionStrategyClassic}}
	_, err := svc.SaveDraft(context.Background(), SaveDraftCmd{Config: cfg})
	if !errors.Is(err, ErrCompletionUnsupported) {
		t.Fatalf("classic SaveDraft error = %v, want ErrCompletionUnsupported", err)
	}
	_, err = svc.SealStep(context.Background(), SealStepCmd{Config: cfg})
	if !errors.Is(err, ErrCompletionUnsupported) {
		t.Fatalf("classic SealStep error = %v, want ErrCompletionUnsupported", err)
	}
}

func TestParseRuntimeConfigCompletionStrategy(t *testing.T) {
	t.Run("omitted defaults classic", func(t *testing.T) {
		raw := []byte(`
workflow:
  name: Classic stream
  steps:
    - id: "1"
      title: Step
      order: 1
      organization: org-a
      substeps:
        - id: "1.1"
          title: Form
          order: 1
          roles: [clerk]
          inputKey: payload
          inputType: formata
          schema: {type: object}
organizations:
  - slug: org-a
    name: Org A
roles:
  - orgSlug: org-a
    slug: clerk
    name: Clerk
`)
		cfg, err := parseRuntimeConfigData("test", raw)
		if err != nil {
			t.Fatalf("parseRuntimeConfigData: %v", err)
		}
		if cfg.Workflow.CompletionStrategy != CompletionStrategyClassic {
			t.Fatalf("CompletionStrategy = %q, want %q", cfg.Workflow.CompletionStrategy, CompletionStrategyClassic)
		}
	})

	t.Run("draft-and-seal accepted", func(t *testing.T) {
		raw := []byte(`
workflow:
  name: Feragame day
  completionStrategy: draft-and-seal
  steps:
    - id: "1"
      title: Production day
      order: 1
      organization: feragame
      substeps:
        - id: "1.1"
          title: Batch loads
          order: 1
          roles: [registrar]
          inputKey: batches
          inputType: formata
          schema: {type: object}
organizations:
  - slug: feragame
    name: Feragame
roles:
  - orgSlug: feragame
    slug: registrar
    name: Registrar
`)
		cfg, err := parseRuntimeConfigData("test", raw)
		if err != nil {
			t.Fatalf("parseRuntimeConfigData: %v", err)
		}
		if cfg.Workflow.CompletionStrategy != CompletionStrategyDraftAndSeal {
			t.Fatalf("CompletionStrategy = %q, want %q", cfg.Workflow.CompletionStrategy, CompletionStrategyDraftAndSeal)
		}
	})

	t.Run("unknown rejected", func(t *testing.T) {
		raw := []byte(`
workflow:
  name: Bad
  completionStrategy: mystery
  steps:
    - id: "1"
      title: Step
      order: 1
      organization: org-a
      substeps:
        - id: "1.1"
          title: Form
          order: 1
          roles: [clerk]
          inputKey: payload
          inputType: formata
          schema: {type: object}
organizations:
  - slug: org-a
    name: Org A
roles:
  - orgSlug: org-a
    slug: clerk
    name: Clerk
`)
		_, err := parseRuntimeConfigData("test", raw)
		if err == nil {
			t.Fatal("expected parse error for unknown completionStrategy")
		}
		if !strings.Contains(err.Error(), "unsupported completionStrategy") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestWorkflowDefCompletionStrategyYAMLRoundTrip(t *testing.T) {
	type wrap struct {
		Workflow WorkflowDef `yaml:"workflow"`
	}
	var decoded wrap
	if err := yaml.Unmarshal([]byte("workflow:\n  name: X\n  completionStrategy: draft-and-seal\n"), &decoded); err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}
	if decoded.Workflow.CompletionStrategy != CompletionStrategyDraftAndSeal {
		t.Fatalf("CompletionStrategy = %q, want %q", decoded.Workflow.CompletionStrategy, CompletionStrategyDraftAndSeal)
	}
}
