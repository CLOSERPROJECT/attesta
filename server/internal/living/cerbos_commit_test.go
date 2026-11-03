package living

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// policySimulatingPDP applies the living commit rule in-process so Commit→Cerbos wiring
// can be tested without a running PDP (role intersection + org match).
func policySimulatingPDP(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Principal struct {
				Attr map[string]any `json:"attr"`
			} `json:"principal"`
			Resource struct {
				Instances map[string]struct {
					Attr map[string]any `json:"attr"`
				} `json:"instances"`
			} `json:"resource"`
			Actions []string `json:"actions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var resourceID string
		var attr map[string]any
		for id, inst := range req.Resource.Instances {
			resourceID = id
			attr = inst.Attr
			break
		}
		effect := "EFFECT_DENY"
		if len(req.Actions) == 1 && req.Actions[0] == "commit" &&
			strAttr(req.Principal.Attr, "orgSlug") == strAttr(attr, "orgSlug") &&
			roleIntersect(req.Principal.Attr["roleSlugs"], attr["rolesAllowed"]) &&
			boolAttr(attr, "sequenceOk") &&
			boolAttr(attr, "readyWhenOk") {
			effect = "EFFECT_ALLOW"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"resourceInstances": map[string]any{
				resourceID: map[string]any{
					"actions": map[string]string{"commit": effect},
				},
			},
		})
	}))
}

func strAttr(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	s, _ := m[key].(string)
	return s
}

func boolAttr(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	b, _ := m[key].(bool)
	return b
}

func roleIntersect(principal any, allowed any) bool {
	var pStr, aStr []string
	pb, _ := json.Marshal(principal)
	ab, _ := json.Marshal(allowed)
	_ = json.Unmarshal(pb, &pStr)
	_ = json.Unmarshal(ab, &aStr)
	for _, p := range pStr {
		for _, a := range aStr {
			if p != "" && p == a {
				return true
			}
		}
	}
	return false
}

func TestCommitWeightCheckDeniedViaCerbosWhenOutOfUnlockOrder(t *testing.T) {
	pdp := policySimulatingPDP(t)
	defer pdp.Close()

	c, domains, _ := testCommitter(t, NewCerbosAuthorizer(pdp.URL, pdp.Client(), time.Now))
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	_, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath:      "intake/weightCheck",
		JobInstanceID: "261019-1st-1",
		Value:         map[string]any{"checkedTruckWeight": 1.0, "checkedLotSum": 1.0},
	})
	var ce *CommitError
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "unauthorized" {
		t.Fatalf("err = %v", err)
	}
	if _, getErr := domains.Get(t.Context(), "feragame"); getErr == nil {
		t.Fatal("domain must stay unset when Unlock order denies")
	}
}

func TestCommitLotRegisterAllowedViaCerbosInUnlockOrder(t *testing.T) {
	pdp := policySimulatingPDP(t)
	defer pdp.Close()

	c, domains, _ := testCommitter(t, NewCerbosAuthorizer(pdp.URL, pdp.Client(), time.Now))
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	truckID := commitTruckArrive(t, c)
	before, err := domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if len(before.Submissions) != 1 {
		t.Fatalf("submissions before register = %#v", before.Submissions)
	}

	result, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath:      GroupTaskPath("intake", "lots", ProvisionalGroupInstanceID, "register"),
		JobInstanceID: truckID,
		Value:         registerLotValue(),
		Mint:          spawningNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	loadID := firstLoadID(result.Domain)
	if result.Domain.Kind("load")[loadID]["numeroFir"] != "FIR-1" {
		t.Fatalf("domain = %#v", result.Domain.Kind("load")[loadID])
	}
	last := result.Domain.Submissions[len(result.Domain.Submissions)-1]
	if last.TaskID != "register" || last.GroupID != "lots" {
		t.Fatalf("submissions = %#v", result.Domain.Submissions)
	}
}

func TestCommitOutOfOrderDoesNotMutateExistingDomain(t *testing.T) {
	pdp := policySimulatingPDP(t)
	defer pdp.Close()

	c, domains, _ := testCommitter(t, NewCerbosAuthorizer(pdp.URL, pdp.Client(), time.Now))
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	truckID := commitTruckArrive(t, c)
	before, err := domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "intake/weightCheck", JobInstanceID: truckID,
		Value: map[string]any{"checkedTruckWeight": 1.0, "checkedLotSum": 1.0},
	})
	var ce *CommitError
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "unauthorized" {
		t.Fatalf("weightCheck before lots close should deny via Unlock: %v", err)
	}
	after, err := domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Submissions) != len(before.Submissions) {
		t.Fatalf("submissions mutated on deny: before=%d after=%d", len(before.Submissions), len(after.Submissions))
	}
}

func TestCommitTruckArriveAllowedViaCerbosWhenRolePermitted(t *testing.T) {
	pdp := policySimulatingPDP(t)
	defer pdp.Close()

	c, _, _ := testCommitter(t, NewCerbosAuthorizer(pdp.URL, pdp.Client(), time.Now))
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	result, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "intake/truckArrive",
		Value:    truckArriveValue(),
		Mint:     startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Domain.Kind("truck")[result.JobInstanceID]["turno"] != "1st" {
		t.Fatalf("domain = %#v", result.Domain)
	}
}

func TestCommitTruckArriveDeniedViaCerbosWhenRoleDisallowed(t *testing.T) {
	pdp := policySimulatingPDP(t)
	defer pdp.Close()

	c, domains, _ := testCommitter(t, NewCerbosAuthorizer(pdp.URL, pdp.Client(), time.Now))
	_, err := c.Commit(t.Context(), "feragame", OperatorIdentity{
		UserID: "gio",
		Org:    "feragame",
		Roles:  []string{"logistic"},
	}, CommitRequest{
		TaskPath: "intake/truckArrive",
		Value:    truckArriveValue(),
		Mint:     startingNewMint(),
	})
	var ce *CommitError
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "unauthorized" {
		t.Fatalf("err = %v", err)
	}
	if _, getErr := domains.Get(t.Context(), "feragame"); getErr == nil {
		t.Fatal("domain should remain unset when Cerbos denies")
	}
}
