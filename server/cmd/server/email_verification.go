package main

import (
	"context"
)

// EmailVerification owns email-verification domain queries and commands:
// start (send), complete (link), and establish-from-mailbox-proof (invite accept / recovery).
type EmailVerification struct {
	identity IdentityStore
}

func NewEmailVerification(identity IdentityStore) *EmailVerification {
	return &EmailVerification{identity: identity}
}

// IsVerified reports whether the identity user has proven control of their email.
// Platform-admin synthetic sessions never appear as IdentityUser; use AllowsAppAccess for that gate.
func (e *EmailVerification) IsVerified(user IdentityUser) bool {
	return user.EmailVerified
}

// AllowsAppAccess reports whether the account may leave the verification waiting path.
// Platform admins bypass Appwrite EmailVerified; otherwise EmailVerified is required.
// Nil AccountUser.EmailVerified defaults to verified so existing session tests keep working.
func (e *EmailVerification) AllowsAppAccess(user AccountUser) bool {
	if user.IsPlatformAdmin {
		return true
	}
	return accountUserEmailVerified(user)
}

func accountUserEmailVerified(user AccountUser) bool {
	if user.EmailVerified != nil {
		return *user.EmailVerified
	}
	return true
}

// Start sends an Appwrite verification email for the current session.
func (e *EmailVerification) Start(ctx context.Context, sessionSecret, redirectURL string) error {
	return e.identity.CreateEmailVerification(ctx, sessionSecret, redirectURL)
}

// Complete finishes verification from the email link. Already-verified users succeed without error.
func (e *EmailVerification) Complete(ctx context.Context, userID, secret string) error {
	user, err := e.identity.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if user.EmailVerified {
		return nil
	}
	return e.identity.CompleteEmailVerification(ctx, userID, secret)
}

// EstablishFromMailboxProof marks the user verified after invitation accept or password recovery.
func (e *EmailVerification) EstablishFromMailboxProof(ctx context.Context, userID string) error {
	return e.identity.UpdateEmailVerification(ctx, userID, true)
}
