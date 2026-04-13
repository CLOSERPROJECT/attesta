package main

import (
	"bytes"
	"errors"
	"log"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHumanReadableTraceabilityTime(t *testing.T) {
	if got := humanReadableTraceabilityTime(time.Time{}); got != "" {
		t.Fatalf("humanReadableTraceabilityTime(zero) = %q, want empty", got)
	}

	value := time.Date(2026, 3, 5, 14, 30, 0, 0, time.FixedZone("CET", 3600))
	if got := humanReadableTraceabilityTime(value); got != "5 Mar 2026 at 13:30 UTC" {
		t.Fatalf("humanReadableTraceabilityTime() = %q", got)
	}
}

func TestDPPDialogIDFragment(t *testing.T) {
	if got := dppDialogIDFragment("   "); got != "step" {
		t.Fatalf("dppDialogIDFragment(blank) = %q, want step", got)
	}
	if got := dppDialogIDFragment(" Step 1 / Review "); got != "step-1-review" {
		t.Fatalf("dppDialogIDFragment(words) = %q, want step-1-review", got)
	}
	if got := dppDialogIDFragment("___"); got != "step" {
		t.Fatalf("dppDialogIDFragment(symbols) = %q, want step", got)
	}
}

func TestFormataStreamCreatorID(t *testing.T) {
	stream := FormataBuilderStream{
		CreatedByUserID: " creator-1 ",
		UpdatedByUserID: " updater-1 ",
	}
	if got := formataStreamCreatorID(stream); got != "creator-1" {
		t.Fatalf("formataStreamCreatorID(created) = %q, want creator-1", got)
	}

	stream.CreatedByUserID = "   "
	if got := formataStreamCreatorID(stream); got != "updater-1" {
		t.Fatalf("formataStreamCreatorID(updated) = %q, want updater-1", got)
	}
}

func TestHomePickerMessage(t *testing.T) {
	if got := homePickerMessage(nil, "error"); got != "" {
		t.Fatalf("homePickerMessage(nil) = %q, want empty", got)
	}

	req := httptest.NewRequest("GET", "/?confirmation=%20saved%20", nil)
	if got := homePickerMessage(req, "confirmation"); got != "saved" {
		t.Fatalf("homePickerMessage() = %q, want saved", got)
	}
}

func TestLogCapabilityCheckError(t *testing.T) {
	originalWriter := log.Writer()
	originalFlags := log.Flags()
	log.SetFlags(0)
	defer log.SetOutput(originalWriter)
	defer log.SetFlags(originalFlags)

	var output bytes.Buffer
	log.SetOutput(&output)

	logCapabilityCheckError(nil, "ignored")
	if output.Len() != 0 {
		t.Fatalf("expected no log output for nil error, got %q", output.String())
	}

	logCapabilityCheckError(errors.New("boom"), "capability %s", "check")
	if !strings.Contains(output.String(), "capability check: boom") {
		t.Fatalf("log output = %q, want capability check: boom", output.String())
	}
}
