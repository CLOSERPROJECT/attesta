package living

import (
	"strings"
	"testing"
)

func TestBindThenMintManyUsesReshapedDollar(t *testing.T) {
	domain := EmptyDomain()
	domain.Container = EntityMap{}
	got, err := RunTransform(domain, RunTransformInput{
		JobInstanceID: "1st",
		Facts: map[string]any{
			"sourceLoadId": "L1",
			"glass": []any{
				map[string]any{"weight": 10.0},
				map[string]any{"weight": nil},
			},
			"plastic": []any{
				map[string]any{"weight": 4.5},
			},
		},
		Catalogs: map[string]any{
			"materials": []any{
				map[string]any{"key": "glass", "label": "Glass"},
				map[string]any{"key": "plastic", "label": "Plastic"},
			},
		},
		Allowed: map[string]struct{}{"container": {}},
		Transform: []any{
			map[string]any{
				"op": "bind",
				"expr": `(
  $rows := $reduce($catalogs.materials, function($acc, $m){(
    $arr := $lookup($, $m.key) ? $lookup($, $m.key) : [];
    $type($arr) != "array" ? $acc : $append($acc, $map(
      $arr[$type(weight)="number" and weight > 0],
      function($r){{"weight": $r.weight, "material": $m.label}}
    ))
  )}, []);
  {"sourceLoadId": sourceLoadId, "rows": $rows}
)`,
			},
			map[string]any{
				"op":     "mintMany",
				"entity": "container",
				"count":  "$count($.rows)",
				"id":     "'C-' & $string($index + 1)",
				"set": map[string]any{
					"weight":       "$.rows[$index].weight",
					"material":     "$.rows[$index].material",
					"sourceLoadId": "$.sourceLoadId",
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Request.Ops) != 2 {
		t.Fatalf("ops = %#v", got.Request.Ops)
	}
	m0 := got.Request.Ops[0].(MintOp)
	if m0.ID != "C-1" || m0.Set["material"] != "Glass" || m0.Set["weight"] != 10.0 || m0.Set["sourceLoadId"] != "L1" {
		t.Fatalf("mint0 = %#v", m0)
	}
	m1 := got.Request.Ops[1].(MintOp)
	if m1.ID != "C-2" || m1.Set["material"] != "Plastic" || m1.Set["weight"] != 4.5 {
		t.Fatalf("mint1 = %#v", m1)
	}
	// Prep facts stay on the request; bind only reshapes `$` for later ops.
	if _, has := got.Request.Facts["rows"]; has {
		t.Fatalf("request facts must stay prep input, got %#v", got.Request.Facts)
	}
}

func TestBindThenMintAndPatch(t *testing.T) {
	domain := EmptyDomain()
	domain.Load = EntityMap{"L1": {"client": "NO"}}
	got, err := RunTransform(domain, RunTransformInput{
		Facts: map[string]any{"rawClient": "SU", "note": "n"},
		Allowed: map[string]struct{}{"load": {}},
		Transform: []any{
			map[string]any{"op": "bind", "expr": `{"client": rawClient, "id": "L2"}`},
			map[string]any{
				"op": "mint", "entity": "load", "id": "$.id", "as": "load",
				"set": map[string]any{"client": "$.client"},
			},
			map[string]any{
				"op": "patch", "entity": "load", "id": `"L1"`,
				"set": map[string]any{"client": "$.client"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Request.Ops) != 2 {
		t.Fatalf("ops = %#v", got.Request.Ops)
	}
	mint := got.Request.Ops[0].(MintOp)
	if mint.ID != "L2" || mint.Set["client"] != "SU" {
		t.Fatalf("mint = %#v", mint)
	}
	patch := got.Request.Ops[1].(PatchOp)
	if patch.ID != "L1" || patch.Set["client"] != "SU" {
		t.Fatalf("patch = %#v", patch)
	}
}

func TestBindErrorsSurfaceAsTransformErrors(t *testing.T) {
	_, err := parseTransformOps([]any{map[string]any{"op": "bind"}}, nil)
	if err == nil || !strings.Contains(err.Error(), "bind requires expr") {
		t.Fatalf("missing expr = %v", err)
	}
	_, err = parseTransformOps([]any{map[string]any{"op": "bind", "expr": "   "}}, nil)
	if err == nil {
		t.Fatal("blank expr")
	}

	_, err = RunTransform(EmptyDomain(), RunTransformInput{
		Facts:     map[string]any{},
		Transform: []any{map[string]any{"op": "bind", "expr": "1+"}},
	})
	var te TransformError
	if err == nil || !asTransformError(err, &te) || te.Kind != "jsonata" {
		t.Fatalf("bad expr = %v", err)
	}

	_, err = RunTransform(EmptyDomain(), RunTransformInput{
		Facts:     map[string]any{},
		Transform: []any{map[string]any{"op": "bind", "expr": "missing.path"}},
	})
	if err == nil || !asTransformError(err, &te) || te.Kind != "type" {
		t.Fatalf("undefined bind = %v", err)
	}

	_, err = parseTransformOps([]any{map[string]any{"op": "fly"}}, nil)
	if err == nil || !asTransformError(err, &te) || te.Kind != "unsupported-op" || te.Op != "fly" {
		t.Fatalf("unsupported = %v", err)
	}
}

func asTransformError(err error, dest *TransformError) bool {
	if err == nil {
		return false
	}
	te, ok := err.(TransformError)
	if !ok {
		return false
	}
	*dest = te
	return true
}
