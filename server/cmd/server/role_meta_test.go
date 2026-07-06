package main

import (
	"context"
	"testing"
)

func testRoleIndexForOrg(orgSlug string, roles map[string]RoleMeta) map[roleMetaKey]RoleMeta {
	index := make(map[roleMetaKey]RoleMeta, len(roles))
	for slug, meta := range roles {
		index[roleMetaKey{OrgSlug: orgSlug, RoleSlug: slug}] = meta
	}
	return index
}

func TestRoleMetaIndexFromIdentity(t *testing.T) {
	server := &Server{
		identity: &fakeIdentityStore{
			listOrganizationsFunc: func(ctx context.Context) ([]IdentityOrg, error) {
				return []IdentityOrg{
					{
						Slug: "org1",
						Roles: []IdentityRole{
							{
								Slug:   "chemist",
								Name:   "Chemist",
								Color:  "var(--role-blue-bg)",
								Border: "var(--role-blue-border)",
							},
						},
					},
					{
						Slug: "org2",
						Roles: []IdentityRole{
							{
								Slug:   "chemist",
								Name:   "Chemist",
								Color:  "var(--role-emerald-bg)",
								Border: "var(--role-emerald-border)",
							},
						},
					},
				}, nil
			},
		},
	}

	index := server.roleMetaIndex(context.Background())
	if len(index) != 2 {
		t.Fatalf("index len = %d, want 2", len(index))
	}
	org1Meta := index[roleMetaKey{OrgSlug: "org1", RoleSlug: "chemist"}]
	if org1Meta.Label != "Chemist" || org1Meta.Palette != "blue" {
		t.Fatalf("org1 chemist meta = %#v, want label Chemist palette blue", org1Meta)
	}
	org2Meta := index[roleMetaKey{OrgSlug: "org2", RoleSlug: "chemist"}]
	if org2Meta.Palette != "emerald" {
		t.Fatalf("org2 chemist palette = %q, want emerald", org2Meta.Palette)
	}
}

func TestRoleMetaForOrgScopedLookup(t *testing.T) {
	index := testRoleIndexForOrg("org1", map[string]RoleMeta{
		"projectmanager": {ID: "projectmanager", Label: "PM Org1", Palette: "blue"},
	})
	index[roleMetaKey{OrgSlug: "org2", RoleSlug: "projectmanager"}] = RoleMeta{
		ID: "projectmanager", Label: "PM Org2", Palette: "emerald",
	}

	got := roleMetaForOrg("org1", "projectmanager", index, nil)
	if got.Palette != "blue" {
		t.Fatalf("org1 palette = %q, want blue", got.Palette)
	}
	got = roleMetaForOrg("org2", "projectmanager", index, nil)
	if got.Palette != "emerald" {
		t.Fatalf("org2 palette = %q, want emerald", got.Palette)
	}
}

func TestRoleMetaForOrgFallbackWhenIdentityUnavailable(t *testing.T) {
	got := roleMetaForOrg("org1", "unknown", map[roleMetaKey]RoleMeta{}, nil)
	if got.Palette != "fallback" || got.Label != "unknown" {
		t.Fatalf("fallback meta = %#v", got)
	}
	if got := (&Server{}).roleMetaIndex(context.Background()); len(got) != 0 {
		t.Fatalf("nil identity index = %#v, want empty", got)
	}
}

func TestRoleMetaForOrgResolvesOrgFromConfigRoles(t *testing.T) {
	index := testRoleIndexForOrg("org1", map[string]RoleMeta{
		"dep1": {ID: "dep1", Label: "Department 1", Palette: "cyan"},
	})
	cfgRoles := []WorkflowRole{{OrgSlug: "org1", Slug: "dep1", Name: "Department 1"}}

	got := roleMetaForOrg("", "dep1", index, cfgRoles)
	if got.Palette != "cyan" || got.Label != "Department 1" {
		t.Fatalf("resolved meta = %#v", got)
	}
}
