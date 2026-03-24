package main

import (
	"context"
	"errors"
	"time"
)

var ErrIdentityNotFound = errors.New("identity not found")
var ErrIdentityUnauthorized = errors.New("identity unauthorized")

// IdentityStore isolates auth and organization data from the Mongo workflow store.
type IdentityStore interface {
	CreateAccount(ctx context.Context, email, password, name string) (IdentityUser, error)
	CreateOrganization(ctx context.Context, sessionSecret, name string) (IdentityOrg, error)
	CreateOrganizationAsAdmin(ctx context.Context, name string) (IdentityOrg, error)
	EnsurePlatformAdminAccount(ctx context.Context, email, password string) error
	AcceptInvite(ctx context.Context, teamID, membershipID, userID, secret string) (IdentitySession, error)
	CreateEmailPasswordSession(ctx context.Context, email, password string) (IdentitySession, error)
	CreateRecovery(ctx context.Context, email, redirectURL string) error
	CompleteRecovery(ctx context.Context, userID, secret, password string) error
	UpdateCurrentPassword(ctx context.Context, sessionSecret, password string) error
	GetSession(ctx context.Context, sessionSecret string) (IdentitySession, error)
	DeleteSession(ctx context.Context, sessionSecret string) error
	GetCurrentUser(ctx context.Context, sessionSecret string) (IdentityUser, error)
	GetUserByID(ctx context.Context, userID string) (IdentityUser, error)
	GetUserByEmail(ctx context.Context, email string) (IdentityUser, error)
	AddOrganizationUserByIDAsAdmin(ctx context.Context, orgSlug, userID string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error)
	InviteOrganizationUser(ctx context.Context, sessionSecret, orgSlug, email, redirectURL string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error)
	ListOrganizations(ctx context.Context) ([]IdentityOrg, error)
	ListOrganizationMemberships(ctx context.Context, orgSlug string) ([]IdentityMembership, error)
	ListOrganizationUsers(ctx context.Context, orgSlug string) ([]IdentityUser, error)
	GetOrganizationBySlug(ctx context.Context, slug string) (*IdentityOrg, error)
	UpdateOrganization(ctx context.Context, sessionSecret, currentSlug, name, logoFileID string, roles []IdentityRole) (IdentityOrg, error)
	UpdateOrganizationAsAdmin(ctx context.Context, currentSlug, name, logoFileID string, roles []IdentityRole) (IdentityOrg, error)
	UpdateOrganizationMembership(ctx context.Context, sessionSecret, orgSlug, membershipID string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error)
	UpdateOrganizationMembershipAsAdmin(ctx context.Context, orgSlug, membershipID string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error)
	UpdateUserLabels(ctx context.Context, userID string, labels []string) (IdentityUser, error)
	DeleteOrganizationMembership(ctx context.Context, sessionSecret, orgSlug, membershipID string) error
	InviteOrganizationUserAsAdmin(ctx context.Context, orgSlug, email, redirectURL string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error)
	UploadOrganizationLogo(ctx context.Context, orgSlug string, upload IdentityFile) (IdentityFile, error)
	GetOrganizationLogo(ctx context.Context, fileID string) (IdentityFile, error)
}

type IdentityUser struct {
	ID              string
	Email           string
	OrgSlug         string
	OrgName         string
	Labels          []string
	IsOrgAdmin      bool
	MembershipID    string
	MembershipRoles []string
	Status          string
	PasswordSet     bool
}

type IdentityOrg struct {
	ID         string
	Slug       string
	Name       string
	LogoFileID string
	Roles      []IdentityRole
}

type IdentityRole struct {
	Slug   string `json:"slug"`
	Name   string `json:"name"`
	Color  string `json:"color,omitempty"`
	Border string `json:"border,omitempty"`
}

type IdentitySession struct {
	Secret    string
	ExpiresAt time.Time
	UserID    string
}

type IdentityFile struct {
	ID          string
	Filename    string
	ContentType string
	Data        []byte
}

type IdentityMembership struct {
	ID              string
	TeamID          string
	UserID          string
	Email           string
	MembershipRoles []string
	RoleSlugs       []string
	IsOrgAdmin      bool
	Confirmed       bool
	InvitedAt       time.Time
	JoinedAt        time.Time
}
