package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestResolveSubstepBodyMode(t *testing.T) {
	tests := []struct {
		name string
		view SubstepBodyView
		want SubstepBodyMode
	}{
		{name: "message from detail", view: SubstepBodyView{Status: "done", DetailMessage: "skipped"}, want: SubstepBodyModeMessage},
		{name: "result when done", view: SubstepBodyView{Status: "done"}, want: SubstepBodyModeResult},
		{name: "actionable when available authorized", view: SubstepBodyView{Status: "available"}, want: SubstepBodyModeActionable},
		{name: "preview when locked", view: SubstepBodyView{Status: "locked", Disabled: true}, want: SubstepBodyModePreview},
		{name: "preview when available but disabled", view: SubstepBodyView{Status: "available", Disabled: true}, want: SubstepBodyModePreview},
		{name: "preview when available but readonly", view: SubstepBodyView{Status: "available", ReadOnly: true}, want: SubstepBodyModePreview},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveSubstepBodyMode(tc.view); got != tc.want {
				t.Fatalf("resolveSubstepBodyMode() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSubstepBodyTemplateRendersResultMode(t *testing.T) {
	tmpl := parseTestTemplates(t)

	action := SubstepBodyView{
		WorkflowKey: "workflow",
		ProcessID:   "process-1",
		SubstepID:   "1.1",
		Title:       "Completed substep",
		InputKey:    "notes",
		Status:      "done",
		Mode:        SubstepBodyModeResult,
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
		Mode:          SubstepBodyModeMessage,
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
		Mode:           SubstepBodyModeResult,
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
