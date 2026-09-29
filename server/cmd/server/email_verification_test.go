package main

import (
	"context"
	"errors"
	"testing"
)

func TestEmailVerificationIsVerified(t *testing.T) {
	ev := NewEmailVerification(&fakeIdentityStore{})

	cases := []struct {
		name string
		user IdentityUser
		want bool
	}{
		{name: "unverified", user: IdentityUser{ID: "u1"}, want: false},
		{name: "verified", user: IdentityUser{ID: "u1", EmailVerified: true}, want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ev.IsVerified(tc.user); got != tc.want {
				t.Fatalf("IsVerified = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestEmailVerificationAllowsAppAccess(t *testing.T) {
	ev := NewEmailVerification(&fakeIdentityStore{})
	verified := true
	unverified := false

	cases := []struct {
		name string
		user AccountUser
		want bool
	}{
		{name: "platform admin unverified", user: AccountUser{IsPlatformAdmin: true, EmailVerified: &unverified}, want: true},
		{name: "verified", user: AccountUser{EmailVerified: &verified}, want: true},
		{name: "unverified", user: AccountUser{EmailVerified: &unverified}, want: false},
		{name: "nil email verified defaults true", user: AccountUser{}, want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ev.AllowsAppAccess(tc.user); got != tc.want {
				t.Fatalf("AllowsAppAccess = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestEmailVerificationStart(t *testing.T) {
	var gotSecret, gotRedirect string
	called := false
	identity := &fakeIdentityStore{
		createEmailVerificationFunc: func(_ context.Context, sessionSecret, redirectURL string) error {
			called = true
			gotSecret = sessionSecret
			gotRedirect = redirectURL
			return nil
		},
	}
	ev := NewEmailVerification(identity)
	if err := ev.Start(context.Background(), "sess-1", "http://attesta.local/verify"); err != nil {
		t.Fatalf("Start error: %v", err)
	}
	if !called {
		t.Fatal("Start did not call CreateEmailVerification")
	}
	if gotSecret != "sess-1" || gotRedirect != "http://attesta.local/verify" {
		t.Fatalf("CreateEmailVerification args = %q %q", gotSecret, gotRedirect)
	}
}

func TestEmailVerificationComplete(t *testing.T) {
	cases := []struct {
		name            string
		emailVerified   bool
		wantCompleteCall bool
		getUserErr      error
		completeErr     error
		wantErr         error
	}{
		{
			name:             "already verified skips complete",
			emailVerified:    true,
			wantCompleteCall: false,
		},
		{
			name:             "not verified calls complete",
			emailVerified:    false,
			wantCompleteCall: true,
		},
		{
			name:        "get user error",
			getUserErr:  ErrIdentityNotFound,
			wantErr:     ErrIdentityNotFound,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			completeCalled := false
			identity := &fakeIdentityStore{
				getUserByIDFunc: func(_ context.Context, userID string) (IdentityUser, error) {
					if tc.getUserErr != nil {
						return IdentityUser{}, tc.getUserErr
					}
					return IdentityUser{ID: userID, EmailVerified: tc.emailVerified}, nil
				},
				completeEmailVerificationFunc: func(_ context.Context, userID, secret string) error {
					completeCalled = true
					if userID != "user-1" || secret != "secret-1" {
						t.Fatalf("CompleteEmailVerification args = %q %q", userID, secret)
					}
					return tc.completeErr
				},
			}
			err := NewEmailVerification(identity).Complete(context.Background(), "user-1", "secret-1")
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("Complete error = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Complete error: %v", err)
			}
			if completeCalled != tc.wantCompleteCall {
				t.Fatalf("CompleteEmailVerification called = %v, want %v", completeCalled, tc.wantCompleteCall)
			}
		})
	}
}

func TestEmailVerificationEstablishFromMailboxProof(t *testing.T) {
	var gotUserID string
	var gotVerified bool
	called := false
	identity := &fakeIdentityStore{
		updateEmailVerificationFunc: func(_ context.Context, userID string, verified bool) error {
			called = true
			gotUserID = userID
			gotVerified = verified
			return nil
		},
	}
	if err := NewEmailVerification(identity).EstablishFromMailboxProof(context.Background(), "user-9"); err != nil {
		t.Fatalf("EstablishFromMailboxProof error: %v", err)
	}
	if !called {
		t.Fatal("EstablishFromMailboxProof did not call UpdateEmailVerification")
	}
	if gotUserID != "user-9" || !gotVerified {
		t.Fatalf("UpdateEmailVerification args = %q %v", gotUserID, gotVerified)
	}
}
