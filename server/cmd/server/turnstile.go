package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const turnstileSiteverifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

var errTurnstileVerification = errors.New("turnstile verification failed")

type turnstileVerifyResponse struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
}

// configuredTurnstileSiteKey returns a key only when server-side validation is enabled.
// Requiring both values avoids presenting a widget that the server does not enforce.
func (s *Server) configuredTurnstileSiteKey() string {
	if strings.TrimSpace(s.turnstileSiteKey) == "" || strings.TrimSpace(s.turnstileSecretKey) == "" {
		return ""
	}
	return s.turnstileSiteKey
}

func (s *Server) verifyTurnstile(r *http.Request) error {
	if s.configuredTurnstileSiteKey() == "" {
		return nil
	}

	token := strings.TrimSpace(r.FormValue("cf-turnstile-response"))
	if token == "" {
		return errTurnstileVerification
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	values := url.Values{
		"secret":   {s.turnstileSecretKey},
		"response": {token},
	}
	if remoteIP := requestRemoteIP(r); remoteIP != "" {
		values.Set("remoteip", remoteIP)
	}
	endpoint := s.turnstileVerifyURL
	if endpoint == "" {
		endpoint = turnstileSiteverifyURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return fmt.Errorf("create Siteverify request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := s.turnstileClient
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("call Siteverify: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("Siteverify status %d: %w", response.StatusCode, errTurnstileVerification)
	}

	var result turnstileVerifyResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode Siteverify response: %w", err)
	}
	if !result.Success {
		return fmt.Errorf("Siteverify rejected token (%s): %w", strings.Join(result.ErrorCodes, ","), errTurnstileVerification)
	}
	return nil
}

func requestRemoteIP(r *http.Request) string {
	if remoteIP := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); net.ParseIP(remoteIP) != nil {
		return remoteIP
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && net.ParseIP(host) != nil {
		return host
	}
	if net.ParseIP(strings.TrimSpace(r.RemoteAddr)) != nil {
		return strings.TrimSpace(r.RemoteAddr)
	}
	return ""
}
