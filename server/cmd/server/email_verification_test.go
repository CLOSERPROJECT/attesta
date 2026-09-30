package main

import (
	"context"
	"errors"
	"testing"
)

func TestEmailVerifiedSuccessMessage(t *testing.T) {
	if got := emailVerifiedSuccessMessage(noticeEmailVerified); got != "Your email is verified." {
		t.Fatalf("emailVerifiedSuccessMessage = %q", got)
	}
	if got := emailVerifiedSuccessMessage("other"); got != "" {
		t.Fatalf("unexpected message %q", got)
	}
}

func TestEmailVerificationAllowsAppAccess(t *testing.T) {
	ev := NewEmailVerification(&fakeIdentityStore{})

	cases := []struct {
		name string
		user AccountUser
		want bool
	}{
		{name: "platform admin unverified", user: AccountUser{IsPlatformAdmin: true, EmailVerified: false}, want: true},
		{name: "platform admin default", user: AccountUser{IsPlatformAdmin: true}, want: true},
		{name: "verified", user: AccountUser{EmailVerified: true}, want: true},
		{name: "unverified", user: AccountUser{EmailVerified: false}, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ev.AllowsAppAccess(tc.user); got != tc.want {
				t.Fatalf("AllowsAppAccess = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestEmailVerificationComplete(t *testing.T) {
	cases := []struct {
		name             string
		emailVerified    bool
		wantCompleteCall bool
		getUserErr       error
		completeErr      error
		wantErr          error
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
			name:       "get user error",
			getUserErr: ErrIdentityNotFound,
			wantErr:    ErrIdentityNotFound,
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
