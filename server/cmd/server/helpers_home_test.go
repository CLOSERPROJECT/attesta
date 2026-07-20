package main

import (
	"context"
	"testing"
	"time"
)

func TestSortHomeProcessList(t *testing.T) {
	base := time.Date(2026, 2, 4, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		sortKey string
		items   []StreamInstanceCard
		wantIDs []string
	}{
		{
			name:    "time asc",
			sortKey: "time_asc",
			items: []StreamInstanceCard{
				{ID: "a", CreatedAtTime: base.Add(2 * time.Hour)},
				{ID: "b", CreatedAtTime: base.Add(-1 * time.Hour)},
				{ID: "c", CreatedAtTime: base},
			},
			wantIDs: []string{"b", "c", "a"},
		},
		{
			name:    "time desc default",
			sortKey: "unknown",
			items: []StreamInstanceCard{
				{ID: "a", CreatedAtTime: base.Add(2 * time.Hour)},
				{ID: "b", CreatedAtTime: base.Add(-1 * time.Hour)},
				{ID: "c", CreatedAtTime: base},
			},
			wantIDs: []string{"a", "c", "b"},
		},
		{
			name:    "progress asc with tie by recent time",
			sortKey: "progress_asc",
			items: []StreamInstanceCard{
				{ID: "a", Percent: 20, CreatedAtTime: base.Add(2 * time.Hour)},
				{ID: "b", Percent: 10, CreatedAtTime: base.Add(-1 * time.Hour)},
				{ID: "c", Percent: 20, CreatedAtTime: base.Add(1 * time.Hour)},
			},
			wantIDs: []string{"b", "a", "c"},
		},
		{
			name:    "progress desc with tie by recent time",
			sortKey: "progress_desc",
			items: []StreamInstanceCard{
				{ID: "a", Percent: 80, CreatedAtTime: base.Add(2 * time.Hour)},
				{ID: "b", Percent: 90, CreatedAtTime: base.Add(-1 * time.Hour)},
				{ID: "c", Percent: 80, CreatedAtTime: base.Add(1 * time.Hour)},
			},
			wantIDs: []string{"b", "a", "c"},
		},
		{
			name:    "status ordering with active first",
			sortKey: "status",
			items: []StreamInstanceCard{
				{ID: "a", Status: "done", Percent: 100, CreatedAtTime: base},
				{ID: "b", Status: "active", Percent: 20, CreatedAtTime: base.Add(-1 * time.Hour)},
				{ID: "c", Status: "blocked", Percent: 10, CreatedAtTime: base.Add(1 * time.Hour)},
			},
			wantIDs: []string{"b", "c", "a"},
		},
		{
			name:    "status ordering puts available before active",
			sortKey: "status",
			items: []StreamInstanceCard{
				{ID: "a", Status: "done", Percent: 100, CreatedAtTime: base},
				{ID: "b", Status: "active", Percent: 20, CreatedAtTime: base.Add(-1 * time.Hour)},
				{ID: "c", Status: "available", Percent: 10, CreatedAtTime: base.Add(1 * time.Hour)},
			},
			wantIDs: []string{"c", "b", "a"},
		},
		{
			name:    "status tie uses percent then time",
			sortKey: "status",
			items: []StreamInstanceCard{
				{ID: "a", Status: "done", Percent: 20, CreatedAtTime: base},
				{ID: "b", Status: "done", Percent: 90, CreatedAtTime: base.Add(-1 * time.Hour)},
				{ID: "c", Status: "done", Percent: 20, CreatedAtTime: base.Add(1 * time.Hour)},
			},
			wantIDs: []string{"b", "c", "a"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			items := append([]StreamInstanceCard(nil), tc.items...)
			sortHomeProcessList(items, tc.sortKey)
			for i, wantID := range tc.wantIDs {
				if items[i].ID != wantID {
					t.Fatalf("index %d id = %q, want %q", i, items[i].ID, wantID)
				}
			}
		})
	}
}

func TestRoleMetaIndexUnavailableUsesFallback(t *testing.T) {
	got := roleMetaForOrg("org1", "dep1", (&Server{}).roleMetaIndex(context.Background()), nil)
	if got.Palette != "fallback" {
		t.Fatalf("palette = %q, want fallback", got.Palette)
	}
}
