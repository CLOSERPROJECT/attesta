package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestSubstepBodyTemplateRendersResultMode(t *testing.T) {
	tmpl := parseTestTemplates(t)

	action := SubstepBodyView{
		WorkflowKey: "workflow",
		ProcessID:   "process-1",
		SubstepID:   "1.1",
		Title:       "Completed substep",
		InputKey:    "notes",
		Status:      "done",
		RoleBadges: []SubstepRoleBadge{
			{ID: "dep1", Label: "Department 1", Palette: "red"},
		},
		Values: []SubstepKV{
			{Key: "notes", Value: "All good"},
		},
		Attachments: []SubstepAttachmentView{
			{
				Key:      "photo",
				URL:      "/w/workflow/process/process-1/substep/1.1/file?id=abc",
				Filename: "photo.jpg",
			},
		},
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "substep_body", action); err != nil {
		t.Fatalf("render substep_body template: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		"Completed by role:",
		`data-role-palette="red"`,
		">Submitted<",
		"<dt>notes</dt>",
		"<dd>All good</dd>",
		"<dt>photo</dt>",
		">Attachments<",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in rendered substep body, got: %s", want, body)
		}
	}
}

func TestSubstepBodyTemplateRendersMessageMode(t *testing.T) {
	tmpl := parseTestTemplates(t)

	action := SubstepBodyView{
		WorkflowKey:   "workflow",
		ProcessID:     "process-1",
		SubstepID:     "2.1",
		Title:         "Skipped substep",
		Status:        "done",
		DetailMessage: "Skipped: not applicable for this batch.",
		RoleBadges: []SubstepRoleBadge{
			{ID: "dep1", Label: "Department 1", Palette: "red"},
		},
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "substep_body", action); err != nil {
		t.Fatalf("render substep_body template: %v", err)
	}
	body := out.String()

	if !strings.Contains(body, "Skipped: not applicable for this batch.") {
		t.Fatalf("expected detail message in substep body, got: %s", body)
	}
	if strings.Contains(body, ">Submitted<") {
		t.Fatalf("expected message mode without submitted block, got: %s", body)
	}
	if strings.Contains(body, `class="substep-body-form"`) {
		t.Fatalf("expected message mode without form, got: %s", body)
	}
}

func TestSubstepBodyTemplateRendersOverrideResultMode(t *testing.T) {
	tmpl := parseTestTemplates(t)

	action := SubstepBodyView{
		WorkflowKey:    "workflow",
		ProcessID:      "process-1",
		SubstepID:      "1.1",
		Title:          "Adapted substep",
		Status:         "done",
		HasOverride:    true,
		OverrideReason: "local source shape",
		RoleBadges: []SubstepRoleBadge{
			{ID: "dep1", Label: "Department 1", Palette: "red"},
		},
		Values: []SubstepKV{
			{Key: "value", Value: "override-ok"},
		},
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "substep_body", action); err != nil {
		t.Fatalf("render substep_body template: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		"Completed with local form adaptation.",
		"Reason: local source shape",
		">Submitted<",
		"<dt>value</dt>",
		"<dd>override-ok</dd>",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in rendered override substep body, got: %s", want, body)
		}
	}
}
