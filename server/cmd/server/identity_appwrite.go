package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/appwrite/sdk-for-go/account"
	"github.com/appwrite/sdk-for-go/appwrite"
	appwriteclient "github.com/appwrite/sdk-for-go/client"
	"github.com/appwrite/sdk-for-go/id"
	"github.com/appwrite/sdk-for-go/models"
	"github.com/appwrite/sdk-for-go/teams"
	"github.com/appwrite/sdk-for-go/users"
)

type appwriteTeamPrefs struct {
	SchemaVersion int            `json:"schemaVersion,omitempty"`
	LogoFileID    string         `json:"logoFileId,omitempty"`
	Roles         []IdentityRole `json:"roles,omitempty"`
}

type appwriteIdentity struct {
	adminClient   appwriteclient.Client
	sessionClient appwriteclient.Client
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
		adminClient.Client = httpClient
		sessionClient.Client = httpClient
	}
	return &appwriteIdentity{
		adminClient:   adminClient,
		sessionClient: sessionClient,
	}
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

func (a *appwriteIdentity) AcceptInvite(ctx context.Context, teamID, membershipID, userID, secret string) (IdentitySession, error) {
	if err := ctx.Err(); err != nil {
		return IdentitySession{}, err
	}
	_, err := teams.New(a.sessionClient).UpdateMembershipStatus(
		strings.TrimSpace(teamID),
		strings.TrimSpace(membershipID),
		strings.TrimSpace(userID),
		strings.TrimSpace(secret),
	)
	if err != nil {
		return IdentitySession{}, normalizeIdentityError(err)
	}
	session, err := account.New(a.sessionClient).CreateSession(strings.TrimSpace(userID), strings.TrimSpace(secret))
	if err != nil {
		return IdentitySession{}, normalizeIdentityError(err)
	}
	return toIdentitySession(session, "")
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
	return toIdentityUser(accountUser, memberships.Memberships), nil
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
	return toIdentityUser(user, memberships.Memberships), nil
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

func (a *appwriteIdentity) GetOrganizationBySlug(ctx context.Context, slug string) (*IdentityOrg, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	team, err := teams.New(a.adminClient).Get(strings.TrimSpace(slug))
	if err != nil {
		return nil, normalizeIdentityError(err)
	}
	org := decodeIdentityOrg(team)
	return &org, nil
}

func cloneAppwriteClient(base appwriteclient.Client, setters ...appwriteclient.ClientOption) (appwriteclient.Client, error) {
	cloned := base
	cloned.Headers = make(map[string]string, len(base.Headers))
	for key, value := range base.Headers {
		cloned.Headers[key] = value
	}
	for _, setter := range setters {
		if err := setter(&cloned); err != nil {
			return appwriteclient.Client{}, err
		}
	}
	return cloned, nil
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
