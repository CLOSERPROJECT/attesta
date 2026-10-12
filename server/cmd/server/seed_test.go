package main

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestSeedEmptyOnlyNoopsWhenMarkerOrgExists(t *testing.T) {
	store := NewMemoryStore()
	identityCalls := 0
	identity := &fakeIdentityStore{
		getOrganizationBySlugFunc: func(_ context.Context, slug string) (*IdentityOrg, error) {
			identityCalls++
			if slug != demoFixtureMarkerOrgSlug {
				t.Fatalf("marker lookup slug = %q, want %q", slug, demoFixtureMarkerOrgSlug)
			}
			return &IdentityOrg{ID: demoFixtureMarkerOrgSlug, Slug: demoFixtureMarkerOrgSlug, Name: "Organization 1"}, nil
		},
		ensurePlatformAdminAccountFunc: func(context.Context, string, string) error {
			t.Fatal("empty-only must not touch identity writes")
			return nil
		},
		createOrganizationAsAdminFunc: func(context.Context, string) (IdentityOrg, error) {
			t.Fatal("empty-only must not create organizations")
			return IdentityOrg{}, nil
		},
	}

	err := Seed(t.Context(), SeedDeps{
		Identity: identity,
		Store:    store,
	})
	if err != nil {
		t.Fatalf("Seed: %v", err)
	}
	if identityCalls != 1 {
		t.Fatalf("marker lookups = %d, want 1", identityCalls)
	}
	cats, err := store.ListCategories(t.Context())
	if err != nil {
		t.Fatalf("ListCategories: %v", err)
	}
	if len(cats) != 0 {
		t.Fatalf("taxonomy changed on empty-only no-op: %d categories", len(cats))
	}
}

func TestSeedAppliesIdentityWhenEmpty(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@attesta.com")
	t.Setenv("ADMIN_PASSWORD", "admin@attesta.com")

	store := NewMemoryStore()
	identity := newSeedFakeIdentity()
	dir := writeSeedTestConfigDir(t)

	err := Seed(t.Context(), SeedDeps{
		Identity:  identity,
		Store:     store,
		ConfigDir: dir,
		RNG:       &fixedCatalogRNG{n: 1},
	})
	if err != nil {
		t.Fatalf("Seed: %v", err)
	}

	org, err := identity.GetOrganizationBySlug(t.Context(), demoFixtureMarkerOrgSlug)
	if err != nil {
		t.Fatalf("marker org missing after seed: %v", err)
	}
	if org.ID != demoFixtureMarkerOrgSlug {
		t.Fatalf("org ID = %q, want shared fixture id %q", org.ID, demoFixtureMarkerOrgSlug)
	}
	if len(org.Roles) == 0 {
		t.Fatal("expected org roles on fixture organization")
	}

	for _, email := range []string{"admin@attesta.com", "admin@org1.com", "admin@org2.com", "admin@org3.com"} {
		user, err := identity.GetUserByEmail(t.Context(), email)
		if err != nil {
			t.Fatalf("user %s: %v", email, err)
		}
		if !user.EmailVerified {
			t.Fatalf("user %s must be email-verified (ADR 0003)", email)
		}
	}

	memberships, err := identity.ListOrganizationMemberships(t.Context(), demoFixtureMarkerOrgSlug)
	if err != nil {
		t.Fatalf("memberships: %v", err)
	}
	if len(memberships) < 2 {
		t.Fatalf("memberships = %d, want platform admin + org admin", len(memberships))
	}
}

func TestSeedPopulatesTaxonomyCatalogAndInstances(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@attesta.com")
	t.Setenv("ADMIN_PASSWORD", "admin@attesta.com")

	store := NewMemoryStore()
	dir := writeSeedTestConfigDir(t)
	if err := Seed(t.Context(), SeedDeps{
		Identity:  newSeedFakeIdentity(),
		Store:     store,
		ConfigDir: dir,
		RNG:       &fixedCatalogRNG{n: 1},
	}); err != nil {
		t.Fatalf("Seed: %v", err)
	}

	cats, err := store.ListCategories(t.Context())
	if err != nil || len(cats) == 0 {
		t.Fatalf("categories=%v err=%v", cats, err)
	}
	streams, err := store.ListFormataBuilderStreams(t.Context())
	if err != nil || len(streams) == 0 {
		t.Fatalf("streams=%v err=%v", streams, err)
	}
	server := &Server{store: store, configDir: dir}
	catalog, err := server.workflowCatalog()
	if err != nil || len(catalog) == 0 {
		t.Fatalf("catalog=%v err=%v", catalog, err)
	}
	var processCount int
	for key := range catalog {
		procs, listErr := store.ListRecentProcessesByWorkflow(t.Context(), key, 100)
		if listErr != nil {
			t.Fatalf("ListRecentProcessesByWorkflow %s: %v", key, listErr)
		}
		processCount += len(procs)
	}
	if processCount == 0 {
		t.Fatal("expected seeded process instances")
	}
}

func TestSeedLeavesEveryFifthStreamEmpty(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@attesta.com")
	t.Setenv("ADMIN_PASSWORD", "admin@attesta.com")

	store := NewMemoryStore()
	dir := writeSeedTestConfigDirWithLeaves(t, 5)
	if err := Seed(t.Context(), SeedDeps{
		Identity:  newSeedFakeIdentity(),
		Store:     store,
		ConfigDir: dir,
		RNG:       &fixedCatalogRNG{n: 0}, // 1 stream per leaf → 5 sorted keys
	}); err != nil {
		t.Fatalf("Seed: %v", err)
	}

	server := &Server{store: store, configDir: dir}
	catalog, err := server.workflowCatalog()
	if err != nil {
		t.Fatalf("workflowCatalog: %v", err)
	}
	if len(catalog) != 5 {
		t.Fatalf("catalog size = %d, want 5", len(catalog))
	}

	var empty, seeded int
	for key := range catalog {
		procs, listErr := store.ListRecentProcessesByWorkflow(t.Context(), key, 0)
		if listErr != nil {
			t.Fatalf("ListRecentProcessesByWorkflow %s: %v", key, listErr)
		}
		switch len(procs) {
		case 0:
			empty++
		case 8:
			seeded++
		default:
			t.Fatalf("%s instances = %d, want 0 or 8", key, len(procs))
		}
	}
	if empty != 1 || seeded != 4 {
		t.Fatalf("empty=%d seeded=%d, want empty=1 seeded=4", empty, seeded)
	}
}

func TestSeedErrorsOnMissingTaxonomyFile(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@attesta.com")
	t.Setenv("ADMIN_PASSWORD", "admin@attesta.com")

	dir := writeSeedTestConfigDir(t)
	err := Seed(t.Context(), SeedDeps{
		Identity:       newSeedFakeIdentity(),
		Store:          NewMemoryStore(),
		ConfigDir:      dir,
		CategoriesPath: filepath.Join(t.TempDir(), "missing.yaml"),
		RNG:            &fixedCatalogRNG{n: 1},
	})
	if err == nil {
		t.Fatal("expected missing taxonomy file error")
	}
	if !strings.Contains(err.Error(), "taxonomy") {
		t.Fatalf("err = %v, want taxonomy wrap", err)
	}
}

func TestSeedReplaceFromArgs(t *testing.T) {
	if !seedReplaceFromArgs([]string{"--replace"}) {
		t.Fatal("expected --replace")
	}
	if seedReplaceFromArgs(nil) {
		t.Fatal("expected no replace")
	}
}

func TestAppwriteIdentityFromEnvRequiresEnv(t *testing.T) {
	t.Setenv("APPWRITE_ENDPOINT", "")
	t.Setenv("APPWRITE_PROJECT_ID", "")
	t.Setenv("APPWRITE_API_KEY", "")
	if _, err := appwriteIdentityFromEnv(appwriteIdentityFromEnvOptions{RequireAll: true}); err == nil {
		t.Fatal("expected missing env error")
	}
}

func TestAppwriteIdentityFromCredentialsRequiresAll(t *testing.T) {
	t.Parallel()
	if _, err := appwriteIdentityFromCredentials(appwriteRuntimeCredentials{}, appwriteIdentityFromEnvOptions{RequireAll: true}); err == nil {
		t.Fatal("expected missing credentials error")
	}
	identity, err := appwriteIdentityFromCredentials(appwriteRuntimeCredentials{
		Endpoint:  "http://appwrite/v1",
		ProjectID: "attesta",
		APIKey:    "key",
	}, appwriteIdentityFromEnvOptions{RequireAll: true})
	if err != nil {
		t.Fatalf("appwriteIdentityFromCredentials: %v", err)
	}
	if identity == nil {
		t.Fatal("expected identity")
	}
}

func TestSeedReplaceRewritesWhenMarkerPresent(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@attesta.com")
	t.Setenv("ADMIN_PASSWORD", "admin@attesta.com")

	store := NewMemoryStore()
	identity := newSeedFakeIdentity()
	dir := writeSeedTestConfigDir(t)
	deps := SeedDeps{
		Identity:  identity,
		Store:     store,
		ConfigDir: dir,
		RNG:       &fixedCatalogRNG{n: 1},
	}
	if err := Seed(t.Context(), deps); err != nil {
		t.Fatalf("first Seed: %v", err)
	}
	firstStreams, _ := store.ListFormataBuilderStreams(t.Context())
	if len(firstStreams) == 0 {
		t.Fatal("expected streams after first seed")
	}

	// Empty-only must no-op.
	if err := Seed(t.Context(), deps); err != nil {
		t.Fatalf("empty-only Seed: %v", err)
	}

	deps.Replace = true
	deps.RNG = &fixedCatalogRNG{n: 0}
	if err := Seed(t.Context(), deps); err != nil {
		t.Fatalf("replace Seed: %v", err)
	}
	org, err := identity.GetOrganizationBySlug(t.Context(), demoFixtureMarkerOrgSlug)
	if err != nil {
		t.Fatalf("marker org after replace: %v", err)
	}
	if org.ID != demoFixtureMarkerOrgSlug {
		t.Fatalf("org ID = %q after replace", org.ID)
	}
	cats, err := store.ListCategories(t.Context())
	if err != nil || len(cats) == 0 {
		t.Fatalf("categories after replace: %v err=%v", cats, err)
	}
}

type seedFakeIdentity struct {
	fakeIdentityStore
	orgs         map[string]IdentityOrg
	usersByEmail map[string]IdentityUser
	usersByID    map[string]IdentityUser
	memberships  map[string][]IdentityMembership
	nextUser     int
}

func newSeedFakeIdentity() *seedFakeIdentity {
	s := &seedFakeIdentity{
		orgs:         map[string]IdentityOrg{},
		usersByEmail: map[string]IdentityUser{},
		usersByID:    map[string]IdentityUser{},
		memberships:  map[string][]IdentityMembership{},
	}
	s.getOrganizationBySlugFunc = func(_ context.Context, slug string) (*IdentityOrg, error) {
		org, ok := s.orgs[slug]
		if !ok {
			return nil, ErrIdentityNotFound
		}
		cp := org
		return &cp, nil
	}
	s.createOrganizationAsAdminFunc = func(_ context.Context, name string) (IdentityOrg, error) {
		slug := canonifySlug(name)
		id := appwriteTeamIDFromSlug(slug)
		org := IdentityOrg{ID: id, Slug: slug, Name: strings.TrimSpace(name), Roles: []IdentityRole{defaultOperatorCatalogRole()}}
		s.orgs[slug] = org
		return org, nil
	}
	s.updateOrganizationAsAdminFunc = func(_ context.Context, currentSlug, name, _ string, roles []IdentityRole) (IdentityOrg, error) {
		org, ok := s.orgs[currentSlug]
		if !ok {
			return IdentityOrg{}, ErrIdentityNotFound
		}
		org.Name = strings.TrimSpace(name)
		org.Slug = canonifySlug(name)
		org.Roles = append([]IdentityRole(nil), roles...)
		delete(s.orgs, currentSlug)
		s.orgs[org.Slug] = org
		return org, nil
	}
	s.deleteOrganizationAsAdminFunc = func(_ context.Context, orgSlug string) error {
		delete(s.orgs, orgSlug)
		delete(s.memberships, orgSlug)
		return nil
	}
	s.ensureUserAccountFunc = func(_ context.Context, email, password, name string) (IdentityUser, error) {
		email = strings.ToLower(strings.TrimSpace(email))
		if email == "" || strings.TrimSpace(password) == "" {
			return IdentityUser{}, nil
		}
		if user, ok := s.usersByEmail[email]; ok {
			return user, nil
		}
		_ = name
		s.nextUser++
		id := "user-" + strings.ReplaceAll(email, "@", "-")
		if strings.Contains(email, "platform_admin") {
			id = "user-platform-admin"
		}
		user := IdentityUser{ID: id, Email: email}
		s.putUser(user)
		return user, nil
	}
	s.getUserByEmailFunc = func(_ context.Context, email string) (IdentityUser, error) {
		user, ok := s.usersByEmail[strings.ToLower(strings.TrimSpace(email))]
		if !ok {
			return IdentityUser{}, ErrIdentityNotFound
		}
		return user, nil
	}
	s.getUserByIDFunc = func(_ context.Context, userID string) (IdentityUser, error) {
		user, ok := s.usersByID[userID]
		if !ok {
			return IdentityUser{}, ErrIdentityNotFound
		}
		return user, nil
	}
	s.updateEmailVerificationFunc = func(_ context.Context, userID string, verified bool) error {
		user, ok := s.usersByID[userID]
		if !ok {
			return ErrIdentityNotFound
		}
		user.EmailVerified = verified
		s.putUser(user)
		return nil
	}
	s.addOrganizationUserByIDAsAdminFunc = func(_ context.Context, orgSlug, userID string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
		if _, ok := s.orgs[orgSlug]; !ok {
			return IdentityMembership{}, ErrIdentityNotFound
		}
		user, ok := s.usersByID[userID]
		if !ok {
			return IdentityMembership{}, ErrIdentityNotFound
		}
		m := IdentityMembership{
			ID:         "m-" + userID + "-" + orgSlug,
			TeamID:     s.orgs[orgSlug].ID,
			OrgSlug:    orgSlug,
			UserID:     userID,
			Email:      user.Email,
			RoleSlugs:  append([]string(nil), roleSlugs...),
			IsOrgAdmin: isOrgAdmin,
			Confirmed:  true,
		}
		s.memberships[orgSlug] = append(s.memberships[orgSlug], m)
		return m, nil
	}
	s.listOrganizationMembershipsFunc = func(_ context.Context, orgSlug string) ([]IdentityMembership, error) {
		return append([]IdentityMembership(nil), s.memberships[orgSlug]...), nil
	}
	s.updateUserLabelsFunc = func(_ context.Context, userID string, labels []string) (IdentityUser, error) {
		user, ok := s.usersByID[userID]
		if !ok {
			return IdentityUser{}, ErrIdentityNotFound
		}
		user.Labels = append([]string(nil), labels...)
		user.IsOrgAdmin = hasIdentityLabel(labels, identityOrgAdminLabel)
		s.putUser(user)
		return user, nil
	}
	return s
}

func (s *seedFakeIdentity) putUser(user IdentityUser) {
	s.usersByID[user.ID] = user
	s.usersByEmail[strings.ToLower(strings.TrimSpace(user.Email))] = user
}

func writeSeedTestConfigDir(t *testing.T) string {
	t.Helper()
	return writeSeedTestConfigDirWithLeaves(t, 1)
}

func writeSeedTestConfigDirWithLeaves(t *testing.T, leafCount int) string {
	t.Helper()
	if leafCount < 1 {
		t.Fatalf("leafCount = %d, want >= 1", leafCount)
	}
	dir := t.TempDir()
	var taxonomy strings.Builder
	taxonomy.WriteString("categories:\n  - slug: supply-chain\n    name: Supply Chain\n    icon: batch-traceability\n    sortOrder: 1\n    subCategories:\n")
	for i := 0; i < leafCount; i++ {
		slug := "leaf-" + string(rune('a'+i))
		name := "Leaf " + string(rune('A'+i))
		if leafCount == 1 {
			slug = "procurement"
			name = "Procurement"
		}
		taxonomy.WriteString("      - slug: " + slug + "\n")
		taxonomy.WriteString("        name: " + name + "\n")
		taxonomy.WriteString("        icon: procurement-workflow\n")
		taxonomy.WriteString("        sortOrder: " + strconv.Itoa(i+1) + "\n")
	}
	if err := os.WriteFile(filepath.Join(dir, "categories.yaml"), []byte(taxonomy.String()), 0o644); err != nil {
		t.Fatalf("write categories: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "demo.yaml"), []byte(minimalCategorizedWorkflowYAML(
		"  categorySlug: supply-chain\n  subCategorySlug: procurement\n",
	)), 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}
	return dir
}

type fixedCatalogRNG struct{ n int }

func (r *fixedCatalogRNG) Intn(n int) int {
	if n <= 0 {
		return 0
	}
	if r.n < 0 {
		return 0
	}
	if r.n >= n {
		return n - 1
	}
	return r.n
}
