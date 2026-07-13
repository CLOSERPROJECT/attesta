// process_completion_test.go
package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestProcessCompletionSentinelErrors(t *testing.T) {
	if ErrProgressUpdate == nil || ErrNotarization == nil {
		t.Fatal("expected sentinel errors to be defined")
	}
	if ErrProgressUpdate.Error() != "process: progress update failed" {
		t.Fatalf("unexpected ErrProgressUpdate message: %q", ErrProgressUpdate.Error())
	}
	if !errors.Is(ErrProgressUpdate, ErrProgressUpdate) {
		t.Fatal("expected errors.Is to match ErrProgressUpdate")
	}
}

func TestEnsureCompletionArtifactsUpdatesDoneStatusAndDPP(t *testing.T) {
	fixedNow := time.Date(2026, 2, 19, 10, 0, 0, 0, time.UTC)
	def := WorkflowDef{
		Steps: []WorkflowStep{{
			StepID: "1",
			Substep: []WorkflowSub{
				{SubstepID: "1.1", Order: 1, Role: "dep1", InputKey: "value", InputType: "formata"},
			},
		}},
	}
	cfg := RuntimeConfig{
		Workflow: def,
		DPP: DPPConfig{
			Enabled:        true,
			GTIN:           "09506000134352",
			LotDefault:     "LOT-DEFAULT",
			SerialStrategy: "process_id_hex",
		},
	}
	store := NewMemoryStore()
	svc := &ProcessService{store: store, now: func() time.Time { return fixedNow }}

	processID := primitive.NewObjectID()
	store.SeedProcess(Process{
		ID:     processID,
		Status: "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "done", Data: map[string]interface{}{"value": "ok"}},
		},
	})
	process, err := store.LoadProcessByID(context.Background(), processID)
	if err != nil {
		t.Fatalf("LoadProcessByID: %v", err)
	}
	process.Progress = normalizeProgressKeys(process.Progress)

	updated := svc.EnsureCompletionArtifacts(context.Background(), cfg, "workflow", process)
	if updated.Status != "done" {
		t.Fatalf("expected status done, got %q", updated.Status)
	}
	if updated.DPP == nil {
		t.Fatal("expected DPP to be generated")
	}
	if updated.DPP.GTIN != "09506000134352" || updated.DPP.Lot != "LOT-DEFAULT" || updated.DPP.Serial != processID.Hex() {
		t.Fatalf("unexpected dpp: %#v", updated.DPP)
	}
}

func TestEnsureCompletionArtifactsNoopForNilAndPending(t *testing.T) {
	svc := &ProcessService{store: NewMemoryStore()}
	cfg := RuntimeConfig{Workflow: WorkflowDef{Steps: []WorkflowStep{{
		StepID:  "1",
		Substep: []WorkflowSub{{SubstepID: "1.1", Order: 1}},
	}}}}

	if got := svc.EnsureCompletionArtifacts(context.Background(), cfg, "workflow", nil); got != nil {
		t.Fatalf("expected nil passthrough, got %#v", got)
	}
	pending := &Process{ID: primitive.NewObjectID(), Progress: map[string]ProcessStep{"1.1": {State: "pending"}}}
	if got := svc.EnsureCompletionArtifacts(context.Background(), cfg, "workflow", pending); got != pending {
		t.Fatalf("expected pending passthrough, got %#v", got)
	}
}

func TestCompleteSubstepMarksProgressAndNotarizes(t *testing.T) {
	fixedNow := time.Date(2026, 2, 2, 14, 0, 0, 0, time.UTC)
	def := WorkflowDef{
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
		ID:     processID,
		Status: "active",
		Progress: map[string]ProcessStep{
			"1_1": {State: "pending"},
		},
	})
	process, err := store.LoadProcessByID(context.Background(), processID)
	if err != nil {
		t.Fatalf("LoadProcessByID: %v", err)
	}
	process.Progress = normalizeProgressKeys(process.Progress)

	payload := map[string]interface{}{"status": "ok"}
	actor := Actor{ID: "user-1", Role: "dep1", RoleSlugs: []string{"dep1"}}
	updated, err := svc.CompleteSubstep(context.Background(), CompleteSubstepCmd{
		Process:     process,
		WorkflowKey: "workflow",
		SubstepID:   "1.1",
		Substep:     def.Steps[0].Substep[0],
		Actor:       actor,
		Payload:     payload,
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
	notaries := store.Notarizations()
	if len(notaries) != 1 {
		t.Fatalf("expected 1 notarization, got %d", len(notaries))
	}
	if notaries[0].SubstepID != "1.1" || notaries[0].Actor.ID != "user-1" {
		t.Fatalf("unexpected notarization: %#v", notaries[0])
	}
}
