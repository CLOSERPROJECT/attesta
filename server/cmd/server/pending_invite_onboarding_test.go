package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/appwrite/sdk-for-go/models"
)

// Feedback loop for: invitee with only an unconfirmed Appwrite membership
// visits /my and must be sent to onboarding (not treated as affiliated).
func TestHandleHomePendingInviteRedirectsToOnboarding(t *testing.T) {
	pendingIdentity := toIdentityUser(
		&models.User{Id: "user-pending", Email: "pending@example.com", Status: true},
		[]models.Membership{{
			Id:       "membership-pending",
			TeamId:   "acme-team",
			TeamName: "Acme Org",
			Confirm:  false,
			Roles:    []string{"owner", "imember"},
		}},
	)

	server := &Server{
		store: NewMemoryStore(),
		tmpl:  parseTestTemplates(t),
		identity: &fakeIdentityStore{
			getSessionFunc: func(ctx context.Context, sessionSecret string) (IdentitySession, error) {
				return IdentitySession{Secret: sessionSecret, ExpiresAt: time.Now().UTC().Add(time.Hour), UserID: "user-pending"}, nil
			},
			getCurrentUserFunc: func(ctx context.Context, sessionSecret string) (IdentityUser, error) {
				return pendingIdentity, nil
			},
		},
		enforceAuth: true,
	}

	req := httptest.NewRequest(http.MethodGet, "/my", nil)
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "session-pending-invite"})
	rec := httptest.NewRecorder()
	server.handleHome(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status=%d body=%q OrgSlug=%q Status=%q IsOrgAdmin=%v (pending invite must not look affiliated)",
			rec.Code, rec.Body.String(), pendingIdentity.OrgSlug, pendingIdentity.Status, pendingIdentity.IsOrgAdmin)
	}
	if loc := rec.Header().Get("Location"); loc != onboardingPath() {
		t.Fatalf("location=%q, want %q (OrgSlug=%q Status=%q)", loc, onboardingPath(), pendingIdentity.OrgSlug, pendingIdentity.Status)
	}
}
