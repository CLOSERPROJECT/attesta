//go:build integration
// +build integration

package living

import (
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

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
		Mint:          startingNewMint(),
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
		Mint:          startingNewMint(),
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
