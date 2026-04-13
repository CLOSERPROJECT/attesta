package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/appwrite/sdk-for-go/account"
	"github.com/appwrite/sdk-for-go/appwrite"
	appwriteclient "github.com/appwrite/sdk-for-go/client"
	appwritefile "github.com/appwrite/sdk-for-go/file"
	"github.com/appwrite/sdk-for-go/id"
	"github.com/appwrite/sdk-for-go/models"
	"github.com/appwrite/sdk-for-go/storage"
	"github.com/appwrite/sdk-for-go/teams"
	"github.com/appwrite/sdk-for-go/users"
)

type appwriteTeamPrefs struct {
	SchemaVersion int            `json:"schemaVersion,omitempty"`
	Slug          string         `json:"slug,omitempty"`
	LogoFileID    string         `json:"logoFileId,omitempty"`
	Roles         []IdentityRole `json:"roles,omitempty"`
}

type appwriteIdentity struct {
	adminClient     appwriteclient.Client
	sessionClient   appwriteclient.Client
	orgAssetsBucket string
}

// NewAppwriteIdentity builds the Appwrite-backed identity adapter used by later auth work.
func NewAppwriteIdentity(endpoint, projectID, apiKey string, httpClient *http.Client) IdentityStore {
	adminClient := appwrite.NewClient(
		appwrite.WithEndpoint(strings.TrimRight(strings.TrimSpace(endpoint), "/")),
		appwrite.WithProject(strings.TrimSpace(projectID)),
		appwrite.WithKey(strings.TrimSpace(apiKey)),
	)
	sessionClient := appwrite.NewClient(
		appwrite.WithEndpoint(strings.TrimRight(strings.TrimSpace(endpoint), "/")),
		appwrite.WithProject(strings.TrimSpace(projectID)),
	)
	if httpClient != nil {
		adminClient.Client = cloneHTTPClient(httpClient, false)
		sessionClient.Client = cloneHTTPClient(httpClient, true)
	}
	return &appwriteIdentity{
		adminClient:     adminClient,
		sessionClient:   sessionClient,
		orgAssetsBucket: appwriteOrgAssetsBucket(),
	}
}

func appwriteOrgAssetsBucket() string {
	bucket := strings.TrimSpace(strings.ToLower(strings.TrimSpace(envOr("APPWRITE_ORG_ASSETS_BUCKET", "org-assets"))))
	if bucket == "" {
		return "org-assets"
	}
	return bucket
}

func (a *appwriteIdentity) CreateEmailPasswordSession(ctx context.Context, email, password string) (IdentitySession, error) {
	if err := ctx.Err(); err != nil {
		return IdentitySession{}, err
	}
	session, err := account.New(a.adminClient).CreateEmailPasswordSession(strings.TrimSpace(email), password)
	if err != nil {
		return IdentitySession{}, normalizeIdentityError(err)
	}
	if err := ctx.Err(); err != nil {
		return IdentitySession{}, err
	}
	return toIdentitySession(session, "")
}

func (a *appwriteIdentity) CreateAccount(ctx context.Context, email, password, name string) (IdentityUser, error) {
	if err := ctx.Err(); err != nil {
		return IdentityUser{}, err
	}
	user, err := account.New(a.sessionClient).Create(
		id.Unique(),
		strings.TrimSpace(email),
		password,
		account.New(a.sessionClient).WithCreateName(strings.TrimSpace(name)),
	)
	if err != nil {
		return IdentityUser{}, normalizeIdentityError(err)
	}
	return toIdentityUser(user, nil), nil
}

func (a *appwriteIdentity) CreateOrganization(ctx context.Context, sessionSecret, name string) (IdentityOrg, error) {
	if err := ctx.Err(); err != nil {
		return IdentityOrg{}, err
	}
	sessionClient, err := cloneAppwriteClient(a.sessionClient, appwrite.WithSession(strings.TrimSpace(sessionSecret)))
	if err != nil {
		return IdentityOrg{}, err
	}
	org, err := a.createOrganizationWithClient(ctx, sessionClient, name)
	if err != nil {
		return IdentityOrg{}, err
	}
	accountUser, err := account.New(sessionClient).Get()
	if err != nil {
		return IdentityOrg{}, normalizeIdentityError(err)
	}
	labels := make([]string, 0, len(accountUser.Labels)+1)
	for _, label := range accountUser.Labels {
		if strings.EqualFold(strings.TrimSpace(label), identityOrgAdminLabel) {
			continue
		}
		labels = append(labels, strings.TrimSpace(label))
	}
	labels = append(labels, identityOrgAdminLabel)
	if _, err := users.New(a.adminClient).UpdateLabels(accountUser.Id, uniqueIdentityStrings(labels)); err != nil {
		return IdentityOrg{}, normalizeIdentityError(err)
	}
	return org, nil
}

func (a *appwriteIdentity) CreateOrganizationAsAdmin(ctx context.Context, name string) (IdentityOrg, error) {
	if err := ctx.Err(); err != nil {
		return IdentityOrg{}, err
	}
	return a.createOrganizationWithClient(ctx, a.adminClient, name)
}

func (a *appwriteIdentity) EnsurePlatformAdminAccount(ctx context.Context, email, password string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	email = strings.ToLower(strings.TrimSpace(email))
	password = strings.TrimSpace(password)
	if email == "" || password == "" {
		return nil
	}
	user, err := a.GetUserByEmail(ctx, email)
	switch {
	case err == nil:
		if _, err := users.New(a.adminClient).UpdatePassword(strings.TrimSpace(user.ID), password); err != nil {
			return normalizeIdentityError(err)
		}
		return nil
	case err != nil && !errors.Is(err, ErrIdentityNotFound):
		return err
	}
	if _, err := users.New(a.adminClient).Create(
		id.Unique(),
		users.New(a.adminClient).WithCreateEmail(email),
		users.New(a.adminClient).WithCreatePassword(password),
		users.New(a.adminClient).WithCreateName("Platform Admin"),
	); err != nil {
		return normalizeIdentityError(err)
	}
	return nil
}

func (a *appwriteIdentity) AcceptInvite(ctx context.Context, teamID, membershipID, userID, secret string) (IdentitySession, error) {
	if err := ctx.Err(); err != nil {
		return IdentitySession{}, err
	}
	sessionClient, err := cloneAppwriteClient(a.sessionClient)
	if err != nil {
		return IdentitySession{}, err
	}
	membership, err := teams.New(sessionClient).UpdateMembershipStatus(
		strings.TrimSpace(teamID),
		strings.TrimSpace(membershipID),
		strings.TrimSpace(userID),
		strings.TrimSpace(secret),
	)
	if err != nil {
		return IdentitySession{}, normalizeIdentityError(err)
	}
	accountUser, err := account.New(sessionClient).Get()
	if err != nil {
		return IdentitySession{}, normalizeIdentityError(err)
	}
	decodedRoles := decodeInviteMembershipRoles(membership.Roles)
	labels := make([]string, 0, len(accountUser.Labels)+len(decodedRoles.BusinessRoles)+1)
	for _, label := range accountUser.Labels {
		if isManagedIdentityLabel(label) {
			continue
		}
		labels = append(labels, strings.TrimSpace(label))
	}
	for _, roleSlug := range decodedRoles.BusinessRoles {
		labels = append(labels, encodeIdentityRoleLabel(roleSlug))
	}
	if decodedRoles.IsOrgAdmin {
		labels = append(labels, identityOrgAdminLabel)
	}
	if _, err := users.New(a.adminClient).UpdateLabels(accountUser.Id, uniqueIdentityStrings(labels)); err != nil {
		return IdentitySession{}, normalizeIdentityError(err)
	}
	session, err := account.New(sessionClient).GetSession("current")
	if err != nil {
		return IdentitySession{}, normalizeIdentityError(err)
	}
	return toIdentitySession(session, appwriteSessionSecretFromJar(sessionClient))
}

func (a *appwriteIdentity) CreateRecovery(ctx context.Context, email, redirectURL string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err := account.New(a.sessionClient).CreateRecovery(strings.TrimSpace(email), strings.TrimSpace(redirectURL))
	return normalizeIdentityError(err)
}

func (a *appwriteIdentity) CompleteRecovery(ctx context.Context, userID, secret, password string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err := account.New(a.sessionClient).UpdateRecovery(strings.TrimSpace(userID), strings.TrimSpace(secret), password)
	return normalizeIdentityError(err)
}

func (a *appwriteIdentity) UpdateCurrentPassword(ctx context.Context, sessionSecret, password string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	sessionClient, err := cloneAppwriteClient(a.sessionClient, appwrite.WithSession(strings.TrimSpace(sessionSecret)))
	if err != nil {
		return err
	}
	_, err = account.New(sessionClient).UpdatePassword(password)
	return normalizeIdentityError(err)
}

func (a *appwriteIdentity) GetSession(ctx context.Context, sessionSecret string) (IdentitySession, error) {
	if err := ctx.Err(); err != nil {
		return IdentitySession{}, err
	}
	sessionClient, err := cloneAppwriteClient(a.sessionClient, appwrite.WithSession(strings.TrimSpace(sessionSecret)))
	if err != nil {
		return IdentitySession{}, err
	}
	session, err := account.New(sessionClient).GetSession("current")
	if err != nil {
		return IdentitySession{}, normalizeIdentityError(err)
	}
	if err := ctx.Err(); err != nil {
		return IdentitySession{}, err
	}
	return toIdentitySession(session, strings.TrimSpace(sessionSecret))
}

func (a *appwriteIdentity) DeleteSession(ctx context.Context, sessionSecret string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	sessionClient, err := cloneAppwriteClient(a.sessionClient, appwrite.WithSession(strings.TrimSpace(sessionSecret)))
	if err != nil {
		return err
	}
	_, err = account.New(sessionClient).DeleteSession("current")
	return normalizeIdentityError(err)
}

func (a *appwriteIdentity) GetCurrentUser(ctx context.Context, sessionSecret string) (IdentityUser, error) {
	if err := ctx.Err(); err != nil {
		return IdentityUser{}, err
	}
	sessionClient, err := cloneAppwriteClient(a.sessionClient, appwrite.WithSession(strings.TrimSpace(sessionSecret)))
	if err != nil {
		return IdentityUser{}, err
	}
	accountUser, err := account.New(sessionClient).Get()
	if err != nil {
		return IdentityUser{}, normalizeIdentityError(err)
	}
	memberships, err := users.New(a.adminClient).ListMemberships(accountUser.Id)
	if err != nil {
		return IdentityUser{}, normalizeIdentityError(err)
	}
	identity := toIdentityUser(accountUser, memberships.Memberships)
	if identity.OrgSlug != "" {
		if org, orgErr := a.getOrganizationByTeamID(ctx, identity.OrgSlug); orgErr == nil && org != nil {
			identity.OrgSlug = org.Slug
			identity.OrgName = org.Name
		}
	}
	return identity, nil
}

func (a *appwriteIdentity) GetUserByID(ctx context.Context, userID string) (IdentityUser, error) {
	if err := ctx.Err(); err != nil {
		return IdentityUser{}, err
	}
	userID = strings.TrimSpace(userID)
	user, err := users.New(a.adminClient).Get(userID)
	if err != nil {
		return IdentityUser{}, normalizeIdentityError(err)
	}
	memberships, err := users.New(a.adminClient).ListMemberships(userID)
	if err != nil {
		return IdentityUser{}, normalizeIdentityError(err)
	}
	identity := toIdentityUser(user, memberships.Memberships)
	if identity.OrgSlug != "" {
		if org, orgErr := a.getOrganizationByTeamID(ctx, identity.OrgSlug); orgErr == nil && org != nil {
			identity.OrgSlug = org.Slug
			identity.OrgName = org.Name
		}
	}
	return identity, nil
}

func (a *appwriteIdentity) GetUserByEmail(ctx context.Context, email string) (IdentityUser, error) {
	if err := ctx.Err(); err != nil {
		return IdentityUser{}, err
	}
	email = strings.ToLower(strings.TrimSpace(email))
	userList, err := users.New(a.adminClient).List(users.New(a.adminClient).WithListSearch(email))
	if err != nil {
		return IdentityUser{}, normalizeIdentityError(err)
	}
	for _, user := range userList.Users {
		if !strings.EqualFold(strings.TrimSpace(user.Email), email) {
			continue
		}
		return a.GetUserByID(ctx, user.Id)
	}
	return IdentityUser{}, ErrIdentityNotFound
}

func (a *appwriteIdentity) AddOrganizationUserByIDAsAdmin(ctx context.Context, orgSlug, userID string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
	if err := ctx.Err(); err != nil {
		return IdentityMembership{}, err
	}
	org, err := a.GetOrganizationBySlug(ctx, orgSlug)
	if err != nil {
		return IdentityMembership{}, err
	}
	membership, err := teams.New(a.adminClient).CreateMembership(
		strings.TrimSpace(org.ID),
		encodeInviteMembershipRoles(roleSlugs, isOrgAdmin),
		teams.New(a.adminClient).WithCreateMembershipUserId(strings.TrimSpace(userID)),
	)
	if err != nil {
		return IdentityMembership{}, normalizeIdentityError(err)
	}
	return a.toIdentityMembership(ctx, membership, org), nil
}

func (a *appwriteIdentity) InviteOrganizationUser(ctx context.Context, sessionSecret, orgSlug, email, redirectURL string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
	if err := ctx.Err(); err != nil {
		return IdentityMembership{}, err
	}
	sessionClient, err := cloneAppwriteClient(a.sessionClient, appwrite.WithSession(strings.TrimSpace(sessionSecret)))
	if err != nil {
		return IdentityMembership{}, err
	}
	return a.inviteOrganizationUserWithClient(ctx, sessionClient, orgSlug, email, redirectURL, roleSlugs, isOrgAdmin)
}

func (a *appwriteIdentity) InviteOrganizationUserAsAdmin(ctx context.Context, orgSlug, email, redirectURL string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
	if err := ctx.Err(); err != nil {
		return IdentityMembership{}, err
	}
	return a.inviteOrganizationUserWithClient(ctx, a.adminClient, orgSlug, email, redirectURL, roleSlugs, isOrgAdmin)
}

func (a *appwriteIdentity) inviteOrganizationUserWithClient(ctx context.Context, client appwriteclient.Client, orgSlug, email, redirectURL string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
	org, err := a.GetOrganizationBySlug(ctx, orgSlug)
	if err != nil {
		return IdentityMembership{}, err
	}
	membership, err := teams.New(client).CreateMembership(
		strings.TrimSpace(org.ID),
		encodeInviteMembershipRoles(roleSlugs, isOrgAdmin),
		teams.New(client).WithCreateMembershipEmail(strings.ToLower(strings.TrimSpace(email))),
		teams.New(client).WithCreateMembershipUrl(strings.TrimSpace(redirectURL)),
	)
	if err != nil {
		return IdentityMembership{}, normalizeIdentityError(err)
	}
	return a.toIdentityMembership(ctx, membership, org), nil
}

func (a *appwriteIdentity) ListOrganizations(ctx context.Context) ([]IdentityOrg, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	teamList, err := teams.New(a.adminClient).List()
	if err != nil {
		return nil, normalizeIdentityError(err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return decodeIdentityOrgs(teamList), nil
}

func (a *appwriteIdentity) ListOrganizationUsers(ctx context.Context, orgSlug string) ([]IdentityUser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	org, err := a.GetOrganizationBySlug(ctx, orgSlug)
	if err != nil {
		return nil, err
	}
	membershipList, err := teams.New(a.adminClient).ListMemberships(strings.TrimSpace(org.ID))
	if err != nil {
		return nil, normalizeIdentityError(err)
	}
	usersForOrg := make([]IdentityUser, 0, len(membershipList.Memberships))
	for _, membership := range membershipList.Memberships {
		if !membership.Confirm {
			continue
		}
		identity := IdentityUser{
			ID:              strings.TrimSpace(membership.UserId),
			Email:           strings.TrimSpace(membership.UserEmail),
			OrgSlug:         strings.TrimSpace(org.Slug),
			OrgName:         strings.TrimSpace(org.Name),
			MembershipID:    strings.TrimSpace(membership.Id),
			MembershipRoles: append([]string(nil), membership.Roles...),
			Status:          "active",
		}
		if !membership.Confirm {
			identity.Status = "pending"
		}
		if identity.ID != "" {
			user, userErr := users.New(a.adminClient).Get(identity.ID)
			if userErr != nil {
				return nil, normalizeIdentityError(userErr)
			}
			accountIdentity := toIdentityUser(user, []models.Membership{membership})
			accountIdentity.OrgSlug = strings.TrimSpace(org.Slug)
			accountIdentity.OrgName = strings.TrimSpace(org.Name)
			identity = accountIdentity
		}
		usersForOrg = append(usersForOrg, identity)
	}
	return usersForOrg, nil
}

func (a *appwriteIdentity) ListOrganizationMemberships(ctx context.Context, orgSlug string) ([]IdentityMembership, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	org, err := a.GetOrganizationBySlug(ctx, orgSlug)
	if err != nil {
		return nil, err
	}
	membershipList, err := teams.New(a.adminClient).ListMemberships(strings.TrimSpace(org.ID))
	if err != nil {
		return nil, normalizeIdentityError(err)
	}
	memberships := make([]IdentityMembership, 0, len(membershipList.Memberships))
	for _, membership := range membershipList.Memberships {
		memberships = append(memberships, a.toIdentityMembership(ctx, &membership, org))
	}
	return memberships, nil
}

func (a *appwriteIdentity) GetOrganizationBySlug(ctx context.Context, slug string) (*IdentityOrg, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, ErrIdentityNotFound
	}
	team, err := teams.New(a.adminClient).Get(slug)
	if err == nil {
		org := decodeIdentityOrg(team)
		return &org, nil
	}
	if !errors.Is(normalizeIdentityError(err), ErrIdentityNotFound) {
		return nil, normalizeIdentityError(err)
	}
	orgs, listErr := a.ListOrganizations(ctx)
	if listErr != nil {
		return nil, listErr
	}
	for _, org := range orgs {
		if strings.EqualFold(strings.TrimSpace(org.Slug), slug) {
			found := org
			return &found, nil
		}
	}
	return nil, ErrIdentityNotFound
}

func (a *appwriteIdentity) UpdateOrganization(ctx context.Context, sessionSecret, currentSlug, name, logoFileID string, roles []IdentityRole) (IdentityOrg, error) {
	if err := ctx.Err(); err != nil {
		return IdentityOrg{}, err
	}
	sessionClient, err := cloneAppwriteClient(a.sessionClient, appwrite.WithSession(strings.TrimSpace(sessionSecret)))
	if err != nil {
		return IdentityOrg{}, err
	}
	return a.updateOrganizationWithClient(ctx, sessionClient, currentSlug, name, logoFileID, roles)
}

func (a *appwriteIdentity) UpdateOrganizationAsAdmin(ctx context.Context, currentSlug, name, logoFileID string, roles []IdentityRole) (IdentityOrg, error) {
	if err := ctx.Err(); err != nil {
		return IdentityOrg{}, err
	}
	return a.updateOrganizationWithClient(ctx, a.adminClient, currentSlug, name, logoFileID, roles)
}

func (a *appwriteIdentity) DeleteOrganizationAsAdmin(ctx context.Context, orgSlug string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	org, err := a.GetOrganizationBySlug(ctx, orgSlug)
	if err != nil {
		return err
	}
	_, err = teams.New(a.adminClient).Delete(strings.TrimSpace(org.ID))
	return normalizeIdentityError(err)
}

func (a *appwriteIdentity) UpdateOrganizationMembership(ctx context.Context, sessionSecret, orgSlug, membershipID string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
	if err := ctx.Err(); err != nil {
		return IdentityMembership{}, err
	}
	sessionClient, err := cloneAppwriteClient(a.sessionClient, appwrite.WithSession(strings.TrimSpace(sessionSecret)))
	if err != nil {
		return IdentityMembership{}, err
	}
	return a.updateOrganizationMembershipWithClient(ctx, sessionClient, orgSlug, membershipID, roleSlugs, isOrgAdmin)
}

func (a *appwriteIdentity) UpdateOrganizationMembershipAsAdmin(ctx context.Context, orgSlug, membershipID string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
	if err := ctx.Err(); err != nil {
		return IdentityMembership{}, err
	}
	return a.updateOrganizationMembershipWithClient(ctx, a.adminClient, orgSlug, membershipID, roleSlugs, isOrgAdmin)
}

func (a *appwriteIdentity) updateOrganizationMembershipWithClient(ctx context.Context, client appwriteclient.Client, orgSlug, membershipID string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
	org, err := a.GetOrganizationBySlug(ctx, orgSlug)
	if err != nil {
		return IdentityMembership{}, err
	}
	membership, err := teams.New(client).UpdateMembership(strings.TrimSpace(org.ID), strings.TrimSpace(membershipID), encodeInviteMembershipRoles(roleSlugs, isOrgAdmin))
	if err != nil {
		return IdentityMembership{}, normalizeIdentityError(err)
	}
	return a.toIdentityMembership(ctx, membership, org), nil
}

func (a *appwriteIdentity) UpdateUserLabels(ctx context.Context, userID string, labels []string) (IdentityUser, error) {
	if err := ctx.Err(); err != nil {
		return IdentityUser{}, err
	}
	user, err := users.New(a.adminClient).UpdateLabels(strings.TrimSpace(userID), uniqueIdentityStrings(labels))
	if err != nil {
		return IdentityUser{}, normalizeIdentityError(err)
	}
	memberships, err := users.New(a.adminClient).ListMemberships(strings.TrimSpace(userID))
	if err != nil {
		return IdentityUser{}, normalizeIdentityError(err)
	}
	identity := toIdentityUser(user, memberships.Memberships)
	if identity.OrgSlug != "" {
		if org, orgErr := a.getOrganizationByTeamID(ctx, identity.OrgSlug); orgErr == nil && org != nil {
			identity.OrgSlug = org.Slug
			identity.OrgName = org.Name
		}
	}
	return identity, nil
}

func (a *appwriteIdentity) DeleteOrganizationMembership(ctx context.Context, sessionSecret, orgSlug, membershipID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	org, err := a.GetOrganizationBySlug(ctx, orgSlug)
	if err != nil {
		return err
	}
	sessionClient, err := cloneAppwriteClient(a.sessionClient, appwrite.WithSession(strings.TrimSpace(sessionSecret)))
	if err != nil {
		return err
	}
	_, err = teams.New(sessionClient).DeleteMembership(strings.TrimSpace(org.ID), strings.TrimSpace(membershipID))
	return normalizeIdentityError(err)
}

func (a *appwriteIdentity) UploadOrganizationLogo(ctx context.Context, orgSlug string, upload IdentityFile) (IdentityFile, error) {
	if err := ctx.Err(); err != nil {
		return IdentityFile{}, err
	}
	tempFile, err := os.CreateTemp("", "attesta-org-logo-*")
	if err != nil {
		return IdentityFile{}, err
	}
	defer os.Remove(tempFile.Name())
	if _, err := tempFile.Write(upload.Data); err != nil {
		_ = tempFile.Close()
		return IdentityFile{}, err
	}
	if err := tempFile.Close(); err != nil {
		return IdentityFile{}, err
	}
	fileID := id.Unique()
	created, err := storage.New(a.adminClient).CreateFile(
		a.orgAssetsBucket,
		fileID,
		appwritefile.NewInputFile(tempFile.Name(), strings.TrimSpace(upload.Filename)),
	)
	if err != nil {
		return IdentityFile{}, normalizeIdentityError(err)
	}
	contentType := strings.TrimSpace(upload.ContentType)
	if contentType == "" {
		contentType = strings.TrimSpace(created.MimeType)
	}
	return IdentityFile{
		ID:          strings.TrimSpace(created.Id),
		Filename:    strings.TrimSpace(created.Name),
		ContentType: contentType,
	}, nil
}

func (a *appwriteIdentity) DeleteOrganizationLogo(ctx context.Context, fileID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err := storage.New(a.adminClient).DeleteFile(a.orgAssetsBucket, strings.TrimSpace(fileID))
	return normalizeIdentityError(err)
}

func (a *appwriteIdentity) GetOrganizationLogo(ctx context.Context, fileID string) (IdentityFile, error) {
	if err := ctx.Err(); err != nil {
		return IdentityFile{}, err
	}
	meta, err := storage.New(a.adminClient).GetFile(a.orgAssetsBucket, strings.TrimSpace(fileID))
	if err != nil {
		return IdentityFile{}, normalizeIdentityError(err)
	}
	body, err := storage.New(a.adminClient).GetFileView(a.orgAssetsBucket, strings.TrimSpace(fileID))
	if err != nil {
		return IdentityFile{}, normalizeIdentityError(err)
	}
	return IdentityFile{
		ID:          strings.TrimSpace(meta.Id),
		Filename:    strings.TrimSpace(meta.Name),
		ContentType: strings.TrimSpace(meta.MimeType),
		Data:        append([]byte(nil), (*body)...),
	}, nil
}

func (a *appwriteIdentity) createOrganizationWithClient(ctx context.Context, client appwriteclient.Client, name string) (IdentityOrg, error) {
	name = strings.TrimSpace(name)
	slug := canonifySlug(name)
	team, err := teams.New(client).Create(slug, name)
	if err != nil {
		return IdentityOrg{}, normalizeIdentityError(err)
	}
	org := decodeIdentityOrg(team)
	org.Slug = slug
	if _, err := teams.New(a.adminClient).UpdatePrefs(team.Id, encodeIdentityOrgPrefs(org)); err != nil {
		return IdentityOrg{}, normalizeIdentityError(err)
	}
	return decodeIdentityOrgFromTeam(team.Id, team.Name, encodeIdentityOrgPrefs(org)), nil
}

func (a *appwriteIdentity) updateOrganizationWithClient(ctx context.Context, client appwriteclient.Client, currentSlug, name, logoFileID string, roles []IdentityRole) (IdentityOrg, error) {
	org, err := a.GetOrganizationBySlug(ctx, currentSlug)
	if err != nil {
		return IdentityOrg{}, err
	}
	name = strings.TrimSpace(name)
	teamID := strings.TrimSpace(org.ID)
	updatedTeam, err := teams.New(client).UpdateName(teamID, name)
	if err != nil {
		return IdentityOrg{}, normalizeIdentityError(err)
	}
	updatedOrg := IdentityOrg{
		ID:         teamID,
		Slug:       canonifySlug(name),
		Name:       strings.TrimSpace(updatedTeam.Name),
		LogoFileID: strings.TrimSpace(logoFileID),
		Roles:      append([]IdentityRole(nil), roles...),
	}
	if _, err := teams.New(a.adminClient).UpdatePrefs(teamID, encodeIdentityOrgPrefs(updatedOrg)); err != nil {
		return IdentityOrg{}, normalizeIdentityError(err)
	}
	return updatedOrg, nil
}

func cloneAppwriteClient(base appwriteclient.Client, setters ...appwriteclient.ClientOption) (appwriteclient.Client, error) {
	cloned := base
	cloned.Headers = make(map[string]string, len(base.Headers))
	for key, value := range base.Headers {
		cloned.Headers[key] = value
	}
	if base.Client != nil {
		cloned.Client = cloneHTTPClient(base.Client, true)
	}
	for _, setter := range setters {
		if err := setter(&cloned); err != nil {
			return appwriteclient.Client{}, err
		}
	}
	return cloned, nil
}

func cloneHTTPClient(base *http.Client, withJar bool) *http.Client {
	if base == nil {
		return nil
	}
	cloned := *base
	if withJar {
		if jar, err := cookiejar.New(nil); err == nil {
			cloned.Jar = jar
		}
	} else {
		cloned.Jar = nil
	}
	return &cloned
}

func appwriteSessionSecretFromJar(client appwriteclient.Client) string {
	if client.Client == nil || client.Client.Jar == nil {
		return ""
	}
	endpoint := strings.TrimSpace(client.Endpoint)
	if endpoint == "" {
		return ""
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return ""
	}
	for _, cookie := range client.Client.Jar.Cookies(parsed) {
		name := strings.TrimSpace(cookie.Name)
		if strings.HasPrefix(name, "a_session_legacy_") || strings.HasPrefix(name, "a_session_") {
			return strings.TrimSpace(cookie.Value)
		}
	}
	return ""
}

func toIdentitySession(session *models.Session, fallbackSecret string) (IdentitySession, error) {
	if session == nil {
		return IdentitySession{}, errors.New("session required")
	}
	expiresAt, err := parseAppwriteTime(session.Expire)
	if err != nil {
		return IdentitySession{}, err
	}
	secret := strings.TrimSpace(session.Secret)
	if secret == "" {
		secret = strings.TrimSpace(fallbackSecret)
	}
	if secret == "" {
		return IdentitySession{}, errors.New("appwrite session missing secret")
	}
	return IdentitySession{
		Secret:    secret,
		ExpiresAt: expiresAt,
		UserID:    strings.TrimSpace(session.UserId),
	}, nil
}

func toIdentityUser(user *models.User, memberships []models.Membership) IdentityUser {
	selected := selectPrimaryMembership(memberships)
	identity := IdentityUser{
		ID:     strings.TrimSpace(user.Id),
		Email:  strings.TrimSpace(user.Email),
		Labels: append([]string(nil), user.Labels...),
		Status: "active",
	}
	identity.PasswordSet = strings.TrimSpace(user.PasswordUpdate) != ""
	if !user.Status {
		identity.Status = "disabled"
	}
	if selected != nil {
		identity.OrgSlug = strings.TrimSpace(selected.TeamId)
		identity.OrgName = strings.TrimSpace(selected.TeamName)
		identity.MembershipID = strings.TrimSpace(selected.Id)
		identity.MembershipRoles = append([]string(nil), selected.Roles...)
		if !selected.Confirm && identity.Status == "active" {
			identity.Status = "pending"
		}
	}
	identity.IsOrgAdmin = hasIdentityLabel(identity.Labels, identityOrgAdminLabel) || hasMembershipRole(identity.MembershipRoles, identityMembershipOwnerRole)
	return identity
}

func selectPrimaryMembership(memberships []models.Membership) *models.Membership {
	for idx := range memberships {
		if memberships[idx].Confirm {
			return &memberships[idx]
		}
	}
	if len(memberships) == 0 {
		return nil
	}
	return &memberships[0]
}

func decodeIdentityOrgs(teamList *models.TeamList) []IdentityOrg {
	if teamList == nil {
		return nil
	}
	type rawList struct {
		Teams []struct {
			ID    string            `json:"$id"`
			Name  string            `json:"name"`
			Prefs appwriteTeamPrefs `json:"prefs"`
		} `json:"teams"`
	}
	var payload rawList
	if err := teamList.Decode(&payload); err == nil {
		orgs := make([]IdentityOrg, 0, len(payload.Teams))
		for _, team := range payload.Teams {
			orgs = append(orgs, decodeIdentityOrgFromTeam(team.ID, team.Name, team.Prefs))
		}
		return orgs
	}
	orgs := make([]IdentityOrg, 0, len(teamList.Teams))
	for _, team := range teamList.Teams {
		orgs = append(orgs, IdentityOrg{
			ID:   strings.TrimSpace(team.Id),
			Slug: strings.TrimSpace(team.Id),
			Name: strings.TrimSpace(team.Name),
		})
	}
	return orgs
}

func decodeIdentityOrg(team *models.Team) IdentityOrg {
	if team == nil {
		return IdentityOrg{}
	}
	type rawTeam struct {
		Prefs appwriteTeamPrefs `json:"prefs"`
	}
	var payload rawTeam
	if err := team.Decode(&payload); err == nil {
		return decodeIdentityOrgFromTeam(team.Id, team.Name, payload.Prefs)
	}
	return IdentityOrg{
		ID:   strings.TrimSpace(team.Id),
		Slug: strings.TrimSpace(team.Id),
		Name: strings.TrimSpace(team.Name),
	}
}

func (a *appwriteIdentity) getOrganizationByTeamID(ctx context.Context, teamID string) (*IdentityOrg, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	team, err := teams.New(a.adminClient).Get(strings.TrimSpace(teamID))
	if err != nil {
		return nil, normalizeIdentityError(err)
	}
	org := decodeIdentityOrg(team)
	return &org, nil
}

func (a *appwriteIdentity) toIdentityMembership(ctx context.Context, membership *models.Membership, org *IdentityOrg) IdentityMembership {
	if membership == nil {
		return IdentityMembership{}
	}
	decodedRoles := decodeInviteMembershipRoles(membership.Roles)
	identity := IdentityMembership{
		ID:              strings.TrimSpace(membership.Id),
		TeamID:          strings.TrimSpace(membership.TeamId),
		UserID:          strings.TrimSpace(membership.UserId),
		Email:           strings.TrimSpace(membership.UserEmail),
		MembershipRoles: append([]string(nil), membership.Roles...),
		RoleSlugs:       append([]string(nil), decodedRoles.BusinessRoles...),
		IsOrgAdmin:      decodedRoles.IsOrgAdmin || hasMembershipRole(membership.Roles, identityMembershipOwnerRole),
		Confirmed:       membership.Confirm,
	}
	if org != nil {
		identity.TeamID = strings.TrimSpace(org.ID)
	}
	if invitedAt, err := parseAppwriteTime(membership.Invited); err == nil {
		identity.InvitedAt = invitedAt
	}
	if joinedAt, err := parseAppwriteTime(membership.Joined); err == nil {
		identity.JoinedAt = joinedAt
	}
	if identity.UserID != "" {
		if user, err := a.GetUserByID(ctx, identity.UserID); err == nil {
			identity.Email = user.Email
			identity.RoleSlugs = decodeIdentityRoleLabels(user.Labels)
			identity.IsOrgAdmin = user.IsOrgAdmin
		}
	}
	return identity
}

func hasIdentityLabel(labels []string, want string) bool {
	want = strings.TrimSpace(want)
	for _, label := range labels {
		if strings.EqualFold(strings.TrimSpace(label), want) {
			return true
		}
	}
	return false
}

func hasMembershipRole(roles []string, want string) bool {
	want = strings.TrimSpace(want)
	for _, role := range roles {
		if strings.EqualFold(strings.TrimSpace(role), want) {
			return true
		}
	}
	return false
}

func parseAppwriteTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, errors.New("missing appwrite timestamp")
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err == nil {
		return parsed, nil
	}
	return time.Parse(time.RFC3339, value)
}

func normalizeIdentityError(err error) error {
	if err == nil {
		return nil
	}
	var appwriteErr *appwriteclient.AppwriteError
	if errors.As(err, &appwriteErr) {
		switch appwriteErr.GetStatusCode() {
		case http.StatusNotFound:
			return ErrIdentityNotFound
		case http.StatusUnauthorized:
			return ErrIdentityUnauthorized
		}
	}
	return err
}
