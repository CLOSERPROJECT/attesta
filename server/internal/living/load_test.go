package living

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoaderEmptyDomainAndActiveConfig(t *testing.T) {
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	if _, err := SeedFixtureConfigStore(configs); err != nil {
		t.Fatal(err)
	}

	loader := &Loader{Domains: domains, Configs: configs}
	result, err := loader.Load(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if result.Org != "feragame" {
		t.Fatalf("org = %q", result.Org)
	}
	if result.ActiveConfigVersion != DefaultFixtureVersion {
		t.Fatalf("active = %q", result.ActiveConfigVersion)
	}
	if len(result.Domain.Load) != 0 || len(result.Domain.Submissions) != 0 || len(result.Domain.JobInstances) != 0 {
		t.Fatalf("domain not empty: %#v", result.Domain)
	}
	if result.Config.ID != "truck-loop" {
		t.Fatalf("config id = %q", result.Config.ID)
	}
	if _, ok := result.Config.Jobs.Get("intake"); !ok {
		t.Fatal("missing intake job")
	}
}

func TestLoaderMissingActiveConfig(t *testing.T) {
	loader := &Loader{
		Domains: NewMemoryDomainStore(),
		Configs: NewMemoryConfigStore(),
	}
	_, err := loader.Load(t.Context(), "feragame")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestHandleLoadVerifiedEmptyDomain(t *testing.T) {
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	if _, err := SeedFixtureConfigStore(configs); err != nil {
		t.Fatal(err)
	}
	h := &Handler{
		Loader: &Loader{Domains: domains, Configs: configs},
		Gate: func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
			return OperatorIdentity{UserID: "u1", Org: "feragame", Roles: []string{"administration"}}, true
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/living/domain", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}

	var got LoadResult
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.ActiveConfigVersion != "1" || got.Config.Org != "feragame" {
		t.Fatalf("got = %#v", got)
	}
	if len(got.Domain.JobInstances) != 0 {
		t.Fatalf("expected empty domain")
	}
}

func TestHandleLoadRejectsWhenGateDenies(t *testing.T) {
	h := &Handler{
		Loader: &Loader{Domains: NewMemoryDomainStore(), Configs: NewMemoryConfigStore()},
		Gate: func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return OperatorIdentity{}, false
		},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/living/domain", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestMemoryStoresRoundTrip(t *testing.T) {
	domains := NewMemoryDomainStore()
	domain := EmptyDomain()
	domain.Load["L1"] = map[string]any{"client": "NO"}
	if err := domains.Put(t.Context(), "feragame", domain); err != nil {
		t.Fatal(err)
	}
	got, err := domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if got.Load["L1"]["client"] != "NO" {
		t.Fatalf("got = %#v", got)
	}

	configs := NewMemoryConfigStore()
	cfg := OrgConfig{ID: "x", Jobs: newOrderedObject()}
	if err := configs.Put(t.Context(), "feragame", "2", cfg); err != nil {
		t.Fatal(err)
	}
	if err := configs.SetActiveVersion(t.Context(), "feragame", "2"); err != nil {
		t.Fatal(err)
	}
	active, err := configs.ActiveVersion(t.Context(), "feragame")
	if err != nil || active != "2" {
		t.Fatalf("active=%q err=%v", active, err)
	}
}
