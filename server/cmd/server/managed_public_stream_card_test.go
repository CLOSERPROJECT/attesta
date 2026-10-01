package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestManagedPublicStreamCardRendersMenuAndMyHref(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var out bytes.Buffer
	view := ManagedPublicStreamCardView{
		Key:          "wf-a",
		CanClone:     true,
		CanEdit:      true,
		EditAction:   "/my/organization/formata-builder?stream=wf-a",
		CanDelete:    true,
		DeleteAction: "/my/streams/wf-a/delete",
		Card: PublicStreamCardView{
			Name: "Alpha",
			Href: "/my/streams/wf-a/",
		},
	}
	if err := tmpl.ExecuteTemplate(&out, "managed_public_stream_card", view); err != nil {
		t.Fatalf("render: %v", err)
	}
	body := out.String()
	for _, want := range []string{
		`href="/my/streams/wf-a/"`,
		`class="public-stream-card-shell"`,
		"Clone",
		`id="delete-workflow-wf-a"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %s", want, body)
		}
	}
	if strings.Contains(body, "public-stream-card-start") {
		t.Fatalf("Start must be absent without Startable, got %s", body)
	}
}

func TestManagedPublicStreamCardHidesMenuWithoutClone(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var out bytes.Buffer
	view := ManagedPublicStreamCardView{
		Key: "wf-a",
		Card: PublicStreamCardView{
			Name: "Alpha",
			Href: "/my/streams/wf-a/",
		},
	}
	if err := tmpl.ExecuteTemplate(&out, "managed_public_stream_card", view); err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(out.String(), "public-stream-card-menu") {
		t.Fatalf("menu should be absent: %s", out.String())
	}
}

func TestManagedPublicStreamCardRendersStartWhenStartable(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var out bytes.Buffer
	view := ManagedPublicStreamCardView{
		Key:         "wf-a",
		Startable:   true,
		StartAction: "/my/streams/wf-a/instance/start",
		Card: PublicStreamCardView{
			Name: "Alpha",
			Href: "/my/streams/wf-a/",
		},
	}
	if err := tmpl.ExecuteTemplate(&out, "managed_public_stream_card", view); err != nil {
		t.Fatalf("render: %v", err)
	}
	body := out.String()
	for _, want := range []string{
		"public-stream-card-start",
		"Start",
		`id="start-instance-wf-a"`,
		`action="/my/streams/wf-a/instance/start"`,
		`name="name"`,
		"New instance",
		"Start instance",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %s", want, body)
		}
	}
}

func TestManagedPublicStreamCardOmitsStartWhenNotStartable(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var out bytes.Buffer
	view := ManagedPublicStreamCardView{
		Key: "wf-a",
		Card: PublicStreamCardView{
			Name: "Alpha",
			Href: "/my/streams/wf-a/",
		},
	}
	if err := tmpl.ExecuteTemplate(&out, "managed_public_stream_card", view); err != nil {
		t.Fatalf("render: %v", err)
	}
	body := out.String()
	for _, gone := range []string{
		"public-stream-card-start",
		`id="start-instance-`,
		`instance/start`,
	} {
		if strings.Contains(body, gone) {
			t.Fatalf("did not expect %q in %s", gone, body)
		}
	}
}
