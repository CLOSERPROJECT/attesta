package main

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestBuildTimelineFormataDisplay(t *testing.T) {
	cfg := testRuntimeConfig()
	process := &Process{
		ID:        primitive.NewObjectID(),
		CreatedAt: time.Now().UTC(),
		Status:    "active",
		Progress: map[string]ProcessStep{
			"1.1": {State: "done", Data: map[string]interface{}{"value": 10.0}},
			"1.2": {State: "done", Data: map[string]interface{}{"note": "batch-1"}},
		},
	}

	timeline := buildTimeline(cfg.Workflow, process, "workflow", map[roleMetaKey]RoleMeta{}, nil, nil)
	if len(timeline) == 0 || len(timeline[0].Substeps) < 2 {
		t.Fatalf("unexpected timeline shape: %#v", timeline)
	}

	valueEntry := timeline[0].Substeps[1]
	if valueEntry.DisplayValue != "map[note:batch-1]" {
		t.Fatalf("expected formata display value, got %q", valueEntry.DisplayValue)
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
				DoneBy: &Actor{ID: "legacy-user", Role: "dep1"},
				Data:   map[string]interface{}{"value": 10.0},
			},
		},
	}

	timeline := buildTimeline(cfg.Workflow, process, "workflow", map[roleMetaKey]RoleMeta{}, nil, nil)
	if len(timeline) == 0 || len(timeline[0].Substeps) == 0 {
		t.Fatalf("unexpected timeline shape: %#v", timeline)
	}
	entry := timeline[0].Substeps[0]
	if entry.DoneBy != "legacy-user" || entry.DoneRole != "dep1" {
		t.Fatalf("unexpected legacy actor render: doneBy=%q doneRole=%q", entry.DoneBy, entry.DoneRole)
	}
	if entry.DoneAt != "26 Feb 2026 at 10:00 UTC" {
		t.Fatalf("doneAt = %q, want %q", entry.DoneAt, "26 Feb 2026 at 10:00 UTC")
	}
}

func TestBuildTimelineTerminatedStreamStatuses(t *testing.T) {
	cfg := testRuntimeConfig()
	process := &Process{
		ID: primitive.NewObjectID(),
		Progress: map[string]ProcessStep{
			"1.1": {State: "done", Data: map[string]interface{}{"value": 10.0}},
		},
		Termination: &ProcessTermination{SubstepID: "1.2"},
	}

	timeline := buildTimeline(cfg.Workflow, process, "workflow", map[roleMetaKey]RoleMeta{}, nil, nil)
	if len(timeline) == 0 || len(timeline[0].Substeps) < 3 {
		t.Fatalf("unexpected timeline shape: %#v", timeline)
	}
	if got := timeline[0].Substeps[1].Status; got != processStatusTerminated {
		t.Fatalf("terminated status = %q, want %s", got, processStatusTerminated)
	}
	if got := timeline[0].Substeps[2].Status; got != "skipped" {
		t.Fatalf("skipped status = %q, want skipped", got)
	}
}

func TestBuildTimelineUsesRoleStyleForAvailableSubsteps(t *testing.T) {
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
						InputType: "formata",
					},
				},
			},
		},
	}
	process := &Process{
		ID:       primitive.NewObjectID(),
		Progress: map[string]ProcessStep{},
	}
	roleMeta := testRoleIndexForOrg("", map[string]RoleMeta{
		"dep1": {ID: "dep1", Label: "Department 1", Palette: "red"},
		"dep2": {ID: "dep2", Label: "Department 2", Palette: "orange"},
	})

	timeline := buildTimeline(def, process, "workflow", roleMeta, nil, nil)
	if len(timeline) == 0 || len(timeline[0].Substeps) == 0 {
		t.Fatalf("unexpected timeline shape: %#v", timeline)
	}
	entry := timeline[0].Substeps[0]
	if entry.Palette != "red" {
		t.Fatalf("unexpected role palette: %q", entry.Palette)
	}
}

func TestBuildTimelineDoneSubstepUsesSelectedRoleStyle(t *testing.T) {
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
						InputType: "formata",
					},
				},
			},
		},
	}
	doneAt := time.Date(2026, 2, 26, 10, 0, 0, 0, time.UTC)
	process := &Process{
		ID: primitive.NewObjectID(),
		Progress: map[string]ProcessStep{
			"1.1": {
				State:  "done",
				DoneAt: &doneAt,
				DoneBy: &Actor{ID: "u2", Role: "dep2"},
				Data:   map[string]interface{}{"value": "ok"},
			},
		},
	}
	roleMeta := testRoleIndexForOrg("", map[string]RoleMeta{
		"dep1": {ID: "dep1", Label: "Department 1", Palette: "red"},
		"dep2": {ID: "dep2", Label: "Department 2", Palette: "orange"},
	})

	timeline := buildTimeline(def, process, "workflow", roleMeta, nil, nil)
	if len(timeline) == 0 || len(timeline[0].Substeps) == 0 {
		t.Fatalf("unexpected timeline shape: %#v", timeline)
	}
	entry := timeline[0].Substeps[0]
	if entry.Status != "done" {
		t.Fatalf("status = %q, want done", entry.Status)
	}
	if entry.Palette != "orange" {
		t.Fatalf("unexpected role palette: %q", entry.Palette)
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
					{SubstepID: "1.1", Title: "A", Order: 1, Role: "dep1", InputKey: "value", InputType: "formata"},
				},
			},
		},
	}
	process := &Process{
		ID:       primitive.NewObjectID(),
		Progress: map[string]ProcessStep{"1.1": {State: "pending"}},
	}

	timeline := buildTimeline(def, process, "workflow", map[roleMetaKey]RoleMeta{}, nil, map[string]string{"org-acme": "Acme Org"})
	if len(timeline) != 1 {
		t.Fatalf("timeline len = %d, want 1", len(timeline))
	}
	if timeline[0].OrgSlug != "org-acme" {
		t.Fatalf("timeline org slug = %q, want %q", timeline[0].OrgSlug, "org-acme")
	}
	if timeline[0].Summary.OrganizationName != "Acme Org" {
		t.Fatalf("timeline org name = %q, want %q", timeline[0].Summary.OrganizationName, "Acme Org")
	}
}

func TestDecorateTimelineOrganizationLogos(t *testing.T) {
	timeline := []TimelineStep{
		{OrgSlug: "org-acme", Summary: StepSummaryView{StepID: "1", OrganizationName: "Acme Org"}},
		{OrgSlug: "org-beta", Summary: StepSummaryView{StepID: "2", OrganizationName: "Beta Org"}},
	}

	decorated := decorateTimelineOrganizationLogos(timeline, map[string]string{
		"org-acme": "/organization/logo/org-acme",
	})

	if decorated[0].Summary.OrgLogoURL != "/organization/logo/org-acme" {
		t.Fatalf("first timeline org logo url = %q", decorated[0].Summary.OrgLogoURL)
	}
	if decorated[1].Summary.OrgLogoURL != "" {
		t.Fatalf("second timeline org logo url = %q, want empty", decorated[1].Summary.OrgLogoURL)
	}
}
