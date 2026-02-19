package main

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestBuildActionListDoneScalarValues(t *testing.T) {
	cfg := testRuntimeConfig()
	process := &Process{
		ID: primitive.NewObjectID(),
		Progress: map[string]ProcessStep{
			"1.1": {State: "done", Data: map[string]interface{}{"value": 10.0}},
		},
	}

	actions := buildActionList(cfg.Workflow, process, "workflow", Actor{Role: "dep1"}, true, map[string]RoleMeta{})
	action := findAction(t, actions, "1.1")
	if action.Status != "done" {
		t.Fatalf("expected status done, got %q", action.Status)
	}
	if len(action.Values) != 1 {
		t.Fatalf("expected 1 value row, got %d", len(action.Values))
	}
	if action.Values[0].Key != "value" || action.Values[0].Value != "10" {
		t.Fatalf("unexpected value row: %#v", action.Values[0])
	}
	if len(action.Attachments) != 0 {
		t.Fatalf("expected no attachments, got %#v", action.Attachments)
	}
}

func TestBuildActionListDoneFileAttachments(t *testing.T) {
	cfg := testRuntimeConfig()
	processID := primitive.NewObjectID()
	attachmentID := primitive.NewObjectID().Hex()
	process := &Process{
		ID: processID,
		Progress: map[string]ProcessStep{
			"1.3": {
				State: "done",
				Data: map[string]interface{}{
					"attachment": map[string]interface{}{
						"attachmentId": attachmentID,
						"filename":     "coa.pdf",
						"sha256":       "abc123",
					},
				},
			},
		},
	}

	actions := buildActionList(cfg.Workflow, process, "workflow", Actor{Role: "dep1"}, true, map[string]RoleMeta{})
	action := findAction(t, actions, "1.3")
	if action.Status != "done" {
		t.Fatalf("expected status done, got %q", action.Status)
	}
	if len(action.Values) != 0 {
		t.Fatalf("expected no scalar values for file step, got %#v", action.Values)
	}
	if len(action.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(action.Attachments))
	}
	got := action.Attachments[0]
	wantURL := "/w/workflow/process/" + processID.Hex() + "/attachment/" + attachmentID + "/file"
	if got.URL != wantURL {
		t.Fatalf("expected URL %q, got %q", wantURL, got.URL)
	}
	if got.Filename != "coa.pdf" {
		t.Fatalf("expected filename coa.pdf, got %q", got.Filename)
	}
	if got.SHA256 != "abc123" {
		t.Fatalf("expected sha abc123, got %q", got.SHA256)
	}
}

func TestBuildActionListDoneFormataValuesAndAttachments(t *testing.T) {
	def := WorkflowDef{
		Steps: []WorkflowStep{
			{
				StepID: "1",
				Substep: []WorkflowSub{
					{
						SubstepID: "1.1",
						Title:     "Form",
						Order:     1,
						Role:      "dep1",
						InputKey:  "payload",
						InputType: "formata",
					},
				},
			},
		},
	}
	doneAt := time.Date(2026, 2, 19, 9, 0, 0, 0, time.UTC)
	processID := primitive.NewObjectID()
	attachmentID := primitive.NewObjectID().Hex()
	process := &Process{
		ID: processID,
		Progress: map[string]ProcessStep{
			"1.1": {
				State:  "done",
				DoneAt: &doneAt,
				DoneBy: &Actor{UserID: "u1", Role: "dep1"},
				Data: map[string]interface{}{
					"payload": map[string]interface{}{
						"details": map[string]interface{}{
							"status": "ok",
							"weight": 42.0,
						},
						"docs": []interface{}{
							map[string]interface{}{
								"attachmentId": attachmentID,
								"filename":     "nested.pdf",
							},
						},
					},
				},
			},
		},
	}

	actions := buildActionList(def, process, "workflow", Actor{Role: "dep1"}, true, map[string]RoleMeta{})
	action := findAction(t, actions, "1.1")
	if action.DoneAt != doneAt.Format(time.RFC3339) {
		t.Fatalf("expected doneAt %q, got %q", doneAt.Format(time.RFC3339), action.DoneAt)
	}
	if action.DoneBy != "u1" || action.DoneRole != "dep1" {
		t.Fatalf("unexpected done actor: %q/%q", action.DoneBy, action.DoneRole)
	}
	if len(action.Values) != 2 {
		t.Fatalf("expected 2 flattened values, got %#v", action.Values)
	}
	if action.Values[0].Key != "details.status" || action.Values[0].Value != "ok" {
		t.Fatalf("unexpected first flattened value: %#v", action.Values[0])
	}
	if action.Values[1].Key != "details.weight" || action.Values[1].Value != "42" {
		t.Fatalf("unexpected second flattened value: %#v", action.Values[1])
	}
	if len(action.Attachments) != 1 {
		t.Fatalf("expected one nested attachment, got %#v", action.Attachments)
	}
	wantURL := "/w/workflow/process/" + processID.Hex() + "/attachment/" + attachmentID + "/file"
	if action.Attachments[0].URL != wantURL {
		t.Fatalf("expected URL %q, got %q", wantURL, action.Attachments[0].URL)
	}
}

func findAction(t *testing.T, actions []ActionView, substepID string) ActionView {
	t.Helper()
	for _, action := range actions {
		if action.SubstepID == substepID {
			return action
		}
	}
	t.Fatalf("action %s not found", substepID)
	return ActionView{}
}
