package org

import "testing"

func mustScope(t *testing.T, cfg OrgConfig) Scope {
	t.Helper()
	scope, err := ScopeFromConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return scope
}
