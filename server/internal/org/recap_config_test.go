package org

import (
	"strings"
	"testing"
)

func recapBaseCfg(task TaskConfig) OrgConfig {
	cfg := validStreamCfg()
	cfg.Domain = map[string]any{
		"entities": map[string]any{
			"lot": map[string]any{
				"fields": []any{"numeroFir", "netWeight", "truckId"},
			},
			"truck": map[string]any{
				"fields": []any{"truckWeight"},
			},
		},
	}
	intake := cfg.Jobs.Values["intake"].(Job)
	intake.Tasks = []TaskConfig{task}
	cfg.Jobs.Values["intake"] = intake
	return cfg
}

func TestValidateRecapConfig(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		task    TaskConfig
		domain  map[string]any // nil → keep recapBaseCfg domain
		wantErr string
	}{
		{
			name: "valid recap",
			task: TaskConfig{
				"id": "arrive",
				"recap": map[string]any{
					"entity": "lot",
					"columns": []any{
						map[string]any{"field": "numeroFir", "title": "FIR"},
						map[string]any{"field": "netWeight"},
					},
				},
			},
		},
		{
			name: "reserved id column",
			task: TaskConfig{
				"id": "arrive",
				"recap": map[string]any{
					"entity": "lot",
					"columns": []any{
						map[string]any{"field": "id", "title": "Id"},
						map[string]any{"field": "numeroFir"},
					},
				},
			},
		},
		{
			name: "task without recap ok",
			task: TaskConfig{"id": "arrive"},
		},
		{
			name: "empty recap ok",
			task: TaskConfig{"id": "arrive", "recap": map[string]any{}},
		},
		{
			name: "unknown kind",
			task: TaskConfig{
				"id": "arrive",
				"recap": map[string]any{
					"entity":  "pallet",
					"columns": []any{map[string]any{"field": "weight"}},
				},
			},
			wantErr: "not in domain.entities",
		},
		{
			name: "unknown field",
			task: TaskConfig{
				"id": "arrive",
				"recap": map[string]any{
					"entity":  "lot",
					"columns": []any{map[string]any{"field": "missingField"}},
				},
			},
			wantErr: "not in domain.entities.lot.fields",
		},
		{
			name: "empty columns",
			task: TaskConfig{
				"id": "arrive",
				"recap": map[string]any{
					"entity":  "lot",
					"columns": []any{},
				},
			},
			wantErr: "at least one column",
		},
		{
			name: "missing fields list on kind",
			task: TaskConfig{
				"id": "arrive",
				"recap": map[string]any{
					"entity":  "lot",
					"columns": []any{map[string]any{"field": "numeroFir"}},
				},
			},
			domain: map[string]any{
				"entities": map[string]any{
					"lot": map[string]any{},
				},
			},
			wantErr: "must declare fields",
		},
		{
			name: "duplicate column fields",
			task: TaskConfig{
				"id": "arrive",
				"recap": map[string]any{
					"entity": "lot",
					"columns": []any{
						map[string]any{"field": "numeroFir"},
						map[string]any{"field": "numeroFir", "title": "again"},
					},
				},
			},
			wantErr: "duplicate",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg := recapBaseCfg(tc.task)
			if tc.domain != nil {
				cfg.Domain = tc.domain
			}
			err := ValidateRecapConfig(cfg)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("ValidateRecapConfig: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err=%v, want substring %q", err, tc.wantErr)
			}
		})
	}
}

func TestValidateOrgConfigRunsRecap(t *testing.T) {
	t.Parallel()
	cfg := recapBaseCfg(TaskConfig{
		"id": "arrive",
		"recap": map[string]any{
			"entity":  "ghost",
			"columns": []any{map[string]any{"field": "x"}},
		},
	})
	err := ValidateOrgConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "not in domain.entities") {
		t.Fatalf("ValidateOrgConfig err=%v, want recap rejection", err)
	}
}

func TestEntityFieldsFromConfig(t *testing.T) {
	t.Parallel()
	got, err := EntityFieldsFromConfig(OrgConfig{
		Domain: map[string]any{
			"entities": map[string]any{
				"lot":   map[string]any{"fields": []any{"a", "b"}},
				"truck": map[string]any{},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got["lot"]) != 2 || got["lot"][0] != "a" || got["lot"][1] != "b" {
		t.Fatalf("lot fields = %#v", got["lot"])
	}
	if _, ok := got["truck"]; ok {
		t.Fatalf("truck without fields must be omitted, got %#v", got)
	}
}
