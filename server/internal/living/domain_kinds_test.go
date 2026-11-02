package living

import (
	"encoding/json"
	"testing"
)

func TestDomainEntitiesWireRoundTripWellKnownAndExtra(t *testing.T) {
	d := EmptyDomain()
	putKind(&d, kindLoad, EntityMap{"L1": {"client": "NO"}})
	putKind(&d, kindPallet, EntityMap{"L1.1": {"loadId": "L1"}})
	putKind(&d, "custom.kind", EntityMap{"c.1": {"n": 1}})

	raw, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	entities, _ := wire["entities"].(map[string]any)
	if entities["load"] == nil || entities["pallet"] == nil || entities["custom.kind"] == nil {
		t.Fatalf("wire entities = %s", raw)
	}
	if _, has := wire["load"]; has {
		t.Fatalf("contract wire must omit privileged load key: %s", raw)
	}

	var got Domain
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	maps := got.EntityMaps()
	if maps[kindLoad]["L1"]["client"] != "NO" {
		t.Fatalf("load = %#v", maps[kindLoad])
	}
	if maps[kindPallet]["L1.1"]["loadId"] != "L1" {
		t.Fatalf("pallet = %#v", maps[kindPallet])
	}
	if maps["custom.kind"]["c.1"]["n"] != float64(1) {
		t.Fatalf("custom = %#v", maps["custom.kind"])
	}
}

func TestEntityMapsMutateClonePreferEntities(t *testing.T) {
	d := EmptyDomain()
	putKind(&d, "widget", EntityMap{"w1": {"ok": true}})

	maps := d.EntityMaps()
	if !maps["widget"]["w1"]["ok"].(bool) {
		t.Fatalf("EntityMaps = %#v", maps)
	}

	got, err := Mutate(d, MutateRequest{
		UserID: "u", JobID: "j", TaskID: "t", Timestamp: "ts",
		Facts: map[string]any{},
		Ops: []MutateOp{
			MintOp{Entity: EntityLoad, ID: "L1", Set: map[string]any{"client": "NO"}},
			MintOp{Entity: EntityType("widget"), ID: "w2", Set: map[string]any{"ok": false}},
			PatchOp{Entity: EntityType("widget"), ID: "w1", Set: map[string]any{"ok": false}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Entities[kindLoad]["L1"]["client"] != "NO" {
		t.Fatalf("mutate load via Entities = %#v", got.Entities)
	}
	if got.Entities["widget"]["w1"]["ok"] != false || got.Entities["widget"]["w2"]["ok"] != false {
		t.Fatalf("mutate widget = %#v", got.Entities["widget"])
	}

	cloned := cloneDomain(got)
	cloned.Entities[kindLoad]["L1"]["client"] = "SU"
	if got.Entities[kindLoad]["L1"]["client"] != "NO" {
		t.Fatal("cloneDomain must deep-copy Entities")
	}
}

func TestMongoCodecUsesEntities(t *testing.T) {
	domain := EmptyDomain()
	putKind(&domain, kindLoad, EntityMap{"a.b": {"k.v": 1}})
	putKind(&domain, "kind.x", EntityMap{"id.1": {"f.g": true}})

	doc := domainToMongoDoc("org", domain)
	if doc.Entities["load"]["a_b"]["k_v"] != 1 {
		t.Fatalf("encode load = %#v", doc.Entities)
	}
	if doc.Entities["kind_x"]["id_1"]["f_g"] != true {
		t.Fatalf("encode kind.x = %#v", doc.Entities)
	}

	got := domainFromMongoDoc(doc)
	if got.Entities[kindLoad]["a.b"]["k.v"] != 1 {
		t.Fatalf("decode Entities load = %#v", got.Entities)
	}
	if got.Entities["kind.x"]["id.1"]["f.g"] != true {
		t.Fatalf("decode Entities extra = %#v", got.Entities)
	}
}
