package main

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestSortHomeProcessList(t *testing.T) {
	base := time.Date(2026, 2, 4, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		sortKey string
		items   []ProcessListItem
		wantIDs []string
	}{
		{
			name:    "time asc",
			sortKey: "time_asc",
			items: []ProcessListItem{
				{ID: "a", CreatedAtTime: base.Add(2 * time.Hour)},
				{ID: "b", CreatedAtTime: base.Add(-1 * time.Hour)},
				{ID: "c", CreatedAtTime: base},
			},
			wantIDs: []string{"b", "c", "a"},
		},
		{
			name:    "time desc default",
			sortKey: "unknown",
			items: []ProcessListItem{
				{ID: "a", CreatedAtTime: base.Add(2 * time.Hour)},
				{ID: "b", CreatedAtTime: base.Add(-1 * time.Hour)},
				{ID: "c", CreatedAtTime: base},
			},
			wantIDs: []string{"a", "c", "b"},
		},
		{
			name:    "progress asc with tie by recent time",
			sortKey: "progress_asc",
			items: []ProcessListItem{
				{ID: "a", Percent: 20, CreatedAtTime: base.Add(2 * time.Hour)},
				{ID: "b", Percent: 10, CreatedAtTime: base.Add(-1 * time.Hour)},
				{ID: "c", Percent: 20, CreatedAtTime: base.Add(1 * time.Hour)},
			},
			wantIDs: []string{"b", "a", "c"},
		},
		{
			name:    "progress desc with tie by recent time",
			sortKey: "progress_desc",
			items: []ProcessListItem{
				{ID: "a", Percent: 80, CreatedAtTime: base.Add(2 * time.Hour)},
				{ID: "b", Percent: 90, CreatedAtTime: base.Add(-1 * time.Hour)},
				{ID: "c", Percent: 80, CreatedAtTime: base.Add(1 * time.Hour)},
			},
			wantIDs: []string{"b", "a", "c"},
		},
		{
			name:    "status ordering with active first",
			sortKey: "status",
			items: []ProcessListItem{
				{ID: "a", Status: "done", Percent: 100, CreatedAtTime: base},
				{ID: "b", Status: "active", Percent: 20, CreatedAtTime: base.Add(-1 * time.Hour)},
				{ID: "c", Status: "blocked", Percent: 10, CreatedAtTime: base.Add(1 * time.Hour)},
			},
			wantIDs: []string{"b", "c", "a"},
		},
		{
			name:    "status tie uses percent then time",
			sortKey: "status",
			items: []ProcessListItem{
				{ID: "a", Status: "done", Percent: 20, CreatedAtTime: base},
				{ID: "b", Status: "done", Percent: 90, CreatedAtTime: base.Add(-1 * time.Hour)},
				{ID: "c", Status: "done", Percent: 20, CreatedAtTime: base.Add(1 * time.Hour)},
			},
			wantIDs: []string{"b", "c", "a"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			items := append([]ProcessListItem(nil), tc.items...)
			sortHomeProcessList(items, tc.sortKey)
			for i, wantID := range tc.wantIDs {
				if items[i].ID != wantID {
					t.Fatalf("index %d id = %q, want %q", i, items[i].ID, wantID)
				}
			}
		})
	}
}

func TestBuildProcessSummary(t *testing.T) {
	def := testRuntimeConfig().Workflow
	now := time.Date(2026, 2, 4, 11, 30, 0, 0, time.UTC)

	active := processWithDone("1.1")
	active.ID = primitive.NewObjectID()
	active.CreatedAt = now
	summary := buildProcessSummary(def, active, "active")
	if summary.ID != active.ID.Hex() {
		t.Fatalf("summary.ID = %q, want %q", summary.ID, active.ID.Hex())
	}
	if summary.Status != "active" {
		t.Fatalf("summary.Status = %q, want active", summary.Status)
	}
	if summary.NextSubstep != "1.2" {
		t.Fatalf("summary.NextSubstep = %q, want 1.2", summary.NextSubstep)
	}
	if summary.NextTitle != "B" || summary.NextRole != "dep1" {
		t.Fatalf("unexpected next summary fields: %#v", summary)
	}

	done := processWithDone("1.1", "1.2", "1.3", "2.1", "2.2", "3.1", "3.2")
	done.ID = primitive.NewObjectID()
	done.CreatedAt = now
	doneSummary := buildProcessSummary(def, done, "done")
	if doneSummary.NextSubstep != "" || doneSummary.NextTitle != "" || doneSummary.NextRole != "" {
		t.Fatalf("expected no next action for done process, got %#v", doneSummary)
	}
}

func TestRoleMetaMapAndActorForRole(t *testing.T) {
	server := &Server{}
	cfg := RuntimeConfig{
		Departments: []Department{
			{ID: "dep1", Name: "Department 1", Color: "#aabbcc", Border: "#112233"},
			{ID: "dep2"},
		},
		Users: []User{
			{ID: "u1", DepartmentID: "dep1"},
		},
	}

	meta := server.roleMetaMap(cfg)
	if meta["dep1"].Label != "Department 1" || meta["dep1"].Color != "#aabbcc" || meta["dep1"].Border != "#112233" {
		t.Fatalf("dep1 role meta mismatch: %#v", meta["dep1"])
	}
	if meta["dep2"].Label != "dep2" {
		t.Fatalf("dep2 label = %q, want dep2", meta["dep2"].Label)
	}
	if meta["dep2"].Color != "#f0f3ea" || meta["dep2"].Border != "#d9e0d0" {
		t.Fatalf("dep2 defaults mismatch: %#v", meta["dep2"])
	}

	known := server.actorForRole(cfg, "dep1", "workflow")
	if known.UserID != "u1" || known.Role != "dep1" {
		t.Fatalf("known actor mismatch: %#v", known)
	}

	unknown := server.actorForRole(cfg, "depX", "workflow")
	if unknown.UserID != "depX" || unknown.Role != "depX" {
		t.Fatalf("unknown actor mismatch: %#v", unknown)
	}

	empty := server.actorForRole(cfg, "", "workflow")
	if empty.UserID != "unknown" || empty.Role != "unknown" {
		t.Fatalf("empty role fallback mismatch: %#v", empty)
	}
}

func TestDefaultRole(t *testing.T) {
	server := &Server{}
	cfg := RuntimeConfig{
		Departments: []Department{{ID: "dep1"}, {ID: "dep2"}},
	}
	if got := server.defaultRole(cfg); got != "dep1" {
		t.Fatalf("defaultRole = %q, want dep1", got)
	}

	if got := server.defaultRole(RuntimeConfig{}); got != "" {
		t.Fatalf("defaultRole for empty config = %q, want empty", got)
	}
}
