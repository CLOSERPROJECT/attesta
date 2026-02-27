package main

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestBuildTimelineFileSubstepDisplay(t *testing.T) {
	cfg := testRuntimeConfig()
	processID := primitive.NewObjectID()
	process := &Process{
		ID:        processID,
		CreatedAt: time.Now().UTC(),
		Status:    "active",
		Progress: map[string]ProcessStep{
			"1.1": {State: "done", Data: map[string]interface{}{"value": 10.0}},
			"1.2": {State: "done", Data: map[string]interface{}{"note": "batch-1"}},
			"1.3": {
				State: "done",
				Data: map[string]interface{}{
					"attachment": map[string]interface{}{
						"attachmentId": primitive.NewObjectID().Hex(),
						"filename":     "cert.pdf",
						"sha256":       "abc",
					},
				},
			},
		},
	}

	timeline := buildTimeline(cfg.Workflow, process, "workflow", map[string]RoleMeta{}, nil)
	if len(timeline) == 0 || len(timeline[0].Substeps) < 3 {
		t.Fatalf("unexpected timeline shape: %#v", timeline)
	}

	fileEntry := timeline[0].Substeps[2]
	if fileEntry.SubstepID != "1.3" {
		t.Fatalf("expected third substep to be 1.3, got %q", fileEntry.SubstepID)
	}
	if fileEntry.FileName != "cert.pdf" {
		t.Fatalf("expected filename cert.pdf, got %q", fileEntry.FileName)
	}
	wantURL := "/w/workflow/process/" + processID.Hex() + "/substep/1.3/file"
	if fileEntry.FileURL != wantURL {
		t.Fatalf("expected file URL %q, got %q", wantURL, fileEntry.FileURL)
	}

	valueEntry := timeline[0].Substeps[1]
	if valueEntry.DisplayValue != "batch-1" {
		t.Fatalf("expected display value batch-1, got %q", valueEntry.DisplayValue)
	}
}

func TestBuildTimelineLegacyActorWithoutOrgSlug(t *testing.T) {
	cfg := testRuntimeConfig()
	doneAt := time.Date(2026, 2, 26, 10, 0, 0, 0, time.UTC)
	process := &Process{
		ID:        primitive.NewObjectID(),
		CreatedAt: time.Now().UTC(),
		Status:    "active",
		Progress: map[string]ProcessStep{
			"1.1": {
				State:  "done",
				DoneAt: &doneAt,
				// Legacy actor shape: no orgSlug/roleSlugs fields.
				DoneBy: &Actor{UserID: "legacy-user", Role: "dep1"},
				Data:   map[string]interface{}{"value": 10.0},
			},
		},
	}

	timeline := buildTimeline(cfg.Workflow, process, "workflow", map[string]RoleMeta{}, nil)
	if len(timeline) == 0 || len(timeline[0].Substeps) == 0 {
		t.Fatalf("unexpected timeline shape: %#v", timeline)
	}
	entry := timeline[0].Substeps[0]
	if entry.DoneBy != "legacy-user" || entry.DoneRole != "dep1" {
		t.Fatalf("unexpected legacy actor render: doneBy=%q doneRole=%q", entry.DoneBy, entry.DoneRole)
	}
}

func TestBuildTimelineIncludesAllAllowedRoleBadges(t *testing.T) {
	def := WorkflowDef{
		Steps: []WorkflowStep{
			{
				StepID: "1",
				Title:  "Step 1",
				Order:  1,
				Substep: []WorkflowSub{
					{
						SubstepID: "1.1",
						Title:     "Multi Role Substep",
						Order:     1,
						Roles:     []string{"dep1", "dep2"},
						InputKey:  "value",
						InputType: "string",
					},
				},
			},
		},
	}
	process := &Process{
		ID:       primitive.NewObjectID(),
		Progress: map[string]ProcessStep{},
	}
	roleMeta := map[string]RoleMeta{
		"dep1": {ID: "dep1", Label: "Department 1", Color: "#aaaaaa", Border: "#111111"},
		"dep2": {ID: "dep2", Label: "Department 2", Color: "#bbbbbb", Border: "#222222"},
	}

	timeline := buildTimeline(def, process, "workflow", roleMeta, nil)
	if len(timeline) == 0 || len(timeline[0].Substeps) == 0 {
		t.Fatalf("unexpected timeline shape: %#v", timeline)
	}
	entry := timeline[0].Substeps[0]
	if len(entry.RoleBadges) != 2 {
		t.Fatalf("role badge count = %d, want 2", len(entry.RoleBadges))
	}
	if entry.RoleBadges[0].ID != "dep1" || entry.RoleBadges[1].ID != "dep2" {
		t.Fatalf("unexpected role badges: %#v", entry.RoleBadges)
	}
	if entry.Role != "dep1, dep2" {
		t.Fatalf("role summary = %q, want %q", entry.Role, "dep1, dep2")
	}
}

func TestBuildTimelineUsesOrganizationNameInStep(t *testing.T) {
	def := WorkflowDef{
		Steps: []WorkflowStep{
			{
				StepID:           "1",
				Title:            "Step 1",
				Order:            1,
				OrganizationSlug: "org-acme",
				Substep: []WorkflowSub{
					{SubstepID: "1.1", Title: "A", Order: 1, Role: "dep1", InputKey: "value", InputType: "string"},
				},
			},
		},
	}
	process := &Process{
		ID:       primitive.NewObjectID(),
		Progress: map[string]ProcessStep{"1.1": {State: "pending"}},
	}

	timeline := buildTimeline(def, process, "workflow", map[string]RoleMeta{}, map[string]string{"org-acme": "Acme Org"})
	if len(timeline) != 1 {
		t.Fatalf("timeline len = %d, want 1", len(timeline))
	}
	if timeline[0].OrgName != "Acme Org" {
		t.Fatalf("timeline org name = %q, want %q", timeline[0].OrgName, "Acme Org")
	}
}
