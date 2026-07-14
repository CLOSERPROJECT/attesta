package main

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestBuildStreamInstanceDetailViewDoesNotFilterByCurrentActiveRole(t *testing.T) {
	process := &Process{
		ID: primitive.NewObjectID(),
		Progress: map[string]ProcessStep{
			"1.1": {State: "pending"},
			"1.2": {State: "pending"},
			"1.3": {State: "pending"},
			"2.1": {State: "pending"},
			"2.2": {State: "pending"},
			"3.1": {State: "pending"},
			"3.2": {State: "pending"},
		},
	}
	server := &Server{}
	actor := Actor{
		Role:      "dep2",
		RoleSlugs: []string{"dep1", "dep2"},
	}
	view := server.buildStreamInstanceDetailView(t.Context(), testRuntimeConfig(), "workflow", process, actor, "", "", false)

	if view.SelectedBody == nil || view.SelectedBody.SubstepID != "1.1" {
		t.Fatalf("expected dep1 step to remain visible after dep2 completion context, got %#v", view.SelectedBody)
	}
}

func TestBuildStreamInstanceDetailViewDoneProcessIncludesDPPAndAttachments(t *testing.T) {
	processID := primitive.NewObjectID()
	process := &Process{
		ID:     processID,
		Status: "done",
		Progress: map[string]ProcessStep{
			"1.1": {
				State: "done",
				Data: map[string]interface{}{
					"value": "ok",
					"attachment": map[string]interface{}{
						"attachmentId": primitive.NewObjectID().Hex(),
						"filename":     "coa.pdf",
					},
				},
			},
		},
		DPP: &ProcessDPP{
			GTIN:   "09506000134352",
			Lot:    "LOT-1",
			Serial: "SER-1",
		},
	}
	server := &Server{}
	view := server.buildStreamInstanceDetailView(t.Context(), testRuntimeConfig(), "workflow", process, Actor{Role: "dep1"}, "", "", false)

	if !view.ProcessDone {
		t.Fatal("expected processDone")
	}
	if view.SelectedBody != nil {
		t.Fatalf("expected no selected body for done process, got %#v", view.SelectedBody)
	}
	if len(view.Attachments) == 0 {
		t.Fatal("expected download attachments for done process")
	}
	if view.DPPURL != "/01/09506000134352/10/LOT-1/21/SER-1" {
		t.Fatalf("DPPURL = %q", view.DPPURL)
	}
	if view.DPPGS1 == "" {
		t.Fatal("expected DPP GS1 element string")
	}
}

func TestApplyDoneByEmailHelpersNoopOnEmptyInput(t *testing.T) {
	server := &Server{}
	def := WorkflowDef{Steps: []WorkflowStep{{StepID: "1", OrganizationSlug: "org-a"}}}

	if got := server.applyDoneByEmailToSubstepViews(t.Context(), def, Actor{OrgSlug: "org-a"}, nil); got != nil {
		t.Fatalf("expected nil passthrough for empty actions, got %#v", got)
	}
	if got := server.applyDoneByEmailToTermination(t.Context(), def, Actor{OrgSlug: "org-a"}, nil); got != nil {
		t.Fatalf("expected nil passthrough for nil termination, got %#v", got)
	}
}
