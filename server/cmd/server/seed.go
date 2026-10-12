package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// Empty-only presence marker: when this org exists and Replace is false, Seed is a no-op.
const demoFixtureMarkerOrgSlug = "organization-1"

// Narrow IdentityStore surface for Seed; full IdentityStore remains the HTTP/session type.
type seedIdentity interface {
	GetOrganizationBySlug(ctx context.Context, slug string) (*IdentityOrg, error)
	DeleteOrganizationAsAdmin(ctx context.Context, orgSlug string) error
	UpdateOrganizationAsAdmin(ctx context.Context, currentSlug, name, logoFileID string, roles []IdentityRole) (IdentityOrg, error)
	CreateOrganizationAsAdmin(ctx context.Context, name string) (IdentityOrg, error)
	UpdateEmailVerification(ctx context.Context, userID string, verified bool) error
	UpdateUserLabels(ctx context.Context, userID string, labels []string) (IdentityUser, error)
	EnsureUserAccount(ctx context.Context, email, password, name string) (IdentityUser, error)
	GetUserByEmail(ctx context.Context, email string) (IdentityUser, error)
	ListOrganizationMemberships(ctx context.Context, orgSlug string) ([]IdentityMembership, error)
	AddOrganizationUserByIDAsAdmin(ctx context.Context, orgSlug, userID string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error)
}

var _ seedIdentity = (*appwriteIdentity)(nil)

type SeedDeps struct {
	Identity seedIdentity
	Store    Store
	// Workflow YAMLs and categories.yaml; empty defaults to "config".
	ConfigDir string
	// Empty uses ConfigDir/categories.yaml.
	CategoriesPath string
	// Rewrite even when the marker org exists (local only).
	Replace bool
	// Nil picks a time-based source in Seed.
	RNG catalogStreamRNG
	Fixture *demoFixture
	Now     func() time.Time
}

// Empty-only (default): if the marker org already exists, returns nil without writing.
// Pass Replace to rewrite.
//
// Apply order is intentional: identity → taxonomy → catalog streams → instances →
// named pilot streams (stable ObjectIDs; left empty for smoke).
//
// Seed does not create the Appwrite project or API key (use prepare for that) and
// does not read ATTESTA_SEED (hosts gate via PrepareDemoStack / ADR-0004).
func Seed(ctx context.Context, deps SeedDeps) error {
	if deps.Identity == nil {
		return fmt.Errorf("seed: identity is required")
	}
	if deps.Store == nil {
		return fmt.Errorf("seed: store is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	if !deps.Replace {
		_, err := deps.Identity.GetOrganizationBySlug(ctx, demoFixtureMarkerOrgSlug)
		switch {
		case err == nil:
			return nil
		case errors.Is(err, ErrIdentityNotFound):
		default:
			return fmt.Errorf("seed: marker org lookup: %w", err)
		}
	}

	fixture := defaultDemoFixture()
	if deps.Fixture != nil {
		fixture = *deps.Fixture
	}
	fixture = demoFixtureWithPlatformAdminFromEnv(fixture)

	if err := seedDemoIdentity(ctx, deps.Identity, fixture, deps.Replace); err != nil {
		return err
	}

	configDir := strings.TrimSpace(deps.ConfigDir)
	if configDir == "" {
		configDir = "config"
	}
	categoriesPath := resolveCategoriesSeedPath(deps.CategoriesPath, configDir)
	if err := seedTaxonomyFromFile(ctx, deps.Store, categoriesPath); err != nil {
		return fmt.Errorf("seed: taxonomy: %w", err)
	}

	rng := deps.RNG
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	now := time.Now().UTC()
	if deps.Now != nil {
		now = deps.Now().UTC()
	}
	catalogResult, err := seedCatalogStreams(ctx, deps.Store, configDir, now, rng)
	if err != nil {
		return fmt.Errorf("seed: catalog streams: %w", err)
	}
	if _, err := seedDemoInstances(ctx, deps.Store, catalogResult.Catalog, now); err != nil {
		return fmt.Errorf("seed: instances: %w", err)
	}
	if _, err := seedNamedPilotStreams(ctx, deps.Store, configDir, now); err != nil {
		return fmt.Errorf("seed: named pilot streams: %w", err)
	}
	return nil
}

func seedDemoIdentity(ctx context.Context, identity seedIdentity, fixture demoFixture, replace bool) error {
	if replace {
		for _, org := range fixture.Orgs {
			slug := demoOrgSlug(org.Name)
			if err := identity.DeleteOrganizationAsAdmin(ctx, slug); err != nil && !errors.Is(err, ErrIdentityNotFound) {
				return fmt.Errorf("seed identity: delete org %s: %w", slug, err)
			}
		}
	}

	orgBySlug := make(map[string]IdentityOrg, len(fixture.Orgs))
	for _, orgDef := range fixture.Orgs {
		slug := demoOrgSlug(orgDef.Name)
		existing, err := identity.GetOrganizationBySlug(ctx, slug)
		switch {
		case err == nil:
			updated, updErr := identity.UpdateOrganizationAsAdmin(ctx, existing.Slug, orgDef.Name, existing.LogoFileID, append([]IdentityRole(nil), orgDef.Roles...))
			if updErr != nil {
				return fmt.Errorf("seed identity: update org %s: %w", slug, updErr)
			}
			orgBySlug[updated.Slug] = updated
		case errors.Is(err, ErrIdentityNotFound):
			created, createErr := identity.CreateOrganizationAsAdmin(ctx, orgDef.Name)
			if createErr != nil {
				return fmt.Errorf("seed identity: create org %s: %w", orgDef.Name, createErr)
			}
			updated, updErr := identity.UpdateOrganizationAsAdmin(ctx, created.Slug, orgDef.Name, created.LogoFileID, append([]IdentityRole(nil), orgDef.Roles...))
			if updErr != nil {
				return fmt.Errorf("seed identity: set roles %s: %w", created.Slug, updErr)
			}
			orgBySlug[updated.Slug] = updated
		default:
			return fmt.Errorf("seed identity: lookup org %s: %w", slug, err)
		}
	}

	for _, userDef := range fixture.Users {
		user, err := ensureDemoFixtureUser(ctx, identity, userDef)
		if err != nil {
			return err
		}
		if err := identity.UpdateEmailVerification(ctx, user.ID, true); err != nil {
			return fmt.Errorf("seed identity: verify %s: %w", userDef.Email, err)
		}
		user.EmailVerified = true

		labels := make([]string, 0, 8)
		if len(userDef.OrgAdminSlugs) > 0 {
			labels = append(labels, identityOrgAdminLabel)
		}
		for _, orgSlug := range userDef.OrgAdminSlugs {
			if _, ok := orgBySlug[orgSlug]; !ok {
				return fmt.Errorf("seed identity: unknown org slug %q for %s", orgSlug, userDef.Email)
			}
			roleSlugs := append([]string(nil), userDef.MembershipRoles[orgSlug]...)
			if err := ensureDemoOrgMembership(ctx, identity, orgSlug, user, roleSlugs, true); err != nil {
				return err
			}
			for _, roleSlug := range roleSlugs {
				if encoded := encodeIdentityRoleLabel(roleSlug); encoded != "" {
					labels = append(labels, encoded)
				}
			}
		}
		if len(labels) > 0 {
			if _, err := identity.UpdateUserLabels(ctx, user.ID, uniqueIdentityStrings(labels)); err != nil {
				return fmt.Errorf("seed identity: labels %s: %w", userDef.Email, err)
			}
		}
	}
	return nil
}

func ensureDemoFixtureUser(ctx context.Context, identity seedIdentity, userDef demoFixtureUser) (IdentityUser, error) {
	email := strings.ToLower(strings.TrimSpace(userDef.Email))
	password := strings.TrimSpace(userDef.Password)
	if email == "" || password == "" {
		return IdentityUser{}, fmt.Errorf("seed identity: user email and password are required")
	}
	user, err := identity.EnsureUserAccount(ctx, email, password, userDef.Name)
	if err != nil {
		return IdentityUser{}, fmt.Errorf("seed identity: ensure user %s: %w", email, err)
	}
	if strings.TrimSpace(user.ID) == "" {
		loaded, loadErr := identity.GetUserByEmail(ctx, email)
		if loadErr != nil {
			return IdentityUser{}, fmt.Errorf("seed identity: load user %s: %w", email, loadErr)
		}
		return loaded, nil
	}
	return user, nil
}

func ensureDemoOrgMembership(ctx context.Context, identity seedIdentity, orgSlug string, user IdentityUser, roleSlugs []string, isOrgAdmin bool) error {
	memberships, err := identity.ListOrganizationMemberships(ctx, orgSlug)
	if err != nil {
		return fmt.Errorf("seed identity: list memberships %s: %w", orgSlug, err)
	}
	for _, m := range memberships {
		if strings.TrimSpace(m.UserID) == strings.TrimSpace(user.ID) || strings.EqualFold(m.Email, user.Email) {
			return nil
		}
	}
	if _, err := identity.AddOrganizationUserByIDAsAdmin(ctx, orgSlug, user.ID, roleSlugs, isOrgAdmin); err != nil {
		return fmt.Errorf("seed identity: add %s to %s: %w", user.Email, orgSlug, err)
	}
	return nil
}
