package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestMakeStreamInstanceDetailReadOnlyClearsSelection(t *testing.T) {
	view := StreamInstanceDetailView{
		SelectedSubstepID: "1.1",
		SelectedBody: &SubstepBodyView{
			SubstepID: "1.1",
			Title:     "Record input",
			Status:    "available",
		},
		Timeline: []TimelineStep{{
			Expanded: true,
			Substeps: []TimelineSubstep{{
				SubstepID: "1.1",
				Title:     "Record input",
				Selected:  true,
				Body: &SubstepBodyView{
					SubstepID: "1.1",
					Title:     "Record input",
					Status:    "available",
				},
			}},
		}},
	}

	got := makeStreamInstanceDetailReadOnly(view, "Preview only.")
	if got.SelectedSubstepID != "" {
		t.Fatalf("SelectedSubstepID = %q, want empty", got.SelectedSubstepID)
	}
	if got.SelectedBody != nil {
		t.Fatalf("SelectedBody = %#v, want nil", got.SelectedBody)
	}
	if got.Timeline[0].Expanded {
		t.Fatal("expected timeline step to be collapsed")
	}
	if got.Timeline[0].Substeps[0].Selected {
		t.Fatal("expected timeline substep to be unselected")
	}
	body := got.Timeline[0].Substeps[0].Body
	if body == nil || !body.ReadOnly || body.Reason != "Preview only." {
		t.Fatalf("expected read-only body with reason, got %#v", body)
	}
}

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

func TestBuildStreamTerminationDetailsViewResolvesIdentityByContext(t *testing.T) {
	server := &Server{
		identity: &fakeIdentityStore{
			getUserByIDFunc: func(ctx context.Context, userID string) (IdentityUser, error) {
				return IdentityUser{ID: userID, Email: "ended@example.com", Status: "active"}, nil
			},
		},
	}
	def := WorkflowDef{Steps: []WorkflowStep{{StepID: "1", OrganizationSlug: "org-a"}}}
	termination := &ProcessTermination{
		Reason:    "supplier cancelled",
		EndedAt:   time.Date(2026, 3, 6, 9, 15, 0, 0, time.UTC),
		Actor:     &Actor{ID: "appwrite:user-1", Role: "dep1"},
		SubstepID: "1.2",
	}

	orgView := server.buildStreamTerminationDetailsView(t.Context(), def, Actor{OrgSlug: "org-a"}, termination)
	if orgView == nil || orgView.EndedBy != "ended@example.com" {
		t.Fatalf("org viewer endedBy = %#v, want email", orgView)
	}

	publicView := server.buildStreamTerminationDetailsView(t.Context(), def, Actor{}, termination)
	if publicView == nil || publicView.EndedBy != "user-1" {
		t.Fatalf("public viewer endedBy = %#v, want stripped user id", publicView)
	}
	if strings.Contains(publicView.EndedBy, "appwrite:") {
		t.Fatalf("public viewer endedBy must not include appwrite prefix, got %q", publicView.EndedBy)
	}
}
