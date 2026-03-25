package main

import (
	"context"
	"strings"
	"time"
)

type fakeIdentityStore struct {
	createAccountFunc                       func(ctx context.Context, email, password, name string) (IdentityUser, error)
	createOrganizationFunc                  func(ctx context.Context, sessionSecret, name string) (IdentityOrg, error)
	createOrganizationAsAdminFunc           func(ctx context.Context, name string) (IdentityOrg, error)
	ensurePlatformAdminAccountFunc          func(ctx context.Context, email, password string) error
	acceptInviteFunc                        func(ctx context.Context, teamID, membershipID, userID, secret string) (IdentitySession, error)
	createEmailPasswordSessionFunc          func(ctx context.Context, email, password string) (IdentitySession, error)
	createRecoveryFunc                      func(ctx context.Context, email, redirectURL string) error
	completeRecoveryFunc                    func(ctx context.Context, userID, secret, password string) error
	updateCurrentPasswordFunc               func(ctx context.Context, sessionSecret, password string) error
	getSessionFunc                          func(ctx context.Context, sessionSecret string) (IdentitySession, error)
	deleteSessionFunc                       func(ctx context.Context, sessionSecret string) error
	getCurrentUserFunc                      func(ctx context.Context, sessionSecret string) (IdentityUser, error)
	getUserByIDFunc                         func(ctx context.Context, userID string) (IdentityUser, error)
	getUserByEmailFunc                      func(ctx context.Context, email string) (IdentityUser, error)
	addOrganizationUserByIDAsAdminFunc      func(ctx context.Context, orgSlug, userID string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error)
	inviteOrganizationUserFunc              func(ctx context.Context, sessionSecret, orgSlug, email, redirectURL string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error)
	listOrganizationsFunc                   func(ctx context.Context) ([]IdentityOrg, error)
	listOrganizationMembershipsFunc         func(ctx context.Context, orgSlug string) ([]IdentityMembership, error)
	listOrganizationUsersFunc               func(ctx context.Context, orgSlug string) ([]IdentityUser, error)
	getOrganizationBySlugFunc               func(ctx context.Context, slug string) (*IdentityOrg, error)
	updateOrganizationFunc                  func(ctx context.Context, sessionSecret, currentSlug, name, logoFileID string, roles []IdentityRole) (IdentityOrg, error)
	updateOrganizationAsAdminFunc           func(ctx context.Context, currentSlug, name, logoFileID string, roles []IdentityRole) (IdentityOrg, error)
	updateOrganizationMembershipFunc        func(ctx context.Context, sessionSecret, orgSlug, membershipID string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error)
	updateOrganizationMembershipAsAdminFunc func(ctx context.Context, orgSlug, membershipID string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error)
	updateUserLabelsFunc                    func(ctx context.Context, userID string, labels []string) (IdentityUser, error)
	deleteOrganizationMembershipFunc        func(ctx context.Context, sessionSecret, orgSlug, membershipID string) error
	inviteOrganizationUserAsAdminFunc       func(ctx context.Context, orgSlug, email, redirectURL string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error)
	uploadOrganizationLogoFunc              func(ctx context.Context, orgSlug string, upload IdentityFile) (IdentityFile, error)
	deleteOrganizationLogoFunc              func(ctx context.Context, fileID string) error
	getOrganizationLogoFunc                 func(ctx context.Context, fileID string) (IdentityFile, error)
}

func (f *fakeIdentityStore) CreateAccount(ctx context.Context, email, password, name string) (IdentityUser, error) {
	if f.createAccountFunc != nil {
		return f.createAccountFunc(ctx, email, password, name)
	}
	return IdentityUser{}, ErrIdentityUnauthorized
}

func (f *fakeIdentityStore) CreateOrganization(ctx context.Context, sessionSecret, name string) (IdentityOrg, error) {
	if f.createOrganizationFunc != nil {
		return f.createOrganizationFunc(ctx, sessionSecret, name)
	}
	return IdentityOrg{}, ErrIdentityUnauthorized
}

func (f *fakeIdentityStore) CreateOrganizationAsAdmin(ctx context.Context, name string) (IdentityOrg, error) {
	if f.createOrganizationAsAdminFunc != nil {
		return f.createOrganizationAsAdminFunc(ctx, name)
	}
	return IdentityOrg{}, ErrIdentityUnauthorized
}

func (f *fakeIdentityStore) EnsurePlatformAdminAccount(ctx context.Context, email, password string) error {
	if f.ensurePlatformAdminAccountFunc != nil {
		return f.ensurePlatformAdminAccountFunc(ctx, email, password)
	}
	return nil
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

func (f *fakeIdentityStore) UpdateCurrentPassword(ctx context.Context, sessionSecret, password string) error {
	if f.updateCurrentPasswordFunc != nil {
		return f.updateCurrentPasswordFunc(ctx, sessionSecret, password)
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

func (f *fakeIdentityStore) GetUserByEmail(ctx context.Context, email string) (IdentityUser, error) {
	if f.getUserByEmailFunc != nil {
		return f.getUserByEmailFunc(ctx, email)
	}
	return IdentityUser{}, ErrIdentityNotFound
}

func (f *fakeIdentityStore) AddOrganizationUserByIDAsAdmin(ctx context.Context, orgSlug, userID string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
	if f.addOrganizationUserByIDAsAdminFunc != nil {
		return f.addOrganizationUserByIDAsAdminFunc(ctx, orgSlug, userID, roleSlugs, isOrgAdmin)
	}
	return IdentityMembership{}, ErrIdentityUnauthorized
}

func (f *fakeIdentityStore) InviteOrganizationUser(ctx context.Context, sessionSecret, orgSlug, email, redirectURL string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
	if f.inviteOrganizationUserFunc != nil {
		return f.inviteOrganizationUserFunc(ctx, sessionSecret, orgSlug, email, redirectURL, roleSlugs, isOrgAdmin)
	}
	return IdentityMembership{}, ErrIdentityUnauthorized
}

func (f *fakeIdentityStore) ListOrganizations(ctx context.Context) ([]IdentityOrg, error) {
	if f.listOrganizationsFunc != nil {
		return f.listOrganizationsFunc(ctx)
	}
	return nil, nil
}

func (f *fakeIdentityStore) ListOrganizationMemberships(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
	if f.listOrganizationMembershipsFunc != nil {
		return f.listOrganizationMembershipsFunc(ctx, orgSlug)
	}
	return nil, nil
}

func (f *fakeIdentityStore) ListOrganizationUsers(ctx context.Context, orgSlug string) ([]IdentityUser, error) {
	if f.listOrganizationUsersFunc != nil {
		return f.listOrganizationUsersFunc(ctx, orgSlug)
	}
	return nil, nil
}

func (f *fakeIdentityStore) GetOrganizationBySlug(ctx context.Context, slug string) (*IdentityOrg, error) {
	if f.getOrganizationBySlugFunc != nil {
		return f.getOrganizationBySlugFunc(ctx, slug)
	}
	return nil, ErrIdentityNotFound
}

func (f *fakeIdentityStore) UpdateOrganization(ctx context.Context, sessionSecret, currentSlug, name, logoFileID string, roles []IdentityRole) (IdentityOrg, error) {
	if f.updateOrganizationFunc != nil {
		return f.updateOrganizationFunc(ctx, sessionSecret, currentSlug, name, logoFileID, roles)
	}
	return IdentityOrg{}, ErrIdentityUnauthorized
}

func (f *fakeIdentityStore) UpdateOrganizationAsAdmin(ctx context.Context, currentSlug, name, logoFileID string, roles []IdentityRole) (IdentityOrg, error) {
	if f.updateOrganizationAsAdminFunc != nil {
		return f.updateOrganizationAsAdminFunc(ctx, currentSlug, name, logoFileID, roles)
	}
	return IdentityOrg{}, ErrIdentityUnauthorized
}

func (f *fakeIdentityStore) UpdateOrganizationMembership(ctx context.Context, sessionSecret, orgSlug, membershipID string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
	if f.updateOrganizationMembershipFunc != nil {
		return f.updateOrganizationMembershipFunc(ctx, sessionSecret, orgSlug, membershipID, roleSlugs, isOrgAdmin)
	}
	return IdentityMembership{}, ErrIdentityUnauthorized
}

func (f *fakeIdentityStore) UpdateOrganizationMembershipAsAdmin(ctx context.Context, orgSlug, membershipID string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
	if f.updateOrganizationMembershipAsAdminFunc != nil {
		return f.updateOrganizationMembershipAsAdminFunc(ctx, orgSlug, membershipID, roleSlugs, isOrgAdmin)
	}
	return IdentityMembership{}, ErrIdentityUnauthorized
}

func (f *fakeIdentityStore) UpdateUserLabels(ctx context.Context, userID string, labels []string) (IdentityUser, error) {
	if f.updateUserLabelsFunc != nil {
		return f.updateUserLabelsFunc(ctx, userID, labels)
	}
	return IdentityUser{}, ErrIdentityUnauthorized
}

func (f *fakeIdentityStore) DeleteOrganizationMembership(ctx context.Context, sessionSecret, orgSlug, membershipID string) error {
	if f.deleteOrganizationMembershipFunc != nil {
		return f.deleteOrganizationMembershipFunc(ctx, sessionSecret, orgSlug, membershipID)
	}
	return nil
}

func (f *fakeIdentityStore) InviteOrganizationUserAsAdmin(ctx context.Context, orgSlug, email, redirectURL string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
	if f.inviteOrganizationUserAsAdminFunc != nil {
		return f.inviteOrganizationUserAsAdminFunc(ctx, orgSlug, email, redirectURL, roleSlugs, isOrgAdmin)
	}
	return IdentityMembership{}, ErrIdentityUnauthorized
}

func (f *fakeIdentityStore) UploadOrganizationLogo(ctx context.Context, orgSlug string, upload IdentityFile) (IdentityFile, error) {
	if f.uploadOrganizationLogoFunc != nil {
		return f.uploadOrganizationLogoFunc(ctx, orgSlug, upload)
	}
	return IdentityFile{}, ErrIdentityUnauthorized
}

func (f *fakeIdentityStore) DeleteOrganizationLogo(ctx context.Context, fileID string) error {
	if f.deleteOrganizationLogoFunc != nil {
		return f.deleteOrganizationLogoFunc(ctx, fileID)
	}
	return nil
}

func (f *fakeIdentityStore) GetOrganizationLogo(ctx context.Context, fileID string) (IdentityFile, error) {
	if f.getOrganizationLogoFunc != nil {
		return f.getOrganizationLogoFunc(ctx, fileID)
	}
	return IdentityFile{}, ErrIdentityNotFound
}

func fakeIdentitySession(secret, userID string, expiresAt time.Time) IdentitySession {
	return IdentitySession{
		Secret:    strings.TrimSpace(secret),
		UserID:    strings.TrimSpace(userID),
		ExpiresAt: expiresAt.UTC(),
	}
}
