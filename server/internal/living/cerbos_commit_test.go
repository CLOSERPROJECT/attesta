package living

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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

func TestCommitCodiceFirDeniedViaCerbosWhenOutOfUnlockOrder(t *testing.T) {
	pdp := policySimulatingPDP(t)
	defer pdp.Close()

	c, domains, _ := testCommitter(t, NewCerbosAuthorizer(pdp.URL, pdp.Client(), time.Now))
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	_, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath:      "intake/codiceFir",
		JobInstanceID: "NIR-261019-NO-1",
		Value:         map[string]any{"codiceFir": "FIR-TOO-EARLY"},
	})
	var ce *CommitError
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "unauthorized" {
		t.Fatalf("err = %v", err)
	}
	if _, getErr := domains.Get(t.Context(), "feragame"); getErr == nil {
		t.Fatal("domain must stay unset when Unlock order denies")
	}
}

func TestCommitCodiceFirAllowedViaCerbosInUnlockOrder(t *testing.T) {
	pdp := policySimulatingPDP(t)
	defer pdp.Close()

	c, domains, _ := testCommitter(t, NewCerbosAuthorizer(pdp.URL, pdp.Client(), time.Now))
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	instanceID := "NIR-261019-NO-1"
	if _, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "intake/register", JobInstanceID: instanceID, Value: registerValue(),
	}); err != nil {
		t.Fatal(err)
	}
	before, err := domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if len(before.Submissions) != 1 {
		t.Fatalf("submissions before codiceFir = %#v", before.Submissions)
	}

	result, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "intake/codiceFir", JobInstanceID: instanceID,
		Value: map[string]any{"codiceFir": "FIR-OK"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Domain.Load[instanceID]["codiceFir"] != "FIR-OK" {
		t.Fatalf("domain = %#v", result.Domain.Load[instanceID])
	}
	if len(result.Domain.Submissions) != 2 || result.Domain.Submissions[1].TaskID != "codiceFir" {
		t.Fatalf("submissions = %#v", result.Domain.Submissions)
	}
}

func TestCommitOutOfOrderDoesNotMutateExistingDomain(t *testing.T) {
	pdp := policySimulatingPDP(t)
	defer pdp.Close()

	c, domains, _ := testCommitter(t, NewCerbosAuthorizer(pdp.URL, pdp.Client(), time.Now))
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	instanceID := "NIR-261019-NO-1"
	if _, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "intake/register", JobInstanceID: instanceID, Value: registerValue(),
	}); err != nil {
		t.Fatal(err)
	}
	before, err := domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}

	// palletWeight is next after codiceFir only; skipping codiceFir must deny and leave Domain intact.
	_, err = c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "intake/palletWeight", JobInstanceID: instanceID,
		Value: map[string]any{},
	})
	var ce *CommitError
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "unauthorized" {
		t.Fatalf("skipping codiceFir should deny via Unlock: %v", err)
	}
	after, err := domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Submissions) != len(before.Submissions) {
		t.Fatalf("submissions mutated on deny: before=%d after=%d", len(before.Submissions), len(after.Submissions))
	}
	if _, hasFir := after.Load[instanceID]["codiceFir"]; hasFir {
		t.Fatal("codiceFir fact must not appear on out-of-order deny")
	}
}

func TestCommitRegisterAllowedViaCerbosWhenRolePermitted(t *testing.T) {
	pdp := policySimulatingPDP(t)
	defer pdp.Close()

	c, _, _ := testCommitter(t, NewCerbosAuthorizer(pdp.URL, pdp.Client(), time.Now))
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	result, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath:      "intake/register",
		JobInstanceID: "NIR-261019-NO-1",
		Value:         registerValue(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Domain.Load["NIR-261019-NO-1"]["client"] != "NO" {
		t.Fatalf("domain = %#v", result.Domain)
	}
}

func TestCommitRegisterDeniedViaCerbosWhenRoleDisallowed(t *testing.T) {
	pdp := policySimulatingPDP(t)
	defer pdp.Close()

	c, domains, _ := testCommitter(t, NewCerbosAuthorizer(pdp.URL, pdp.Client(), time.Now))
	_, err := c.Commit(t.Context(), "feragame", OperatorIdentity{
		UserID: "gio",
		Org:    "feragame",
		Roles:  []string{"logistic"},
	}, CommitRequest{
		TaskPath:      "intake/register",
		JobInstanceID: "NIR-261019-NO-1",
		Value:         registerValue(),
	})
	var ce *CommitError
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "unauthorized" {
		t.Fatalf("err = %v", err)
	}
	if _, getErr := domains.Get(t.Context(), "feragame"); getErr == nil {
		t.Fatal("domain should remain unset when Cerbos denies")
	}
}

func TestCommitRegisterIntegrationCerbosPolicy(t *testing.T) {
	cerbosURL := strings.TrimSpace(os.Getenv("CERBOS_URL"))
	if cerbosURL == "" {
		cerbosURL = "http://localhost:3592"
	}
	if !cerbosHealthy(cerbosURL) {
		t.Skipf("skip: cerbos unavailable at %s", cerbosURL)
	}

	auth := NewCerbosAuthorizer(cerbosURL, http.DefaultClient, time.Now)
	c, domains, _ := testCommitter(t, auth)

	allowOp := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	if _, err := c.Commit(t.Context(), "feragame", allowOp, CommitRequest{
		TaskPath:      "intake/register",
		JobInstanceID: "NIR-261019-NO-ALLOW",
		Value:         registerValue(),
	}); err != nil {
		t.Fatalf("administration should be allowed by living policy: %v", err)
	}

	denyDomains := NewMemoryDomainStore()
	denyConfigs := NewMemoryConfigStore()
	if _, err := SeedFixtureConfigStore(denyConfigs); err != nil {
		t.Fatal(err)
	}
	denyCommitter := &Committer{
		Domains:    denyDomains,
		Configs:    denyConfigs,
		Authorizer: auth,
		Now:        func() time.Time { return time.Date(2026, 10, 29, 15, 0, 0, 0, time.UTC) },
	}
	_, err := denyCommitter.Commit(t.Context(), "feragame", OperatorIdentity{
		UserID: "gio", Org: "feragame", Roles: []string{"logistic"},
	}, CommitRequest{
		TaskPath:      "intake/register",
		JobInstanceID: "NIR-261019-NO-DENY",
		Value:         registerValue(),
	})
	var ce *CommitError
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "unauthorized" {
		t.Fatalf("logistic should be denied: %v", err)
	}
	if _, getErr := denyDomains.Get(t.Context(), "feragame"); getErr == nil {
		t.Fatal("domain should remain unset on deny")
	}
	_ = domains
}

func cerbosHealthy(url string) bool {
	endpoint := strings.TrimSuffix(url, "/") + "/_cerbos/health"
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
