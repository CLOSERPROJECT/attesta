package main

import "strings"

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
	MembershipRoles map[string][]string
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
			{
				Name: "Feragame",
				Roles: []IdentityRole{
					{Slug: "chemist", Name: "Chemist", Palette: "blue"},
				},
			},
		},
		Users: []demoFixtureUser{
			{
				Email:         "admin@attesta.com",
				Password:      "admin@attesta.com",
				Name:          "Platform Admin",
				PlatformAdmin: true,
				OrgAdminSlugs: []string{"organization-1", "organization-2", "organization-3", "feragame"},
			},
			{
				Email:         "admin@org1.com",
				Password:      "admin@org1.com",
				Name:          "Org 1 Admin",
				OrgAdminSlugs: []string{"organization-1"},
				MembershipRoles: map[string][]string{
					"organization-1": {"chemist", "mechanic", "supervisor"},
				},
			},
			{
				Email:         "admin@org2.com",
				Password:      "admin@org2.com",
				Name:          "Org 2 Admin",
				OrgAdminSlugs: []string{"organization-2"},
				MembershipRoles: map[string][]string{
					"organization-2": {"analyst"},
				},
			},
			{
				Email:         "admin@org3.com",
				Password:      "admin@org3.com",
				Name:          "Org 3 Admin",
				OrgAdminSlugs: []string{"organization-3"},
				MembershipRoles: map[string][]string{
					"organization-3": {"qainspector"},
				},
			},
			{
				Email:         "admin@feragame.com",
				Password:      "admin@feragame.com",
				Name:          "Feragame Admin",
				OrgAdminSlugs: []string{"feragame"},
				MembershipRoles: map[string][]string{
					"feragame": {"chemist"},
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
