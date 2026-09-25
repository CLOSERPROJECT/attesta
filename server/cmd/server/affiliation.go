package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Domain errors for affiliation submit/approve/reject/leave flows.
var (
	ErrAffiliationAlreadyAffiliated      = errors.New("affiliation: already affiliated")
	ErrAffiliationPendingExists          = errors.New("affiliation: pending request exists")
	ErrAffiliationNotFound               = errors.New("affiliation: request not found")
	ErrAffiliationNotPending             = errors.New("affiliation: request is not pending")
	ErrAffiliationInvalidName            = errors.New("affiliation: organization name is required")
	ErrAffiliationInvalidRoles           = errors.New("affiliation: invalid roles")
	ErrAffiliationOrganizationSlugExists = errors.New("affiliation: organization slug already exists")
	ErrAffiliationNotAffiliated          = errors.New("affiliation: not affiliated")
	ErrAffiliationSoleOrgAdmin           = errors.New("affiliation: sole organization admin")
	ErrAffiliationInviteAcceptFailed     = errors.New("affiliation: invite accept failed")
)

const (
	MailKindOrgCreationSubmitted = "affiliation.organization_creation.submitted"
	MailKindOrgCreationApproved  = "affiliation.organization_creation.approved"
	MailKindOrgCreationRejected  = "affiliation.organization_creation.rejected"
	MailKindJoinSubmitted        = "affiliation.join.submitted"
	MailKindJoinApproved         = "affiliation.join.approved"
	MailKindJoinRejected         = "affiliation.join.rejected"
)

type AffiliationRequestStatus string

const (
	AffiliationStatusPending  AffiliationRequestStatus = "pending"
	AffiliationStatusApproved AffiliationRequestStatus = "approved"
	AffiliationStatusRejected AffiliationRequestStatus = "rejected"
)

type JoinRequest struct {
	ID              primitive.ObjectID       `bson:"_id,omitempty"`
	RequesterUserID string                   `bson:"requesterUserId"`
	RequesterEmail  string                   `bson:"requesterEmail"`
	OrgSlug         string                   `bson:"orgSlug"`
	RoleSlugs       []string                 `bson:"roleSlugs,omitempty"`
	Status          AffiliationRequestStatus `bson:"status"`
	RejectReason    string                   `bson:"rejectReason,omitempty"`
	DecidedByUserID string                   `bson:"decidedByUserId,omitempty"`
	CreatedAt       time.Time                `bson:"createdAt"`
	UpdatedAt       time.Time                `bson:"updatedAt"`
	DecidedAt       time.Time                `bson:"decidedAt,omitempty"`
}

type OrganizationCreationRequest struct {
	ID              primitive.ObjectID       `bson:"_id,omitempty"`
	RequesterUserID string                   `bson:"requesterUserId"`
	RequesterEmail  string                   `bson:"requesterEmail"`
	ProposedName    string                   `bson:"proposedName"`
	ProposedSlug    string                   `bson:"proposedSlug"`
	Status          AffiliationRequestStatus `bson:"status"`
	RejectReason    string                   `bson:"rejectReason,omitempty"`
	DecidedByUserID string                   `bson:"decidedByUserId,omitempty"`
	CreatedAt       time.Time                `bson:"createdAt"`
	UpdatedAt       time.Time                `bson:"updatedAt"`
	DecidedAt       time.Time                `bson:"decidedAt,omitempty"`
}

// affiliationStore is the persistence port for join and organization-creation intents.
// Pending list methods return status==pending only; list-all is not part of this port.
type affiliationStore interface {
	InsertJoinRequest(ctx context.Context, req JoinRequest) (JoinRequest, error)
	LoadJoinRequestByID(ctx context.Context, id primitive.ObjectID) (*JoinRequest, error)
	UpdateJoinRequest(ctx context.Context, req JoinRequest) (JoinRequest, error)
	DeleteJoinRequest(ctx context.Context, id primitive.ObjectID) error
	FindPendingJoinRequestByUser(ctx context.Context, userID string) (*JoinRequest, error)
	ListPendingJoinRequestsByOrg(ctx context.Context, orgSlug string) ([]JoinRequest, error)

	InsertOrganizationCreationRequest(ctx context.Context, req OrganizationCreationRequest) (OrganizationCreationRequest, error)
	LoadOrganizationCreationRequestByID(ctx context.Context, id primitive.ObjectID) (*OrganizationCreationRequest, error)
	UpdateOrganizationCreationRequest(ctx context.Context, req OrganizationCreationRequest) (OrganizationCreationRequest, error)
	DeleteOrganizationCreationRequest(ctx context.Context, id primitive.ObjectID) error
	FindPendingOrganizationCreationRequestByUser(ctx context.Context, userID string) (*OrganizationCreationRequest, error)
	ListPendingOrganizationCreationRequests(ctx context.Context) ([]OrganizationCreationRequest, error)
}

// Affiliation owns affiliation-domain queries and join / organization-creation /
// leave / invitation commands.
type Affiliation struct {
	identity                  IdentityStore
	store                     affiliationStore
	mailer                    Mailer
	now                       func() time.Time
	platformAdminNotifyEmails []string
}

func NewAffiliation(identity IdentityStore, store affiliationStore, mailer Mailer, now func() time.Time, platformAdminNotifyEmails []string) *Affiliation {
	if mailer == nil {
		mailer = noopMailer{}
	}
	if now == nil {
		now = time.Now
	}
	return &Affiliation{
		identity:                  identity,
		store:                     store,
		mailer:                    mailer,
		now:                       now,
		platformAdminNotifyEmails: normalizePlatformAdminNotifyEmails(platformAdminNotifyEmails),
	}
}

func normalizePlatformAdminNotifyEmails(emails []string) []string {
	if len(emails) == 0 {
		return nil
	}
	out := make([]string, 0, len(emails))
	for _, email := range emails {
		email = strings.TrimSpace(email)
		if email == "" {
			continue
		}
		out = append(out, email)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (a *Affiliation) IsAffiliated(user IdentityUser) bool {
	return strings.TrimSpace(user.OrgSlug) != ""
}

// ensureInviteOrgSlugCompatible returns nil when user may be invited into orgSlug.
// Unaffiliated users are always compatible; affiliated users must already match orgSlug.
func (a *Affiliation) ensureInviteOrgSlugCompatible(user IdentityUser, orgSlug string) error {
	if !a.IsAffiliated(user) {
		return nil
	}
	if strings.EqualFold(strings.TrimSpace(user.OrgSlug), strings.TrimSpace(orgSlug)) {
		return nil
	}
	return ErrAffiliationAlreadyAffiliated
}

// ensureInviteAcceptCompatible blocks affiliated users from accepting invites into a different org.
// Unaffiliated users and GetUserByID not-found (new invitees) are allowed through.
func (a *Affiliation) ensureInviteAcceptCompatible(ctx context.Context, userID, teamID string) error {
	user, err := a.identity.GetUserByID(ctx, userID)
	switch {
	case err == nil:
		// continue
	case errors.Is(err, ErrIdentityNotFound):
		return nil
	default:
		return err
	}
	if !a.IsAffiliated(user) {
		return nil
	}
	org, err := a.identity.GetOrganizationBySlug(ctx, user.OrgSlug)
	switch {
	case err == nil:
		// continue
	case errors.Is(err, ErrIdentityNotFound):
		return ErrAffiliationAlreadyAffiliated
	default:
		return err
	}
	if org == nil || strings.TrimSpace(org.ID) != teamID {
		return ErrAffiliationAlreadyAffiliated
	}
	return nil
}

// LeaveOrganization removes the session user from their organization when allowed.
// Sole org admins are blocked until another org admin exists.
func (a *Affiliation) LeaveOrganization(ctx context.Context, sessionSecret string, user IdentityUser) error {
	if !a.IsAffiliated(user) {
		return ErrAffiliationNotAffiliated
	}

	current, err := a.identity.GetCurrentUser(ctx, sessionSecret)
	if err != nil {
		return err
	}
	if strings.TrimSpace(current.ID) != strings.TrimSpace(user.ID) {
		return ErrIdentityUnauthorized
	}
	if !a.IsAffiliated(current) {
		return ErrAffiliationNotAffiliated
	}

	orgSlug := strings.TrimSpace(current.OrgSlug)
	if current.IsOrgAdmin {
		users, listErr := a.identity.ListOrganizationUsers(ctx, orgSlug)
		if listErr != nil {
			return listErr
		}
		adminCount := 0
		for _, orgUser := range users {
			if orgUser.IsOrgAdmin {
				adminCount++
			}
		}
		if adminCount < 2 {
			return ErrAffiliationSoleOrgAdmin
		}
	}

	membershipID := strings.TrimSpace(current.MembershipID)
	if membershipID == "" {
		return ErrIdentityNotFound
	}
	if err := a.identity.DeleteOrganizationMembership(ctx, sessionSecret, orgSlug, membershipID); err != nil {
		return err
	}

	return a.stripManagedIdentityLabels(ctx, current.ID)
}

func (a *Affiliation) PendingJoinRequestForUser(ctx context.Context, userID string) (*JoinRequest, error) {
	return a.store.FindPendingJoinRequestByUser(ctx, userID)
}

func (a *Affiliation) PendingOrganizationCreationRequestForUser(ctx context.Context, userID string) (*OrganizationCreationRequest, error) {
	return a.store.FindPendingOrganizationCreationRequestByUser(ctx, userID)
}

// WithdrawPendingJoinRequest deletes the caller's pending join request (as if never submitted).
func (a *Affiliation) WithdrawPendingJoinRequest(ctx context.Context, user IdentityUser) error {
	uid := strings.TrimSpace(user.ID)
	if uid == "" {
		return ErrAffiliationNotFound
	}
	pending, err := a.PendingJoinRequestForUser(ctx, uid)
	if err != nil {
		return err
	}
	if pending == nil {
		return ErrAffiliationNotFound
	}
	if pending.Status != AffiliationStatusPending {
		return ErrAffiliationNotPending
	}
	if err := a.store.DeleteJoinRequest(ctx, pending.ID); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return ErrAffiliationNotFound
		}
		return err
	}
	return nil
}

// WithdrawPendingOrganizationCreationRequest deletes the caller's pending org-creation request.
func (a *Affiliation) WithdrawPendingOrganizationCreationRequest(ctx context.Context, user IdentityUser) error {
	uid := strings.TrimSpace(user.ID)
	if uid == "" {
		return ErrAffiliationNotFound
	}
	pending, err := a.PendingOrganizationCreationRequestForUser(ctx, uid)
	if err != nil {
		return err
	}
	if pending == nil {
		return ErrAffiliationNotFound
	}
	if pending.Status != AffiliationStatusPending {
		return ErrAffiliationNotPending
	}
	if err := a.store.DeleteOrganizationCreationRequest(ctx, pending.ID); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return ErrAffiliationNotFound
		}
		return err
	}
	return nil
}

func (a *Affiliation) HasPendingAffiliationIntent(ctx context.Context, userID string) (bool, error) {
	join, err := a.PendingJoinRequestForUser(ctx, userID)
	if err != nil {
		return false, err
	}
	if join != nil {
		return true, nil
	}
	orgReq, err := a.PendingOrganizationCreationRequestForUser(ctx, userID)
	if err != nil {
		return false, err
	}
	return orgReq != nil, nil
}

// PendingInvite is an unconfirmed Appwrite team membership offered to the user.
type PendingInvite struct {
	MembershipID string
	OrgSlug      string
	OrgName      string
	RoleSlugs    []string
	IsOrgAdmin   bool
	InvitedAt    time.Time
}

func (a *Affiliation) ListPendingInvitesForUser(ctx context.Context, userID string) ([]PendingInvite, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" || a.identity == nil {
		return nil, nil
	}
	memberships, err := a.identity.ListUserMemberships(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]PendingInvite, 0)
	for _, membership := range memberships {
		if membership.Confirmed {
			continue
		}
		orgSlug := strings.TrimSpace(membership.OrgSlug)
		if orgSlug == "" {
			orgSlug = strings.TrimSpace(membership.TeamID)
		}
		if orgSlug == "" || strings.TrimSpace(membership.ID) == "" {
			continue
		}
		orgName := strings.TrimSpace(membership.OrgName)
		if orgName == "" {
			orgName = orgSlug
		}
		out = append(out, PendingInvite{
			MembershipID: strings.TrimSpace(membership.ID),
			OrgSlug:      orgSlug,
			OrgName:      orgName,
			RoleSlugs:    append([]string(nil), membership.RoleSlugs...),
			IsOrgAdmin:   membership.IsOrgAdmin,
			InvitedAt:    membership.InvitedAt,
		})
	}
	return out, nil
}

func (a *Affiliation) findPendingInvite(ctx context.Context, userID, membershipID string) (PendingInvite, error) {
	membershipID = strings.TrimSpace(membershipID)
	if membershipID == "" {
		return PendingInvite{}, ErrAffiliationNotFound
	}
	invites, err := a.ListPendingInvitesForUser(ctx, userID)
	if err != nil {
		return PendingInvite{}, err
	}
	for _, invite := range invites {
		if invite.MembershipID == membershipID {
			return invite, nil
		}
	}
	return PendingInvite{}, ErrAffiliationNotFound
}

// RejectPendingInvite deletes an unconfirmed invite membership for the current user.
func (a *Affiliation) RejectPendingInvite(ctx context.Context, user IdentityUser, membershipID string) error {
	invite, err := a.findPendingInvite(ctx, user.ID, membershipID)
	if err != nil {
		return err
	}
	return a.identity.DeleteOrganizationMembershipAsAdmin(ctx, invite.OrgSlug, invite.MembershipID)
}

func (a *Affiliation) SubmitJoinRequest(ctx context.Context, user IdentityUser, orgSlug string, roleSlugs []string) (JoinRequest, error) {
	orgSlug = strings.TrimSpace(orgSlug)
	if orgSlug == "" {
		return JoinRequest{}, ErrAffiliationNotFound
	}
	if a.IsAffiliated(user) {
		return JoinRequest{}, ErrAffiliationAlreadyAffiliated
	}
	pending, err := a.HasPendingAffiliationIntent(ctx, user.ID)
	if err != nil {
		return JoinRequest{}, err
	}
	if pending {
		return JoinRequest{}, ErrAffiliationPendingExists
	}

	org, err := a.loadOrganizationBySlug(ctx, orgSlug)
	if err != nil {
		return JoinRequest{}, err
	}
	roles, err := validateJoinRequestRoles(org, roleSlugs)
	if err != nil {
		return JoinRequest{}, err
	}

	saved, err := a.store.InsertJoinRequest(ctx, JoinRequest{
		RequesterUserID: strings.TrimSpace(user.ID),
		RequesterEmail:  strings.TrimSpace(user.Email),
		OrgSlug:         org.Slug,
		RoleSlugs:       roles,
		Status:          AffiliationStatusPending,
	})
	if err != nil {
		return JoinRequest{}, err
	}

	a.notifyJoinSubmitted(ctx, saved, org)
	return saved, nil
}

func (a *Affiliation) ListPendingJoinRequests(ctx context.Context, orgSlug string) ([]JoinRequest, error) {
	return a.store.ListPendingJoinRequestsByOrg(ctx, strings.TrimSpace(orgSlug))
}

func (a *Affiliation) ApproveJoinRequest(ctx context.Context, requestID primitive.ObjectID, decidedBy IdentityUser) (JoinRequest, error) {
	req, err := a.loadPendingJoinRequest(ctx, requestID)
	if err != nil {
		return JoinRequest{}, err
	}
	if err := ensureDeciderOrgMatches(decidedBy, req.OrgSlug); err != nil {
		return JoinRequest{}, err
	}

	requester, err := a.identity.GetUserByID(ctx, req.RequesterUserID)
	if err != nil {
		return JoinRequest{}, err
	}
	if a.IsAffiliated(requester) {
		return JoinRequest{}, ErrAffiliationAlreadyAffiliated
	}

	org, err := a.loadOrganizationBySlug(ctx, req.OrgSlug)
	if err != nil {
		return JoinRequest{}, err
	}
	roles, err := validateJoinRequestRoles(org, req.RoleSlugs)
	if err != nil {
		return JoinRequest{}, err
	}

	now := a.now().UTC()
	req.Status = AffiliationStatusApproved
	req.RoleSlugs = roles
	req.RejectReason = ""
	req.DecidedByUserID = strings.TrimSpace(decidedBy.ID)
	req.DecidedAt = now
	req.UpdatedAt = now
	updated, err := a.store.UpdateJoinRequest(ctx, req)
	if err != nil {
		return JoinRequest{}, err
	}

	if err := a.grantOrganizationMembership(ctx, org.Slug, req.RequesterUserID, roles, false); err != nil {
		_ = a.compensateJoinRequestToPending(ctx, updated)
		return JoinRequest{}, err
	}

	if email := strings.TrimSpace(updated.RequesterEmail); email != "" {
		a.notify(ctx, MailMessage{
			Kind:    MailKindJoinApproved,
			To:      []string{email},
			Subject: "Join request approved: " + org.Name,
			Body: fmt.Sprintf(
				"Your request to join organization %q (slug %q) was approved.\n\nOpen: %s",
				org.Name, org.Slug, a.actionURL("/my/organization"),
			),
		})
	}
	return updated, nil
}

func (a *Affiliation) RejectJoinRequest(ctx context.Context, requestID primitive.ObjectID, decidedBy IdentityUser, reason string) (JoinRequest, error) {
	req, err := a.loadPendingJoinRequest(ctx, requestID)
	if err != nil {
		return JoinRequest{}, err
	}
	if err := ensureDeciderOrgMatches(decidedBy, req.OrgSlug); err != nil {
		return JoinRequest{}, err
	}

	now := a.now().UTC()
	req.Status = AffiliationStatusRejected
	req.RejectReason = strings.TrimSpace(reason)
	req.DecidedByUserID = strings.TrimSpace(decidedBy.ID)
	req.DecidedAt = now
	req.UpdatedAt = now
	updated, err := a.store.UpdateJoinRequest(ctx, req)
	if err != nil {
		return JoinRequest{}, err
	}

	if email := strings.TrimSpace(updated.RequesterEmail); email != "" {
		body := fmt.Sprintf(
			"Your request to join organization %q was rejected.",
			updated.OrgSlug,
		)
		if updated.RejectReason != "" {
			body += " Reason: " + updated.RejectReason
		}
		body += "\n\nOpen: " + a.actionURL("/my/onboarding")
		a.notify(ctx, MailMessage{
			Kind:    MailKindJoinRejected,
			To:      []string{email},
			Subject: "Join request rejected: " + updated.OrgSlug,
			Body:    body,
		})
	}
	return updated, nil
}

func (a *Affiliation) SubmitOrganizationCreationRequest(ctx context.Context, user IdentityUser, name string) (OrganizationCreationRequest, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return OrganizationCreationRequest{}, ErrAffiliationInvalidName
	}
	proposedSlug := canonifySlug(name)
	if proposedSlug == "" {
		return OrganizationCreationRequest{}, ErrAffiliationInvalidName
	}
	if a.IsAffiliated(user) {
		return OrganizationCreationRequest{}, ErrAffiliationAlreadyAffiliated
	}
	pending, err := a.HasPendingAffiliationIntent(ctx, user.ID)
	if err != nil {
		return OrganizationCreationRequest{}, err
	}
	if pending {
		return OrganizationCreationRequest{}, ErrAffiliationPendingExists
	}
	if err := a.ensureOrganizationSlugAvailable(ctx, proposedSlug); err != nil {
		return OrganizationCreationRequest{}, err
	}

	saved, err := a.store.InsertOrganizationCreationRequest(ctx, OrganizationCreationRequest{
		RequesterUserID: strings.TrimSpace(user.ID),
		RequesterEmail:  strings.TrimSpace(user.Email),
		ProposedName:    name,
		ProposedSlug:    proposedSlug,
		Status:          AffiliationStatusPending,
	})
	if err != nil {
		return OrganizationCreationRequest{}, err
	}

	if len(a.platformAdminNotifyEmails) > 0 {
		a.notify(ctx, MailMessage{
			Kind:    MailKindOrgCreationSubmitted,
			To:      append([]string(nil), a.platformAdminNotifyEmails...),
			Subject: "Organization creation request: " + name,
			Body: fmt.Sprintf(
				"Requester %s submitted an organization creation request for %q (slug %q).\n\nOpen: %s",
				strings.TrimSpace(user.Email), name, proposedSlug, a.actionURL("/admin/organizations"),
			),
		})
	}
	return saved, nil
}

func (a *Affiliation) ListPendingOrganizationCreationRequests(ctx context.Context) ([]OrganizationCreationRequest, error) {
	return a.store.ListPendingOrganizationCreationRequests(ctx)
}

func (a *Affiliation) ApproveOrganizationCreationRequest(ctx context.Context, requestID primitive.ObjectID, decidedBy IdentityUser) (OrganizationCreationRequest, IdentityOrg, error) {
	req, err := a.loadPendingOrganizationCreationRequest(ctx, requestID)
	if err != nil {
		return OrganizationCreationRequest{}, IdentityOrg{}, err
	}

	requester, err := a.identity.GetUserByID(ctx, req.RequesterUserID)
	if err != nil {
		return OrganizationCreationRequest{}, IdentityOrg{}, err
	}
	if a.IsAffiliated(requester) {
		return OrganizationCreationRequest{}, IdentityOrg{}, ErrAffiliationAlreadyAffiliated
	}
	if err := a.ensureOrganizationSlugAvailable(ctx, req.ProposedSlug); err != nil {
		return OrganizationCreationRequest{}, IdentityOrg{}, err
	}

	now := a.now().UTC()
	req.Status = AffiliationStatusApproved
	req.RejectReason = ""
	req.DecidedByUserID = strings.TrimSpace(decidedBy.ID)
	req.DecidedAt = now
	req.UpdatedAt = now
	updated, err := a.store.UpdateOrganizationCreationRequest(ctx, req)
	if err != nil {
		return OrganizationCreationRequest{}, IdentityOrg{}, err
	}

	org, err := a.identity.CreateOrganizationAsAdmin(ctx, req.ProposedName)
	if err != nil {
		_ = a.compensateOrganizationCreationRequestToPending(ctx, updated)
		return OrganizationCreationRequest{}, IdentityOrg{}, err
	}
	if err := a.grantOrganizationMembership(ctx, org.Slug, req.RequesterUserID, nil, true); err != nil {
		a.compensateOrganizationCreationAfterIdentity(ctx, updated, org)
		return OrganizationCreationRequest{}, IdentityOrg{}, err
	}

	if email := strings.TrimSpace(updated.RequesterEmail); email != "" {
		a.notify(ctx, MailMessage{
			Kind:    MailKindOrgCreationApproved,
			To:      []string{email},
			Subject: "Organization creation approved: " + updated.ProposedName,
			Body: fmt.Sprintf(
				"Your request to create organization %q (slug %q) was approved.\n\nOpen: %s",
				updated.ProposedName, updated.ProposedSlug, a.actionURL("/my/organization"),
			),
		})
	}
	return updated, org, nil
}

func (a *Affiliation) RejectOrganizationCreationRequest(ctx context.Context, requestID primitive.ObjectID, decidedBy IdentityUser, reason string) (OrganizationCreationRequest, error) {
	req, err := a.loadPendingOrganizationCreationRequest(ctx, requestID)
	if err != nil {
		return OrganizationCreationRequest{}, err
	}

	now := a.now().UTC()
	req.Status = AffiliationStatusRejected
	req.RejectReason = strings.TrimSpace(reason)
	req.DecidedByUserID = strings.TrimSpace(decidedBy.ID)
	req.DecidedAt = now
	req.UpdatedAt = now
	updated, err := a.store.UpdateOrganizationCreationRequest(ctx, req)
	if err != nil {
		return OrganizationCreationRequest{}, err
	}

	if email := strings.TrimSpace(updated.RequesterEmail); email != "" {
		body := fmt.Sprintf(
			"Your request to create organization %q (slug %q) was rejected.",
			updated.ProposedName, updated.ProposedSlug,
		)
		if updated.RejectReason != "" {
			body += " Reason: " + updated.RejectReason
		}
		body += "\n\nOpen: " + a.actionURL("/my/onboarding")
		a.notify(ctx, MailMessage{
			Kind:    MailKindOrgCreationRejected,
			To:      []string{email},
			Subject: "Organization creation rejected: " + updated.ProposedName,
			Body:    body,
		})
	}
	return updated, nil
}

func (a *Affiliation) compensateJoinRequestToPending(ctx context.Context, req JoinRequest) error {
	req.Status, req.RejectReason, req.DecidedByUserID, req.DecidedAt, req.UpdatedAt = clearAffiliationDecision(a.now().UTC())
	_, err := a.store.UpdateJoinRequest(ctx, req)
	return err
}

func (a *Affiliation) compensateOrganizationCreationRequestToPending(ctx context.Context, req OrganizationCreationRequest) error {
	req.Status, req.RejectReason, req.DecidedByUserID, req.DecidedAt, req.UpdatedAt = clearAffiliationDecision(a.now().UTC())
	_, err := a.store.UpdateOrganizationCreationRequest(ctx, req)
	return err
}

// compensateOrganizationCreationAfterIdentity reverts the request to pending and best-effort
// deletes the org created during approve. If DeleteOrganizationAsAdmin fails or is unavailable,
// status is still reverted so the request can be retried after manual cleanup.
func (a *Affiliation) compensateOrganizationCreationAfterIdentity(ctx context.Context, req OrganizationCreationRequest, org IdentityOrg) {
	_ = a.compensateOrganizationCreationRequestToPending(ctx, req)
	if slug := strings.TrimSpace(org.Slug); slug != "" {
		_ = a.identity.DeleteOrganizationAsAdmin(ctx, slug)
	}
}

func clearAffiliationDecision(now time.Time) (status AffiliationRequestStatus, rejectReason, decidedByUserID string, decidedAt, updatedAt time.Time) {
	return AffiliationStatusPending, "", "", time.Time{}, now
}

func mapPendingAffiliationLoad(err error, found bool, status AffiliationRequestStatus) error {
	switch {
	case err == nil && found:
		if status != AffiliationStatusPending {
			return ErrAffiliationNotPending
		}
		return nil
	case errors.Is(err, mongo.ErrNoDocuments), err == nil && !found:
		return ErrAffiliationNotFound
	default:
		return err
	}
}

func (a *Affiliation) loadPendingOrganizationCreationRequest(ctx context.Context, requestID primitive.ObjectID) (OrganizationCreationRequest, error) {
	req, err := a.store.LoadOrganizationCreationRequestByID(ctx, requestID)
	var status AffiliationRequestStatus
	if req != nil {
		status = req.Status
	}
	if mapErr := mapPendingAffiliationLoad(err, req != nil, status); mapErr != nil {
		return OrganizationCreationRequest{}, mapErr
	}
	return *req, nil
}

func (a *Affiliation) loadPendingJoinRequest(ctx context.Context, requestID primitive.ObjectID) (JoinRequest, error) {
	req, err := a.store.LoadJoinRequestByID(ctx, requestID)
	var status AffiliationRequestStatus
	if req != nil {
		status = req.Status
	}
	if mapErr := mapPendingAffiliationLoad(err, req != nil, status); mapErr != nil {
		return JoinRequest{}, mapErr
	}
	return *req, nil
}

func (a *Affiliation) loadOrganizationBySlug(ctx context.Context, slug string) (IdentityOrg, error) {
	org, err := a.identity.GetOrganizationBySlug(ctx, strings.TrimSpace(slug))
	switch {
	case err == nil && org != nil:
		return *org, nil
	case errors.Is(err, ErrIdentityNotFound), err == nil && org == nil:
		return IdentityOrg{}, ErrAffiliationNotFound
	default:
		return IdentityOrg{}, err
	}
}

func (a *Affiliation) ensureOrganizationSlugAvailable(ctx context.Context, slug string) error {
	existing, err := a.identity.GetOrganizationBySlug(ctx, slug)
	switch {
	case err == nil && existing != nil:
		return ErrAffiliationOrganizationSlugExists
	case errors.Is(err, ErrIdentityNotFound), err == nil && existing == nil:
		return nil
	default:
		return err
	}
}

func ensureDeciderOrgMatches(decidedBy IdentityUser, orgSlug string) error {
	decidedOrg := strings.TrimSpace(decidedBy.OrgSlug)
	if decidedOrg == "" {
		return nil
	}
	if !strings.EqualFold(decidedOrg, strings.TrimSpace(orgSlug)) {
		return ErrAffiliationNotFound
	}
	return nil
}

// RequestableJoinRoles returns Organization roles that may be named on a Join request.
// Org admin is membership standing, not an Organization role, and is never requestable here.
func (a *Affiliation) RequestableJoinRoles(org IdentityOrg) []Role {
	return requestableJoinRoles(org)
}

func requestableJoinRoles(org IdentityOrg) []Role {
	roles := rolesFromIdentityOrg(org)
	out := make([]Role, 0, len(roles))
	for _, role := range roles {
		if containsRole([]string{role.Slug}, "org-admin") || containsRole([]string{role.Slug}, "org_admin") {
			continue
		}
		out = append(out, role)
	}
	return out
}

func validateJoinRequestRoles(org IdentityOrg, roleSlugs []string) ([]string, error) {
	roles := canonifyRoleSlugs(roleSlugs)
	if len(roles) == 0 {
		return nil, ErrAffiliationInvalidRoles
	}
	allowed := make(map[string]struct{}, len(org.Roles))
	for _, role := range requestableJoinRoles(org) {
		allowed[canonifySlug(role.Slug)] = struct{}{}
	}
	for _, slug := range roles {
		if _, ok := allowed[slug]; !ok {
			return nil, ErrAffiliationInvalidRoles
		}
	}
	return roles, nil
}

// grantOrganizationMembership adds the user to the organization and stamps managed labels
// (role labels and/or org-admin). On stamp failure it best-effort deletes the membership
// just created and returns the stamp error.
func (a *Affiliation) grantOrganizationMembership(ctx context.Context, orgSlug, userID string, roles []string, isOrgAdmin bool) error {
	orgSlug = strings.TrimSpace(orgSlug)
	userID = strings.TrimSpace(userID)
	membership, err := a.identity.AddOrganizationUserByIDAsAdmin(ctx, orgSlug, userID, roles, isOrgAdmin)
	if err != nil {
		return err
	}
	if err := a.stampManagedMembershipLabels(ctx, userID, roles, isOrgAdmin); err != nil {
		if id := strings.TrimSpace(membership.ID); id != "" {
			_ = a.identity.DeleteOrganizationMembershipAsAdmin(ctx, orgSlug, id)
		}
		return err
	}
	return nil
}

func (a *Affiliation) stampManagedMembershipLabels(ctx context.Context, userID string, roleSlugs []string, isOrgAdmin bool) error {
	user, err := a.identity.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	labels := make([]string, 0, len(user.Labels)+len(roleSlugs)+1)
	for _, label := range user.Labels {
		if isManagedIdentityLabel(label) {
			continue
		}
		labels = append(labels, strings.TrimSpace(label))
	}
	for _, roleSlug := range roleSlugs {
		labels = append(labels, encodeIdentityRoleLabel(roleSlug))
	}
	if isOrgAdmin {
		labels = append(labels, identityOrgAdminLabel)
	}
	_, err = a.identity.UpdateUserLabels(ctx, userID, uniqueIdentityStrings(labels))
	return err
}

// stripManagedIdentityLabels removes role and org-admin labels from the user, leaving
// any non-managed labels in place. Missing users are treated as already cleaned.
func (a *Affiliation) stripManagedIdentityLabels(ctx context.Context, userID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil
	}
	targetUser, getErr := a.identity.GetUserByID(ctx, userID)
	if getErr != nil {
		if errors.Is(getErr, ErrIdentityNotFound) {
			return nil
		}
		return getErr
	}
	labels := make([]string, 0, len(targetUser.Labels))
	for _, label := range targetUser.Labels {
		if isManagedIdentityLabel(label) {
			continue
		}
		labels = append(labels, strings.TrimSpace(label))
	}
	_, err := a.identity.UpdateUserLabels(ctx, userID, labels)
	return err
}

func (a *Affiliation) notifyJoinSubmitted(ctx context.Context, req JoinRequest, org IdentityOrg) {
	users, err := a.identity.ListOrganizationUsers(ctx, org.Slug)
	if err != nil {
		log.Printf("affiliation mail %s: list org admins for %s failed: %v", MailKindJoinSubmitted, org.Slug, err)
		return
	}
	to := make([]string, 0)
	seen := map[string]struct{}{}
	for _, user := range users {
		if !user.IsOrgAdmin {
			continue
		}
		email := strings.TrimSpace(user.Email)
		if email == "" {
			continue
		}
		key := strings.ToLower(email)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		to = append(to, email)
	}
	if len(to) == 0 {
		log.Printf("affiliation mail %s: no org admins for %s", MailKindJoinSubmitted, org.Slug)
		return
	}
	a.notify(ctx, MailMessage{
		Kind:    MailKindJoinSubmitted,
		To:      to,
		Subject: "Join request: " + org.Name,
		Body: fmt.Sprintf(
			"Requester %s submitted a join request for organization %q (slug %q) with roles %v.\n\nOpen: %s",
			strings.TrimSpace(req.RequesterEmail), org.Name, org.Slug, req.RoleSlugs, a.actionURL("/my/organization/members"),
		),
	})
}

func (a *Affiliation) actionURL(path string) string {
	if a.mailer == nil {
		return mailAbsoluteURL("", path)
	}
	return a.mailer.AbsoluteURL(path)
}

func (a *Affiliation) notify(ctx context.Context, msg MailMessage) {
	if a.mailer == nil {
		return
	}
	if err := a.mailer.Send(ctx, msg); err != nil {
		log.Printf("affiliation mail %s failed: %v", msg.Kind, err)
	}
}
