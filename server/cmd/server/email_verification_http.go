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
	http.Redirect(w, r, s.postVerificationAppPathForUserID(r, userID), http.StatusSeeOther)
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
	view := VerifyEmailView{
		PageBase:       s.pageBaseForUser(user, "verify_email_body", "", ""),
		Email:          strings.TrimSpace(user.Email),
		Confirmation:   confirmation,
		Error:          errMsg,
		ResendDisabled: s.emailVerificationResendTooSoon(r),
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

func (s *Server) emailVerificationResendTooSoon(r *http.Request) bool {
	cookie, err := r.Cookie(emailVerificationResendCookie)
	if err != nil || cookie == nil {
		return false
	}
	unix, err := strconv.ParseInt(strings.TrimSpace(cookie.Value), 10, 64)
	if err != nil || unix <= 0 {
		return false
	}
	elapsed := s.nowUTC().Sub(time.Unix(unix, 0))
	return elapsed >= 0 && elapsed < emailVerificationResendCooldown
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
