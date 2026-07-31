package main

import "testing"

func TestUserCanAccessStream(t *testing.T) {
	cfg := RuntimeConfig{
		Organizations: []WorkflowOrganization{{Slug: "acme", Name: "Acme"}},
		Roles: []WorkflowRole{
			{OrgSlug: "acme", Slug: "operator", Name: "Operator"},
			{OrgSlug: "other", Slug: "reviewer", Name: "Reviewer"},
		},
	}
	otherOrgCfg := RuntimeConfig{
		Organizations: []WorkflowOrganization{{Slug: "other", Name: "Other"}},
		Roles:         []WorkflowRole{{OrgSlug: "other", Slug: "reviewer", Name: "Reviewer"}},
	}

	cases := []struct {
		name string
		user *AccountUser
		cfg  RuntimeConfig
		want bool
	}{
		{
			name: "platform admin sees all",
			user: &AccountUser{IsPlatformAdmin: true},
			cfg:  otherOrgCfg,
			want: true,
		},
		{
			name: "org admin sees org streams without workflow role",
			user: &AccountUser{OrgSlug: "acme", RoleSlugs: []string{"org-admin"}},
			cfg:  cfg,
			want: true,
		},
		{
			name: "org admin does not see other org streams",
			user: &AccountUser{OrgSlug: "acme", RoleSlugs: []string{"org-admin"}},
			cfg:  otherOrgCfg,
			want: false,
		},
		{
			name: "role member matching org+role",
			user: &AccountUser{OrgSlug: "acme", RoleSlugs: []string{"operator"}},
			cfg:  cfg,
			want: true,
		},
		{
			name: "role member wrong role",
			user: &AccountUser{OrgSlug: "acme", RoleSlugs: []string{"reviewer"}},
			cfg:  cfg,
			want: false,
		},
		{
			name: "outsider",
			user: &AccountUser{OrgSlug: "stranger", RoleSlugs: []string{"operator"}},
			cfg:  cfg,
			want: false,
		},
		{
			name: "nil user",
			user: nil,
			cfg:  cfg,
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := userCanAccessStream(tc.user, tc.cfg); got != tc.want {
				t.Fatalf("userCanAccessStream = %v, want %v", got, tc.want)
			}
		})
	}
}
