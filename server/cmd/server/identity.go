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
	CreateEmailPasswordSession(ctx context.Context, email, password string) (IdentitySession, error)
	GetSession(ctx context.Context, sessionSecret string) (IdentitySession, error)
	DeleteSession(ctx context.Context, sessionSecret string) error
	GetCurrentUser(ctx context.Context, sessionSecret string) (IdentityUser, error)
	GetUserByID(ctx context.Context, userID string) (IdentityUser, error)
	ListOrganizations(ctx context.Context) ([]IdentityOrg, error)
	GetOrganizationBySlug(ctx context.Context, slug string) (*IdentityOrg, error)
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
