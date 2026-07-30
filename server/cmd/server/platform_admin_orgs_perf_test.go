package main

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestPlatformAdminViewDoesNotCallHydratedMembershipList(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")

	var hydratedCalls atomic.Int64
	var liteCalls atomic.Int64

	orgs := []IdentityOrg{
		{ID: "team-1", Slug: "accepted", Name: "Accepted Org"},
		{ID: "team-2", Slug: "pending", Name: "Pending Org"},
		{ID: "team-3", Slug: "missing", Name: "Missing Org"},
	}

	identity := &fakeIdentityStore{
		listOrganizationsFunc: func(ctx context.Context) ([]IdentityOrg, error) {
			return orgs, nil
		},
		listOrganizationMembershipsFunc: func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
			hydratedCalls.Add(1)
			t.Fatalf("platform admin view must not call ListOrganizationMemberships (%s)", orgSlug)
			return nil, nil
		},
		listOrganizationMembershipsLiteFunc: func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
			liteCalls.Add(1)
			switch orgSlug {
			case "accepted":
				return []IdentityMembership{
					{Email: "admin@example.com", Confirmed: true, IsOrgAdmin: true},
					{Email: "owner@example.com", Confirmed: true, IsOrgAdmin: true},
				}, nil
			case "pending":
				return []IdentityMembership{
					{Email: "pending-owner@example.com", Confirmed: false, IsOrgAdmin: true},
				}, nil
			default:
				return nil, nil
			}
		},
	}

	server := &Server{identity: identity, authorizer: fakeAuthorizer{}}
	view := server.platformAdminView(&AccountUser{Email: "admin@example.com", IsPlatformAdmin: true}, "", PlatformAdminErrors{})

	if hydratedCalls.Load() != 0 {
		t.Fatalf("hydratedCalls = %d", hydratedCalls.Load())
	}
	if liteCalls.Load() != 3 {
		t.Fatalf("liteCalls = %d, want 3", liteCalls.Load())
	}
	if len(view.Organizations) != 3 {
		t.Fatalf("organizations = %#v", view.Organizations)
	}
	if view.Organizations[0].OrgAdminStatus != "At least one org admin accepted" {
		t.Fatalf("accepted status = %q", view.Organizations[0].OrgAdminStatus)
	}
	if view.Organizations[2].OrgAdminStatus != "All org admin invites pending" {
		t.Fatalf("pending status = %q", view.Organizations[2].OrgAdminStatus)
	}
	if view.Organizations[1].OrgAdminStatus != "No org admin" {
		t.Fatalf("missing status = %q", view.Organizations[1].OrgAdminStatus)
	}
}

func TestPlatformAdminOrganizationRowsPreservesOrderUnderParallelFetch(t *testing.T) {
	var calls atomic.Int64
	identity := &fakeIdentityStore{
		listOrganizationMembershipsLiteFunc: func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
			n := calls.Add(1)
			// Stagger later orgs so unordered append would scramble results.
			if orgSlug == "a" {
				time.Sleep(30 * time.Millisecond)
			}
			_ = n
			return []IdentityMembership{{Email: orgSlug + "-owner@example.com", IsOrgAdmin: true, Confirmed: true}}, nil
		},
	}
	orgs := []Organization{{Slug: "a", Name: "A"}, {Slug: "b", Name: "B"}, {Slug: "c", Name: "C"}}
	rows := platformAdminOrganizationRows(context.Background(), orgs, identity)
	if len(rows) != 3 || rows[0].Slug != "a" || rows[1].Slug != "b" || rows[2].Slug != "c" {
		t.Fatalf("rows = %#v", rows)
	}
	if rows[0].OrgAdminEmails[0] != "a-owner@example.com" || rows[2].OrgAdminEmails[0] != "c-owner@example.com" {
		t.Fatalf("emails = %#v", rows)
	}
}

func TestFilterPlatformOrgAdminMembershipsDropsNonAdmins(t *testing.T) {
	in := []IdentityMembership{
		{Email: "owner@example.com", IsOrgAdmin: true, Confirmed: true},
		{Email: "member@example.com", IsOrgAdmin: false, Confirmed: true},
		{Email: "pending@example.com", IsOrgAdmin: true, Confirmed: false},
	}
	got := filterPlatformOrgAdminMemberships(in)
	if len(got) != 2 {
		t.Fatalf("got = %#v", got)
	}
	if got[0].Email != "owner@example.com" || got[1].Email != "pending@example.com" {
		t.Fatalf("got = %#v", got)
	}
}

func TestPlatformAdminViewUsesOrganizationPageTotal(t *testing.T) {
	t.Setenv("ADMIN_EMAIL", "admin@example.com")
	t.Setenv("ADMIN_PASSWORD", "change-me")
	identity := &fakeIdentityStore{
		listOrganizationsPageFunc: func(ctx context.Context, opts IdentityOrgListOptions) (IdentityOrgPage, error) {
			if opts.Limit != 12 || opts.Offset != 12 || opts.Search != "ac" {
				t.Fatalf("opts = %#v", opts)
			}
			return IdentityOrgPage{
				Total: 25,
				Organizations: []IdentityOrg{
					{ID: "t13", Slug: "org-13", Name: "Org 13"},
				},
			}, nil
		},
		listOrganizationMembershipsLiteFunc: func(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
			return nil, nil
		},
	}
	server := &Server{identity: identity, authorizer: fakeAuthorizer{}}
	view := server.platformAdminView(&AccountUser{Email: "admin@example.com", IsPlatformAdmin: true}, "", PlatformAdminErrors{SearchQuery: "ac", Page: 2})
	if view.MatchedOrganizations != 25 || view.TotalPages != 3 || view.CurrentPage != 2 || len(view.Organizations) != 1 {
		t.Fatalf("view paging = matched=%d pages=%d page=%d rows=%d", view.MatchedOrganizations, view.TotalPages, view.CurrentPage, len(view.Organizations))
	}
}
