package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestStreamPageNewInstanceDialogMarkup(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	view := HomeView{
		PageBase: PageBase{
			WorkflowKey:  "workflow",
			WorkflowPath: "/w/workflow",
			WorkflowName: "Demo workflow",
		},
		StatusFilter:  "all",
		Sort:          "time_desc",
		FilterOptions: testHomeFilterOptions(),
		ProcessGroups: testHomeActiveProcessGroups(nil, "all", "time_desc", 1),
	}
	if err := tmpl.ExecuteTemplate(&out, "home_body", view); err != nil {
		t.Fatalf("render home_body: %v", err)
	}
	body := out.String()

	dialogStart := strings.Index(body, `id="new-instance-dialog"`)
	if dialogStart == -1 {
		t.Fatalf("expected new-instance dialog, got:\n%s", body)
	}
	dialogEnd := strings.Index(body[dialogStart:], `</dialog>`)
	if dialogEnd == -1 {
		t.Fatal("expected closing dialog tag for new-instance dialog")
	}
	dialog := body[dialogStart : dialogStart+dialogEnd]

	for _, want := range []string{
		`class="dialog"`,
		`class="dialog-card"`,
		`class="dialog-head"`,
		`class="dialog-title"`,
		`class="dialog-subtitle"`,
		`class="dialog-actions"`,
		`class="dialog-title">New instance</h3>`,
		"Give this stream instance a brief name",
		`class="btn btn-ghost btn-icon dialog-close"`,
	} {
		if !strings.Contains(dialog, want) {
			t.Fatalf("expected %q in new-instance dialog markup, got:\n%s", want, dialog)
		}
	}

	headIdx := strings.Index(dialog, `class="dialog-head"`)
	titleIdx := strings.Index(dialog, `class="dialog-title"`)
	actionsIdx := strings.Index(dialog, `class="dialog-actions"`)
	if headIdx == -1 || titleIdx == -1 || actionsIdx == -1 {
		t.Fatal("expected dialog-head, title, and actions in new-instance dialog")
	}
	if !(headIdx < titleIdx && titleIdx < actionsIdx) {
		t.Fatalf("expected dialog head before title before actions")
	}
}

func TestPlatformAdminCreateOrgDialogMarkup(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	view := PlatformAdminView{
	}
	if err := tmpl.ExecuteTemplate(&out, "platform_admin_body", view); err != nil {
		t.Fatalf("render platform_admin_body: %v", err)
	}
	body := out.String()

	dialogStart := strings.Index(body, `id="create-org-dialog"`)
	if dialogStart == -1 {
		t.Fatalf("expected create-org dialog, got:\n%s", body)
	}
	dialogEnd := strings.Index(body[dialogStart:], `</dialog>`)
	if dialogEnd == -1 {
		t.Fatal("expected closing dialog tag for create-org dialog")
	}
	dialog := body[dialogStart : dialogStart+dialogEnd]

	for _, want := range []string{
		`class="dialog"`,
		`class="dialog-card"`,
		`class="dialog-head"`,
		`class="dialog-title"`,
		`class="dialog-subtitle"`,
		`class="dialog-title">New organization</h3>`,
		"Add a new organization to the platform",
		`class="btn btn-ghost btn-icon dialog-close"`,
		"Create organization",
	} {
		if !strings.Contains(dialog, want) {
			t.Fatalf("expected %q in create-org dialog markup, got:\n%s", want, dialog)
		}
	}

	headIdx := strings.Index(dialog, `class="dialog-head"`)
	cardIdx := strings.Index(dialog, `class="dialog-card"`)
	if cardIdx == -1 || headIdx == -1 || !(cardIdx < headIdx) {
		t.Fatalf("expected dialog-card before dialog-head")
	}
}

func TestProcessTerminateDialogMarkup(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	view := ProcessPageView{
		PageBase: PageBase{
			WorkflowKey:  "workflow",
			WorkflowPath: "/w/workflow",
		},
		ProcessID: "process-1",
		Detail: StreamInstanceDetailView{
			WorkflowKey:      "workflow",
			ProcessID:        "process-1",
			CanTerminate:     true,
			TerminateAction:  "/w/workflow/process/process-1/terminate",
			TerminateSubstep: "2.1",
		},
	}
	if err := tmpl.ExecuteTemplate(&out, "process_content.html", view); err != nil {
		t.Fatalf("render process_content.html: %v", err)
	}
	body := out.String()

	dialogStart := strings.Index(body, `id="terminate-process-dialog"`)
	if dialogStart == -1 {
		t.Fatalf("expected terminate-process dialog, got:\n%s", body)
	}
	dialogEnd := strings.Index(body[dialogStart:], `</dialog>`)
	if dialogEnd == -1 {
		t.Fatal("expected closing dialog tag for terminate-process dialog")
	}
	dialog := body[dialogStart : dialogStart+dialogEnd]

	for _, want := range []string{
		`class="dialog"`,
		`class="dialog-card"`,
		`class="dialog-head"`,
		`class="dialog-title u-text-danger"`,
		`class="dialog-subtitle"`,
		`class="dialog-actions"`,
		"Danger zone - End stream early",
		"Available substep",
		"2.1",
		`action="/w/workflow/process/process-1/terminate"`,
	} {
		if !strings.Contains(dialog, want) {
			t.Fatalf("expected %q in terminate-process dialog markup, got:\n%s", want, dialog)
		}
	}

	headIdx := strings.Index(dialog, `class="dialog-head"`)
	dangerIdx := strings.Index(dialog, `class="dialog-title u-text-danger"`)
	actionsIdx := strings.Index(dialog, `class="dialog-actions"`)
	if headIdx == -1 || dangerIdx == -1 || actionsIdx == -1 {
		t.Fatal("expected dialog-head, danger title, and actions in terminate dialog")
	}
	if !(headIdx < dangerIdx && dangerIdx < actionsIdx) {
		t.Fatalf("expected dialog head before danger title before actions")
	}
}

func TestStreamPreviewDialogPageScopedMarkup(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	view := HomeView{
		PageBase: PageBase{
			WorkflowKey:  "workflow",
			WorkflowPath: "/w/workflow",
			WorkflowName: "Demo workflow",
		},
		StatusFilter:  "all",
		Sort:          "time_desc",
		FilterOptions: testHomeFilterOptions(),
		ProcessGroups: testHomeActiveProcessGroups(nil, "all", "time_desc", 1),
	}
	if err := tmpl.ExecuteTemplate(&out, "home_body", view); err != nil {
		t.Fatalf("render home_body: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`id="stream-preview-dialog"`,
		`class="dialog"`,
		`class="dialog-card"`,
		`class="stream-preview-body"`,
		"Stream preview",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in stream preview dialog markup, got excerpt:\n%s", want, body)
		}
	}
	if strings.Contains(body, `stream-preview-dialog-card`) {
		t.Fatal("expected stream preview card sizing class removed from markup")
	}
}
