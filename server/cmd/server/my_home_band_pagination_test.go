package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestMyHomeYourTurnBareGETRedirectsToMy(t *testing.T) {
	server, sessionID := newMyHomeBandPaginationServer(t, 1, 0)

	req := httptest.NewRequest(http.MethodGet, "/my/home/your-turn?page=1", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleMyRoutes(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d, want %d; body=%q", rec.Code, http.StatusFound, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); loc != appHomePath {
		t.Fatalf("Location=%q, want %q", loc, appHomePath)
	}
}

func TestMyHomeUpcomingBareGETRedirectsToMy(t *testing.T) {
	server, sessionID := newMyHomeBandPaginationServer(t, 0, 1)

	req := httptest.NewRequest(http.MethodGet, "/my/home/upcoming?page=1", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleMyRoutes(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d, want %d; body=%q", rec.Code, http.StatusFound, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); loc != appHomePath {
		t.Fatalf("Location=%q, want %q", loc, appHomePath)
	}
}

func TestMyHomeYourTurnHTMXFragmentPaginatesAndClamps(t *testing.T) {
	server, sessionID := newMyHomeBandPaginationServer(t, myHomeBandPageSize+2, 0)

	req := httptest.NewRequest(http.MethodGet, "/my/home/your-turn?page=99", nil)
	req.Header.Set("HX-Request", "true")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleMyRoutes(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d; body=%q", rec.Code, http.StatusOK, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, "<html") || strings.Contains(body, `class="topbar"`) {
		t.Fatalf("HTMX fragment must not include layout, got:\n%s", body)
	}
	if !strings.Contains(body, `id="my-home-your-turn"`) {
		t.Fatalf("expected stable section id, got:\n%s", body)
	}
	if !strings.Contains(body, `class="pagination`) {
		t.Fatalf("expected shared .pagination, got:\n%s", body)
	}
	if !strings.Contains(body, `hx-target="#my-home-your-turn"`) {
		t.Fatalf("expected hx-target for section swap, got:\n%s", body)
	}
	if strings.Contains(body, `hx-push-url`) {
		t.Fatalf("PushURL must be false, got:\n%s", body)
	}
	// Clamped to last page: 7 items → page 2 has 2 rows.
	if got := strings.Count(body, `class="list-row"`); got != 2 {
		t.Fatalf("list-row count=%d, want 2 on clamped last page; body:\n%s", got, body)
	}
	if !strings.Contains(body, `/my/home/your-turn?page=2`) {
		t.Fatalf("expected page=2 links after clamp, got:\n%s", body)
	}
}

func TestMyHomeUpcomingHTMXFragmentPageSlice(t *testing.T) {
	server, sessionID := newMyHomeBandPaginationServer(t, 0, myHomeBandPageSize+1)

	req := httptest.NewRequest(http.MethodGet, "/my/home/upcoming?page=1", nil)
	req.Header.Set("HX-Request", "true")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleMyRoutes(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `id="my-home-upcoming"`) {
		t.Fatalf("expected stable section id, got:\n%s", body)
	}
	if !strings.Contains(body, `class="pagination`) {
		t.Fatalf("expected shared .pagination, got:\n%s", body)
	}
	if !strings.Contains(body, `hx-target="#my-home-upcoming"`) {
		t.Fatalf("expected hx-target, got:\n%s", body)
	}
	if strings.Contains(body, `hx-push-url`) {
		t.Fatalf("PushURL must be false, got:\n%s", body)
	}
	if got := strings.Count(body, `class="list-row"`); got != myHomeBandPageSize {
		t.Fatalf("list-row count=%d, want %d; body:\n%s", got, myHomeBandPageSize, body)
	}
}

func TestOperatorHomeYourTurnAndUpcomingShowAtMostPageSize(t *testing.T) {
	server, sessionID := newMyHomeBandPaginationServer(t, myHomeBandPageSize+3, myHomeBandPageSize+2)

	req := httptest.NewRequest(http.MethodGet, "/my", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleHome(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	yourTurnStart := strings.Index(body, `id="my-home-your-turn"`)
	upcomingStart := strings.Index(body, `id="my-home-upcoming"`)
	if yourTurnStart < 0 || upcomingStart < 0 || yourTurnStart >= upcomingStart {
		t.Fatalf("expected both band ids in order, yourTurn=%d upcoming=%d", yourTurnStart, upcomingStart)
	}
	yourTurnSection := body[yourTurnStart:upcomingStart]
	if got := strings.Count(yourTurnSection, `class="list-row"`); got != myHomeBandPageSize {
		t.Fatalf("Your turn list-row count=%d, want %d", got, myHomeBandPageSize)
	}
	upcomingSection := body[upcomingStart:]
	chooseIdx := strings.Index(upcomingSection, "Choose a stream")
	if chooseIdx >= 0 {
		upcomingSection = upcomingSection[:chooseIdx]
	}
	if got := strings.Count(upcomingSection, `class="list-row"`); got != myHomeBandPageSize {
		t.Fatalf("Upcoming list-row count=%d, want %d", got, myHomeBandPageSize)
	}
	for _, want := range []string{
		`hx-target="#my-home-your-turn"`,
		`hx-target="#my-home-upcoming"`,
		`/my/home/your-turn?page=`,
		`/my/home/upcoming?page=`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q on /my page 1, got:\n%s", want, body)
		}
	}
	// Scope to band sections — the rest of /my may use hx-push-url elsewhere.
	for name, section := range map[string]string{
		"Your turn": yourTurnSection,
		"Upcoming":  upcomingSection,
	} {
		if strings.Contains(section, `hx-push-url="true"`) {
			t.Fatalf("%s band pager must not push URL, got hx-push-url=true in:\n%s", name, section)
		}
	}
	// Chrome Attention still reflects full unpaginated your-turn count.
	if !strings.Contains(body, `attention-dot`) {
		t.Fatalf("expected Attention dots from full your-turn count, got:\n%s", body)
	}
}

func TestNormalizeMyHomeBandPage(t *testing.T) {
	if got := normalizeMyHomeBandPage(0, 0); got != 1 {
		t.Fatalf("empty list page 0 → %d, want 1", got)
	}
	if got := normalizeMyHomeBandPage(99, myHomeBandPageSize+2); got != 2 {
		t.Fatalf("clamp high → %d, want 2", got)
	}
	if got := normalizeMyHomeBandPage(2, myHomeBandPageSize*2); got != 2 {
		t.Fatalf("in-range → %d, want 2", got)
	}
}

// newMyHomeBandPaginationServer seeds yourTurnCount org1-actionable processes and
// upcomingCount processes waiting on org2. Uses the two-org workflow fixture.
func newMyHomeBandPaginationServer(t *testing.T, yourTurnCount, upcomingCount int) (*Server, string) {
	t.Helper()
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-home-band-page"
	tempDir := t.TempDir()
	writeTwoOrgWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"))

	store := NewMemoryStore()
	for i := 0; i < yourTurnCount; i++ {
		store.SeedProcess(Process{
			WorkflowKey: "workflow",
			Name:        "Your turn " + strconv.Itoa(i+1),
			Status:      processStatusActive,
			CreatedAt:   now.Add(-time.Duration(i) * time.Minute),
			Progress: map[string]ProcessStep{
				"1.1": {State: "pending"},
				"2.1": {State: "pending"},
			},
		})
	}
	for i := 0; i < upcomingCount; i++ {
		store.SeedProcess(Process{
			WorkflowKey: "workflow",
			Name:        "Upcoming " + strconv.Itoa(i+1),
			Status:      processStatusActive,
			CreatedAt:   now.Add(-time.Duration(100+i) * time.Minute),
			Progress: map[string]ProcessStep{
				"1.1": {State: "done", DoneAt: &now},
				"2.1": {State: "pending"},
			},
		})
	}

	user := AccountUser{
		IdentityUserID: "member-1",
		Email:          "member@org1.example",
		OrgSlug:        "org1",
		RoleSlugs:      []string{"dep1"},
		Status:         "active",
	}
	server := &Server{
		identity:    testIdentityForSessions(now, map[string]AccountUser{sessionID: user}),
		store:       store,
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		configDir:   tempDir,
		now:         func() time.Time { return now },
	}
	return server, sessionID
}

func TestMyHomeJoinRequestsBareGETRedirectsToMy(t *testing.T) {
	server, sessionID := newMyHomeJoinRequestsPaginationServer(t, 1)

	req := httptest.NewRequest(http.MethodGet, "/my/home/join-requests?page=1", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleMyRoutes(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d, want %d; body=%q", rec.Code, http.StatusFound, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); loc != appHomePath {
		t.Fatalf("Location=%q, want %q", loc, appHomePath)
	}
}

func TestMyHomeJoinRequestsHTMXFragmentPaginates(t *testing.T) {
	server, sessionID := newMyHomeJoinRequestsPaginationServer(t, myHomeBandPageSize+1)

	req := httptest.NewRequest(http.MethodGet, "/my/home/join-requests?page=1", nil)
	req.Header.Set("HX-Request", "true")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleMyRoutes(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `id="my-home-join-requests"`) {
		t.Fatalf("expected stable section id, got:\n%s", body)
	}
	if !strings.Contains(body, `class="pagination`) {
		t.Fatalf("expected shared .pagination, got:\n%s", body)
	}
	if !strings.Contains(body, `hx-target="#my-home-join-requests"`) {
		t.Fatalf("expected hx-target, got:\n%s", body)
	}
	if strings.Contains(body, `hx-push-url`) {
		t.Fatalf("PushURL must be false, got:\n%s", body)
	}
	if got := strings.Count(body, `class="list-row"`); got != myHomeBandPageSize {
		t.Fatalf("list-row count=%d, want %d; body:\n%s", got, myHomeBandPageSize, body)
	}
}

func TestOperatorHomeJoinRequestsShowAtMostPageSize(t *testing.T) {
	server, sessionID := newMyHomeJoinRequestsPaginationServer(t, myHomeBandPageSize+2)

	req := httptest.NewRequest(http.MethodGet, "/my", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleHome(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	start := strings.Index(body, `id="my-home-join-requests"`)
	if start < 0 {
		t.Fatalf("expected join-requests band on /my, got:\n%s", body)
	}
	section := body[start:]
	if end := strings.Index(section, `class="my-home-rule"`); end > 0 {
		section = section[:end]
	}
	if got := strings.Count(section, `class="list-row"`); got != myHomeBandPageSize {
		t.Fatalf("Join requests list-row count=%d, want %d", got, myHomeBandPageSize)
	}
	if !strings.Contains(body, `/my/home/join-requests?page=`) {
		t.Fatalf("expected join-requests pager links, got:\n%s", body)
	}
}

func TestMyHomeOrgCreationBareGETRedirectsToMy(t *testing.T) {
	server := newMyHomeOrgCreationPaginationServer(t, 1)
	sessionID := platformAdminSessionValue()

	req := httptest.NewRequest(http.MethodGet, "/my/home/org-creation?page=1", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleMyRoutes(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status=%d, want %d; body=%q", rec.Code, http.StatusFound, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); loc != appHomePath {
		t.Fatalf("Location=%q, want %q", loc, appHomePath)
	}
}

func TestMyHomeOrgCreationHTMXFragmentPaginates(t *testing.T) {
	server := newMyHomeOrgCreationPaginationServer(t, myHomeBandPageSize+1)
	sessionID := platformAdminSessionValue()

	req := httptest.NewRequest(http.MethodGet, "/my/home/org-creation?page=1", nil)
	req.Header.Set("HX-Request", "true")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: sessionID})
	rec := httptest.NewRecorder()
	server.handleMyRoutes(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `id="my-home-org-creation"`) {
		t.Fatalf("expected stable section id, got:\n%s", body)
	}
	if !strings.Contains(body, `class="pagination`) {
		t.Fatalf("expected shared .pagination, got:\n%s", body)
	}
	if !strings.Contains(body, `hx-target="#my-home-org-creation"`) {
		t.Fatalf("expected hx-target, got:\n%s", body)
	}
	if strings.Contains(body, `hx-push-url`) {
		t.Fatalf("PushURL must be false, got:\n%s", body)
	}
	if got := strings.Count(body, `class="list-row"`); got != myHomeBandPageSize {
		t.Fatalf("list-row count=%d, want %d; body:\n%s", got, myHomeBandPageSize, body)
	}
}

func newMyHomeJoinRequestsPaginationServer(t *testing.T, joinCount int) (*Server, string) {
	t.Helper()
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	sessionID := "session-home-join-page"
	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "Main workflow", "string")

	store := NewMemoryStore()
	for i := 0; i < joinCount; i++ {
		if _, err := store.InsertJoinRequest(context.Background(), JoinRequest{
			RequesterUserID: "joiner-" + strconv.Itoa(i+1),
			RequesterEmail:  "joiner" + strconv.Itoa(i+1) + "@example.com",
			OrgSlug:         "acme",
			RoleSlugs:       []string{"viewer"},
			Status:          AffiliationStatusPending,
			CreatedAt:       now.Add(-time.Duration(i) * time.Minute),
			UpdatedAt:       now,
		}); err != nil {
			t.Fatalf("InsertJoinRequest: %v", err)
		}
	}

	user := AccountUser{
		IdentityUserID: "admin-1",
		Email:          "admin@acme.example",
		OrgSlug:        "acme",
		RoleSlugs:      []string{"org_admin"},
		Status:         "active",
	}
	identity := testIdentityForSessions(now, map[string]AccountUser{sessionID: user})
	identity.getOrganizationBySlugFunc = func(ctx context.Context, slug string) (*IdentityOrg, error) {
		return &IdentityOrg{
			ID: "team-1", Slug: "acme", Name: "Acme Org",
			Roles: []IdentityRole{{Slug: "viewer", Name: "Viewer"}},
		}, nil
	}

	server := &Server{
		identity:    identity,
		store:       store,
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		configDir:   tempDir,
		now:         func() time.Time { return now },
	}
	return server, sessionID
}

func newMyHomeOrgCreationPaginationServer(t *testing.T, orgCount int) *Server {
	t.Helper()
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	tempDir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"), "Main workflow", "string")

	store := NewMemoryStore()
	for i := 0; i < orgCount; i++ {
		n := strconv.Itoa(i + 1)
		if _, err := store.InsertOrganizationCreationRequest(context.Background(), OrganizationCreationRequest{
			RequesterUserID: "founder-" + n,
			RequesterEmail:  "founder" + n + "@example.com",
			ProposedName:    "Org " + n,
			ProposedSlug:    "org-" + n,
			Status:          AffiliationStatusPending,
			CreatedAt:       now.Add(-time.Duration(i) * time.Minute),
			UpdatedAt:       now,
		}); err != nil {
			t.Fatalf("InsertOrganizationCreationRequest: %v", err)
		}
	}

	return &Server{
		identity: &fakeIdentityStore{
			listOrganizationsPageFunc: func(ctx context.Context, opts IdentityOrgListOptions) (IdentityOrgPage, error) {
				return IdentityOrgPage{}, nil
			},
		},
		store:       store,
		tmpl:        parseTestTemplates(t),
		authorizer:  fakeAuthorizer{},
		enforceAuth: true,
		configDir:   tempDir,
		now:         func() time.Time { return now },
	}
}
