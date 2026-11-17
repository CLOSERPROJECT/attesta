package org

import (
	"strings"
	"testing"
)

func TestValidateTaskConfigsRejectsUnlockFrom(t *testing.T) {
	err := validateTaskConfigs([]TaskConfig{
		{"id": "open"},
		{"id": "next", "unlockFrom": "open"},
	})
	if err == nil || !strings.Contains(err.Error(), "unlockFrom") || !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("unlockFrom = %v", err)
	}
}

func TestValidateTaskConfigsRejectsUnlockFromAnyValue(t *testing.T) {
	cases := []any{"open", "", 1, true, nil}
	for _, raw := range cases {
		err := validateTaskConfigs([]TaskConfig{
			{"id": "open"},
			{"id": "next", "unlockFrom": raw},
		})
		if err == nil || !strings.Contains(err.Error(), "not supported") {
			t.Fatalf("unlockFrom=%#v err=%v, want not supported", raw, err)
		}
	}
}
