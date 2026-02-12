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

	timeline := buildTimeline(cfg.Workflow, process, "workflow", map[string]RoleMeta{})
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
