package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestBuildPublicStreamRunCardsOnlyCompletedCappedWithDPP(t *testing.T) {
	doneAt := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	def := WorkflowDef{Steps: []WorkflowStep{{Substep: []WorkflowSub{{SubstepID: "1.1"}}}}}

	processes := []Process{
		{ID: primitive.NewObjectID(), Status: processStatusActive, Progress: map[string]ProcessStep{"1_1": {State: "pending"}}},
		{ID: primitive.NewObjectID(), Status: processStatusTerminated, Termination: &ProcessTermination{EndedAt: doneAt}},
		{
			ID:     primitive.NewObjectID(),
			Status: processStatusDone,
			Progress: map[string]ProcessStep{
				"1_1": {State: "done", DoneAt: &doneAt},
			},
			DPP: &ProcessDPP{GTIN: "09506000134352", Lot: "LOT-1", Serial: "SER-1", GeneratedAt: doneAt},
		},
		{
			ID:     primitive.NewObjectID(),
			Status: processStatusDone,
			Progress: map[string]ProcessStep{
				"1_1": {State: "done", DoneAt: ptrTime(doneAt.Add(-time.Hour))},
			},
		},
	}
	// Newest-first input: builder must preserve ListRecent order among completed only.
	ordered := []Process{processes[2], processes[3], processes[0], processes[1]}
	cards := buildPublicStreamRunCards(def, ordered)
	if len(cards) != 2 {
		t.Fatalf("len = %d, want 2 completed", len(cards))
	}
	if cards[0].DigitalLink != digitalLinkURL("09506000134352", "LOT-1", "SER-1") {
		t.Fatalf("first DigitalLink = %q", cards[0].DigitalLink)
	}
	if !cards[0].PassportChip || cards[0].StatusLabel != "Completed" {
		t.Fatalf("first card = %#v", cards[0])
	}
	if cards[1].DigitalLink != "" || cards[1].PassportChip {
		t.Fatalf("second card must be non-link, got %#v", cards[1])
	}
}

func TestBuildPublicStreamRunCardsRespectsCap(t *testing.T) {
	def := WorkflowDef{}
	doneAt := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	var processes []Process
	for i := 0; i < publicStreamRecentRunLimit+3; i++ {
		at := doneAt.Add(-time.Duration(i) * time.Hour)
		processes = append(processes, Process{
			ID:       primitive.NewObjectID(),
			Status:   processStatusDone,
			Progress: map[string]ProcessStep{"1_1": {State: "done", DoneAt: &at}},
		})
	}
	cards := buildPublicStreamRunCards(def, processes)
	if len(cards) != publicStreamRecentRunLimit {
		t.Fatalf("len = %d, want %d", len(cards), publicStreamRecentRunLimit)
	}
}

func TestPublicStreamRunCardTemplateLinkVsStatic(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var linked bytes.Buffer
	if err := tmpl.ExecuteTemplate(&linked, "public_stream_run_card", PublicStreamRunCardView{
		StatusLabel:  "Completed",
		CompletedAt:  "1 Jul 2026 at 12:00 UTC",
		DigitalLink:  "/01/09506000134352/10/LOT-1/21/SER-1",
		PassportChip: true,
	}); err != nil {
		t.Fatalf("linked render: %v", err)
	}
	lb := linked.String()
	if !strings.Contains(lb, `href="/01/09506000134352/10/LOT-1/21/SER-1"`) {
		t.Fatalf("expected DPP href, got: %s", lb)
	}
	if !strings.Contains(lb, "DPP") {
		t.Fatalf("expected DPP cue, got: %s", lb)
	}

	var static bytes.Buffer
	if err := tmpl.ExecuteTemplate(&static, "public_stream_run_card", PublicStreamRunCardView{
		StatusLabel: "Completed",
		CompletedAt: "1 Jul 2026 at 11:00 UTC",
	}); err != nil {
		t.Fatalf("static render: %v", err)
	}
	sb := static.String()
	if strings.Contains(sb, "href=") {
		t.Fatalf("static card must not link, got: %s", sb)
	}
	if !strings.Contains(sb, `<article class="public-stream-run-card"`) {
		t.Fatalf("expected article, got: %s", sb)
	}
}
