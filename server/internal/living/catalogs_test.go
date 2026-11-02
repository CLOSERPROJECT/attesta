package living

import (
	"testing"
)

func TestOrgConfigCatalogsJSONRoundTrip(t *testing.T) {
	raw := []byte(`{
		"id": "c",
		"version": "1",
		"catalogs": {"tareKinds": {"euro": 25}, "clients": {"NO": "Nord"}},
		"jobs": {}
	}`)
	var cfg OrgConfig
	if err := cfg.UnmarshalJSON(raw); err != nil {
		t.Fatal(err)
	}
	tare, _ := cfg.Catalogs["tareKinds"].(map[string]any)
	if tare["euro"] != float64(25) {
		t.Fatalf("catalogs.tareKinds.euro = %#v", tare["euro"])
	}
}

func TestDomainBindingsExposeCatalogs(t *testing.T) {
	bindings := domainBindings(EmptyDomain(), "inst-1", map[string]any{
		"tareKinds": map[string]any{"euro": 25},
	})
	catalogs, ok := bindings["catalogs"].(map[string]any)
	if !ok {
		t.Fatalf("bindings.catalogs = %#v", bindings["catalogs"])
	}
	tare, _ := catalogs["tareKinds"].(map[string]any)
	if tare["euro"] != 25 {
		t.Fatalf("catalogs.tareKinds = %#v", tare)
	}
	got, err := evaluateExpr("$lookup($catalogs.tareKinds, 'euro')", map[string]any{}, bindings)
	if err != nil {
		t.Fatal(err)
	}
	if got != 25 {
		t.Fatalf("$catalogs lookup = %#v, want 25", got)
	}
}

func TestDomainBindingsEmptyCatalogsWhenNil(t *testing.T) {
	bindings := domainBindings(EmptyDomain(), "", nil)
	catalogs, ok := bindings["catalogs"].(map[string]any)
	if !ok || len(catalogs) != 0 {
		t.Fatalf("nil catalogs → empty map, got %#v", bindings["catalogs"])
	}
}

func TestRunTransformBindingsIncludeCatalogs(t *testing.T) {
	domain := EmptyDomain()
	putKind(&domain, "load", EntityMap{})
	got, err := RunTransform(domain, RunTransformInput{
		JobInstanceID: "L1",
		Facts:         map[string]any{},
		Transform: []any{
			map[string]any{
				"op":     "mint",
				"entity": "load",
				"id":     "'L1'",
				"set": map[string]any{
					"note": "$lookup($catalogs.tareKinds, 'euro')",
				},
			},
		},
		Allowed:  map[string]struct{}{"load": {}},
		Catalogs: map[string]any{"tareKinds": map[string]any{"euro": 25}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got.Bindings["catalogs"].(map[string]any); !ok {
		t.Fatalf("transform bindings missing catalogs: %#v", got.Bindings)
	}
	if len(got.Request.Ops) != 1 {
		t.Fatalf("ops = %#v", got.Request.Ops)
	}
	mint, ok := got.Request.Ops[0].(MintOp)
	if !ok {
		t.Fatalf("op = %#v", got.Request.Ops[0])
	}
	if mint.Set["note"] != 25 {
		t.Fatalf("set.note = %#v, want 25 from $catalogs", mint.Set["note"])
	}
}

func TestBuildFormBindingsNoTareKindsInjection(t *testing.T) {
	domain := EmptyDomain()
	putKind(&domain, "load", EntityMap{"L1": {"client": "NO"}})
	bindings := buildFormBindings(domain, AvailableTaskRef{
		JobID: "intake", TaskID: "palletWeight", JobInstanceID: "L1",
	}, map[string]any{
		"tareKinds": map[string]any{"euro": 25, "industrial": 30},
	})
	domainBag, _ := bindings["domain"].(map[string]any)
	if _, has := domainBag["tareKinds"]; has {
		t.Fatal("buildFormBindings must not copy tareKinds into domain")
	}
	catalogs, _ := bindings["catalogs"].(map[string]any)
	tare, _ := catalogs["tareKinds"].(map[string]any)
	if tare["euro"] != float64(25) && tare["euro"] != 25 {
		t.Fatalf("bindings.catalogs.tareKinds = %#v", tare)
	}
}

func TestResolveReadWhereUsesCatalogs(t *testing.T) {
	domain := domainFromKinds(map[string]EntityMap{
		"load": {
			"L1": {"client": "NO"},
			"L2": {"client": "SU"},
		},
	})
	ids, err := ResolveRead(domain, ReadDef{
		Entity: EntityType("load"),
		Where:  "client = $lookup($catalogs.clients, 'nord')",
		Key:    "id",
	}, "", "", map[string]any{"clients": map[string]any{"nord": "NO"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != "L1" {
		t.Fatalf("ids = %#v, want [L1] via $catalogs", ids)
	}
}

func TestRunSubmitHTTPEffectsUsesCatalogs(t *testing.T) {
	port := &recordingHttpPort{responses: []HttpPortResponse{{URL: "blob:ok"}}}
	result := runSubmitHTTPEffects(t.Context(), port, []SubmitHttpEffect{
		{
			Method: `"POST"`,
			URL:    `"/" & $lookup($catalogs.clients, 'nord')`,
			Label:  `"L"`,
			Action: "link",
		},
	}, map[string]any{}, nil, EmptyDomain(), "", map[string]any{"clients": map[string]any{"nord": "NO"}})
	if len(result.Errors) != 0 {
		t.Fatalf("errors = %#v", result.Errors)
	}
	if len(port.calls) != 1 || port.calls[0].URL != "/NO" {
		t.Fatalf("calls = %#v, want URL /NO from $catalogs", port.calls)
	}
}

func TestBuildTaskFormBundleExposesCatalogsNotTaskTareKinds(t *testing.T) {
	cfg := OrgConfig{
		ID:      "probe",
		Version: "1",
		Catalogs: map[string]any{
			"tareKinds": map[string]any{"euro": 25},
		},
		Jobs: OrderedObject{
			Keys: []string{"intake"},
			Values: map[string]any{
				"intake": Job{
					Repeatable: true,
					InstanceID: `"x"`,
					Label:      `"x"`,
					Tasks: []TaskConfig{
						{
							"id":    "register",
							"roles": []any{"administration"},
							"schema": map[string]any{
								"type":       "object",
								"properties": map[string]any{},
							},
							"tareKinds": map[string]any{"euro": 99},
						},
					},
				},
			},
		},
	}
	bundle, err := BuildTaskFormBundle(cfg, EmptyDomain(), AvailableTaskRef{
		JobID: "intake", TaskID: "register",
	})
	if err != nil {
		t.Fatal(err)
	}
	domainBag, _ := bundle.Bindings["domain"].(map[string]any)
	if _, has := domainBag["tareKinds"]; has {
		t.Fatal("Task-local tareKinds must not land in bindings.domain")
	}
	catalogs, _ := bundle.Bindings["catalogs"].(map[string]any)
	tare, _ := catalogs["tareKinds"].(map[string]any)
	if tare["euro"] != float64(25) {
		t.Fatalf("bindings.catalogs.tareKinds = %#v, want org catalogs", tare)
	}
}
