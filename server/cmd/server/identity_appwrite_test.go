package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	appwriteclient "github.com/appwrite/sdk-for-go/client"
	"github.com/appwrite/sdk-for-go/models"
)

func TestAppwriteIdentityCreateEmailPasswordSession(t *testing.T) {
	var method string
	var path string
	var projectHeader string
	var keyHeader string
	var body map[string]interface{}

	appwriteAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		projectHeader = r.Header.Get("X-Appwrite-Project")
		keyHeader = r.Header.Get("X-Appwrite-Key")
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"$id":"session-1",
			"userId":"user-1",
			"expire":"2026-03-18T10:11:12Z",
			"secret":"session-secret"
		}`))
	}))
	defer appwriteAPI.Close()

	identity := NewAppwriteIdentity(appwriteAPI.URL+"/v1", "project-1", "api-key-1", appwriteAPI.Client())

	session, err := identity.CreateEmailPasswordSession(context.Background(), "user@example.com", "pw-123456789012")
	if err != nil {
		t.Fatalf("CreateEmailPasswordSession error: %v", err)
	}

	if method != http.MethodPost {
		t.Fatalf("method = %q, want POST", method)
	}
	if path != "/v1/account/sessions/email" {
		t.Fatalf("path = %q, want /v1/account/sessions/email", path)
	}
	if projectHeader != "project-1" {
		t.Fatalf("project header = %q, want project-1", projectHeader)
	}
	if keyHeader != "api-key-1" {
		t.Fatalf("key header = %q, want api-key-1", keyHeader)
	}
	if body["email"] != "user@example.com" {
		t.Fatalf("email payload = %#v, want user@example.com", body["email"])
	}
	if body["password"] != "pw-123456789012" {
		t.Fatalf("password payload = %#v", body["password"])
	}
	if session.Secret != "session-secret" {
		t.Fatalf("session secret = %q, want session-secret", session.Secret)
	}
	if session.UserID != "user-1" {
		t.Fatalf("session user id = %q, want user-1", session.UserID)
	}
	if !session.ExpiresAt.Equal(time.Date(2026, time.March, 18, 10, 11, 12, 0, time.UTC)) {
		t.Fatalf("expires at = %s", session.ExpiresAt)
	}
}

func TestAppwriteIdentityOrganizationOperations(t *testing.T) {
	t.Setenv("APPWRITE_ORG_ASSETS_BUCKET", "org-assets")

	var createTeamBody map[string]interface{}
	var createTeamSessionHeader string
	var createTeamKeyHeader string
	var updateNameBody map[string]interface{}
	var updateNameSessionHeader string
	var updateNameKeyHeader string
	var updatePrefsBodies []map[string]interface{}
	var updatedLabelsBody map[string]interface{}
	var createFileCalled bool
	var createFileContentType string
	var createFileBody []byte
	var updateLabelsBody map[string]interface{}

	appwriteAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/teams":
			createTeamSessionHeader = r.Header.Get("X-Appwrite-Session")
			createTeamKeyHeader = r.Header.Get("X-Appwrite-Key")
			if err := json.NewDecoder(r.Body).Decode(&createTeamBody); err != nil {
				t.Fatalf("decode create team: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"$id":"fresh-org","name":"Fresh Org","prefs":{"schemaVersion":1,"slug":"fresh-org"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/account":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"$id":"user-1","email":"owner@example.com","status":true,"labels":[]}`))
		case r.Method == http.MethodPut && r.URL.Path == "/v1/users/user-1/labels":
			if err := json.NewDecoder(r.Body).Decode(&updatedLabelsBody); err != nil {
				t.Fatalf("decode labels: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"$id":"user-1","email":"owner@example.com","status":true,"labels":["attesta:org-admin"]}`))
		case r.Method == http.MethodPut && r.URL.Path == "/v1/users/member-1/labels":
			if err := json.NewDecoder(r.Body).Decode(&updateLabelsBody); err != nil {
				t.Fatalf("decode update labels: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"$id":"member-1","email":"member@example.com","status":true,"labels":["attesta:role:qa-reviewer","attesta:org-admin"]}`))
		case r.Method == http.MethodPut && r.URL.Path == "/v1/teams/fresh-org/prefs":
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode prefs: %v", err)
			}
			updatePrefsBodies = append(updatePrefsBodies, body)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"schemaVersion":1,"slug":"fresh-org"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/teams/fresh-org":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"$id":"fresh-org","name":"Fresh Org","prefs":{"schemaVersion":1,"slug":"fresh-org","roles":[{"slug":"qa-reviewer","name":"QA Reviewer"}]}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/teams/fresh-org/memberships":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"total":1,"memberships":[{"$id":"membership-1","userId":"member-1","userEmail":"member@example.com","teamId":"fresh-org","teamName":"Fresh Org","confirm":true,"roles":["member"]}]}`))
		case r.Method == http.MethodPut && r.URL.Path == "/v1/teams/fresh-org":
			updateNameSessionHeader = r.Header.Get("X-Appwrite-Session")
			updateNameKeyHeader = r.Header.Get("X-Appwrite-Key")
			if err := json.NewDecoder(r.Body).Decode(&updateNameBody); err != nil {
				t.Fatalf("decode update team: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"$id":"fresh-org","name":"Updated Org"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/users/member-1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"$id":"member-1","email":"member@example.com","status":true,"labels":["attesta:role:qa-reviewer"]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/users/member-1/memberships":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"total":1,"memberships":[{"$id":"membership-1","userId":"member-1","userEmail":"member@example.com","teamId":"fresh-org","teamName":"Fresh Org","confirm":true,"roles":["owner"]}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/storage/buckets/org-assets/files/logo-1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"$id":"logo-1","name":"logo.png","mimeType":"image/png"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/storage/buckets/org-assets/files/logo-1/view":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("PNG"))
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/storage/buckets/org-assets/files/"):
			http.NotFound(w, r)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/storage/buckets/org-assets/files":
			createFileCalled = true
			createFileContentType = r.Header.Get("Content-Type")
			var err error
			createFileBody, err = io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read upload body: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"$id":"logo-1","name":"logo.png","mimeType":"image/png"}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer appwriteAPI.Close()

	identity := NewAppwriteIdentity(appwriteAPI.URL+"/v1", "project-1", "api-key-1", appwriteAPI.Client())

	createdOrg, err := identity.CreateOrganization(context.Background(), "session-secret", "Fresh Org")
	if err != nil {
		t.Fatalf("CreateOrganization error: %v", err)
	}
	if createdOrg.Slug != "fresh-org" || createdOrg.Name != "Fresh Org" {
		t.Fatalf("created org = %#v", createdOrg)
	}
	if createTeamBody["teamId"] != "fresh-org" || createTeamBody["name"] != "Fresh Org" {
		t.Fatalf("create team body = %#v", createTeamBody)
	}
	labels, _ := updatedLabelsBody["labels"].([]interface{})
	if len(labels) != 1 || labels[0] != identityOrgAdminLabel {
		t.Fatalf("labels body = %#v", updatedLabelsBody)
	}
	if createTeamSessionHeader != "session-secret" {
		t.Fatalf("create team session header = %q, want session-secret", createTeamSessionHeader)
	}

	uploadedLogo, err := identity.UploadOrganizationLogo(context.Background(), "fresh-org", IdentityFile{
		Filename: "logo.png",
		Data:     []byte("PNG"),
	})
	if err != nil {
		t.Fatalf("UploadOrganizationLogo error: %v", err)
	}
	if !createFileCalled || uploadedLogo.ID != "logo-1" {
		t.Fatalf("upload result = %#v called=%v", uploadedLogo, createFileCalled)
	}
	if createFileContentType == "" || !strings.HasPrefix(createFileContentType, "multipart/form-data") {
		t.Fatalf("upload content type = %q", createFileContentType)
	}
	if !strings.Contains(string(createFileBody), "logo.png") {
		t.Fatalf("upload body = %q", string(createFileBody))
	}
	if uploadedLogo.ContentType != "image/png" {
		t.Fatalf("uploaded logo content type = %q, want image/png", uploadedLogo.ContentType)
	}

	updatedOrg, err := identity.UpdateOrganization(context.Background(), "session-secret", "fresh-org", "Updated Org", uploadedLogo.ID, []IdentityRole{{Slug: "qa-reviewer", Name: "QA Reviewer"}})
	if err != nil {
		t.Fatalf("UpdateOrganization error: %v", err)
	}
	if updatedOrg.Slug != "updated-org" || updatedOrg.LogoFileID != "logo-1" {
		t.Fatalf("updated org = %#v", updatedOrg)
	}
	if updateNameBody["name"] != "Updated Org" {
		t.Fatalf("update name body = %#v", updateNameBody)
	}
	if updateNameSessionHeader != "session-secret" {
		t.Fatalf("update name session header = %q, want session-secret", updateNameSessionHeader)
	}
	if len(updatePrefsBodies) != 2 {
		t.Fatalf("prefs calls = %d, want 2", len(updatePrefsBodies))
	}
	prefs, _ := updatePrefsBodies[1]["prefs"].(map[string]interface{})
	if prefs["slug"] != "updated-org" || prefs["logoFileId"] != "logo-1" {
		t.Fatalf("update prefs body = %#v", updatePrefsBodies[1])
	}

	logo, err := identity.GetOrganizationLogo(context.Background(), "logo-1")
	if err != nil {
		t.Fatalf("GetOrganizationLogo error: %v", err)
	}
	if logo.Filename != "logo.png" || logo.ContentType != "image/png" || string(logo.Data) != "PNG" {
		t.Fatalf("logo = %#v", logo)
	}

	orgUsers, err := identity.ListOrganizationUsers(context.Background(), "fresh-org")
	if err != nil {
		t.Fatalf("ListOrganizationUsers error: %v", err)
	}
	if len(orgUsers) != 1 || orgUsers[0].Email != "member@example.com" || !containsRole(decodeIdentityRoleLabels(orgUsers[0].Labels), "qa-reviewer") {
		t.Fatalf("org users = %#v", orgUsers)
	}

	updatedUser, err := identity.UpdateUserLabels(context.Background(), "member-1", []string{identityOrgAdminLabel, encodeIdentityRoleLabel("qa-reviewer")})
	if err != nil {
		t.Fatalf("UpdateUserLabels error: %v", err)
	}
	if updatedUser.Email != "member@example.com" || !updatedUser.IsOrgAdmin || updatedUser.OrgSlug != "fresh-org" {
		t.Fatalf("updated user = %#v", updatedUser)
	}
	updatedUserLabels, _ := updateLabelsBody["labels"].([]interface{})
	if len(updatedUserLabels) != 2 {
		t.Fatalf("update labels body = %#v", updateLabelsBody)
	}

	createdAdminOrg, err := identity.CreateOrganizationAsAdmin(context.Background(), "Fresh Org")
	if err != nil {
		t.Fatalf("CreateOrganizationAsAdmin error: %v", err)
	}
	if createdAdminOrg.Slug != "fresh-org" || createTeamKeyHeader != "api-key-1" {
		t.Fatalf("created admin org = %#v headers session=%q key=%q", createdAdminOrg, createTeamSessionHeader, createTeamKeyHeader)
	}
	if createTeamSessionHeader != "" {
		t.Fatalf("create admin team session header = %q, want empty", createTeamSessionHeader)
	}

	updatedAdminOrg, err := identity.UpdateOrganizationAsAdmin(context.Background(), "fresh-org", "Updated Org", "logo-1", []IdentityRole{{Slug: "qa-reviewer", Name: "QA Reviewer"}})
	if err != nil {
		t.Fatalf("UpdateOrganizationAsAdmin error: %v", err)
	}
	if updatedAdminOrg.Slug != "updated-org" || updateNameKeyHeader != "api-key-1" {
		t.Fatalf("updated admin org = %#v headers session=%q key=%q", updatedAdminOrg, updateNameSessionHeader, updateNameKeyHeader)
	}
	if updateNameSessionHeader != "" {
		t.Fatalf("update admin team session header = %q, want empty", updateNameSessionHeader)
	}
}

func TestAppwriteIdentityMembershipOperations(t *testing.T) {
	var inviteSessionHeader string
	var inviteKeyHeader string
	var inviteBody map[string]interface{}
	var updateMembershipSessionHeader string
	var updateMembershipKeyHeader string
	var updateMembershipBody map[string]interface{}
	var deleteMembershipSessionHeader string
	var userSearch string

	appwriteAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/teams/acme":
			http.NotFound(w, r)
		case r.Method == http.MethodGet && r.URL.Path == "/v1/teams":
			_, _ = w.Write([]byte(`{"total":1,"teams":[{"$id":"acme-team","name":"Acme Org","prefs":{"schemaVersion":1,"slug":"acme"}}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/users":
			userSearch = r.URL.Query().Get("search")
			_, _ = w.Write([]byte(`{"total":1,"users":[{"$id":"member-1","email":"member@example.com","status":true,"labels":["attesta:role:qa-reviewer"]}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/users/member-1":
			_, _ = w.Write([]byte(`{"$id":"member-1","email":"member@example.com","status":true,"labels":["attesta:role:qa-reviewer"]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/users/member-1/memberships":
			_, _ = w.Write([]byte(`{"total":1,"memberships":[{"$id":"membership-1","userId":"member-1","userEmail":"member@example.com","teamId":"acme-team","teamName":"Acme Org","confirm":true,"roles":["owner"]}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/teams/acme-team":
			_, _ = w.Write([]byte(`{"$id":"acme-team","name":"Acme Org","prefs":{"schemaVersion":1,"slug":"acme"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/teams/acme-team/memberships":
			_, _ = w.Write([]byte(`{"total":2,"memberships":[
				{"$id":"membership-1","userId":"member-1","userEmail":"member@example.com","teamId":"acme-team","teamName":"Acme Org","invited":"2026-03-10T10:00:00Z","joined":"2026-03-10T11:00:00Z","confirm":true,"roles":["owner"]},
				{"$id":"membership-2","userId":"","userEmail":"pending@example.com","teamId":"acme-team","teamName":"Acme Org","invited":"2026-03-10T10:00:00Z","confirm":false,"roles":["member","attesta-role:approver"]}
			]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/teams/acme-team/memberships":
			inviteSessionHeader = r.Header.Get("X-Appwrite-Session")
			inviteKeyHeader = r.Header.Get("X-Appwrite-Key")
			if err := json.NewDecoder(r.Body).Decode(&inviteBody); err != nil {
				t.Fatalf("decode invite body: %v", err)
			}
			_, _ = w.Write([]byte(`{"$id":"membership-3","userId":"","userEmail":"invitee@example.com","teamId":"acme-team","teamName":"Acme Org","confirm":false,"roles":["owner","attesta-role:approver"]}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/v1/teams/acme-team/memberships/membership-1":
			updateMembershipSessionHeader = r.Header.Get("X-Appwrite-Session")
			updateMembershipKeyHeader = r.Header.Get("X-Appwrite-Key")
			if err := json.NewDecoder(r.Body).Decode(&updateMembershipBody); err != nil {
				t.Fatalf("decode update membership body: %v", err)
			}
			_, _ = w.Write([]byte(`{"$id":"membership-1","userId":"member-1","userEmail":"member@example.com","teamId":"acme-team","teamName":"Acme Org","confirm":true,"roles":["member","attesta-role:approver"]}`))
		case r.Method == http.MethodPut && r.URL.Path == "/v1/users/member-1/labels":
			_, _ = w.Write([]byte(`{"$id":"member-1","email":"member@example.com","status":true,"labels":["attesta:role:approver","attesta:org-admin"]}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/teams/acme-team/memberships/membership-1":
			deleteMembershipSessionHeader = r.Header.Get("X-Appwrite-Session")
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer appwriteAPI.Close()

	identity := NewAppwriteIdentity(appwriteAPI.URL+"/v1", "project-1", "api-key-1", appwriteAPI.Client())

	user, err := identity.GetUserByEmail(context.Background(), "member@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail error: %v", err)
	}
	if userSearch != "member@example.com" || user.Email != "member@example.com" || user.OrgSlug != "acme" {
		t.Fatalf("user = %#v search=%q", user, userSearch)
	}

	membership, err := identity.InviteOrganizationUser(context.Background(), "session-secret", "acme", "invitee@example.com", "https://attesta.example/invite/accept", []string{"approver"}, true)
	if err != nil {
		t.Fatalf("InviteOrganizationUser error: %v", err)
	}
	if inviteSessionHeader != "session-secret" || membership.Email != "invitee@example.com" || !membership.IsOrgAdmin {
		t.Fatalf("invite = %#v header=%q", membership, inviteSessionHeader)
	}
	if inviteBody["email"] != "invitee@example.com" || inviteBody["url"] != "https://attesta.example/invite/accept" {
		t.Fatalf("invite body = %#v", inviteBody)
	}

	memberships, err := identity.ListOrganizationMemberships(context.Background(), "acme")
	if err != nil {
		t.Fatalf("ListOrganizationMemberships error: %v", err)
	}
	if len(memberships) != 2 {
		t.Fatalf("memberships = %#v", memberships)
	}
	if !memberships[0].Confirmed || memberships[0].Email != "member@example.com" || !memberships[0].IsOrgAdmin {
		t.Fatalf("membership[0] = %#v", memberships[0])
	}
	if memberships[1].Confirmed || memberships[1].RoleSlugs[0] != "approver" {
		t.Fatalf("membership[1] = %#v", memberships[1])
	}

	updatedMembership, err := identity.UpdateOrganizationMembership(context.Background(), "session-secret", "acme", "membership-1", []string{"approver"}, false)
	if err != nil {
		t.Fatalf("UpdateOrganizationMembership error: %v", err)
	}
	if updateMembershipSessionHeader != "session-secret" || updatedMembership.RoleSlugs[0] != "qa-reviewer" {
		t.Fatalf("updated membership = %#v header=%q", updatedMembership, updateMembershipSessionHeader)
	}
	if roles, _ := updateMembershipBody["roles"].([]interface{}); len(roles) != 2 {
		t.Fatalf("update membership body = %#v", updateMembershipBody)
	}

	updatedUser, err := identity.UpdateUserLabels(context.Background(), "member-1", []string{encodeIdentityRoleLabel("approver"), identityOrgAdminLabel})
	if err != nil {
		t.Fatalf("UpdateUserLabels error: %v", err)
	}
	if !updatedUser.IsOrgAdmin || updatedUser.OrgSlug != "acme" {
		t.Fatalf("updated user = %#v", updatedUser)
	}

	if err := identity.DeleteOrganizationMembership(context.Background(), "session-secret", "acme", "membership-1"); err != nil {
		t.Fatalf("DeleteOrganizationMembership error: %v", err)
	}
	if deleteMembershipSessionHeader != "session-secret" {
		t.Fatalf("delete session header = %q", deleteMembershipSessionHeader)
	}

	adminInvite, err := identity.InviteOrganizationUserAsAdmin(context.Background(), "acme", "invitee@example.com", "https://attesta.example/invite/accept", []string{"approver"}, true)
	if err != nil {
		t.Fatalf("InviteOrganizationUserAsAdmin error: %v", err)
	}
	if adminInvite.Email != "invitee@example.com" || inviteKeyHeader != "api-key-1" {
		t.Fatalf("admin invite = %#v key=%q session=%q", adminInvite, inviteKeyHeader, inviteSessionHeader)
	}
	if inviteSessionHeader != "" {
		t.Fatalf("admin invite session header = %q, want empty", inviteSessionHeader)
	}

	adminUpdatedMembership, err := identity.UpdateOrganizationMembershipAsAdmin(context.Background(), "acme", "membership-1", []string{"approver"}, true)
	if err != nil {
		t.Fatalf("UpdateOrganizationMembershipAsAdmin error: %v", err)
	}
	if adminUpdatedMembership.Email != "member@example.com" || updateMembershipKeyHeader != "api-key-1" {
		t.Fatalf("admin updated membership = %#v key=%q session=%q", adminUpdatedMembership, updateMembershipKeyHeader, updateMembershipSessionHeader)
	}
	if updateMembershipSessionHeader != "" {
		t.Fatalf("admin update session header = %q, want empty", updateMembershipSessionHeader)
	}
}

func TestAppwriteIdentityAddOrganizationUserByIDAsAdmin(t *testing.T) {
	var createMembershipKeyHeader string
	var createMembershipSessionHeader string
	var createMembershipBody map[string]interface{}

	appwriteAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/teams/acme":
			http.NotFound(w, r)
		case r.Method == http.MethodGet && r.URL.Path == "/v1/teams":
			_, _ = w.Write([]byte(`{"total":1,"teams":[{"$id":"acme-team","name":"Acme Org","prefs":{"schemaVersion":1,"slug":"acme"}}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/teams/acme-team":
			_, _ = w.Write([]byte(`{"$id":"acme-team","name":"Acme Org","prefs":{"schemaVersion":1,"slug":"acme"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/users/user-9":
			_, _ = w.Write([]byte(`{"$id":"user-9","email":"admin@example.com","status":true,"labels":["attesta:org-admin"]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/users/user-9/memberships":
			_, _ = w.Write([]byte(`{"total":1,"memberships":[{"$id":"membership-3","userId":"user-9","userEmail":"admin@example.com","teamId":"acme-team","teamName":"Acme Org","confirm":true,"roles":["owner"]}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/teams/acme-team/memberships":
			createMembershipKeyHeader = r.Header.Get("X-Appwrite-Key")
			createMembershipSessionHeader = r.Header.Get("X-Appwrite-Session")
			if err := json.NewDecoder(r.Body).Decode(&createMembershipBody); err != nil {
				t.Fatalf("decode create membership body: %v", err)
			}
			_, _ = w.Write([]byte(`{"$id":"membership-3","userId":"user-9","userEmail":"admin@example.com","teamId":"acme-team","teamName":"Acme Org","confirm":true,"roles":["owner"]}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer appwriteAPI.Close()

	identity := NewAppwriteIdentity(appwriteAPI.URL+"/v1", "project-1", "api-key-1", appwriteAPI.Client())

	membership, err := identity.AddOrganizationUserByIDAsAdmin(context.Background(), "acme", "user-9", nil, true)
	if err != nil {
		t.Fatalf("AddOrganizationUserByIDAsAdmin error: %v", err)
	}
	if createMembershipKeyHeader != "api-key-1" || createMembershipSessionHeader != "" {
		t.Fatalf("headers key=%q session=%q", createMembershipKeyHeader, createMembershipSessionHeader)
	}
	if createMembershipBody["userId"] != "user-9" {
		t.Fatalf("create membership body = %#v", createMembershipBody)
	}
	if _, ok := createMembershipBody["email"]; ok {
		t.Fatalf("create membership body unexpectedly had email: %#v", createMembershipBody)
	}
	if membership.UserID != "user-9" || !membership.IsOrgAdmin || !membership.Confirmed {
		t.Fatalf("membership = %#v", membership)
	}
}

func TestAppwriteIdentityMembershipOperationErrors(t *testing.T) {
	appwriteAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/teams/missing":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"not found","code":404}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/teams":
			_, _ = w.Write([]byte(`{"total":0,"teams":[]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/users":
			_, _ = w.Write([]byte(`{"total":0,"users":[]}`))
		default:
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"message":"unauthorized","code":401}`))
		}
	}))
	defer appwriteAPI.Close()

	identity := NewAppwriteIdentity(appwriteAPI.URL+"/v1", "project-1", "api-key-1", appwriteAPI.Client())

	if _, err := identity.GetUserByEmail(context.Background(), "missing@example.com"); !errors.Is(err, ErrIdentityNotFound) {
		t.Fatalf("GetUserByEmail error = %v, want %v", err, ErrIdentityNotFound)
	}
	if _, err := identity.InviteOrganizationUser(context.Background(), "session-secret", "missing", "user@example.com", "https://attesta.example/invite/accept", nil, false); !errors.Is(err, ErrIdentityNotFound) {
		t.Fatalf("InviteOrganizationUser error = %v, want %v", err, ErrIdentityNotFound)
	}
	if _, err := identity.ListOrganizationMemberships(context.Background(), "missing"); !errors.Is(err, ErrIdentityNotFound) {
		t.Fatalf("ListOrganizationMemberships error = %v, want %v", err, ErrIdentityNotFound)
	}
	if _, err := identity.UpdateOrganizationMembership(context.Background(), "session-secret", "missing", "membership-1", nil, false); !errors.Is(err, ErrIdentityNotFound) {
		t.Fatalf("UpdateOrganizationMembership error = %v, want %v", err, ErrIdentityNotFound)
	}
	if err := identity.DeleteOrganizationMembership(context.Background(), "session-secret", "missing", "membership-1"); !errors.Is(err, ErrIdentityNotFound) {
		t.Fatalf("DeleteOrganizationMembership error = %v, want %v", err, ErrIdentityNotFound)
	}
}

func TestAppwriteIdentityListOrganizationMembershipsPendingMembership(t *testing.T) {
	appwriteAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/teams/acme":
			_, _ = w.Write([]byte(`{"$id":"acme","name":"Acme Org","prefs":{"schemaVersion":1,"slug":"acme"}}`))
		case "/v1/teams/acme/memberships":
			_, _ = w.Write([]byte(`{"total":1,"memberships":[{"$id":"membership-1","userId":"","userEmail":"pending@example.com","teamId":"acme","teamName":"Acme Org","confirm":false,"roles":["member","attesta-role:approver"]}]}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer appwriteAPI.Close()

	identity := NewAppwriteIdentity(appwriteAPI.URL+"/v1", "project-1", "api-key-1", appwriteAPI.Client())
	memberships, err := identity.ListOrganizationMemberships(context.Background(), "acme")
	if err != nil {
		t.Fatalf("ListOrganizationMemberships error: %v", err)
	}
	if len(memberships) != 1 || memberships[0].Email != "pending@example.com" || memberships[0].Confirmed {
		t.Fatalf("memberships = %#v", memberships)
	}
}

func TestAppwriteIdentityCreateAccountAndRecovery(t *testing.T) {
	var createPath string
	var createBody map[string]interface{}
	var recoveryPath string
	var recoveryBody map[string]interface{}
	var completePath string
	var completeBody map[string]interface{}

	appwriteAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/account":
			createPath = r.URL.Path
			if err := json.NewDecoder(r.Body).Decode(&createBody); err != nil {
				t.Fatalf("decode create account body: %v", err)
			}
			_, _ = w.Write([]byte(`{"$id":"user-1","email":"new@example.com","status":true,"labels":[]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/account/recovery":
			recoveryPath = r.URL.Path
			if err := json.NewDecoder(r.Body).Decode(&recoveryBody); err != nil {
				t.Fatalf("decode recovery body: %v", err)
			}
			_, _ = w.Write([]byte(`{"userId":"user-1","secret":"secret-1"}`))
		case r.Method == http.MethodPut && r.URL.Path == "/v1/account/recovery":
			completePath = r.URL.Path
			if err := json.NewDecoder(r.Body).Decode(&completeBody); err != nil {
				t.Fatalf("decode complete body: %v", err)
			}
			_, _ = w.Write([]byte(`{"userId":"user-1","secret":"secret-1"}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer appwriteAPI.Close()

	identity := NewAppwriteIdentity(appwriteAPI.URL+"/v1", "project-1", "api-key-1", appwriteAPI.Client())

	user, err := identity.CreateAccount(context.Background(), "new@example.com", "very-secure-password", "")
	if err != nil {
		t.Fatalf("CreateAccount error: %v", err)
	}
	if createPath != "/v1/account" || createBody["email"] != "new@example.com" || createBody["password"] != "very-secure-password" {
		t.Fatalf("create account request = %q %#v", createPath, createBody)
	}
	if user.Email != "new@example.com" || user.ID != "user-1" {
		t.Fatalf("user = %#v", user)
	}

	if err := identity.CreateRecovery(context.Background(), "new@example.com", "http://attesta.local/reset/confirm"); err != nil {
		t.Fatalf("CreateRecovery error: %v", err)
	}
	if recoveryPath != "/v1/account/recovery" || recoveryBody["email"] != "new@example.com" || recoveryBody["url"] != "http://attesta.local/reset/confirm" {
		t.Fatalf("recovery request = %q %#v", recoveryPath, recoveryBody)
	}

	if err := identity.CompleteRecovery(context.Background(), "user-1", "secret-1", "updated-password"); err != nil {
		t.Fatalf("CompleteRecovery error: %v", err)
	}
	if completePath != "/v1/account/recovery" || completeBody["userId"] != "user-1" || completeBody["secret"] != "secret-1" || completeBody["password"] != "updated-password" {
		t.Fatalf("complete recovery request = %q %#v", completePath, completeBody)
	}
}

func TestAppwriteIdentityGetCurrentUserHydratesMembership(t *testing.T) {
	var accountSessionHeader string
	var membershipsKeyHeader string

	appwriteAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/account":
			accountSessionHeader = r.Header.Get("X-Appwrite-Session")
			_, _ = w.Write([]byte(`{
				"$id":"user-1",
				"email":"org-admin@example.com",
				"status":true,
				"labels":["attesta:org-admin","attesta:role:qa-reviewer"]
			}`))
		case "/v1/users/user-1/memberships":
			membershipsKeyHeader = r.Header.Get("X-Appwrite-Key")
			_, _ = w.Write([]byte(`{
				"total":1,
				"memberships":[
					{
						"$id":"membership-1",
						"userId":"user-1",
						"teamId":"acme",
						"teamName":"Acme Org",
						"confirm":true,
						"roles":["owner","member"]
					}
					]
				}`))
		case "/v1/teams/acme":
			_, _ = w.Write([]byte(`{
				"$id":"acme",
				"name":"Acme Org",
				"prefs":{"schemaVersion":1,"slug":"acme"}
			}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer appwriteAPI.Close()

	identity := NewAppwriteIdentity(appwriteAPI.URL+"/v1", "project-1", "api-key-1", appwriteAPI.Client())

	user, err := identity.GetCurrentUser(context.Background(), "session-secret")
	if err != nil {
		t.Fatalf("GetCurrentUser error: %v", err)
	}

	if accountSessionHeader != "session-secret" {
		t.Fatalf("account session header = %q, want session-secret", accountSessionHeader)
	}
	if membershipsKeyHeader != "api-key-1" {
		t.Fatalf("memberships key header = %q, want api-key-1", membershipsKeyHeader)
	}
	if user.ID != "user-1" {
		t.Fatalf("user id = %q, want user-1", user.ID)
	}
	if user.Email != "org-admin@example.com" {
		t.Fatalf("email = %q, want org-admin@example.com", user.Email)
	}
	if user.OrgSlug != "acme" || user.OrgName != "Acme Org" {
		t.Fatalf("org = %#v", user)
	}
	if user.MembershipID != "membership-1" {
		t.Fatalf("membership id = %q, want membership-1", user.MembershipID)
	}
	if len(user.MembershipRoles) != 2 || user.MembershipRoles[0] != "owner" || user.MembershipRoles[1] != "member" {
		t.Fatalf("membership roles = %#v", user.MembershipRoles)
	}
	if !user.IsOrgAdmin {
		t.Fatal("expected org admin")
	}
	if user.Status != "active" {
		t.Fatalf("status = %q, want active", user.Status)
	}
}

func TestAppwriteIdentityListOrganizationsDecodesPrefs(t *testing.T) {
	appwriteAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/teams" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"total":1,
			"teams":[
				{
					"$id":"acme",
					"name":"Acme Org",
					"prefs":{
						"schemaVersion":1,
						"logoFileId":"logo-file-1",
						"roles":[
							{"slug":"qa-reviewer","name":"QA Reviewer","color":"#123456","border":"solid"},
							{"slug":"qa-approver","name":"QA Approver","color":"#654321","border":"dashed"}
						]
					}
				}
			]
		}`))
	}))
	defer appwriteAPI.Close()

	identity := NewAppwriteIdentity(appwriteAPI.URL+"/v1", "project-1", "api-key-1", appwriteAPI.Client())

	orgs, err := identity.ListOrganizations(context.Background())
	if err != nil {
		t.Fatalf("ListOrganizations error: %v", err)
	}
	if len(orgs) != 1 {
		t.Fatalf("len(orgs) = %d, want 1", len(orgs))
	}
	if orgs[0].ID != "acme" || orgs[0].Slug != "acme" || orgs[0].Name != "Acme Org" {
		t.Fatalf("org = %#v", orgs[0])
	}
	if orgs[0].LogoFileID != "logo-file-1" {
		t.Fatalf("logo = %q, want logo-file-1", orgs[0].LogoFileID)
	}
	if len(orgs[0].Roles) != 2 {
		t.Fatalf("roles = %#v", orgs[0].Roles)
	}
	if orgs[0].Roles[0].Slug != "qa-reviewer" || orgs[0].Roles[1].Slug != "qa-approver" {
		t.Fatalf("roles = %#v", orgs[0].Roles)
	}
}

func TestAppwriteIdentityNormalizesNotFound(t *testing.T) {
	appwriteAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
	}))
	defer appwriteAPI.Close()

	identity := NewAppwriteIdentity(appwriteAPI.URL+"/v1", "project-1", "api-key-1", appwriteAPI.Client())

	_, err := identity.GetOrganizationBySlug(context.Background(), "missing")
	if !errors.Is(err, ErrIdentityNotFound) {
		t.Fatalf("error = %v, want %v", err, ErrIdentityNotFound)
	}
}

func TestAppwriteIdentityGetSessionAndDeleteSession(t *testing.T) {
	var getSessionHeader string
	var deleteSessionHeader string

	appwriteAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/account/sessions/current":
			getSessionHeader = r.Header.Get("X-Appwrite-Session")
			_, _ = w.Write([]byte(`{
				"$id":"session-1",
				"userId":"user-1",
				"expire":"2026-03-18T10:11:12Z"
			}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/account/sessions/current":
			deleteSessionHeader = r.Header.Get("X-Appwrite-Session")
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer appwriteAPI.Close()

	identity := NewAppwriteIdentity(appwriteAPI.URL+"/v1", "project-1", "api-key-1", appwriteAPI.Client())

	session, err := identity.GetSession(context.Background(), "session-secret")
	if err != nil {
		t.Fatalf("GetSession error: %v", err)
	}
	if getSessionHeader != "session-secret" {
		t.Fatalf("get session header = %q, want session-secret", getSessionHeader)
	}
	if session.Secret != "session-secret" {
		t.Fatalf("session secret = %q, want session-secret", session.Secret)
	}

	if err := identity.DeleteSession(context.Background(), "session-secret"); err != nil {
		t.Fatalf("DeleteSession error: %v", err)
	}
	if deleteSessionHeader != "session-secret" {
		t.Fatalf("delete session header = %q, want session-secret", deleteSessionHeader)
	}
}

func TestAppwriteIdentityAcceptInvite(t *testing.T) {
	var membershipPath string
	var sessionPath string
	var membershipBody map[string]interface{}

	appwriteAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPatch && r.URL.Path == "/v1/teams/acme/memberships/membership-1/status":
			membershipPath = r.URL.Path
			if err := json.NewDecoder(r.Body).Decode(&membershipBody); err != nil {
				t.Fatalf("decode membership body: %v", err)
			}
			http.SetCookie(w, &http.Cookie{Name: "a_session_project-1", Value: "session-secret", Path: "/"})
			_, _ = w.Write([]byte(`{"$id":"membership-1","teamId":"acme","userId":"user-1","confirm":true,"roles":["member"]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/account/sessions/current":
			sessionPath = r.URL.Path
			_, _ = w.Write([]byte(`{"$id":"session-1","userId":"user-1","expire":"2026-03-18T10:11:12Z","secret":"session-secret"}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer appwriteAPI.Close()

	identity := NewAppwriteIdentity(appwriteAPI.URL+"/v1", "project-1", "api-key-1", appwriteAPI.Client())

	session, err := identity.AcceptInvite(context.Background(), "acme", "membership-1", "user-1", "secret-1")
	if err != nil {
		t.Fatalf("AcceptInvite error: %v", err)
	}
	if membershipPath != "/v1/teams/acme/memberships/membership-1/status" {
		t.Fatalf("membership path = %q", membershipPath)
	}
	if membershipBody["userId"] != "user-1" || membershipBody["secret"] != "secret-1" {
		t.Fatalf("membership body = %#v", membershipBody)
	}
	if sessionPath != "/v1/account/sessions/current" {
		t.Fatalf("session path = %q", sessionPath)
	}
	if session.Secret != "session-secret" {
		t.Fatalf("session secret = %q, want session-secret", session.Secret)
	}
}

func TestAppwriteIdentityGetUserByIDAndGetOrganizationBySlug(t *testing.T) {
	appwriteAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/users/user-2":
			_, _ = w.Write([]byte(`{
				"$id":"user-2",
				"email":"reviewer@example.com",
				"status":true,
				"labels":["attesta:role:qa-reviewer"]
			}`))
		case "/v1/users/user-2/memberships":
			_, _ = w.Write([]byte(`{
				"total":1,
				"memberships":[
					{
						"$id":"membership-2",
						"userId":"user-2",
						"teamId":"acme",
						"teamName":"Acme Org",
						"confirm":false,
						"roles":["owner","attesta-role:qa-reviewer"]
					}
				]
			}`))
		case "/v1/teams/acme":
			_, _ = w.Write([]byte(`{
				"$id":"acme",
				"name":"Acme Org",
				"prefs":{
					"schemaVersion":1,
					"logoFileId":"logo-file-1",
					"roles":[{"slug":"qa-reviewer","name":"QA Reviewer","color":"#123456","border":"solid"}]
				}
			}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer appwriteAPI.Close()

	identity := NewAppwriteIdentity(appwriteAPI.URL+"/v1", "project-1", "api-key-1", appwriteAPI.Client())

	user, err := identity.GetUserByID(context.Background(), "user-2")
	if err != nil {
		t.Fatalf("GetUserByID error: %v", err)
	}
	if user.Status != "pending" {
		t.Fatalf("status = %q, want pending", user.Status)
	}
	if !user.IsOrgAdmin {
		t.Fatal("expected org admin derived from owner membership role")
	}

	org, err := identity.GetOrganizationBySlug(context.Background(), "acme")
	if err != nil {
		t.Fatalf("GetOrganizationBySlug error: %v", err)
	}
	if org.LogoFileID != "logo-file-1" {
		t.Fatalf("logo = %q, want logo-file-1", org.LogoFileID)
	}
	if len(org.Roles) != 1 || org.Roles[0].Slug != "qa-reviewer" {
		t.Fatalf("roles = %#v", org.Roles)
	}
}

func TestAppwriteIdentityListOrganizationUsersSkipsPendingAndKeepsMembershipData(t *testing.T) {
	appwriteAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/teams/acme":
			_, _ = w.Write([]byte(`{"$id":"acme","name":"Acme Org","prefs":{"schemaVersion":1,"slug":"acme"}}`))
		case "/v1/teams/acme/memberships":
			_, _ = w.Write([]byte(`{
				"total":2,
				"memberships":[
					{"$id":"membership-1","userId":"","userEmail":"service@example.com","teamId":"acme","teamName":"Acme Org","confirm":true,"roles":["member"]},
					{"$id":"membership-2","userId":"pending-1","userEmail":"pending@example.com","teamId":"acme","teamName":"Acme Org","confirm":false,"roles":["member"]}
				]
			}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer appwriteAPI.Close()

	identity := NewAppwriteIdentity(appwriteAPI.URL+"/v1", "project-1", "api-key-1", appwriteAPI.Client())

	users, err := identity.ListOrganizationUsers(context.Background(), "acme")
	if err != nil {
		t.Fatalf("ListOrganizationUsers error: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("users = %#v", users)
	}
	if users[0].Email != "service@example.com" || users[0].MembershipID != "membership-1" || users[0].Status != "active" {
		t.Fatalf("user = %#v", users[0])
	}
}

func TestAppwriteIdentityUnauthorizedOrganizationAndStoragePaths(t *testing.T) {
	appwriteAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"unauthorized"}`, http.StatusUnauthorized)
	}))
	defer appwriteAPI.Close()

	identity := NewAppwriteIdentity(appwriteAPI.URL+"/v1", "project-1", "api-key-1", appwriteAPI.Client())

	tests := []struct {
		name string
		run  func() error
	}{
		{
			name: "create organization",
			run: func() error {
				_, err := identity.CreateOrganization(context.Background(), "session-secret", "Acme Org")
				return err
			},
		},
		{
			name: "accept invite",
			run: func() error {
				_, err := identity.AcceptInvite(context.Background(), "acme", "membership-1", "user-1", "secret-1")
				return err
			},
		},
		{
			name: "get user by email",
			run: func() error {
				_, err := identity.GetUserByEmail(context.Background(), "user@example.com")
				return err
			},
		},
		{
			name: "invite user",
			run: func() error {
				_, err := identity.InviteOrganizationUser(context.Background(), "session-secret", "acme", "user@example.com", "https://attesta.example/invite/accept", []string{"approver"}, false)
				return err
			},
		},
		{
			name: "list organization users",
			run: func() error {
				_, err := identity.ListOrganizationUsers(context.Background(), "acme")
				return err
			},
		},
		{
			name: "update organization",
			run: func() error {
				_, err := identity.UpdateOrganization(context.Background(), "session-secret", "acme", "Updated Org", "", nil)
				return err
			},
		},
		{
			name: "update organization membership",
			run: func() error {
				_, err := identity.UpdateOrganizationMembership(context.Background(), "session-secret", "acme", "membership-1", []string{"approver"}, false)
				return err
			},
		},
		{
			name: "update user labels",
			run: func() error {
				_, err := identity.UpdateUserLabels(context.Background(), "user-1", []string{encodeIdentityRoleLabel("approver")})
				return err
			},
		},
		{
			name: "delete organization membership",
			run: func() error {
				return identity.DeleteOrganizationMembership(context.Background(), "session-secret", "acme", "membership-1")
			},
		},
		{
			name: "upload organization logo",
			run: func() error {
				_, err := identity.UploadOrganizationLogo(context.Background(), "acme", IdentityFile{
					Filename: "logo.png",
					Data:     []byte("PNG"),
				})
				return err
			},
		},
		{
			name: "get organization logo",
			run: func() error {
				_, err := identity.GetOrganizationLogo(context.Background(), "logo-1")
				return err
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); !errors.Is(err, ErrIdentityUnauthorized) {
				t.Fatalf("error = %v, want %v", err, ErrIdentityUnauthorized)
			}
		})
	}
}

func TestAppwriteIdentityCreateOrganizationKeepsExistingAdminLabel(t *testing.T) {
	var updatedLabelsBody map[string]interface{}

	appwriteAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/teams":
			_, _ = w.Write([]byte(`{"$id":"fresh-org","name":"Fresh Org","prefs":{"schemaVersion":1,"slug":"fresh-org"}}`))
		case "/v1/teams/fresh-org/prefs":
			_, _ = w.Write([]byte(`{"schemaVersion":1,"slug":"fresh-org"}`))
		case "/v1/account":
			_, _ = w.Write([]byte(`{"$id":"user-1","email":"owner@example.com","status":true,"labels":["attesta:org-admin"]}`))
		case "/v1/users/user-1/labels":
			if err := json.NewDecoder(r.Body).Decode(&updatedLabelsBody); err != nil {
				t.Fatalf("decode labels body: %v", err)
			}
			_, _ = w.Write([]byte(`{"$id":"user-1","email":"owner@example.com","status":true,"labels":["attesta:org-admin"]}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer appwriteAPI.Close()

	identity := NewAppwriteIdentity(appwriteAPI.URL+"/v1", "project-1", "api-key-1", appwriteAPI.Client())

	if _, err := identity.CreateOrganization(context.Background(), "session-secret", "Fresh Org"); err != nil {
		t.Fatalf("CreateOrganization error: %v", err)
	}
	labels, _ := updatedLabelsBody["labels"].([]interface{})
	if len(labels) != 1 || labels[0] != identityOrgAdminLabel {
		t.Fatalf("labels body = %#v", updatedLabelsBody)
	}
}

func TestAppwriteIdentityAcceptInviteSessionFailure(t *testing.T) {
	appwriteAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/teams/acme/memberships/membership-1/status":
			http.SetCookie(w, &http.Cookie{Name: "a_session_project-1", Value: "session-secret", Path: "/"})
			_, _ = w.Write([]byte(`{"$id":"membership-1","teamId":"acme","userId":"user-1","confirm":true,"roles":["member"]}`))
		case "/v1/account/sessions/current":
			http.Error(w, `{"message":"session failed"}`, http.StatusInternalServerError)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer appwriteAPI.Close()

	identity := NewAppwriteIdentity(appwriteAPI.URL+"/v1", "project-1", "api-key-1", appwriteAPI.Client())

	if _, err := identity.AcceptInvite(context.Background(), "acme", "membership-1", "user-1", "secret-1"); err == nil {
		t.Fatal("expected AcceptInvite error")
	}
}

func TestNewAppwriteIdentitySessionClientHasCookieJar(t *testing.T) {
	identityStore := NewAppwriteIdentity("http://example.invalid/v1", "project-1", "api-key-1", http.DefaultClient)
	identity, ok := identityStore.(*appwriteIdentity)
	if !ok {
		t.Fatalf("identity type = %T", identityStore)
	}
	if identity.sessionClient.Client == nil {
		t.Fatal("session client http client is nil")
	}
	if identity.sessionClient.Client.Jar == nil {
		t.Fatal("session client cookie jar is nil")
	}
	if identity.adminClient.Client == nil {
		t.Fatal("admin client http client is nil")
	}
	if identity.adminClient.Client.Jar != nil {
		t.Fatal("admin client cookie jar should be nil")
	}
}

func TestAppwriteIdentityGetOrganizationLogoViewFailure(t *testing.T) {
	appwriteAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/storage/buckets/org-assets/files/logo-1":
			_, _ = w.Write([]byte(`{"$id":"logo-1","name":"logo.png","mimeType":"image/png"}`))
		case "/v1/storage/buckets/org-assets/files/logo-1/view":
			http.Error(w, `{"message":"missing"}`, http.StatusNotFound)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer appwriteAPI.Close()

	identity := NewAppwriteIdentity(appwriteAPI.URL+"/v1", "project-1", "api-key-1", appwriteAPI.Client())

	if _, err := identity.GetOrganizationLogo(context.Background(), "logo-1"); !errors.Is(err, ErrIdentityNotFound) {
		t.Fatalf("GetOrganizationLogo error = %v, want %v", err, ErrIdentityNotFound)
	}
}

func TestAppwriteIdentityDirectTeamLookupAndCurrentUserFallback(t *testing.T) {
	appwriteAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/account":
			_, _ = w.Write([]byte(`{"$id":"user-1","email":"org-admin@example.com","status":true,"labels":["attesta:org-admin"]}`))
		case "/v1/users/user-1/memberships":
			_, _ = w.Write([]byte(`{"total":1,"memberships":[{"$id":"membership-1","userId":"user-1","teamId":"team-1","teamName":"Acme Org","confirm":true,"roles":["owner"]}]}`))
		case "/v1/teams/missing":
			http.NotFound(w, r)
		case "/v1/teams/team-1":
			http.NotFound(w, r)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer appwriteAPI.Close()

	identityStore := NewAppwriteIdentity(appwriteAPI.URL+"/v1", "project-1", "api-key-1", appwriteAPI.Client())
	identity := identityStore.(*appwriteIdentity)

	org, err := identity.getOrganizationByTeamID(context.Background(), "missing")
	if !errors.Is(err, ErrIdentityNotFound) || org != nil {
		t.Fatalf("getOrganizationByTeamID = %#v %v", org, err)
	}

	user, err := identity.GetCurrentUser(context.Background(), "session-secret")
	if err != nil {
		t.Fatalf("GetCurrentUser error: %v", err)
	}
	if user.OrgSlug != "team-1" || user.OrgName != "Acme Org" {
		t.Fatalf("user = %#v", user)
	}
}

func TestIdentityAppwriteHelpers(t *testing.T) {
	if got := encodeIdentityRoleLabel("   "); got != "" {
		t.Fatalf("encodeIdentityRoleLabel blank = %q, want empty", got)
	}
	t.Setenv("APPWRITE_ORG_ASSETS_BUCKET", "")
	if got := appwriteOrgAssetsBucket(); got != "org-assets" {
		t.Fatalf("appwriteOrgAssetsBucket = %q, want org-assets", got)
	}
	t.Setenv("APPWRITE_ORG_ASSETS_BUCKET", " ORG-ASSETS ")
	if got := appwriteOrgAssetsBucket(); got != "org-assets" {
		t.Fatalf("appwriteOrgAssetsBucket normalized = %q, want org-assets", got)
	}
	if _, err := parseAppwriteTime(""); err == nil {
		t.Fatal("expected parseAppwriteTime error")
	}

	selected := selectPrimaryMembership([]models.Membership{
		{Id: "pending-1", Confirm: false},
		{Id: "pending-2", Confirm: false},
	})
	if selected == nil || selected.Id != "pending-1" {
		t.Fatalf("selected = %#v, want pending-1", selected)
	}
	if !hasMembershipRole([]string{"member", "owner"}, identityMembershipOwnerRole) {
		t.Fatal("expected owner membership role")
	}
	if hasMembershipRole([]string{"member"}, identityMembershipOwnerRole) {
		t.Fatal("did not expect owner membership role")
	}

	client := appwriteclient.Client{}
	if got := appwriteSessionSecretFromJar(client); got != "" {
		t.Fatalf("appwriteSessionSecretFromJar without http client = %q", got)
	}
	client.Endpoint = "://bad-url"
	client.Client = &http.Client{}
	if got := appwriteSessionSecretFromJar(client); got != "" {
		t.Fatalf("appwriteSessionSecretFromJar with invalid endpoint = %q", got)
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar.New error: %v", err)
	}
	parsed, err := url.Parse("https://appwrite.example/v1")
	if err != nil {
		t.Fatalf("url.Parse error: %v", err)
	}
	jar.SetCookies(parsed, []*http.Cookie{
		{Name: "ignored", Value: "x"},
		{Name: "a_session_project-1", Value: "session-secret"},
	})
	client.Endpoint = parsed.String()
	client.Client = &http.Client{Jar: jar}
	if got := appwriteSessionSecretFromJar(client); got != "session-secret" {
		t.Fatalf("appwriteSessionSecretFromJar = %q, want session-secret", got)
	}
}

func TestAppwriteIdentityUpdateCurrentPassword(t *testing.T) {
	var sessionHeader string
	var body map[string]interface{}

	appwriteAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/v1/account/password" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		sessionHeader = r.Header.Get("X-Appwrite-Session")
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode password body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"$id":"user-1"}`))
	}))
	defer appwriteAPI.Close()

	identity := NewAppwriteIdentity(appwriteAPI.URL+"/v1", "project-1", "api-key-1", appwriteAPI.Client())

	if err := identity.UpdateCurrentPassword(context.Background(), "session-secret", "updated-password"); err != nil {
		t.Fatalf("UpdateCurrentPassword error: %v", err)
	}
	if sessionHeader != "session-secret" {
		t.Fatalf("session header = %q, want session-secret", sessionHeader)
	}
	if body["password"] != "updated-password" {
		t.Fatalf("body = %#v", body)
	}
}

func TestAppwriteIdentityCanceledContext(t *testing.T) {
	identity := NewAppwriteIdentity("http://example.invalid/v1", "project-1", "api-key-1", nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name string
		run  func() error
	}{
		{
			name: "create session",
			run: func() error {
				_, err := identity.CreateEmailPasswordSession(ctx, "user@example.com", "pw")
				return err
			},
		},
		{
			name: "create account",
			run: func() error {
				_, err := identity.CreateAccount(ctx, "user@example.com", "password", "")
				return err
			},
		},
		{
			name: "create organization",
			run: func() error {
				_, err := identity.CreateOrganization(ctx, "session-secret", "Acme Org")
				return err
			},
		},
		{
			name: "accept invite",
			run: func() error {
				_, err := identity.AcceptInvite(ctx, "acme", "membership-1", "user-1", "secret-1")
				return err
			},
		},
		{
			name: "create recovery",
			run: func() error {
				return identity.CreateRecovery(ctx, "user@example.com", "http://attesta.local/reset/confirm")
			},
		},
		{
			name: "complete recovery",
			run: func() error {
				return identity.CompleteRecovery(ctx, "user-1", "secret-1", "password")
			},
		},
		{
			name: "update current password",
			run: func() error {
				return identity.UpdateCurrentPassword(ctx, "session-secret", "password")
			},
		},
		{
			name: "get session",
			run: func() error {
				_, err := identity.GetSession(ctx, "session-secret")
				return err
			},
		},
		{
			name: "delete session",
			run: func() error {
				return identity.DeleteSession(ctx, "session-secret")
			},
		},
		{
			name: "get current user",
			run: func() error {
				_, err := identity.GetCurrentUser(ctx, "session-secret")
				return err
			},
		},
		{
			name: "get user by id",
			run: func() error {
				_, err := identity.GetUserByID(ctx, "user-1")
				return err
			},
		},
		{
			name: "get user by email",
			run: func() error {
				_, err := identity.GetUserByEmail(ctx, "user@example.com")
				return err
			},
		},
		{
			name: "add organization user by id as admin",
			run: func() error {
				_, err := identity.AddOrganizationUserByIDAsAdmin(ctx, "acme", "user-1", nil, true)
				return err
			},
		},
		{
			name: "invite organization user",
			run: func() error {
				_, err := identity.InviteOrganizationUser(ctx, "session-secret", "acme", "user@example.com", "https://attesta.example/invite/accept", nil, false)
				return err
			},
		},
		{
			name: "invite organization user as admin",
			run: func() error {
				_, err := identity.InviteOrganizationUserAsAdmin(ctx, "acme", "user@example.com", "https://attesta.example/invite/accept", nil, true)
				return err
			},
		},
		{
			name: "list organizations",
			run: func() error {
				_, err := identity.ListOrganizations(ctx)
				return err
			},
		},
		{
			name: "list organization users",
			run: func() error {
				_, err := identity.ListOrganizationUsers(ctx, "acme")
				return err
			},
		},
		{
			name: "list organization memberships",
			run: func() error {
				_, err := identity.ListOrganizationMemberships(ctx, "acme")
				return err
			},
		},
		{
			name: "get organization by slug",
			run: func() error {
				_, err := identity.GetOrganizationBySlug(ctx, "acme")
				return err
			},
		},
		{
			name: "create organization as admin",
			run: func() error {
				_, err := identity.CreateOrganizationAsAdmin(ctx, "Acme Org")
				return err
			},
		},
		{
			name: "update organization",
			run: func() error {
				_, err := identity.UpdateOrganization(ctx, "session-secret", "acme", "Updated Org", "", nil)
				return err
			},
		},
		{
			name: "update organization as admin",
			run: func() error {
				_, err := identity.UpdateOrganizationAsAdmin(ctx, "acme", "Updated Org", "", nil)
				return err
			},
		},
		{
			name: "update organization membership",
			run: func() error {
				_, err := identity.UpdateOrganizationMembership(ctx, "session-secret", "acme", "membership-1", nil, false)
				return err
			},
		},
		{
			name: "update organization membership as admin",
			run: func() error {
				_, err := identity.UpdateOrganizationMembershipAsAdmin(ctx, "acme", "membership-1", nil, true)
				return err
			},
		},
		{
			name: "update user labels",
			run: func() error {
				_, err := identity.UpdateUserLabels(ctx, "user-1", nil)
				return err
			},
		},
		{
			name: "delete organization membership",
			run: func() error {
				return identity.DeleteOrganizationMembership(ctx, "session-secret", "acme", "membership-1")
			},
		},
		{
			name: "upload organization logo",
			run: func() error {
				_, err := identity.UploadOrganizationLogo(ctx, "acme", IdentityFile{Filename: "logo.png", Data: []byte("PNG")})
				return err
			},
		},
		{
			name: "get organization logo",
			run: func() error {
				_, err := identity.GetOrganizationLogo(ctx, "logo-1")
				return err
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); !errors.Is(err, context.Canceled) {
				t.Fatalf("error = %v, want %v", err, context.Canceled)
			}
		})
	}
}

func TestAppwriteIdentityNormalizesUnauthorizedErrors(t *testing.T) {
	appwriteAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"unauthorized"}`, http.StatusUnauthorized)
	}))
	defer appwriteAPI.Close()

	identity := NewAppwriteIdentity(appwriteAPI.URL+"/v1", "project-1", "api-key-1", appwriteAPI.Client())

	if _, err := identity.CreateAccount(context.Background(), "user@example.com", "password", ""); !errors.Is(err, ErrIdentityUnauthorized) {
		t.Fatalf("CreateAccount error = %v, want %v", err, ErrIdentityUnauthorized)
	}
	if err := identity.CreateRecovery(context.Background(), "user@example.com", "http://attesta.local/reset/confirm"); !errors.Is(err, ErrIdentityUnauthorized) {
		t.Fatalf("CreateRecovery error = %v, want %v", err, ErrIdentityUnauthorized)
	}
	if err := identity.CompleteRecovery(context.Background(), "user-1", "secret-1", "password"); !errors.Is(err, ErrIdentityUnauthorized) {
		t.Fatalf("CompleteRecovery error = %v, want %v", err, ErrIdentityUnauthorized)
	}
}

func TestAppwriteIdentityAdditionalUnauthorizedPaths(t *testing.T) {
	appwriteAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"unauthorized"}`, http.StatusUnauthorized)
	}))
	defer appwriteAPI.Close()

	identity := NewAppwriteIdentity(appwriteAPI.URL+"/v1", "project-1", "api-key-1", appwriteAPI.Client())

	if _, err := identity.CreateEmailPasswordSession(context.Background(), "user@example.com", "password"); !errors.Is(err, ErrIdentityUnauthorized) {
		t.Fatalf("CreateEmailPasswordSession error = %v, want %v", err, ErrIdentityUnauthorized)
	}
	if _, err := identity.GetSession(context.Background(), "session-secret"); !errors.Is(err, ErrIdentityUnauthorized) {
		t.Fatalf("GetSession error = %v, want %v", err, ErrIdentityUnauthorized)
	}
	if _, err := identity.GetCurrentUser(context.Background(), "session-secret"); !errors.Is(err, ErrIdentityUnauthorized) {
		t.Fatalf("GetCurrentUser error = %v, want %v", err, ErrIdentityUnauthorized)
	}
	if _, err := identity.ListOrganizations(context.Background()); !errors.Is(err, ErrIdentityUnauthorized) {
		t.Fatalf("ListOrganizations error = %v, want %v", err, ErrIdentityUnauthorized)
	}
}

func TestIdentityAppwriteDecodeFallbacks(t *testing.T) {
	orgs := decodeIdentityOrgs(&models.TeamList{
		Teams: []models.Team{{Id: "acme", Name: "Acme Org"}},
	})
	if len(orgs) != 1 || orgs[0].Slug != "acme" || orgs[0].LogoFileID != "" {
		t.Fatalf("orgs = %#v", orgs)
	}

	org := decodeIdentityOrg(&models.Team{Id: "beta", Name: "Beta Org"})
	if org.Slug != "beta" || org.Name != "Beta Org" || org.LogoFileID != "" {
		t.Fatalf("org = %#v", org)
	}
}

func TestCloneAppwriteClientOptionError(t *testing.T) {
	base := appwriteclient.Client{
		Headers: map[string]string{"X-Test": "value"},
	}

	_, err := cloneAppwriteClient(base, func(clt *appwriteclient.Client) error {
		return errors.New("clone failed")
	})
	if err == nil || err.Error() != "clone failed" {
		t.Fatalf("error = %v, want clone failed", err)
	}
}

func TestIdentityAppwriteNilAndErrorBranches(t *testing.T) {
	if got := decodeIdentityOrgs(nil); got != nil {
		t.Fatalf("decodeIdentityOrgs(nil) = %#v, want nil", got)
	}
	if got := decodeIdentityOrg(nil); got.ID != "" || got.Slug != "" || got.Name != "" || got.LogoFileID != "" || len(got.Roles) != 0 {
		t.Fatalf("decodeIdentityOrg(nil) = %#v", got)
	}
	if got := selectPrimaryMembership(nil); got != nil {
		t.Fatalf("selectPrimaryMembership(nil) = %#v, want nil", got)
	}
	if err := normalizeIdentityError(nil); err != nil {
		t.Fatalf("normalizeIdentityError(nil) = %v, want nil", err)
	}
	if _, err := toIdentitySession(nil, ""); err == nil {
		t.Fatal("expected nil session error")
	}
	if _, err := toIdentitySession(&models.Session{
		UserId: "user-1",
		Expire: "2026-03-18T10:11:12Z",
	}, ""); err == nil {
		t.Fatal("expected missing secret error")
	}
}
