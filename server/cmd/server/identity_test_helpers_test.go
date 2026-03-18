package main

import (
	"context"
	"strings"
	"time"
)

type fakeIdentityStore struct {
	createAccountFunc              func(ctx context.Context, email, password, name string) (IdentityUser, error)
	acceptInviteFunc               func(ctx context.Context, teamID, membershipID, userID, secret string) (IdentitySession, error)
	createEmailPasswordSessionFunc func(ctx context.Context, email, password string) (IdentitySession, error)
	createRecoveryFunc             func(ctx context.Context, email, redirectURL string) error
	completeRecoveryFunc           func(ctx context.Context, userID, secret, password string) error
	getSessionFunc                 func(ctx context.Context, sessionSecret string) (IdentitySession, error)
	deleteSessionFunc              func(ctx context.Context, sessionSecret string) error
	getCurrentUserFunc             func(ctx context.Context, sessionSecret string) (IdentityUser, error)
	getUserByIDFunc                func(ctx context.Context, userID string) (IdentityUser, error)
	listOrganizationsFunc          func(ctx context.Context) ([]IdentityOrg, error)
	getOrganizationBySlugFunc      func(ctx context.Context, slug string) (*IdentityOrg, error)
}

func (f *fakeIdentityStore) CreateAccount(ctx context.Context, email, password, name string) (IdentityUser, error) {
	if f.createAccountFunc != nil {
		return f.createAccountFunc(ctx, email, password, name)
	}
	return IdentityUser{}, ErrIdentityUnauthorized
}

func (f *fakeIdentityStore) AcceptInvite(ctx context.Context, teamID, membershipID, userID, secret string) (IdentitySession, error) {
	if f.acceptInviteFunc != nil {
		return f.acceptInviteFunc(ctx, teamID, membershipID, userID, secret)
	}
	return IdentitySession{}, ErrIdentityUnauthorized
}

func (f *fakeIdentityStore) CreateEmailPasswordSession(ctx context.Context, email, password string) (IdentitySession, error) {
	if f.createEmailPasswordSessionFunc != nil {
		return f.createEmailPasswordSessionFunc(ctx, email, password)
	}
	return IdentitySession{}, ErrIdentityUnauthorized
}

func (f *fakeIdentityStore) CreateRecovery(ctx context.Context, email, redirectURL string) error {
	if f.createRecoveryFunc != nil {
		return f.createRecoveryFunc(ctx, email, redirectURL)
	}
	return nil
}

func (f *fakeIdentityStore) CompleteRecovery(ctx context.Context, userID, secret, password string) error {
	if f.completeRecoveryFunc != nil {
		return f.completeRecoveryFunc(ctx, userID, secret, password)
	}
	return nil
}

func (f *fakeIdentityStore) GetSession(ctx context.Context, sessionSecret string) (IdentitySession, error) {
	if f.getSessionFunc != nil {
		return f.getSessionFunc(ctx, sessionSecret)
	}
	return IdentitySession{}, ErrIdentityUnauthorized
}

func (f *fakeIdentityStore) DeleteSession(ctx context.Context, sessionSecret string) error {
	if f.deleteSessionFunc != nil {
		return f.deleteSessionFunc(ctx, sessionSecret)
	}
	return nil
}

func (f *fakeIdentityStore) GetCurrentUser(ctx context.Context, sessionSecret string) (IdentityUser, error) {
	if f.getCurrentUserFunc != nil {
		return f.getCurrentUserFunc(ctx, sessionSecret)
	}
	return IdentityUser{}, ErrIdentityUnauthorized
}

func (f *fakeIdentityStore) GetUserByID(ctx context.Context, userID string) (IdentityUser, error) {
	if f.getUserByIDFunc != nil {
		return f.getUserByIDFunc(ctx, userID)
	}
	return IdentityUser{}, ErrIdentityNotFound
}

func (f *fakeIdentityStore) ListOrganizations(ctx context.Context) ([]IdentityOrg, error) {
	if f.listOrganizationsFunc != nil {
		return f.listOrganizationsFunc(ctx)
	}
	return nil, nil
}

func (f *fakeIdentityStore) GetOrganizationBySlug(ctx context.Context, slug string) (*IdentityOrg, error) {
	if f.getOrganizationBySlugFunc != nil {
		return f.getOrganizationBySlugFunc(ctx, slug)
	}
	return nil, ErrIdentityNotFound
}

func fakeIdentitySession(secret, userID string, expiresAt time.Time) IdentitySession {
	return IdentitySession{
		Secret:    strings.TrimSpace(secret),
		UserID:    strings.TrimSpace(userID),
		ExpiresAt: expiresAt.UTC(),
	}
}
