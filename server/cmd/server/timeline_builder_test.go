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
				DoneBy: &Actor{ID: "legacy-user", Role: "dep1"},
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
	if entry.DoneAt != "26 Feb 2026 at 10:00 UTC" {
		t.Fatalf("doneAt = %q, want %q", entry.DoneAt, "26 Feb 2026 at 10:00 UTC")
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
	if string(entry.RoleColor) != "#aaaaaa" || string(entry.RoleBorder) != "#111111" {
		t.Fatalf("unexpected role style values: color=%q border=%q", entry.RoleColor, entry.RoleBorder)
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
						InputType: "string",
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
	roleMeta := map[string]RoleMeta{
		"dep1": {ID: "dep1", Label: "Department 1", Color: "#aaaaaa", Border: "#111111"},
		"dep2": {ID: "dep2", Label: "Department 2", Color: "#bbbbbb", Border: "#222222"},
	}

	timeline := buildTimeline(def, process, "workflow", roleMeta, nil)
	if len(timeline) == 0 || len(timeline[0].Substeps) == 0 {
		t.Fatalf("unexpected timeline shape: %#v", timeline)
	}
	entry := timeline[0].Substeps[0]
	if entry.Status != "done" {
		t.Fatalf("status = %q, want done", entry.Status)
	}
	if string(entry.RoleColor) != "#bbbbbb" || string(entry.RoleBorder) != "#222222" {
		t.Fatalf("unexpected role style values: color=%q border=%q", entry.RoleColor, entry.RoleBorder)
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
	if timeline[0].OrgSlug != "org-acme" {
		t.Fatalf("timeline org slug = %q, want %q", timeline[0].OrgSlug, "org-acme")
	}
	if timeline[0].OrgName != "Acme Org" {
		t.Fatalf("timeline org name = %q, want %q", timeline[0].OrgName, "Acme Org")
	}
}

func TestDecorateTimelineOrganizationLogos(t *testing.T) {
	timeline := []TimelineStep{
		{StepID: "1", OrgSlug: "org-acme", OrgName: "Acme Org"},
		{StepID: "2", OrgSlug: "org-beta", OrgName: "Beta Org"},
	}

	decorated := decorateTimelineOrganizationLogos(timeline, map[string]string{
		"org-acme": "/organization/logo/org-acme",
	})

	if decorated[0].OrgLogoURL != "/organization/logo/org-acme" {
		t.Fatalf("first timeline org logo url = %q", decorated[0].OrgLogoURL)
	}
	if decorated[1].OrgLogoURL != "" {
		t.Fatalf("second timeline org logo url = %q, want empty", decorated[1].OrgLogoURL)
	}
}
