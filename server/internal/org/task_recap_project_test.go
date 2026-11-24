package org

import (
	"testing"
)

func TestProjectTaskRecapMintAndPatchSameEntityKeepsMinted(t *testing.T) {
	cfg := OrgConfig{
		Jobs: OrderedObject{
			Keys: []string{"j"},
			Values: map[string]any{
				"j": Job{Tasks: []TaskConfig{{
					"id": "t",
					"recap": map[string]any{
						"entity": "lot",
						"columns": []any{
							map[string]any{"field": "client", "title": "Cliente"},
							map[string]any{"field": "netWeight"},
						},
					},
				}}},
			},
		},
	}
	domain := domainFromKinds(map[string]EntityMap{
		"lot": {
			"L1": {"client": "CO", "netWeight": 50.0, "notes": "later"},
		},
	})
	domain.Submissions = []Submission{
		{
			JobID: "j", TaskID: "t", JobInstanceID: "i1",
			Timestamp: "2026-01-01T10:00:00Z", UserID: "ada", UserName: "Ada",
			EntitiesTouched: []EntityTouch{{Op: "mint", Entity: "lot", ID: "L1"}},
			Facts:           map[string]any{"client": "CO"},
		},
		{
			JobID: "j", TaskID: "t", JobInstanceID: "i1",
			Timestamp: "2026-01-01T11:00:00Z", UserID: "bob", UserName: "Bob",
			EntitiesTouched: []EntityTouch{{Op: "patch", Entity: "lot", ID: "L1"}},
			Facts:           map[string]any{"netWeight": 50.0},
		},
	}

	table, err := ProjectTaskRecap(cfg, domain, SubmissionAddress{JobID: "j", TaskID: "t", JobInstanceID: "i1"})
	if err != nil {
		t.Fatal(err)
	}
	if table.Mode != taskRecapModeEntity || len(table.Rows) != 1 {
		t.Fatalf("table = %#v", table)
	}
	row := table.Rows[0]
	if row.Key != "L1" || row.Touch != "minted" {
		t.Fatalf("row = %#v", row)
	}
	if row.UserID != "bob" || row.UserName != "Bob" || row.SubmissionID != "2026-01-01T11:00:00Z" {
		t.Fatalf("latest actor = %#v", row)
	}
	if row.Cells["client"] != "CO" || row.Cells["netWeight"] != 50.0 {
		t.Fatalf("current cells = %#v", row.Cells)
	}
	if columnByKey(table.Columns, "client").Title != "Cliente" ||
		columnByKey(table.Columns, "netWeight").Title != "netWeight" {
		t.Fatalf("columns = %#v", table.Columns)
	}
}

func TestProjectTaskRecapMintManyTwoEntities(t *testing.T) {
	cfg := OrgConfig{
		Jobs: OrderedObject{
			Keys: []string{"j"},
			Values: map[string]any{
				"j": Job{Tasks: []TaskConfig{{
					"id": "t",
					"recap": map[string]any{
						"entity":  "pallet",
						"columns": []any{map[string]any{"field": "lotId"}, map[string]any{"field": "weight"}},
					},
				}}},
			},
		},
	}
	domain := domainFromKinds(map[string]EntityMap{
		"pallet": {
			"P.1": {"lotId": "L1", "weight": 10.0},
			"P.2": {"lotId": "L1", "weight": 20.0},
		},
	})
	domain.Submissions = []Submission{{
		JobID: "j", TaskID: "t", JobInstanceID: "L1",
		Timestamp: "2026-01-01T10:00:00Z", UserID: "ada",
		EntitiesTouched: []EntityTouch{
			{Op: "mint", Entity: "pallet", ID: "P.1"},
			{Op: "mint", Entity: "pallet", ID: "P.2"},
		},
	}}

	table, err := ProjectTaskRecap(cfg, domain, SubmissionAddress{JobID: "j", TaskID: "t", JobInstanceID: "L1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(table.Rows) != 2 {
		t.Fatalf("rows = %#v", table.Rows)
	}
	if table.Rows[0].Key != "P.1" || table.Rows[1].Key != "P.2" {
		t.Fatalf("keys = %q %q", table.Rows[0].Key, table.Rows[1].Key)
	}
	if table.Rows[0].Touch != "minted" || table.Rows[1].Cells["weight"] != 20.0 {
		t.Fatalf("rows = %#v", table.Rows)
	}
}

func TestProjectTaskRecapPatchesKeepLatestActorAcrossSubmissions(t *testing.T) {
	cfg := OrgConfig{
		Jobs: OrderedObject{
			Keys: []string{"j"},
			Values: map[string]any{
				"j": Job{Tasks: []TaskConfig{{
					"id": "weigh",
					"recap": map[string]any{
						"entity":  "pallet",
						"columns": []any{map[string]any{"field": "weight"}},
					},
				}}},
			},
		},
	}
	domain := domainFromKinds(map[string]EntityMap{
		"pallet": {
			"P.1": {"weight": 99.0},
			"P.2": {"weight": 11.0},
		},
	})
	domain.Submissions = []Submission{
		{
			JobID: "j", TaskID: "weigh", JobInstanceID: "L1",
			Timestamp: "2026-01-01T10:00:00Z", UserID: "ada", UserName: "Ada",
			EntitiesTouched: []EntityTouch{{Op: "patch", Entity: "pallet", ID: "P.1"}},
		},
		{
			JobID: "j", TaskID: "weigh", JobInstanceID: "L1",
			Timestamp: "2026-01-01T10:00:01Z", UserID: "bob", UserName: "Bob",
			EntitiesTouched: []EntityTouch{{Op: "patch", Entity: "pallet", ID: "P.2"}},
		},
		{
			JobID: "j", TaskID: "weigh", JobInstanceID: "L1",
			Timestamp: "2026-01-01T10:00:02Z", UserID: "cara", UserName: "Cara",
			EntitiesTouched: []EntityTouch{{Op: "patch", Entity: "pallet", ID: "P.1"}},
		},
	}

	table, err := ProjectTaskRecap(cfg, domain, SubmissionAddress{JobID: "j", TaskID: "weigh", JobInstanceID: "L1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(table.Rows) != 2 {
		t.Fatalf("rows = %#v", table.Rows)
	}
	// Chronological by latest-touch time: P.2 @ :01, then P.1 @ :02.
	if table.Rows[0].Key != "P.2" || table.Rows[0].UserName != "Bob" {
		t.Fatalf("first = %#v", table.Rows[0])
	}
	if table.Rows[1].Key != "P.1" || table.Rows[1].UserName != "Cara" || table.Rows[1].Touch != "patched" {
		t.Fatalf("second = %#v", table.Rows[1])
	}
	if table.Rows[1].Cells["weight"] != 99.0 {
		t.Fatalf("current weight = %#v", table.Rows[1].Cells)
	}
}

func TestProjectTaskRecapJobInstanceScoping(t *testing.T) {
	cfg := OrgConfig{
		Jobs: OrderedObject{
			Keys: []string{"j"},
			Values: map[string]any{
				"j": Job{Tasks: []TaskConfig{{
					"id": "t",
					"recap": map[string]any{
						"entity":  "lot",
						"columns": []any{map[string]any{"field": "client"}},
					},
				}}},
			},
		},
	}
	domain := domainFromKinds(map[string]EntityMap{
		"lot": {
			"A": {"client": "CO"},
			"B": {"client": "DA"},
		},
	})
	domain.Submissions = []Submission{
		{
			JobID: "j", TaskID: "t", JobInstanceID: "truck-a",
			Timestamp: "t1", UserID: "ada",
			EntitiesTouched: []EntityTouch{{Op: "mint", Entity: "lot", ID: "A"}},
		},
		{
			JobID: "j", TaskID: "t", JobInstanceID: "truck-b",
			Timestamp: "t2", UserID: "bob",
			EntitiesTouched: []EntityTouch{{Op: "mint", Entity: "lot", ID: "B"}},
		},
	}

	table, err := ProjectTaskRecap(cfg, domain, SubmissionAddress{JobID: "j", TaskID: "t", JobInstanceID: "truck-a"})
	if err != nil {
		t.Fatal(err)
	}
	if len(table.Rows) != 1 || table.Rows[0].Key != "A" {
		t.Fatalf("scoped = %#v", table.Rows)
	}
}

func TestProjectTaskRecapSkipsMissingDomainEntity(t *testing.T) {
	cfg := OrgConfig{
		Jobs: OrderedObject{
			Keys: []string{"j"},
			Values: map[string]any{
				"j": Job{Tasks: []TaskConfig{{
					"id": "t",
					"recap": map[string]any{
						"entity":  "lot",
						"columns": []any{map[string]any{"field": "client"}},
					},
				}}},
			},
		},
	}
	domain := domainFromKinds(map[string]EntityMap{
		"lot": {"keep": {"client": "CO"}},
	})
	domain.Submissions = []Submission{{
		JobID: "j", TaskID: "t", JobInstanceID: "i",
		Timestamp: "t1", UserID: "ada",
		EntitiesTouched: []EntityTouch{
			{Op: "mint", Entity: "lot", ID: "gone"},
			{Op: "mint", Entity: "lot", ID: "keep"},
		},
	}}

	table, err := ProjectTaskRecap(cfg, domain, SubmissionAddress{JobID: "j", TaskID: "t", JobInstanceID: "i"})
	if err != nil {
		t.Fatal(err)
	}
	if len(table.Rows) != 1 || table.Rows[0].Key != "keep" {
		t.Fatalf("rows = %#v", table.Rows)
	}
}

func TestProjectTaskRecapSubmissionModeUnionKeys(t *testing.T) {
	cfg := OrgConfig{
		Jobs: OrderedObject{
			Keys: []string{"j"},
			Values: map[string]any{
				"j": Job{Tasks: []TaskConfig{{
					"id": "t",
					"schema": map[string]any{
						"properties": map[string]any{
							"a": map[string]any{"title": "Alpha"},
							"b": map[string]any{"title": "Bravo"},
						},
					},
				}}},
			},
		},
	}
	domain := EmptyDomain()
	domain.Submissions = []Submission{
		{
			JobID: "j", TaskID: "t", JobInstanceID: "i",
			Timestamp: "t1", UserID: "ada",
			Facts: map[string]any{"b": 1},
		},
		{
			JobID: "j", TaskID: "t", JobInstanceID: "i",
			Timestamp: "t2", UserID: "bob", UserName: "Bob",
			Facts: map[string]any{"a": 2, "c": 3},
		},
	}

	table, err := ProjectTaskRecap(cfg, domain, SubmissionAddress{JobID: "j", TaskID: "t", JobInstanceID: "i"})
	if err != nil {
		t.Fatal(err)
	}
	if table.Mode != taskRecapModeSubmission {
		t.Fatalf("mode = %q", table.Mode)
	}
	if len(table.Columns) != 3 || table.Columns[0].Key != "b" || table.Columns[0].Title != "Bravo" {
		t.Fatalf("columns = %#v", table.Columns)
	}
	if table.Columns[1].Key != "a" || table.Columns[1].Title != "Alpha" || table.Columns[2].Key != "c" || table.Columns[2].Title != "c" {
		t.Fatalf("union = %#v", table.Columns)
	}
	if len(table.Rows) != 2 || table.Rows[0].Touch != "" || table.Rows[1].UserName != "Bob" {
		t.Fatalf("rows = %#v", table.Rows)
	}
	if table.Rows[0].Key != "t1" || table.Rows[0].Cells["b"] != 1 {
		t.Fatalf("first = %#v", table.Rows[0])
	}
}
