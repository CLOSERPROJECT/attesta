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

// Living Feragame demo org / operators (Cerbos Commit demos on /app).
const (
	demoFeragameOrgSlug           = "feragame"
	demoOrgAdminUserEmail         = "admin@feragame.com"
	demoAdministrationUserEmail    = "administration@feragame.com"
	demoLogisticUserEmail = "logistic@feragame.com"
	demoLineUserEmail     = "line@feragame.com"
)

func feragameOrgDef() demoFixtureOrg {
	return demoFixtureOrg{
		Name: "Feragame",
		Roles: []IdentityRole{
			{Slug: "administration", Name: "Administration", Palette: "blue"},
			{Slug: "logistic", Name: "Logistic", Palette: "cyan"},
			{Slug: "line", Name: "Line", Palette: "emerald"},
		},
	}
}

func feragameLivingUsers() []demoFixtureUser {
	return []demoFixtureUser{
		{
			Email:         demoOrgAdminUserEmail,
			Password:      demoOrgAdminUserEmail,
			Name:          "Living All Roles",
			OrgAdminSlugs: []string{demoFeragameOrgSlug},
			MembershipRoles: map[string][]string{
				demoFeragameOrgSlug: {"administration", "logistic", "line"},
			},
		},
		{
			Email:         demoAdministrationUserEmail,
			Password:      demoAdministrationUserEmail,
			Name:          "Living Administration",
			OrgAdminSlugs: []string{demoFeragameOrgSlug},
			MembershipRoles: map[string][]string{
				demoFeragameOrgSlug: {"administration"},
			},
		},
		{
			Email:         demoLogisticUserEmail,
			Password:      demoLogisticUserEmail,
			Name:          "Living Logistic",
			OrgAdminSlugs: []string{demoFeragameOrgSlug},
			MembershipRoles: map[string][]string{
				demoFeragameOrgSlug: {"logistic"},
			},
		},
		{
			Email:         demoLineUserEmail,
			Password:      demoLineUserEmail,
			Name:          "Living Line",
			OrgAdminSlugs: []string{demoFeragameOrgSlug},
			MembershipRoles: map[string][]string{
				demoFeragameOrgSlug: {"line"},
			},
		},
	}
}

// Scoped so Feragame hard-reset never touches orgs 1–3 / the Formata marker org.
func feragameDemoFixture() demoFixture {
	users := []demoFixtureUser{{
		Email:         "admin@attesta.com",
		Password:      "admin@attesta.com",
		Name:          "Platform Admin",
		PlatformAdmin: true,
		OrgAdminSlugs: []string{demoFeragameOrgSlug},
	}}
	users = append(users, feragameLivingUsers()...)
	return demoFixture{
		Orgs:  []demoFixtureOrg{feragameOrgDef()},
		Users: users,
	}
}

func defaultDemoFixture() demoFixture {
	users := []demoFixtureUser{
		{
			Email:         "admin@attesta.com",
			Password:      "admin@attesta.com",
			Name:          "Platform Admin",
			PlatformAdmin: true,
			OrgAdminSlugs: []string{"organization-1", "organization-2", "organization-3", demoFeragameOrgSlug},
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
	}
	users = append(users, feragameLivingUsers()...)
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
			feragameOrgDef(),
		},
		Users: users,
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
