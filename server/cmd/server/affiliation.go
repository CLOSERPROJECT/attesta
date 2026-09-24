package main

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Domain errors reserved for submit/approve/reject flows in later tickets.
var (
	ErrAffiliationAlreadyAffiliated = errors.New("affiliation: already affiliated")
	ErrAffiliationPendingExists     = errors.New("affiliation: pending request exists")
	ErrAffiliationNotFound          = errors.New("affiliation: request not found")
	ErrAffiliationNotPending        = errors.New("affiliation: request is not pending")
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

// Affiliation owns affiliation-domain queries and persistence wrappers.
// Submit/approve/reject/leave/list-for-admin are intentionally not implemented yet.
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

func (a *Affiliation) SaveOrganizationCreationRequest(ctx context.Context, req OrganizationCreationRequest) (OrganizationCreationRequest, error) {
	return a.store.InsertOrganizationCreationRequest(ctx, req)
}

func (a *Affiliation) LoadOrganizationCreationRequestByID(ctx context.Context, id primitive.ObjectID) (*OrganizationCreationRequest, error) {
	return a.store.LoadOrganizationCreationRequestByID(ctx, id)
}
