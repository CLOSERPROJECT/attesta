package main

import "strings"

const demoFixtureDefaultPassword = "change-me"

type demoFixtureOrg struct {
	Name  string
	Roles []IdentityRole
}

type demoFixtureUser struct {
	Email           string
	Password        string
	Name            string
	PlatformAdmin   bool
	OrgAdminSlugs   []string
	MembershipRoles map[string][]string // orgSlug → business role slugs
}

type demoFixture struct {
	Orgs  []demoFixtureOrg
	Users []demoFixtureUser
}

func defaultDemoFixture() demoFixture {
	return demoFixture{
		Orgs: []demoFixtureOrg{
			{
				Name: "Organization 1",
				Roles: []IdentityRole{
					{Slug: "chemist", Name: "Chemist", Palette: "blue"},
					{Slug: "mechanic", Name: "Mechanic", Palette: "blue"},
					{Slug: "supervisor", Name: "Supervisor", Palette: "cyan"},
				},
			},
			{
				Name: "Organization 2",
				Roles: []IdentityRole{
					{Slug: "analyst", Name: "Analyst", Palette: "emerald"},
				},
			},
			{
				Name: "Organization 3",
				Roles: []IdentityRole{
					{Slug: "qainspector", Name: "QA Inspector", Palette: "pink"},
				},
			},
		},
		Users: []demoFixtureUser{
			{
				Email:         "platform_admin@example.org",
				Password:      demoFixtureDefaultPassword,
				Name:          "Platform Admin",
				PlatformAdmin: true,
				OrgAdminSlugs: []string{"organization-1", "organization-2", "organization-3"},
			},
			{
				Email:         "org1@admin.org",
				Password:      demoFixtureDefaultPassword,
				Name:          "Org 1 Admin",
				OrgAdminSlugs: []string{"organization-1"},
				MembershipRoles: map[string][]string{
					"organization-1": {"chemist", "mechanic", "supervisor"},
				},
			},
			{
				Email:         "org2@admin.org",
				Password:      demoFixtureDefaultPassword,
				Name:          "Org 2 Admin",
				OrgAdminSlugs: []string{"organization-2"},
				MembershipRoles: map[string][]string{
					"organization-2": {"analyst"},
				},
			},
			{
				Email:         "org3@admin.org",
				Password:      demoFixtureDefaultPassword,
				Name:          "Org 3 Admin",
				OrgAdminSlugs: []string{"organization-3"},
				MembershipRoles: map[string][]string{
					"organization-3": {"qainspector"},
				},
			},
		},
	}
}

func demoFixtureWithPlatformAdminFromEnv(base demoFixture) demoFixture {
	email, password, ok := platformAdminCredentials()
	if !ok {
		return base
	}
	out := base
	out.Users = append([]demoFixtureUser(nil), base.Users...)
	for i := range out.Users {
		if !out.Users[i].PlatformAdmin {
			continue
		}
		out.Users[i].Email = email
		out.Users[i].Password = password
		return out
	}
	return out
}

func demoOrgSlug(name string) string {
	return canonifySlug(strings.TrimSpace(name))
}
