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

// Domain errors for affiliation submit/approve/reject flows.
var (
	ErrAffiliationAlreadyAffiliated      = errors.New("affiliation: already affiliated")
	ErrAffiliationPendingExists          = errors.New("affiliation: pending request exists")
	ErrAffiliationNotFound               = errors.New("affiliation: request not found")
	ErrAffiliationNotPending             = errors.New("affiliation: request is not pending")
	ErrAffiliationInvalidName            = errors.New("affiliation: organization name is required")
	ErrAffiliationInvalidRoles           = errors.New("affiliation: invalid roles")
	ErrAffiliationOrganizationSlugExists = errors.New("affiliation: organization slug already exists")
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

// Affiliation owns affiliation-domain queries and join / organization-creation commands.
// Leave remains for a later ticket.
type Affiliation struct {
	identity IdentityStore
	store    Store
	mailer   Mailer
	now      func() time.Time
}

func NewAffiliation(identity IdentityStore, store Store, mailer Mailer, now func() time.Time) *Affiliation {
	if mailer == nil {
		mailer = noopMailer{}
	}
	if now == nil {
		now = time.Now
	}
	return &Affiliation{
		identity: identity,
		store:    store,
		mailer:   mailer,
		now:      now,
	}
}

func (a *Affiliation) IsAffiliated(user IdentityUser) bool {
	return strings.TrimSpace(user.OrgSlug) != ""
}

func (a *Affiliation) PendingJoinRequestForUser(ctx context.Context, userID string) (*JoinRequest, error) {
	return a.store.FindPendingJoinRequestByUser(ctx, userID)
}

func (a *Affiliation) PendingOrganizationCreationRequestForUser(ctx context.Context, userID string) (*OrganizationCreationRequest, error) {
	return a.store.FindPendingOrganizationCreationRequestByUser(ctx, userID)
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

func (a *Affiliation) SaveJoinRequest(ctx context.Context, req JoinRequest) (JoinRequest, error) {
	return a.store.InsertJoinRequest(ctx, req)
}

func (a *Affiliation) LoadJoinRequestByID(ctx context.Context, id primitive.ObjectID) (*JoinRequest, error) {
	return a.store.LoadJoinRequestByID(ctx, id)
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

	saved, err := a.SaveJoinRequest(ctx, JoinRequest{
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
	all, err := a.store.ListJoinRequestsByOrg(ctx, strings.TrimSpace(orgSlug))
	if err != nil {
		return nil, err
	}
	pending := make([]JoinRequest, 0, len(all))
	for _, req := range all {
		if req.Status == AffiliationStatusPending {
			pending = append(pending, req)
		}
	}
	return pending, nil
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

	if _, err := a.identity.AddOrganizationUserByIDAsAdmin(ctx, org.Slug, req.RequesterUserID, roles, false); err != nil {
		return JoinRequest{}, err
	}
	if err := a.stampJoinRequestRoleLabels(ctx, req.RequesterUserID, roles); err != nil {
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

	if email := strings.TrimSpace(updated.RequesterEmail); email != "" {
		a.notify(ctx, MailMessage{
			Kind:    MailKindJoinApproved,
			To:      []string{email},
			Subject: "Join request approved: " + org.Name,
			Body: fmt.Sprintf(
				"Your request to join organization %q (slug %q) was approved.",
				org.Name, org.Slug,
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
		a.notify(ctx, MailMessage{
			Kind:    MailKindJoinRejected,
			To:      []string{email},
			Subject: "Join request rejected: " + updated.OrgSlug,
			Body:    body,
		})
	}
	return updated, nil
}

func (a *Affiliation) SaveOrganizationCreationRequest(ctx context.Context, req OrganizationCreationRequest) (OrganizationCreationRequest, error) {
	return a.store.InsertOrganizationCreationRequest(ctx, req)
}

func (a *Affiliation) LoadOrganizationCreationRequestByID(ctx context.Context, id primitive.ObjectID) (*OrganizationCreationRequest, error) {
	return a.store.LoadOrganizationCreationRequestByID(ctx, id)
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

	saved, err := a.SaveOrganizationCreationRequest(ctx, OrganizationCreationRequest{
		RequesterUserID: strings.TrimSpace(user.ID),
		RequesterEmail:  strings.TrimSpace(user.Email),
		ProposedName:    name,
		ProposedSlug:    proposedSlug,
		Status:          AffiliationStatusPending,
	})
	if err != nil {
		return OrganizationCreationRequest{}, err
	}

	if adminEmail, _, ok := platformAdminCredentials(); ok {
		a.notify(ctx, MailMessage{
			Kind:    MailKindOrgCreationSubmitted,
			To:      []string{adminEmail},
			Subject: "Organization creation request: " + name,
			Body: fmt.Sprintf(
				"Requester %s submitted an organization creation request for %q (slug %q).",
				strings.TrimSpace(user.Email), name, proposedSlug,
			),
		})
	}
	return saved, nil
}

func (a *Affiliation) ListPendingOrganizationCreationRequests(ctx context.Context) ([]OrganizationCreationRequest, error) {
	all, err := a.store.ListOrganizationCreationRequests(ctx)
	if err != nil {
		return nil, err
	}
	pending := make([]OrganizationCreationRequest, 0, len(all))
	for _, req := range all {
		if req.Status == AffiliationStatusPending {
			pending = append(pending, req)
		}
	}
	return pending, nil
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

	org, err := a.identity.CreateOrganizationAsAdmin(ctx, req.ProposedName)
	if err != nil {
		return OrganizationCreationRequest{}, IdentityOrg{}, err
	}
	if _, err := a.identity.AddOrganizationUserByIDAsAdmin(ctx, org.Slug, req.RequesterUserID, nil, true); err != nil {
		return OrganizationCreationRequest{}, IdentityOrg{}, err
	}
	if err := a.stampOrgAdminLabel(ctx, req.RequesterUserID); err != nil {
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

	if email := strings.TrimSpace(updated.RequesterEmail); email != "" {
		a.notify(ctx, MailMessage{
			Kind:    MailKindOrgCreationApproved,
			To:      []string{email},
			Subject: "Organization creation approved: " + updated.ProposedName,
			Body: fmt.Sprintf(
				"Your request to create organization %q (slug %q) was approved.",
				updated.ProposedName, updated.ProposedSlug,
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
		a.notify(ctx, MailMessage{
			Kind:    MailKindOrgCreationRejected,
			To:      []string{email},
			Subject: "Organization creation rejected: " + updated.ProposedName,
			Body:    body,
		})
	}
	return updated, nil
}

func (a *Affiliation) loadPendingOrganizationCreationRequest(ctx context.Context, requestID primitive.ObjectID) (OrganizationCreationRequest, error) {
	req, err := a.store.LoadOrganizationCreationRequestByID(ctx, requestID)
	switch {
	case err == nil && req != nil:
		if req.Status != AffiliationStatusPending {
			return OrganizationCreationRequest{}, ErrAffiliationNotPending
		}
		return *req, nil
	case errors.Is(err, mongo.ErrNoDocuments), err == nil && req == nil:
		return OrganizationCreationRequest{}, ErrAffiliationNotFound
	default:
		return OrganizationCreationRequest{}, err
	}
}

func (a *Affiliation) loadPendingJoinRequest(ctx context.Context, requestID primitive.ObjectID) (JoinRequest, error) {
	req, err := a.store.LoadJoinRequestByID(ctx, requestID)
	switch {
	case err == nil && req != nil:
		if req.Status != AffiliationStatusPending {
			return JoinRequest{}, ErrAffiliationNotPending
		}
		return *req, nil
	case errors.Is(err, mongo.ErrNoDocuments), err == nil && req == nil:
		return JoinRequest{}, ErrAffiliationNotFound
	default:
		return JoinRequest{}, err
	}
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

func validateJoinRequestRoles(org IdentityOrg, roleSlugs []string) ([]string, error) {
	roles := canonifyRoleSlugs(roleSlugs)
	if len(roles) == 0 {
		return nil, ErrAffiliationInvalidRoles
	}
	for _, slug := range roles {
		if containsRole([]string{slug}, "org-admin") || containsRole([]string{slug}, "org_admin") {
			return nil, ErrAffiliationInvalidRoles
		}
		if !identityOrgHasRole(org, slug) {
			return nil, ErrAffiliationInvalidRoles
		}
	}
	return roles, nil
}

func (a *Affiliation) stampOrgAdminLabel(ctx context.Context, userID string) error {
	user, err := a.identity.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	labels := make([]string, 0, len(user.Labels)+1)
	for _, label := range user.Labels {
		if strings.EqualFold(strings.TrimSpace(label), identityOrgAdminLabel) {
			continue
		}
		labels = append(labels, strings.TrimSpace(label))
	}
	labels = append(labels, identityOrgAdminLabel)
	_, err = a.identity.UpdateUserLabels(ctx, userID, uniqueIdentityStrings(labels))
	return err
}

func (a *Affiliation) stampJoinRequestRoleLabels(ctx context.Context, userID string, roleSlugs []string) error {
	user, err := a.identity.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	labels := make([]string, 0, len(user.Labels)+len(roleSlugs))
	for _, label := range user.Labels {
		if isManagedIdentityLabel(label) {
			continue
		}
		labels = append(labels, strings.TrimSpace(label))
	}
	for _, roleSlug := range roleSlugs {
		labels = append(labels, encodeIdentityRoleLabel(roleSlug))
	}
	_, err = a.identity.UpdateUserLabels(ctx, userID, uniqueIdentityStrings(labels))
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
			"Requester %s submitted a join request for organization %q (slug %q) with roles %v.",
			strings.TrimSpace(req.RequesterEmail), org.Name, org.Slug, req.RoleSlugs,
		),
	})
}

func (a *Affiliation) notify(ctx context.Context, msg MailMessage) {
	if a.mailer == nil {
		return
	}
	if err := a.mailer.Send(ctx, msg); err != nil {
		log.Printf("affiliation mail %s failed: %v", msg.Kind, err)
	}
}
