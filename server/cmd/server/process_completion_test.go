// process_completion_test.go
package main

import (
	"errors"
	"testing"
)

func TestProcessCompletionSentinelErrors(t *testing.T) {
	if ErrProgressUpdate == nil || ErrNotarization == nil {
		t.Fatal("expected sentinel errors to be defined")
	}
	if ErrProgressUpdate.Error() != "process: progress update failed" {
		t.Fatalf("unexpected ErrProgressUpdate message: %q", ErrProgressUpdate.Error())
	}
	if !errors.Is(ErrProgressUpdate, ErrProgressUpdate) {
		t.Fatal("expected errors.Is to match ErrProgressUpdate")
	}
}
