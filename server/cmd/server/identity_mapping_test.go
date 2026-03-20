package main

import "testing"

func TestIdentityRoleLabelsRoundTrip(t *testing.T) {
	labels := []string{
		identityOrgAdminLabel,
		encodeIdentityRoleLabel("qa-reviewer"),
		"ignored",
		encodeIdentityRoleLabel("qa-approver"),
		encodeIdentityRoleLabel("qa-reviewer"),
	}

	roleSlugs := decodeIdentityRoleLabels(labels)

	if len(roleSlugs) != 2 {
		t.Fatalf("len(roleSlugs) = %d, want 2", len(roleSlugs))
	}
	if roleSlugs[0] != "qa-reviewer" || roleSlugs[1] != "qa-approver" {
		t.Fatalf("roleSlugs = %#v", roleSlugs)
	}
}

func TestIdentityOrgPrefsRoundTrip(t *testing.T) {
	org := IdentityOrg{
		ID:         "acme",
		Slug:       "acme",
		Name:       "Acme Org",
		LogoFileID: "logo-file-1",
		Roles: []IdentityRole{
			{Slug: "qa-reviewer", Name: "QA Reviewer", Color: "#123456", Border: "solid"},
			{Slug: "qa-approver", Name: "QA Approver", Color: "#654321", Border: "dashed"},
		},
	}

	prefs := encodeIdentityOrgPrefs(org)
	decoded := decodeIdentityOrgFromTeam(org.ID, org.Name, prefs)

	if prefs.SchemaVersion != identityTeamPrefsSchemaVersion {
		t.Fatalf("schema version = %d, want %d", prefs.SchemaVersion, identityTeamPrefsSchemaVersion)
	}
	if decoded.ID != org.ID || decoded.Slug != org.Slug || decoded.Name != org.Name {
		t.Fatalf("decoded org = %#v", decoded)
	}
	if decoded.LogoFileID != org.LogoFileID {
		t.Fatalf("logo = %q, want %q", decoded.LogoFileID, org.LogoFileID)
	}
	if len(decoded.Roles) != 2 {
		t.Fatalf("roles = %#v", decoded.Roles)
	}
	if decoded.Roles[0].Slug != "qa-reviewer" || decoded.Roles[1].Slug != "qa-approver" {
		t.Fatalf("roles = %#v", decoded.Roles)
	}
}

func TestInviteMembershipRolesRoundTrip(t *testing.T) {
	encoded := encodeInviteMembershipRoles([]string{"qa-reviewer", "qa-approver", "qa-reviewer"}, true)
	decoded := decodeInviteMembershipRoles(encoded)

	if len(encoded) != 3 {
		t.Fatalf("encoded = %#v", encoded)
	}
	if encoded[0] != identityMembershipOwnerRole {
		t.Fatalf("team role = %q, want %q", encoded[0], identityMembershipOwnerRole)
	}
	if !decoded.IsOrgAdmin {
		t.Fatal("expected org admin")
	}
	if len(decoded.MembershipRoles) != 1 || decoded.MembershipRoles[0] != identityMembershipOwnerRole {
		t.Fatalf("membership roles = %#v", decoded.MembershipRoles)
	}
	if len(decoded.BusinessRoles) != 2 {
		t.Fatalf("business roles = %#v", decoded.BusinessRoles)
	}
	if decoded.BusinessRoles[0] != "qa-reviewer" || decoded.BusinessRoles[1] != "qa-approver" {
		t.Fatalf("business roles = %#v", decoded.BusinessRoles)
	}
}
