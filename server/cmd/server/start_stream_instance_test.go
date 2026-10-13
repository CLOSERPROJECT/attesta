package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestStartStreamInstanceTemplateCardTrigger(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var out bytes.Buffer
	view := StartStreamInstanceView{
		DialogID:    "start-instance-wf-a",
		StartAction: "/my/streams/wf-a/instance/start",
		ReturnPath:  "/?category=supply-chain&subCategory=procurement",
		CardTrigger: true,
	}
	if err := tmpl.ExecuteTemplate(&out, "start_stream_instance", view); err != nil {
		t.Fatalf("render: %v", err)
	}
	body := out.String()
	for _, want := range []string{
		`class="btn btn-primary btn-lg public-stream-card-start"`,
		`id="start-instance-wf-a"`,
		`data-start-stream-instance`,
		`action="/my/streams/wf-a/instance/start"`,
		`name="next"`,
		`value="/?category=supply-chain&amp;subCategory=procurement"`,
		`required`,
		`data-start-stream-submit`,
		`disabled`,
		"New instance",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %s", want, body)
		}
	}
	if strings.Contains(body, "Start instance") {
		t.Fatalf("submit label must be Start, got %s", body)
	}
}

func TestStartStreamInstanceTemplateHeaderTrigger(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var out bytes.Buffer
	view := StartStreamInstanceView{
		DialogID:    "new-instance-dialog",
		StartAction: "/my/streams/workflow/instance/start",
		ReturnPath:  "/my/streams/workflow/",
	}
	if err := tmpl.ExecuteTemplate(&out, "start_stream_instance", view); err != nil {
		t.Fatalf("render: %v", err)
	}
	body := out.String()
	if strings.Contains(body, "public-stream-card-start") {
		t.Fatalf("header trigger must not use card class, got %s", body)
	}
	if !strings.Contains(body, `class="btn btn-primary"`) {
		t.Fatalf("expected header btn-primary, got %s", body)
	}
}

func TestSafeReturnPath(t *testing.T) {
	if got := safeReturnPath("/streams/a", "/my"); got != "/streams/a" {
		t.Fatalf("got %q", got)
	}
	if got := safeReturnPath("//evil.com", "/my"); got != "/my" {
		t.Fatalf("protocol-relative got %q", got)
	}
	if got := safeReturnPath("https://evil.com", "/my"); got != "/my" {
		t.Fatalf("absolute URL got %q", got)
	}
	if got := safeReturnPath("", "/my"); got != "/my" {
		t.Fatalf("empty got %q", got)
	}
}
