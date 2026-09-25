package main

import "testing"

func TestCountOrgAdmins(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "secret")

	tests := []struct {
		name  string
		users []IdentityUser
		want  int
	}{
		{name: "empty", want: 0},
		{
			name: "counts org admins only",
			users: []IdentityUser{
				{ID: "a", IsOrgAdmin: true},
				{ID: "m", IsOrgAdmin: false},
				{ID: "b", IsOrgAdmin: true},
			},
			want: 2,
		},
		{
			name: "skips platform admin identity users",
			users: []IdentityUser{
				{ID: "a", Email: "owner@example.com", IsOrgAdmin: true},
				{ID: "p", Email: "admin@example.com", IsOrgAdmin: true},
			},
			want: 1,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := CountOrgAdmins(tc.users); got != tc.want {
				t.Fatalf("CountOrgAdmins = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestCanDeleteMember(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                string
		actorKey, targetKey string
		wantAllowed         bool
		wantReason          string
	}{
		{name: "peer allowed", actorKey: "admin-1", targetKey: "member-1", wantAllowed: true},
		{
			name:        "self denied",
			actorKey:    "admin-1",
			targetKey:   "admin-1",
			wantAllowed: false,
			wantReason:  reasonSelfDelete,
		},
		{name: "empty keys treated as match denied", actorKey: "", targetKey: "", wantAllowed: false, wantReason: reasonSelfDelete},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := CanDeleteMember(tc.actorKey, tc.targetKey)
			if got.Allowed != tc.wantAllowed || got.Reason != tc.wantReason {
				t.Fatalf("CanDeleteMember = %+v, want Allowed=%v Reason=%q", got, tc.wantAllowed, tc.wantReason)
			}
		})
	}
}

func TestCanChangeOrgAdmin(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                       string
		targetIsAdmin, wantAdmin   bool
		otherAdminCount            int
		wantAllowed                bool
		wantReason                 string
	}{
		{name: "promote member", targetIsAdmin: false, wantAdmin: true, otherAdminCount: 0, wantAllowed: true},
		{name: "keep admin", targetIsAdmin: true, wantAdmin: true, otherAdminCount: 0, wantAllowed: true},
		{name: "demote with peers", targetIsAdmin: true, wantAdmin: false, otherAdminCount: 1, wantAllowed: true},
		{
			name:            "demote sole denied",
			targetIsAdmin:   true,
			wantAdmin:       false,
			otherAdminCount: 0,
			wantAllowed:     false,
			wantReason:      reasonSoleOrgAdminDemote,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := CanChangeOrgAdmin(tc.targetIsAdmin, tc.wantAdmin, tc.otherAdminCount)
			if got.Allowed != tc.wantAllowed || got.Reason != tc.wantReason {
				t.Fatalf("CanChangeOrgAdmin = %+v, want Allowed=%v Reason=%q", got, tc.wantAllowed, tc.wantReason)
			}
		})
	}
}

func TestCanLeaveOrganization(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		isOrgAdmin  bool
		adminCount  int
		wantAllowed bool
		wantReason  string
	}{
		{name: "member always", isOrgAdmin: false, adminCount: 1, wantAllowed: true},
		{name: "admin with peer", isOrgAdmin: true, adminCount: 2, wantAllowed: true},
		{
			name:        "sole admin denied",
			isOrgAdmin:  true,
			adminCount:  1,
			wantAllowed: false,
			wantReason:  reasonSoleOrgAdminLeave,
		},
		{
			name:        "zero admins denied for admin",
			isOrgAdmin:  true,
			adminCount:  0,
			wantAllowed: false,
			wantReason:  reasonSoleOrgAdminLeave,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := CanLeaveOrganization(tc.isOrgAdmin, tc.adminCount)
			if got.Allowed != tc.wantAllowed || got.Reason != tc.wantReason {
				t.Fatalf("CanLeaveOrganization = %+v, want Allowed=%v Reason=%q", got, tc.wantAllowed, tc.wantReason)
			}
		})
	}
}

func TestCanDeleteCatalogRole(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		catalogLen  int
		inUse       bool
		wantAllowed bool
		wantReason  string
	}{
		{name: "unused with peers", catalogLen: 2, inUse: false, wantAllowed: true},
		{name: "in use", catalogLen: 3, inUse: true, wantAllowed: false, wantReason: reasonRoleInUse},
		{name: "last role", catalogLen: 1, inUse: false, wantAllowed: false, wantReason: reasonLastCatalogRole},
		{name: "in use wins over last", catalogLen: 1, inUse: true, wantAllowed: false, wantReason: reasonRoleInUse},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := CanDeleteCatalogRole(tc.catalogLen, tc.inUse)
			if got.Allowed != tc.wantAllowed || got.Reason != tc.wantReason {
				t.Fatalf("CanDeleteCatalogRole = %+v, want Allowed=%v Reason=%q", got, tc.wantAllowed, tc.wantReason)
			}
		})
	}
}

func TestCanEditCatalogRole(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		inUse       bool
		wantAllowed bool
		wantReason  string
	}{
		{name: "unused", inUse: false, wantAllowed: true},
		{name: "in use", inUse: true, wantAllowed: false, wantReason: reasonRoleInUse},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := CanEditCatalogRole(tc.inUse)
			if got.Allowed != tc.wantAllowed || got.Reason != tc.wantReason {
				t.Fatalf("CanEditCatalogRole = %+v, want Allowed=%v Reason=%q", got, tc.wantAllowed, tc.wantReason)
			}
		})
	}
}
