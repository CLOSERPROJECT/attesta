package main

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func (s *Server) handleEmailVerificationWaiting(w http.ResponseWriter, r *http.Request) {
	user, session, ok := s.requireAuthenticatedPage(w, r)
	if !ok {
		return
	}
	if s.emailVerificationService().AllowsAppAccess(*user) {
		http.Redirect(w, r, s.postVerificationAppPath(r, user), http.StatusSeeOther)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.renderEmailVerificationWaiting(w, r, user, emailVerificationNoticeMessage(requestNotice(r)), "")
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			logAndHTTPError(w, r, http.StatusBadRequest, "invalid form", err, "failed to parse email verification resend form")
			return
		}
		if session == nil || strings.TrimSpace(session.Secret) == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if s.emailVerificationResendTooSoon(r) {
			s.renderEmailVerificationWaiting(w, r, user, "", emailVerificationNoticeMessage(noticeVerificationResendWait))
			return
		}
		if err := s.emailVerificationService().Start(r.Context(), session.Secret, verifyRedirectURL(r)); err != nil {
			logRequestError(r, err, "failed to resend email verification for %s", user.Email)
			s.renderEmailVerificationWaiting(w, r, user, "", emailVerificationNoticeMessage(noticeVerificationSendFailed))
			return
		}
		s.setEmailVerificationResendCookie(w, r)
		http.Redirect(w, r, emailVerificationPath()+"?notice="+url.QueryEscape(noticeVerificationSent), http.StatusSeeOther)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleEmailVerificationConfirm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.identity == nil {
		http.NotFound(w, r)
		return
	}
	userID, secret := emailVerificationConfirmParams(r)
	if userID == "" || secret == "" {
		s.redirectEmailVerificationFailure(w, r)
		return
	}
	if err := s.emailVerificationService().Complete(r.Context(), userID, secret); err != nil {
		logRequestError(r, err, "failed to complete email verification for %s", userID)
		s.redirectEmailVerificationFailure(w, r)
		return
	}
	http.Redirect(w, r, s.mailboxProofSuccessPath(r, mailboxProofConfirm, userID, false), http.StatusSeeOther)
}

// mailboxProofKind selects the post-establish (or post-complete) success destination.
// Only confirm attaches noticeEmailVerified; invite accept and recovery do not.
type mailboxProofKind int

const (
	mailboxProofConfirm mailboxProofKind = iota
	mailboxProofInviteAccept
	mailboxProofPasswordRecovery
)

// mailboxProofEstablish is the fail-closed HTTP outcome of EstablishFromMailboxProof
// for invite accept and password recovery. When OK is false, an error response was
// already written; callers must not continue onto success redirects or session write.
type mailboxProofEstablish struct {
	OK           bool
	RedirectPath string
}

// establishMailboxProof calls EstablishFromMailboxProof and fail-closes on error
// (HTTP 500). Soft-log-and-continue onto /my or login-as-verified is intentionally forbidden.
func (s *Server) establishMailboxProof(w http.ResponseWriter, r *http.Request, userID string, kind mailboxProofKind, inviteNeedsPassword bool) mailboxProofEstablish {
	action := mailboxProofAction(kind)
	if err := s.emailVerificationService().EstablishFromMailboxProof(r.Context(), userID); err != nil {
		logAndHTTPError(w, r, http.StatusInternalServerError, "failed to verify email", err,
			"failed to establish email verification after %s for %s", action, userID)
		return mailboxProofEstablish{}
	}
	return mailboxProofEstablish{
		OK:           true,
		RedirectPath: s.mailboxProofSuccessPath(r, kind, userID, inviteNeedsPassword),
	}
}

func mailboxProofAction(kind mailboxProofKind) string {
	switch kind {
	case mailboxProofInviteAccept:
		return "invite accept"
	case mailboxProofPasswordRecovery:
		return "password recovery"
	default:
		return "mailbox proof"
	}
}

// mailboxProofSuccessPath is the shared success destination after mailbox proof
// establishes verification (invite/recovery) or after confirm Completes.
func (s *Server) mailboxProofSuccessPath(r *http.Request, kind mailboxProofKind, userID string, inviteNeedsPassword bool) string {
	switch kind {
	case mailboxProofConfirm:
		return pathWithNotice(s.postVerificationAppPathForUserID(r, userID), noticeEmailVerified)
	case mailboxProofInviteAccept:
		if inviteNeedsPassword {
			return "/invite/password"
		}
		return appHomePath
	case mailboxProofPasswordRecovery:
		return pathWithNotice("/login", noticePasswordResetSuccess)
	default:
		return appHomePath
	}
}

func emailVerificationConfirmParams(r *http.Request) (string, string) {
	query := r.URL.Query()
	return strings.TrimSpace(query.Get("userId")), strings.TrimSpace(query.Get("secret"))
}

func (s *Server) redirectEmailVerificationFailure(w http.ResponseWriter, r *http.Request) {
	notice := url.QueryEscape(noticeVerificationFailed)
	if s.enforceAuth {
		if _, _, err := s.currentUser(r); err == nil {
			http.Redirect(w, r, emailVerificationPath()+"?notice="+notice, http.StatusSeeOther)
			return
		}
	}
	http.Redirect(w, r, "/login?notice="+notice, http.StatusSeeOther)
}

func (s *Server) renderEmailVerificationWaiting(w http.ResponseWriter, r *http.Request, user *AccountUser, confirmation, errMsg string) {
	resend := s.emailVerificationResendState(r)
	view := VerifyEmailView{
		PageBase:               s.pageBaseForUser(user, "verify_email_body", "", ""),
		Email:                  strings.TrimSpace(user.Email),
		Confirmation:           confirmation,
		Error:                  errMsg,
		ResendDisabled:         resend.Disabled,
		ResendAvailableAt:      resend.AvailableAt,
		ResendRemainingSeconds: resend.RemainingSeconds,
	}
	if err := s.tmpl.ExecuteTemplate(w, "verify_email.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) postVerificationAppPath(r *http.Request, user *AccountUser) string {
	if user != nil && s.affiliationService().IsAffiliated(identityUserForAffiliation(user)) {
		return appHomePath
	}
	return onboardingPath()
}

func (s *Server) postVerificationAppPathForUserID(r *http.Request, userID string) string {
	if s.identity == nil {
		return appHomePath
	}
	user, err := s.identity.GetUserByID(r.Context(), userID)
	if err != nil {
		return appHomePath
	}
	if s.affiliationService().IsAffiliated(user) {
		return appHomePath
	}
	return onboardingPath()
}

type emailVerificationResendState struct {
	Disabled         bool
	AvailableAt      int64
	RemainingSeconds int
}

func (s *Server) emailVerificationResendTooSoon(r *http.Request) bool {
	return s.emailVerificationResendState(r).Disabled
}

func (s *Server) emailVerificationResendState(r *http.Request) emailVerificationResendState {
	cookie, err := r.Cookie(emailVerificationResendCookie)
	if err != nil || cookie == nil {
		return emailVerificationResendState{}
	}
	unix, err := strconv.ParseInt(strings.TrimSpace(cookie.Value), 10, 64)
	if err != nil || unix <= 0 {
		return emailVerificationResendState{}
	}
	availableAt := time.Unix(unix, 0).Add(emailVerificationResendCooldown)
	remaining := availableAt.Sub(s.nowUTC())
	if remaining <= 0 {
		return emailVerificationResendState{}
	}
	secs := int((remaining + time.Second - 1) / time.Second)
	if secs < 1 {
		secs = 1
	}
	return emailVerificationResendState{
		Disabled:         true,
		AvailableAt:      availableAt.Unix(),
		RemainingSeconds: secs,
	}
}

func (s *Server) setEmailVerificationResendCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     emailVerificationResendCookie,
		Value:    strconv.FormatInt(s.nowUTC().Unix(), 10),
		Path:     "/",
		MaxAge:   int(emailVerificationResendCooldown.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   shouldSecureCookie(r),
	})
}
