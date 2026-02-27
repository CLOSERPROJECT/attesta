package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

type WorkflowDef struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" yaml:"-"`
	Name        string             `bson:"name" yaml:"name"`
	Description string             `bson:"description,omitempty" yaml:"description,omitempty"`
	Steps       []WorkflowStep     `bson:"steps" yaml:"steps"`
}

type WorkflowStep struct {
	StepID           string        `bson:"stepId" yaml:"id"`
	Title            string        `bson:"title" yaml:"title"`
	Order            int           `bson:"order" yaml:"order"`
	OrganizationSlug string        `bson:"organization,omitempty" yaml:"organization"`
	Substep          []WorkflowSub `bson:"substeps" yaml:"substeps"`
}

type WorkflowSub struct {
	SubstepID string                 `bson:"substepId" yaml:"id"`
	Title     string                 `bson:"title" yaml:"title"`
	Order     int                    `bson:"order" yaml:"order"`
	Role      string                 `bson:"role,omitempty" yaml:"role,omitempty"`
	Roles     []string               `bson:"roles,omitempty" yaml:"roles,omitempty"`
	InputKey  string                 `bson:"inputKey" yaml:"inputKey"`
	InputType string                 `bson:"inputType" yaml:"inputType"`
	Schema    map[string]interface{} `bson:"schema,omitempty" yaml:"schema,omitempty"`
	UISchema  map[string]interface{} `bson:"uiSchema,omitempty" yaml:"uiSchema,omitempty"`
}

type Process struct {
	ID            primitive.ObjectID     `bson:"_id,omitempty"`
	WorkflowDefID primitive.ObjectID     `bson:"workflowDefId"`
	WorkflowKey   string                 `bson:"workflowKey,omitempty"`
	CreatedAt     time.Time              `bson:"createdAt"`
	CreatedBy     string                 `bson:"createdBy"`
	Status        string                 `bson:"status"`
	Progress      map[string]ProcessStep `bson:"progress"`
	DPP           *ProcessDPP            `bson:"dpp,omitempty"`
}

type ProcessDPP struct {
	GTIN        string    `bson:"gtin"`
	Lot         string    `bson:"lot"`
	Serial      string    `bson:"serial"`
	GeneratedAt time.Time `bson:"generatedAt"`
}

type ProcessStep struct {
	State  string                 `bson:"state"`
	DoneAt *time.Time             `bson:"doneAt,omitempty"`
	DoneBy *Actor                 `bson:"doneBy,omitempty"`
	Data   map[string]interface{} `bson:"data,omitempty"`
}

type Actor struct {
	UserID      string   `bson:"userId"`
	Role        string   `bson:"role"`
	OrgSlug     string   `bson:"orgSlug,omitempty"`
	RoleSlugs   []string `bson:"roleSlugs,omitempty"`
	WorkflowKey string   `bson:"workflowKey,omitempty"`
}

type Notarization struct {
	ID         primitive.ObjectID     `bson:"_id,omitempty"`
	ProcessID  primitive.ObjectID     `bson:"processId"`
	SubstepID  string                 `bson:"substepId"`
	Payload    map[string]interface{} `bson:"payload"`
	Actor      Actor                  `bson:"actor"`
	CreatedAt  time.Time              `bson:"createdAt"`
	FakeNotary FakeNotary             `bson:"fakeNotary"`
}

type FakeNotary struct {
	Method string `bson:"method"`
	Digest string `bson:"digest"`
}

type AttachmentPayload struct {
	AttachmentID string
	Filename     string
	ContentType  string
	Size         int64
	SHA256       string
}

type Server struct {
	mongo          *mongo.Client
	store          Store
	tmpl           *template.Template
	authorizer     Authorizer
	sse            *SSEHub
	now            func() time.Time
	configProvider func() (RuntimeConfig, error)
	workflowDefID  primitive.ObjectID
	configDir      string
	configMu       sync.Mutex
	catalogModTime map[string]time.Time
	catalog        map[string]RuntimeConfig
	viteDevServer  string
	enforceAuth    bool
}

type SSEHub struct {
	mu     sync.Mutex
	stream map[string]map[chan string]struct{}
}

type TimelineSubstep struct {
	SubstepID    string
	Title        string
	Role         string
	RoleBadges   []TimelineRoleBadge
	RoleLabel    string
	RoleColor    string
	RoleBorder   string
	Status       string
	DoneBy       string
	DoneRole     string
	DoneAt       string
	DisplayValue string
	FileName     string
	FileSHA256   string
	FileURL      string
}

type TimelineRoleBadge struct {
	ID     string
	Label  string
	Color  string
	Border string
}

type TimelineStep struct {
	StepID   string
	Title    string
	Substeps []TimelineSubstep
}

type NotarizedAttachment struct {
	AttachmentID string `json:"attachment_id"`
	Filename     string `json:"filename"`
	ContentType  string `json:"content_type"`
	SizeBytes    int64  `json:"size_bytes"`
	SHA256       string `json:"sha256"`
}

type NotarizedSubstep struct {
	SubstepID  string                 `json:"substep_id"`
	Title      string                 `json:"title"`
	Role       string                 `json:"role"`
	Status     string                 `json:"status"`
	DoneAt     string                 `json:"done_at,omitempty"`
	DoneBy     string                 `json:"done_by,omitempty"`
	DoneRole   string                 `json:"done_role,omitempty"`
	Payload    map[string]interface{} `json:"payload,omitempty"`
	Digest     string                 `json:"digest,omitempty"`
	Attachment *NotarizedAttachment   `json:"attachment,omitempty"`
}

type NotarizedStep struct {
	StepID   string             `json:"step_id"`
	Title    string             `json:"title"`
	Substeps []NotarizedSubstep `json:"substeps"`
}

type MerkleLeaf struct {
	SubstepID string `json:"substep_id"`
	Hash      string `json:"hash"`
}

type MerkleTree struct {
	Leaves []MerkleLeaf `json:"leaves"`
	Levels [][]string   `json:"levels"`
	Root   string       `json:"root"`
}

type NotarizedProcessExport struct {
	ProcessID string          `json:"process_id"`
	CreatedAt string          `json:"created_at"`
	Status    string          `json:"status"`
	Steps     []NotarizedStep `json:"steps"`
	Merkle    MerkleTree      `json:"merkle"`
}

type ActionView struct {
	ProcessID     string
	SubstepID     string
	Title         string
	Role          string
	AllowedRoles  []string
	RoleBadges    []ActionRoleBadge
	MatchingRoles []string
	RoleLabel     string
	RoleColor     string
	RoleBorder    string
	InputKey      string
	InputType     string
	FormSchema    string
	FormUISchema  string
	Status        string
	DoneAt        string
	DoneBy        string
	DoneRole      string
	Values        []ActionKV
	Attachments   []ActionAttachmentView
	Disabled      bool
	Reason        string
}

type ActionRoleBadge struct {
	ID     string
	Label  string
	Color  string
	Border string
}

type ActionKV struct {
	Key   string
	Value string
}

type ActionAttachmentView struct {
	Filename string
	URL      string
	SHA256   string
}

type ActionTodo struct {
	ProcessID string
	SubstepID string
	Title     string
	Role      string
	Status    string
}

type Department struct {
	ID     string `yaml:"id"`
	Name   string `yaml:"name"`
	Color  string `yaml:"color"`
	Border string `yaml:"border"`
}

type User struct {
	ID           string `yaml:"id"`
	Name         string `yaml:"name"`
	DepartmentID string `yaml:"departmentId"`
}

type RuntimeConfig struct {
	Workflow      WorkflowDef            `yaml:"workflow"`
	Organizations []WorkflowOrganization `yaml:"organizations"`
	Roles         []WorkflowRole         `yaml:"roles"`
	Departments   []Department           `yaml:"departments"`
	Users         []User                 `yaml:"users"`
	DPP           DPPConfig              `yaml:"dpp"`
}

type WorkflowOrganization struct {
	Slug   string `yaml:"slug"`
	Name   string `yaml:"name"`
	Color  string `yaml:"color"`
	Border string `yaml:"border"`
}

type WorkflowRole struct {
	OrgSlug string `yaml:"orgSlug"`
	Slug    string `yaml:"slug"`
	Name    string `yaml:"name"`
	Color   string `yaml:"color"`
	Border  string `yaml:"border"`
}

type DPPConfig struct {
	Enabled            bool   `yaml:"enabled"`
	GTIN               string `yaml:"gtin"`
	LotInputKey        string `yaml:"lotInputKey"`
	LotDefault         string `yaml:"lotDefault"`
	SerialInputKey     string `yaml:"serialInputKey"`
	SerialStrategy     string `yaml:"serialStrategy"`
	ProductName        string `yaml:"productName"`
	ProductDescription string `yaml:"productDescription"`
	OwnerName          string `yaml:"ownerName"`
}

type RoleMeta struct {
	ID     string
	Label  string
	Color  string
	Border string
}

type UserView struct {
	ID         string
	Name       string
	Role       string
	RoleLabel  string
	RoleColor  string
	RoleBorder string
}

type ProcessSummary struct {
	ID          string
	Status      string
	CreatedAt   string
	NextSubstep string
	NextTitle   string
	NextRole    string
}

type PageBase struct {
	Body          string
	ViteDevServer string
	WorkflowKey   string
	WorkflowName  string
	WorkflowPath  string
	ShowOrgsLink  bool
	ShowMyOrgLink bool
}

type BackofficeLandingView struct {
	PageBase
	Users []UserView
}

type WorkflowOption struct {
	Key         string
	Name        string
	Description string
	Counts      WorkflowProcessCounts
}

type WorkflowProcessCounts struct {
	NotStarted int
	Started    int
	Terminated int
}

type WorkflowPickerView struct {
	PageBase
	Workflows []WorkflowOption
}

type HomeWorkflowPickerView struct {
	WorkflowPickerView
}

type BackofficeWorkflowPickerView struct {
	WorkflowPickerView
}

type DepartmentDashboardView struct {
	PageBase
	CurrentUser     Actor
	RoleLabel       string
	TodoActions     []ActionTodo
	ActiveProcesses []ProcessSummary
	DoneProcesses   []ProcessSummary
}

type DashboardView struct {
	PageBase
	UserID          string
	RoleSlugs       []string
	TodoActions     []ActionTodo
	ActiveProcesses []ProcessSummary
	DoneProcesses   []ProcessSummary
}

type DepartmentProcessView struct {
	PageBase
	CurrentUser Actor
	RoleLabel   string
	ProcessID   string
	Actions     []ActionView
	Error       string
	Timeline    []TimelineStep
}

type ActionListView struct {
	WorkflowKey string
	ProcessID   string
	CurrentUser Actor
	Actions     []ActionView
	Error       string
	Timeline    []TimelineStep
}

type ProcessListItem struct {
	ID              string
	Status          string
	CreatedAt       string
	CreatedAtTime   time.Time
	DoneSubsteps    int
	TotalSubsteps   int
	Percent         int
	LastNotarizedAt string
	LastDigestShort string
}

type HomeView struct {
	PageBase
	LatestProcessID string
	Sort            string
	Processes       []ProcessListItem
	History         []ProcessListItem
}

type LoginView struct {
	PageBase
	Next  string
	Email string
	Error string
}

type InviteView struct {
	PageBase
	Token string
	Email string
	Org   string
	Roles []string
	Error string
}

type ResetRequestView struct {
	PageBase
	Email        string
	ResetLink    string
	Confirmation string
}

type ResetSetView struct {
	PageBase
	Token string
	Error string
}

type PlatformAdminView struct {
	PageBase
	Organizations []Organization
	InviteLink    string
	Error         string
}

type OrgAdminView struct {
	PageBase
	Organization Organization
	Roles        []Role
	Users        []OrgAdminUserRow
	Invites      []OrgAdminInviteRow
	InviteLink   string
	Error        string
}

type OrgAdminRoleOption struct {
	Slug       string
	Name       string
	RoleColor  string
	RoleBorder string
	Selected   bool
}

type OrgAdminUserRow struct {
	UserID      string
	Email       string
	Status      string
	Activated   bool
	RoleOptions []OrgAdminRoleOption
}

type OrgAdminInviteRow struct {
	Email     string
	RoleSlugs []string
	CreatedAt time.Time
	ExpiresAt time.Time
	UsedAt    *time.Time
	Status    string
}

type WorkflowRefValidationError struct {
	Messages []string
}

func (e *WorkflowRefValidationError) Error() string {
	if e == nil || len(e.Messages) == 0 {
		return "workflow references are invalid"
	}
	return "workflow references are invalid: " + strings.Join(e.Messages, "; ")
}

type ProcessPageView struct {
	PageBase
	ProcessID   string
	Timeline    []TimelineStep
	ActionList  ActionListView
	DPPURL      string
	DPPGS1      string
	Attachments []ProcessDownloadAttachment
}

type ProcessDownloadAttachment struct {
	SubstepID string
	Filename  string
	URL       string
}

type DPPPageView struct {
	PageBase
	ProcessID    string
	DigitalLink  string
	GTIN         string
	Lot          string
	Serial       string
	IssuedAt     string
	Workflow     WorkflowDef
	Traceability []DPPTraceabilityStep
	Export       NotarizedProcessExport
}

type DPPTraceabilityStep struct {
	StepID   string
	Title    string
	Substeps []DPPTraceabilitySubstep
}

type DPPTraceabilitySubstep struct {
	SubstepID  string
	Title      string
	Role       string
	Status     string
	DoneAt     string
	DoneBy     string
	Digest     string
	Values     []DPPTraceabilityValue
	FileName   string
	FileSHA256 string
	FileURL    string
}

type DPPTraceabilityValue struct {
	Key   string
	Value string
}

type workflowContextKey struct{}

type workflowContextValue struct {
	Key string
	Cfg RuntimeConfig
}

func main() {
	ctx := context.Background()
	mongoURI := envOr("MONGODB_URI", "mongodb://localhost:27017")
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal(err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal(err)
	}

	db := client.Database("closer_demo")
	tmpl, err := template.ParseGlob("templates/*.html")
	if err != nil {
		log.Fatal(err)
	}

	defaultConfigPath := envOr("WORKFLOW_CONFIG", "config/workflow.yaml")
	configDir := strings.TrimSpace(os.Getenv("WORKFLOW_CONFIG_DIR"))
	if configDir == "" {
		configDir = filepath.Dir(defaultConfigPath)
	}

	server := &Server{
		mongo:         client,
		store:         &MongoStore{db: db},
		tmpl:          tmpl,
		authorizer:    NewCerbosAuthorizer(envOr("CERBOS_URL", "http://localhost:3592"), http.DefaultClient, time.Now),
		sse:           newSSEHub(),
		now:           time.Now,
		workflowDefID: primitive.NewObjectID(),
		configDir:     configDir,
		viteDevServer: strings.TrimRight(strings.TrimSpace(os.Getenv("VITE_DEV_SERVER")), "/"),
		enforceAuth:   true,
	}
	if err := ensureStoreIndexes(ctx, server.store); err != nil {
		log.Printf("warning: failed to ensure auth indexes: %v", err)
	}
	if err := bootstrapPlatformAdmin(ctx, server.store, server.now); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("../web/dist"))))
	mux.HandleFunc("/docs", server.handleDocs)
	mux.HandleFunc("/docs/", server.handleDocs)
	mux.HandleFunc("/01/", server.handleDigitalLinkDPP)
	mux.HandleFunc("/login", server.handleLogin)
	mux.HandleFunc("/logout", server.handleLogout)
	mux.HandleFunc("/invite/", server.handleInvite)
	mux.HandleFunc("/reset", server.handleResetRequest)
	mux.HandleFunc("/reset/", server.handleResetSet)
	mux.HandleFunc("/dashboard", server.handleDashboard)
	mux.HandleFunc("/dashboard/", server.handleDashboard)
	mux.HandleFunc("/admin/orgs", server.handleAdminOrgs)
	mux.HandleFunc("/admin/orgs/", server.handleAdminOrgs)
	mux.HandleFunc("/org-admin/roles", server.handleOrgAdminRoles)
	mux.HandleFunc("/org-admin/users", server.handleOrgAdminUsers)
	mux.HandleFunc("/w/", server.handleWorkflowRoutes)
	mux.HandleFunc("/", server.handleHome)
	mux.HandleFunc("/process/start", server.handleLegacyStartProcess)
	mux.HandleFunc("/process/", server.handleLegacyProcessRoutes)
	mux.HandleFunc("/backoffice", server.handleLegacyBackoffice)
	mux.HandleFunc("/backoffice/", server.handleLegacyBackoffice)
	mux.HandleFunc("/events", server.handleEvents)

	addr := ":3000"
	log.Printf("server listening on %s", addr)
	if err := http.ListenAndServe(addr, logRequests(mux)); err != nil {
		log.Fatal(err)
	}
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func boolEnvOr(key string, fallback bool) bool {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if raw == "" {
		return fallback
	}
	switch raw {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func intEnvOr(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func sessionTTLDays() int {
	days := intEnvOr("SESSION_TTL_DAYS", 30)
	if days <= 0 {
		return 30
	}
	return days
}

func safeNextPath(r *http.Request, fallback string) string {
	next := strings.TrimSpace(r.URL.Query().Get("next"))
	if r.Method == http.MethodPost {
		_ = r.ParseForm()
		if formNext := strings.TrimSpace(r.FormValue("next")); formNext != "" {
			next = formNext
		}
	}
	if next == "" || !strings.HasPrefix(next, "/") {
		return fallback
	}
	return next
}

func shouldSecureCookie(r *http.Request) bool {
	if boolEnvOr("COOKIE_SECURE", false) {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "https") {
		return true
	}
	return r.TLS != nil
}

func newSessionID() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func (s *Server) readSession(r *http.Request) (*Session, error) {
	cookie, err := r.Cookie("attesta_session")
	if err != nil {
		return nil, err
	}
	sessionID := strings.TrimSpace(cookie.Value)
	if sessionID == "" {
		return nil, mongo.ErrNoDocuments
	}
	session, err := s.store.LoadSessionByID(r.Context(), sessionID)
	if err != nil {
		return nil, err
	}
	if session.ExpiresAt.Before(s.nowUTC()) {
		_ = s.store.DeleteSession(r.Context(), sessionID)
		return nil, mongo.ErrNoDocuments
	}
	return session, nil
}

func (s *Server) currentUser(r *http.Request) (*AccountUser, *Session, error) {
	session, err := s.readSession(r)
	if err != nil {
		return nil, nil, err
	}
	user, err := s.store.GetUserByUserID(r.Context(), session.UserID)
	if err != nil {
		return nil, nil, err
	}
	return user, session, nil
}

func (s *Server) requireAuthenticatedPage(w http.ResponseWriter, r *http.Request) (*AccountUser, *Session, bool) {
	if !s.enforceAuth {
		return &AccountUser{UserID: "legacy-user"}, nil, true
	}
	user, session, err := s.currentUser(r)
	if err == nil {
		return user, session, true
	}
	target := "/login?next=" + url.QueryEscape(r.URL.RequestURI())
	http.Redirect(w, r, target, http.StatusSeeOther)
	return nil, nil, false
}

func (s *Server) requireAuthenticatedPost(w http.ResponseWriter, r *http.Request) (*AccountUser, *Session, bool) {
	if !s.enforceAuth {
		return &AccountUser{UserID: "legacy-user"}, nil, true
	}
	user, session, err := s.currentUser(r)
	if err == nil {
		return user, session, true
	}
	http.Error(w, "unauthorized", http.StatusUnauthorized)
	return nil, nil, false
}

func bootstrapPlatformAdmin(ctx context.Context, store Store, now func() time.Time) error {
	if store == nil {
		return nil
	}
	anyoneCanCreate := boolEnvOr("ANYONE_CAN_CREATE_ACCOUNT", true)
	adminEmail := strings.ToLower(strings.TrimSpace(os.Getenv("ADMIN_EMAIL")))
	adminPassword := strings.TrimSpace(os.Getenv("ADMIN_PASSWORD"))
	if !anyoneCanCreate && (adminEmail == "" || adminPassword == "") {
		return errors.New("ADMIN_EMAIL and ADMIN_PASSWORD are required when ANYONE_CAN_CREATE_ACCOUNT=false")
	}
	if adminEmail == "" || adminPassword == "" {
		return nil
	}

	if _, err := store.GetUserByEmail(ctx, adminEmail); err == nil {
		return nil
	} else if !errors.Is(err, mongo.ErrNoDocuments) {
		return err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	userID := "platform-admin-" + canonifySlug(strings.Split(adminEmail, "@")[0])
	_, err = store.CreateUser(ctx, AccountUser{
		UserID:          userID,
		Email:           adminEmail,
		PasswordHash:    string(passwordHash),
		RoleSlugs:       []string{},
		Status:          "active",
		IsPlatformAdmin: true,
		CreatedAt:       now().UTC(),
	})
	return err
}

func attachmentMaxBytes() int64 {
	const defaultMaxBytes = int64(25 * 1024 * 1024)
	raw := strings.TrimSpace(os.Getenv("ATTACHMENT_MAX_BYTES"))
	if raw == "" {
		return defaultMaxBytes
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return defaultMaxBytes
	}
	return value
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func workflowPath(key string) string {
	return "/w/" + strings.TrimSpace(key)
}

func (s *Server) pageBase(body, workflowKey, workflowName string) PageBase {
	base := PageBase{
		Body:          body,
		ViteDevServer: s.viteDevServer,
		WorkflowKey:   strings.TrimSpace(workflowKey),
		WorkflowName:  strings.TrimSpace(workflowName),
	}
	if base.WorkflowKey != "" {
		base.WorkflowPath = workflowPath(base.WorkflowKey)
	}
	return base
}

func userIsOrgAdmin(user *AccountUser) bool {
	if user == nil {
		return false
	}
	if !containsRole(user.RoleSlugs, "org-admin") && !containsRole(user.RoleSlugs, "org_admin") {
		return false
	}
	return strings.TrimSpace(user.OrgSlug) != "" && user.OrgID != nil
}

func (s *Server) pageBaseForUser(user *AccountUser, body, workflowKey, workflowName string) PageBase {
	base := s.pageBase(body, workflowKey, workflowName)
	if user == nil {
		return base
	}
	base.ShowOrgsLink = user.IsPlatformAdmin
	base.ShowMyOrgLink = userIsOrgAdmin(user)
	return base
}

func deriveProcessStatus(def WorkflowDef, process *Process) string {
	if process == nil {
		return "active"
	}
	status := strings.TrimSpace(process.Status)
	if status == "" {
		status = "active"
	}
	if status != "done" && isProcessDone(def, process) {
		status = "done"
	}
	return status
}

func workflowProcessCounts(def WorkflowDef, processes []Process) WorkflowProcessCounts {
	counts := WorkflowProcessCounts{}
	for _, process := range processes {
		process.Progress = normalizeProgressKeys(process.Progress)
		status := deriveProcessStatus(def, &process)
		doneCount, _, _ := processProgressStats(def, &process)
		switch {
		case status == "done":
			counts.Terminated++
		case doneCount == 0:
			counts.NotStarted++
		default:
			counts.Started++
		}
	}
	return counts
}

func (s *Server) workflowOptions(ctx context.Context) ([]WorkflowOption, error) {
	catalog, err := s.workflowCatalog()
	if err != nil {
		return nil, err
	}
	keys := sortedWorkflowKeys(catalog)
	options := make([]WorkflowOption, 0, len(keys))
	for _, key := range keys {
		cfg := catalog[key]
		options = append(options, WorkflowOption{
			Key:         key,
			Name:        cfg.Workflow.Name,
			Description: strings.TrimSpace(cfg.Workflow.Description),
			Counts:      WorkflowProcessCounts{},
		})
		if s.store == nil {
			continue
		}
		processes, listErr := s.store.ListRecentProcessesByWorkflow(ctx, key, 0)
		if listErr != nil {
			return nil, listErr
		}
		options[len(options)-1].Counts = workflowProcessCounts(cfg.Workflow, processes)
	}
	return options, nil
}

func (s *Server) selectedWorkflow(r *http.Request) (string, RuntimeConfig, error) {
	if value := r.Context().Value(workflowContextKey{}); value != nil {
		if selected, ok := value.(workflowContextValue); ok {
			if err := s.validateWorkflowRefs(r.Context(), selected.Cfg); err != nil {
				return "", RuntimeConfig{}, err
			}
			return selected.Key, selected.Cfg, nil
		}
	}
	cfg, err := s.runtimeConfig()
	if err != nil {
		return "", RuntimeConfig{}, err
	}
	if err := s.validateWorkflowRefs(r.Context(), cfg); err != nil {
		return "", RuntimeConfig{}, err
	}
	return s.defaultWorkflowKey(), cfg, nil
}

func (s *Server) validateWorkflowRefs(ctx context.Context, cfg RuntimeConfig) error {
	if s == nil || s.store == nil {
		return nil
	}
	if !s.enforceAuth {
		return nil
	}
	if len(cfg.Organizations) == 0 && len(cfg.Roles) == 0 {
		return nil
	}

	messages := []string{}
	yamlOrgs := map[string]struct{}{}
	for _, org := range cfg.Organizations {
		slug := strings.TrimSpace(org.Slug)
		if slug == "" {
			continue
		}
		yamlOrgs[slug] = struct{}{}
		if _, err := s.store.GetOrganizationBySlug(ctx, slug); err != nil {
			messages = append(messages, "missing organization slug "+slug)
		}
	}

	type roleRef struct {
		orgSlug string
	}
	yamlRoles := map[string]roleRef{}
	for _, role := range cfg.Roles {
		orgSlug := strings.TrimSpace(role.OrgSlug)
		roleSlug := strings.TrimSpace(role.Slug)
		if orgSlug == "" || roleSlug == "" {
			continue
		}
		yamlRoles[roleSlug] = roleRef{orgSlug: orgSlug}
		if _, err := s.store.GetRoleBySlug(ctx, orgSlug, roleSlug); err != nil {
			messages = append(messages, "missing role slug "+orgSlug+"/"+roleSlug)
		}
	}

	for _, step := range cfg.Workflow.Steps {
		stepOrg := strings.TrimSpace(step.OrganizationSlug)
		if stepOrg != "" {
			if _, ok := yamlOrgs[stepOrg]; !ok {
				messages = append(messages, "step "+step.StepID+" references organization not in yaml: "+stepOrg)
			}
		}
		for _, sub := range step.Substep {
			roles := sub.Roles
			if len(roles) == 0 && strings.TrimSpace(sub.Role) != "" {
				roles = []string{strings.TrimSpace(sub.Role)}
			}
			if len(roles) == 0 {
				messages = append(messages, "substep "+sub.SubstepID+" has no roles")
				continue
			}
			for _, roleSlug := range roles {
				trimmedRole := strings.TrimSpace(roleSlug)
				roleMeta, ok := yamlRoles[trimmedRole]
				if !ok {
					messages = append(messages, "substep "+sub.SubstepID+" references role not in yaml: "+trimmedRole)
					continue
				}
				if stepOrg != "" && roleMeta.orgSlug != stepOrg {
					messages = append(messages, "substep "+sub.SubstepID+" role "+trimmedRole+" not in step organization "+stepOrg)
					continue
				}
				if _, err := s.store.GetRoleBySlug(ctx, roleMeta.orgSlug, trimmedRole); err != nil {
					messages = append(messages, "missing role slug "+roleMeta.orgSlug+"/"+trimmedRole)
				}
			}
		}
	}

	if len(messages) == 0 {
		return nil
	}
	return &WorkflowRefValidationError{Messages: dedupeStrings(messages)}
}

func dedupeStrings(items []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func normalizeHomeSortKey(value string) string {
	switch value {
	case "time_asc", "time_desc", "progress_asc", "progress_desc", "status":
		return value
	default:
		return "time_desc"
	}
}

func countWorkflowSubsteps(def WorkflowDef) int {
	count := 0
	for _, step := range sortedSteps(def) {
		count += len(step.Substep)
	}
	return count
}

func processProgressStats(def WorkflowDef, process *Process) (doneCount int, lastAt string, lastDigestShort string) {
	if process == nil {
		return 0, "", ""
	}
	var latest time.Time
	first := true
	for _, sub := range orderedSubsteps(def) {
		progress, ok := process.Progress[sub.SubstepID]
		if !ok || progress.State != "done" {
			continue
		}
		doneCount++
		if progress.DoneAt == nil {
			continue
		}
		if first || progress.DoneAt.After(latest) {
			latest = *progress.DoneAt
			first = false
			digest := ""
			if progress.Data != nil {
				digest = digestPayload(progress.Data)
			}
			if len(digest) > 12 {
				digest = digest[:12]
			}
			lastDigestShort = digest
		}
	}
	if !first {
		lastAt = latest.Format(time.RFC3339)
	}
	return doneCount, lastAt, lastDigestShort
}

func sortHomeProcessList(items []ProcessListItem, sortKey string) {
	switch sortKey {
	case "time_asc":
		sort.Slice(items, func(i, j int) bool { return items[i].CreatedAtTime.Before(items[j].CreatedAtTime) })
	case "progress_asc":
		sort.Slice(items, func(i, j int) bool {
			if items[i].Percent == items[j].Percent {
				return items[i].CreatedAtTime.After(items[j].CreatedAtTime)
			}
			return items[i].Percent < items[j].Percent
		})
	case "progress_desc":
		sort.Slice(items, func(i, j int) bool {
			if items[i].Percent == items[j].Percent {
				return items[i].CreatedAtTime.After(items[j].CreatedAtTime)
			}
			return items[i].Percent > items[j].Percent
		})
	case "status":
		sort.Slice(items, func(i, j int) bool {
			if items[i].Status == items[j].Status {
				if items[i].Percent == items[j].Percent {
					return items[i].CreatedAtTime.After(items[j].CreatedAtTime)
				}
				return items[i].Percent > items[j].Percent
			}
			if items[i].Status == "active" {
				return true
			}
			if items[j].Status == "active" {
				return false
			}
			return items[i].Status < items[j].Status
		})
	default:
		sort.Slice(items, func(i, j int) bool { return items[i].CreatedAtTime.After(items[j].CreatedAtTime) })
	}
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	user, _, ok := s.requireAuthenticatedPage(w, r)
	if !ok {
		return
	}
	options, err := s.workflowOptions(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	view := HomeWorkflowPickerView{
		WorkflowPickerView: WorkflowPickerView{
			PageBase:  s.pageBaseForUser(user, "home_picker_body", "", ""),
			Workflows: options,
		},
	}
	if err := s.tmpl.ExecuteTemplate(w, "home.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

const swaggerUIPage = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Attesta API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  <style>
    html, body { margin: 0; padding: 0; }
    #swagger-ui { min-height: 100vh; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.ui = SwaggerUIBundle({
      url: '/docs/openapi3.json',
      dom_id: '#swagger-ui',
      deepLinking: true
    });
  </script>
</body>
</html>`

func (s *Server) handleDocs(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/docs":
		http.Redirect(w, r, "/docs/", http.StatusMovedPermanently)
		return
	case "/docs/":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, swaggerUIPage)
		return
	case "/docs/openapi3.json":
		s.serveOpenAPIFile(w, r, "openapi3.json", "application/json; charset=utf-8")
		return
	case "/docs/openapi3.yaml":
		s.serveOpenAPIFile(w, r, "openapi3.yaml", "application/yaml; charset=utf-8")
		return
	default:
		http.NotFound(w, r)
	}
}

func openAPIDocCandidates(filename string) []string {
	return []string{
		filepath.Join("gen", "http", filename),
		filepath.Join("..", "gen", "http", filename),
		filepath.Join("..", "..", "gen", "http", filename),
	}
}

func (s *Server) serveOpenAPIFile(w http.ResponseWriter, r *http.Request, filename, contentType string) {
	var foundPath string
	for _, candidate := range openAPIDocCandidates(filename) {
		if _, err := os.Stat(candidate); err == nil {
			foundPath = candidate
			break
		}
	}
	if foundPath == "" {
		http.Error(w, "OpenAPI spec not found. Run `task goa:generate`.", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", contentType)
	http.ServeFile(w, r, foundPath)
}

func (s *Server) issueSession(w http.ResponseWriter, r *http.Request, user *AccountUser) error {
	if user == nil {
		return errors.New("user required")
	}
	now := s.nowUTC()
	if err := s.store.SetUserLastLogin(r.Context(), user.UserID, now); err != nil {
		return err
	}
	sessionID, err := newSessionID()
	if err != nil {
		return err
	}
	expiresAt := now.Add(time.Duration(sessionTTLDays()) * 24 * time.Hour)
	session := Session{
		SessionID:   sessionID,
		UserID:      user.UserID,
		UserMongoID: user.ID,
		OrgID:       user.OrgID,
		CreatedAt:   now,
		LastLoginAt: now,
		ExpiresAt:   expiresAt,
	}
	if _, err := s.store.CreateSession(r.Context(), session); err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "attesta_session",
		Value:    sessionID,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   shouldSecureCookie(r),
	})
	return nil
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		view := LoginView{
			PageBase: s.pageBase("login_body", "", ""),
			Next:     safeNextPath(r, "/"),
		}
		if err := s.tmpl.ExecuteTemplate(w, "login.html", view); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
		password := strings.TrimSpace(r.FormValue("password"))
		next := safeNextPath(r, "/")

		user, err := s.store.GetUserByEmail(r.Context(), email)
		if err != nil || user == nil {
			view := LoginView{
				PageBase: s.pageBase("login_body", "", ""),
				Email:    email,
				Next:     next,
				Error:    "Invalid email or password.",
			}
			w.WriteHeader(http.StatusUnauthorized)
			_ = s.tmpl.ExecuteTemplate(w, "login.html", view)
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
			view := LoginView{
				PageBase: s.pageBase("login_body", "", ""),
				Email:    email,
				Next:     next,
				Error:    "Invalid email or password.",
			}
			w.WriteHeader(http.StatusUnauthorized)
			_ = s.tmpl.ExecuteTemplate(w, "login.html", view)
			return
		}

		if err := s.issueSession(w, r, user); err != nil {
			http.Error(w, "login failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, next, http.StatusSeeOther)
		return
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if cookie, err := r.Cookie("attesta_session"); err == nil && strings.TrimSpace(cookie.Value) != "" {
		_ = s.store.DeleteSession(r.Context(), strings.TrimSpace(cookie.Value))
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "attesta_session",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   shouldSecureCookie(r),
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func validatePassword(value string) error {
	password := strings.TrimSpace(value)
	if len(password) < 12 {
		return errors.New("password must be at least 12 characters")
	}
	return nil
}

func (s *Server) loadActiveInvite(ctx context.Context, token string) (*Invite, error) {
	invite, err := s.store.LoadInviteByTokenHash(ctx, token)
	if err != nil {
		return nil, err
	}
	now := s.nowUTC()
	if invite.UsedAt != nil {
		return nil, errors.New("invite already used")
	}
	if !invite.ExpiresAt.IsZero() && invite.ExpiresAt.Before(now) {
		return nil, errors.New("invite expired")
	}
	return invite, nil
}

func (s *Server) handleInvite(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/invite/"))
	if token == "" || strings.Contains(token, "/") {
		http.NotFound(w, r)
		return
	}

	invite, err := s.loadActiveInvite(r.Context(), token)
	if err != nil {
		http.Error(w, "invalid or expired invite", http.StatusBadRequest)
		return
	}

	orgName := invite.OrgID.Hex()
	if org, err := s.store.ListOrganizations(r.Context()); err == nil {
		for _, item := range org {
			if item.ID == invite.OrgID {
				orgName = item.Name
				break
			}
		}
	}

	switch r.Method {
	case http.MethodGet:
		view := InviteView{
			PageBase: s.pageBase("invite_body", "", ""),
			Token:    token,
			Email:    invite.Email,
			Org:      orgName,
			Roles:    invite.RoleSlugs,
		}
		if err := s.tmpl.ExecuteTemplate(w, "invite.html", view); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		password := r.FormValue("password")
		if err := validatePassword(password); err != nil {
			view := InviteView{
				PageBase: s.pageBase("invite_body", "", ""),
				Token:    token,
				Email:    invite.Email,
				Org:      orgName,
				Roles:    invite.RoleSlugs,
				Error:    err.Error(),
			}
			w.WriteHeader(http.StatusBadRequest)
			_ = s.tmpl.ExecuteTemplate(w, "invite.html", view)
			return
		}

		user, err := s.store.GetUserByUserID(r.Context(), invite.UserID)
		if err != nil {
			http.Error(w, "invalid invite user", http.StatusBadRequest)
			return
		}
		if strings.ToLower(strings.TrimSpace(user.Email)) != strings.ToLower(strings.TrimSpace(invite.Email)) {
			http.Error(w, "invite email mismatch", http.StatusBadRequest)
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(strings.TrimSpace(password)), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "failed to set password", http.StatusInternalServerError)
			return
		}
		if err := s.store.SetUserPasswordHash(r.Context(), user.UserID, string(hash)); err != nil {
			http.Error(w, "failed to set password", http.StatusInternalServerError)
			return
		}
		if err := s.store.MarkInviteUsed(r.Context(), token, s.nowUTC()); err != nil {
			http.Error(w, "failed to finalize invite", http.StatusBadRequest)
			return
		}
		updated, err := s.store.GetUserByUserID(r.Context(), user.UserID)
		if err != nil {
			http.Error(w, "failed to load user", http.StatusInternalServerError)
			return
		}
		if err := s.issueSession(w, r, updated); err != nil {
			http.Error(w, "failed to login", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func resetTTLHrs() int {
	hours := intEnvOr("RESET_TTL_HOURS", 24)
	if hours <= 0 {
		return 24
	}
	return hours
}

func (s *Server) handleResetRequest(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		view := ResetRequestView{PageBase: s.pageBase("reset_request_body", "", "")}
		if err := s.tmpl.ExecuteTemplate(w, "reset_request.html", view); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
		view := ResetRequestView{
			PageBase:     s.pageBase("reset_request_body", "", ""),
			Email:        email,
			Confirmation: "If the account exists, a reset link has been created.",
			ResetLink:    "",
		}

		user, err := s.store.GetUserByEmail(r.Context(), email)
		if err == nil && user != nil {
			token, tokenErr := newSessionID()
			if tokenErr != nil {
				http.Error(w, "failed to create reset token", http.StatusInternalServerError)
				return
			}
			_, createErr := s.store.CreatePasswordReset(r.Context(), PasswordReset{
				Email:     email,
				UserID:    user.UserID,
				TokenHash: token,
				ExpiresAt: s.nowUTC().Add(time.Duration(resetTTLHrs()) * time.Hour),
				CreatedAt: s.nowUTC(),
			})
			if createErr == nil {
				view.ResetLink = "/reset/" + token
			}
		}

		if err := s.tmpl.ExecuteTemplate(w, "reset_request.html", view); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) loadActivePasswordReset(ctx context.Context, token string) (*PasswordReset, error) {
	reset, err := s.store.LoadPasswordResetByTokenHash(ctx, token)
	if err != nil {
		return nil, err
	}
	now := s.nowUTC()
	if reset.UsedAt != nil {
		return nil, errors.New("reset token already used")
	}
	if !reset.ExpiresAt.IsZero() && reset.ExpiresAt.Before(now) {
		return nil, errors.New("reset token expired")
	}
	return reset, nil
}

func (s *Server) handleResetSet(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/reset/"))
	if token == "" || strings.Contains(token, "/") {
		http.NotFound(w, r)
		return
	}
	reset, err := s.loadActivePasswordReset(r.Context(), token)
	if err != nil {
		http.Error(w, "invalid or expired reset token", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		view := ResetSetView{
			PageBase: s.pageBase("reset_set_body", "", ""),
			Token:    token,
		}
		if err := s.tmpl.ExecuteTemplate(w, "reset_set.html", view); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		password := r.FormValue("password")
		if err := validatePassword(password); err != nil {
			view := ResetSetView{
				PageBase: s.pageBase("reset_set_body", "", ""),
				Token:    token,
				Error:    err.Error(),
			}
			w.WriteHeader(http.StatusBadRequest)
			_ = s.tmpl.ExecuteTemplate(w, "reset_set.html", view)
			return
		}

		user, err := s.store.GetUserByUserID(r.Context(), reset.UserID)
		if err != nil {
			http.Error(w, "invalid reset user", http.StatusBadRequest)
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(strings.TrimSpace(password)), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "failed to reset password", http.StatusInternalServerError)
			return
		}
		if err := s.store.SetUserPasswordHash(r.Context(), user.UserID, string(hash)); err != nil {
			http.Error(w, "failed to reset password", http.StatusInternalServerError)
			return
		}
		if err := s.store.MarkPasswordResetUsed(r.Context(), token, s.nowUTC()); err != nil {
			http.Error(w, "failed to finalize reset", http.StatusBadRequest)
			return
		}
		updated, err := s.store.GetUserByUserID(r.Context(), user.UserID)
		if err != nil {
			http.Error(w, "failed to load user", http.StatusInternalServerError)
			return
		}
		if err := s.issueSession(w, r, updated); err != nil {
			http.Error(w, "failed to login", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func randomUserIDFromEmail(email string) string {
	local := strings.TrimSpace(strings.Split(strings.ToLower(strings.TrimSpace(email)), "@")[0])
	if local == "" {
		local = "user"
	}
	return canonifySlug(local) + "-" + fmt.Sprintf("%d", time.Now().UnixNano())
}

func isDuplicateSlugError(err error) bool {
	if err == nil {
		return false
	}
	if mongo.IsDuplicateKeyError(err) {
		return true
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(message, "slug already exists") ||
		strings.Contains(message, "role already exists") ||
		strings.Contains(message, "duplicate key")
}

func requestedRoleSlugs(form url.Values) []string {
	roles := canonifyRoleSlugs(form["roles"])
	if len(roles) > 0 {
		return roles
	}
	legacyRole := strings.TrimSpace(form.Get("role"))
	if legacyRole == "" {
		return []string{}
	}
	return canonifyRoleSlugs([]string{legacyRole})
}

func accountMatchesOrg(user *AccountUser, orgID primitive.ObjectID, orgSlug string) bool {
	if user == nil || user.OrgID == nil {
		return false
	}
	if *user.OrgID != orgID {
		return false
	}
	return strings.TrimSpace(user.OrgSlug) == strings.TrimSpace(orgSlug)
}

func ensureStoreIndexes(ctx context.Context, store Store) error {
	mongoStore, ok := store.(*MongoStore)
	if !ok || mongoStore == nil {
		return nil
	}
	return mongoStore.EnsureAuthIndexes(ctx)
}

func (s *Server) requirePlatformAdmin(w http.ResponseWriter, r *http.Request) (*AccountUser, bool) {
	user, _, ok := s.requireAuthenticatedPage(w, r)
	if !ok {
		return nil, false
	}
	if !user.IsPlatformAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return nil, false
	}
	return user, true
}

func (s *Server) requireOrgAdmin(w http.ResponseWriter, r *http.Request) (*AccountUser, bool) {
	user, _, ok := s.requireAuthenticatedPage(w, r)
	if !ok {
		return nil, false
	}
	if !userIsOrgAdmin(user) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return nil, false
	}
	return user, true
}

func (s *Server) renderPlatformAdmin(w http.ResponseWriter, user *AccountUser, inviteLink, errMsg string) {
	orgs, _ := s.store.ListOrganizations(context.Background())
	view := PlatformAdminView{
		PageBase:      s.pageBaseForUser(user, "platform_admin_body", "", ""),
		Organizations: orgs,
		InviteLink:    strings.TrimSpace(inviteLink),
		Error:         strings.TrimSpace(errMsg),
	}
	if err := s.tmpl.ExecuteTemplate(w, "platform_admin.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleAdminOrgs(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.requirePlatformAdmin(w, r)
	if !ok {
		return
	}
	path := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/admin/orgs"))
	if path == "" || path == "/" {
		switch r.Method {
		case http.MethodGet:
			s.renderPlatformAdmin(w, admin, "", "")
			return
		case http.MethodPost:
			if err := r.ParseForm(); err != nil {
				http.Error(w, "invalid form", http.StatusBadRequest)
				return
			}
			name := strings.TrimSpace(r.FormValue("name"))
			color := strings.TrimSpace(r.FormValue("color"))
			border := strings.TrimSpace(r.FormValue("border"))
			if name == "" {
				s.renderPlatformAdmin(w, admin, "", "organization name is required")
				return
			}
			if existing, err := s.store.GetOrganizationBySlug(r.Context(), canonifySlug(name)); err == nil && existing != nil {
				s.renderPlatformAdmin(w, admin, "", "organization slug already exists")
				return
			}
			if _, err := s.store.CreateOrganization(r.Context(), Organization{
				Name:      name,
				Color:     color,
				Border:    border,
				CreatedAt: s.nowUTC(),
			}); err != nil {
				if isDuplicateSlugError(err) {
					s.renderPlatformAdmin(w, admin, "", "organization slug already exists")
					return
				}
				s.renderPlatformAdmin(w, admin, "", "failed to create organization")
				return
			}
			http.Redirect(w, r, "/admin/orgs", http.StatusSeeOther)
			return
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
	}

	orgSlug := strings.Trim(strings.TrimPrefix(path, "/"), "/")
	org, err := s.store.GetOrganizationBySlug(r.Context(), orgSlug)
	if err != nil || org == nil {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		s.renderPlatformAdmin(w, admin, "", "")
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	if email == "" {
		s.renderPlatformAdmin(w, admin, "", "email is required")
		return
	}

	roleSlug := "org-admin"
	if _, roleErr := s.store.GetRoleBySlug(r.Context(), org.Slug, roleSlug); roleErr != nil {
		_, _ = s.store.CreateRole(r.Context(), Role{
			OrgID:     org.ID,
			OrgSlug:   org.Slug,
			Slug:      roleSlug,
			Name:      "Org Admin",
			Color:     org.Color,
			Border:    org.Border,
			CreatedAt: s.nowUTC(),
		})
	}

	user, userErr := s.store.GetUserByEmail(r.Context(), email)
	if userErr != nil || user == nil {
		orgID := org.ID
		createdUser, createErr := s.store.CreateUser(r.Context(), AccountUser{
			UserID:    randomUserIDFromEmail(email),
			OrgID:     &orgID,
			OrgSlug:   org.Slug,
			Email:     email,
			RoleSlugs: []string{roleSlug},
			Status:    "invited",
			CreatedAt: s.nowUTC(),
		})
		if createErr != nil {
			s.renderPlatformAdmin(w, admin, "", "failed to create org admin user")
			return
		}
		user = &createdUser
	} else {
		_ = s.store.SetUserRoles(r.Context(), user.UserID, append(user.RoleSlugs, roleSlug))
	}

	token, tokenErr := newSessionID()
	if tokenErr != nil {
		s.renderPlatformAdmin(w, admin, "", "failed to create invite")
		return
	}
	if _, inviteErr := s.store.CreateInvite(r.Context(), Invite{
		OrgID:           org.ID,
		Email:           email,
		UserID:          user.UserID,
		RoleSlugs:       []string{roleSlug},
		TokenHash:       token,
		ExpiresAt:       s.nowUTC().Add(7 * 24 * time.Hour),
		CreatedAt:       s.nowUTC(),
		CreatedByUserID: admin.UserID,
	}); inviteErr != nil {
		s.renderPlatformAdmin(w, admin, "", "failed to create invite")
		return
	}
	s.renderPlatformAdmin(w, admin, "/invite/"+token, "")
}

func (s *Server) renderOrgAdmin(w http.ResponseWriter, user *AccountUser, orgSlug, inviteLink, errMsg string) {
	org, err := s.store.GetOrganizationBySlug(context.Background(), orgSlug)
	if err != nil || org == nil {
		http.Error(w, "organization not found", http.StatusNotFound)
		return
	}
	roles, _ := s.store.ListRolesByOrg(context.Background(), orgSlug)
	users, _ := s.store.ListUsersByOrgID(context.Background(), org.ID)
	invites := []Invite{}
	if user != nil {
		invites, _ = s.store.ListInvitesByCreator(context.Background(), user.UserID, org.ID)
	}

	orgUsers := make([]OrgAdminUserRow, 0, len(users))
	for _, orgUser := range users {
		if strings.EqualFold(strings.TrimSpace(orgUser.Status), "deleted") {
			continue
		}
		roleOptions := make([]OrgAdminRoleOption, 0, len(roles))
		for _, role := range roles {
			roleOptions = append(roleOptions, OrgAdminRoleOption{
				Slug:       role.Slug,
				Name:       role.Name,
				RoleColor:  role.Color,
				RoleBorder: role.Border,
				Selected:   containsRole(orgUser.RoleSlugs, role.Slug),
			})
		}
		orgUsers = append(orgUsers, OrgAdminUserRow{
			UserID:      orgUser.UserID,
			Email:       orgUser.Email,
			Status:      orgUser.Status,
			Activated:   strings.TrimSpace(orgUser.PasswordHash) != "",
			RoleOptions: roleOptions,
		})
	}

	now := s.nowUTC()
	orgInvites := make([]OrgAdminInviteRow, 0, len(invites))
	for _, invite := range invites {
		status := "pending"
		if invite.UsedAt != nil {
			status = "accepted"
		} else if invite.ExpiresAt.Before(now) {
			status = "expired"
		}
		orgInvites = append(orgInvites, OrgAdminInviteRow{
			Email:     invite.Email,
			RoleSlugs: append([]string(nil), invite.RoleSlugs...),
			CreatedAt: invite.CreatedAt,
			ExpiresAt: invite.ExpiresAt,
			UsedAt:    invite.UsedAt,
			Status:    status,
		})
	}

	view := OrgAdminView{
		PageBase:     s.pageBaseForUser(user, "org_admin_body", "", ""),
		Organization: *org,
		Roles:        roles,
		Users:        orgUsers,
		Invites:      orgInvites,
		InviteLink:   strings.TrimSpace(inviteLink),
		Error:        strings.TrimSpace(errMsg),
	}
	if err := s.tmpl.ExecuteTemplate(w, "org_admin.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleOrgAdminRoles(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireOrgAdmin(w, r)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.renderOrgAdmin(w, user, user.OrgSlug, "", "")
		return
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		name := strings.TrimSpace(r.FormValue("name"))
		color := strings.TrimSpace(r.FormValue("color"))
		border := strings.TrimSpace(r.FormValue("border"))
		if name == "" {
			s.renderOrgAdmin(w, user, user.OrgSlug, "", "role name is required")
			return
		}
		if existing, err := s.store.GetRoleBySlug(r.Context(), user.OrgSlug, canonifySlug(name)); err == nil && existing != nil {
			s.renderOrgAdmin(w, user, user.OrgSlug, "", "role slug already exists")
			return
		}
		_, err := s.store.CreateRole(r.Context(), Role{
			OrgID:     *user.OrgID,
			OrgSlug:   user.OrgSlug,
			Name:      name,
			Color:     color,
			Border:    border,
			CreatedAt: s.nowUTC(),
		})
		if err != nil {
			if isDuplicateSlugError(err) {
				s.renderOrgAdmin(w, user, user.OrgSlug, "", "role slug already exists")
				return
			}
			s.renderOrgAdmin(w, user, user.OrgSlug, "", "failed to create role")
			return
		}
		s.renderOrgAdmin(w, user, user.OrgSlug, "", "")
		return
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleOrgAdminUsers(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.requireOrgAdmin(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		s.renderOrgAdmin(w, admin, admin.OrgSlug, "", "")
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	intent := strings.TrimSpace(r.FormValue("intent"))
	if intent == "" {
		intent = "invite"
	}

	switch intent {
	case "invite":
		email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
		if email == "" {
			s.renderOrgAdmin(w, admin, admin.OrgSlug, "", "email is required")
			return
		}

		selectedRoles := requestedRoleSlugs(r.Form)
		for _, roleSlug := range selectedRoles {
			if _, err := s.store.GetRoleBySlug(r.Context(), admin.OrgSlug, roleSlug); err != nil {
				s.renderOrgAdmin(w, admin, admin.OrgSlug, "", "role not found")
				return
			}
		}

		user, userErr := s.store.GetUserByEmail(r.Context(), email)
		if userErr != nil || user == nil {
			orgID := *admin.OrgID
			created, createErr := s.store.CreateUser(r.Context(), AccountUser{
				UserID:    randomUserIDFromEmail(email),
				OrgID:     &orgID,
				OrgSlug:   admin.OrgSlug,
				Email:     email,
				RoleSlugs: selectedRoles,
				Status:    "invited",
				CreatedAt: s.nowUTC(),
			})
			if createErr != nil {
				s.renderOrgAdmin(w, admin, admin.OrgSlug, "", "failed to create user")
				return
			}
			user = &created
		} else {
			if !accountMatchesOrg(user, *admin.OrgID, admin.OrgSlug) {
				s.renderOrgAdmin(w, admin, admin.OrgSlug, "", "email already belongs to another organization")
				return
			}
			mergedRoles := canonifyRoleSlugs(append(append([]string{}, user.RoleSlugs...), selectedRoles...))
			if err := s.store.SetUserRoles(r.Context(), user.UserID, mergedRoles); err != nil {
				s.renderOrgAdmin(w, admin, admin.OrgSlug, "", "failed to update user roles")
				return
			}
		}

		token, tokenErr := newSessionID()
		if tokenErr != nil {
			s.renderOrgAdmin(w, admin, admin.OrgSlug, "", "failed to create invite")
			return
		}
		if _, err := s.store.CreateInvite(r.Context(), Invite{
			OrgID:           *admin.OrgID,
			Email:           email,
			UserID:          user.UserID,
			RoleSlugs:       selectedRoles,
			TokenHash:       token,
			ExpiresAt:       s.nowUTC().Add(7 * 24 * time.Hour),
			CreatedAt:       s.nowUTC(),
			CreatedByUserID: admin.UserID,
		}); err != nil {
			s.renderOrgAdmin(w, admin, admin.OrgSlug, "", "failed to create invite")
			return
		}
		s.renderOrgAdmin(w, admin, admin.OrgSlug, "/invite/"+token, "")
		return
	case "set_roles":
		userID := strings.TrimSpace(r.FormValue("userId"))
		if userID == "" {
			s.renderOrgAdmin(w, admin, admin.OrgSlug, "", "user is required")
			return
		}
		target, err := s.store.GetUserByUserID(r.Context(), userID)
		if err != nil || target == nil {
			s.renderOrgAdmin(w, admin, admin.OrgSlug, "", "user not found")
			return
		}
		if !accountMatchesOrg(target, *admin.OrgID, admin.OrgSlug) {
			s.renderOrgAdmin(w, admin, admin.OrgSlug, "", "user does not belong to your organization")
			return
		}
		selectedRoles := requestedRoleSlugs(r.Form)
		for _, roleSlug := range selectedRoles {
			if _, err := s.store.GetRoleBySlug(r.Context(), admin.OrgSlug, roleSlug); err != nil {
				s.renderOrgAdmin(w, admin, admin.OrgSlug, "", "role not found")
				return
			}
		}
		if target.UserID == admin.UserID && !containsRole(selectedRoles, "org-admin") {
			s.renderOrgAdmin(w, admin, admin.OrgSlug, "", "cannot remove org-admin from your own account")
			return
		}
		if err := s.store.SetUserRoles(r.Context(), target.UserID, selectedRoles); err != nil {
			s.renderOrgAdmin(w, admin, admin.OrgSlug, "", "failed to update user roles")
			return
		}
		s.renderOrgAdmin(w, admin, admin.OrgSlug, "", "")
		return
	case "delete_user":
		userID := strings.TrimSpace(r.FormValue("userId"))
		if userID == "" {
			s.renderOrgAdmin(w, admin, admin.OrgSlug, "", "user is required")
			return
		}
		if userID == admin.UserID {
			s.renderOrgAdmin(w, admin, admin.OrgSlug, "", "cannot delete yourself")
			return
		}
		target, err := s.store.GetUserByUserID(r.Context(), userID)
		if err != nil || target == nil {
			s.renderOrgAdmin(w, admin, admin.OrgSlug, "", "user not found")
			return
		}
		if !accountMatchesOrg(target, *admin.OrgID, admin.OrgSlug) {
			s.renderOrgAdmin(w, admin, admin.OrgSlug, "", "user does not belong to your organization")
			return
		}
		if err := s.store.DisableUser(r.Context(), userID); err != nil {
			s.renderOrgAdmin(w, admin, admin.OrgSlug, "", "failed to delete user")
			return
		}
		s.renderOrgAdmin(w, admin, admin.OrgSlug, "", "")
		return
	default:
		s.renderOrgAdmin(w, admin, admin.OrgSlug, "", "unsupported action")
		return
	}
}

func cloneRequestWithPath(r *http.Request, path string) *http.Request {
	clone := r.Clone(r.Context())
	if clone.URL != nil {
		copied := *clone.URL
		copied.Path = path
		clone.URL = &copied
	}
	clone.RequestURI = path
	return clone
}

func (s *Server) handleWorkflowRoutes(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.requireAuthenticatedPage(w, r); !ok {
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/w/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		http.NotFound(w, r)
		return
	}
	workflowKey := strings.TrimSpace(parts[0])
	cfg, err := s.workflowByKey(workflowKey)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	scopedReq := r.WithContext(context.WithValue(r.Context(), workflowContextKey{}, workflowContextValue{
		Key: workflowKey,
		Cfg: cfg,
	}))
	if len(parts) == 1 || (len(parts) == 2 && parts[1] == "") {
		s.handleWorkflowHome(w, scopedReq)
		return
	}
	rest := "/" + strings.Join(parts[1:], "/")
	switch {
	case rest == "/process/start":
		s.handleStartProcess(w, cloneRequestWithPath(scopedReq, rest))
		return
	case strings.HasPrefix(rest, "/process/"):
		s.handleProcessRoutes(w, cloneRequestWithPath(scopedReq, rest))
		return
	case rest == "/backoffice" || strings.HasPrefix(rest, "/backoffice/"):
		s.handleBackoffice(w, cloneRequestWithPath(scopedReq, rest))
		return
	case rest == "/dashboard" || rest == "/dashboard/" || strings.HasPrefix(rest, "/dashboard/"):
		s.handleDashboard(w, cloneRequestWithPath(scopedReq, rest))
		return
	case rest == "/events":
		s.handleEvents(w, cloneRequestWithPath(scopedReq, rest))
		return
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleWorkflowHome(w http.ResponseWriter, r *http.Request) {
	user, _, ok := s.requireAuthenticatedPage(w, r)
	if !ok {
		return
	}
	workflowKey, cfg, err := s.selectedWorkflow(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctx := r.Context()
	latestID := ""
	if latest, err := s.loadLatestProcess(ctx, workflowKey); err == nil {
		latestID = latest.ID.Hex()
	}
	sortKey := normalizeHomeSortKey(strings.TrimSpace(r.URL.Query().Get("sort")))
	processesRaw, err := s.store.ListRecentProcessesByWorkflow(ctx, workflowKey, 0)
	if err != nil {
		processesRaw = nil
	}

	totalSubsteps := countWorkflowSubsteps(cfg.Workflow)
	var processes []ProcessListItem
	var history []ProcessListItem
	for _, process := range processesRaw {
		process.Progress = normalizeProgressKeys(process.Progress)
		status := deriveProcessStatus(cfg.Workflow, &process)
		doneCount, lastAt, lastDigest := processProgressStats(cfg.Workflow, &process)
		percent := 0
		if totalSubsteps > 0 {
			percent = int(float64(doneCount) / float64(totalSubsteps) * 100)
		}
		item := ProcessListItem{
			ID:              process.ID.Hex(),
			Status:          status,
			CreatedAt:       process.CreatedAt.Format(time.RFC3339),
			CreatedAtTime:   process.CreatedAt,
			DoneSubsteps:    doneCount,
			TotalSubsteps:   totalSubsteps,
			Percent:         percent,
			LastNotarizedAt: lastAt,
			LastDigestShort: lastDigest,
		}
		processes = append(processes, item)
		if status == "done" {
			history = append(history, item)
		}
	}

	sortHomeProcessList(processes, sortKey)
	sortHomeProcessList(history, sortKey)

	view := HomeView{
		PageBase:        s.pageBaseForUser(user, "home_body", workflowKey, cfg.Workflow.Name),
		LatestProcessID: latestID,
		Sort:            sortKey,
		Processes:       processes,
		History:         history,
	}
	if err := s.tmpl.ExecuteTemplate(w, "home.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleStartProcess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	workflowKey, cfg, err := s.selectedWorkflow(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctx := r.Context()
	process := Process{
		WorkflowDefID: s.workflowDefID,
		WorkflowKey:   workflowKey,
		CreatedAt:     s.nowUTC(),
		CreatedBy:     "demo",
		Status:        "active",
		Progress:      map[string]ProcessStep{},
	}
	for _, step := range sortedSteps(cfg.Workflow) {
		for _, sub := range sortedSubsteps(step) {
			process.Progress[encodeProgressKey(sub.SubstepID)] = ProcessStep{State: "pending"}
		}
	}
	id, err := s.store.InsertProcess(ctx, process)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, role := range s.roles(cfg) {
		s.sse.Broadcast("role:"+workflowKey+":"+role, "role-updated")
	}
	http.Redirect(w, r, fmt.Sprintf("%s/process/%s", workflowPath(workflowKey), id.Hex()), http.StatusSeeOther)
}

func (s *Server) defaultWorkflowKey() string {
	catalog, err := s.workflowCatalog()
	if err == nil {
		if _, ok := catalog["workflow"]; ok {
			return "workflow"
		}
		keys := sortedWorkflowKeys(catalog)
		if len(keys) > 0 {
			return keys[0]
		}
	}
	base := strings.TrimSpace(filepath.Base(strings.TrimSpace(s.configDir)))
	if base == "" || base == "." || base == string(filepath.Separator) || base == "config" {
		return "workflow"
	}
	return strings.TrimSpace(base)
}

func (s *Server) resolveLegacyProcessWorkflowKey(ctx context.Context, processID string) (string, bool) {
	process, err := s.loadProcess(ctx, processID)
	if err != nil {
		return "", false
	}
	if key := strings.TrimSpace(process.WorkflowKey); key != "" {
		return key, true
	}
	return s.defaultWorkflowKey(), true
}

func (s *Server) processBelongsToWorkflow(process *Process, workflowKey string) bool {
	if process == nil {
		return false
	}
	current := strings.TrimSpace(process.WorkflowKey)
	if current == workflowKey {
		return true
	}
	return current == "" && workflowKey == s.defaultWorkflowKey()
}

func legacyProcessPath(path string) (processID string, parts []string, ok bool) {
	raw := strings.TrimPrefix(path, "/process/")
	parts = strings.Split(raw, "/")
	if len(parts) == 0 || parts[0] == "" {
		return "", nil, false
	}
	return parts[0], parts, true
}

func (s *Server) handleLegacyStartProcess(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		http.Error(w, "workflow context required", http.StatusBadRequest)
		return
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func (s *Server) handleLegacyProcessRoutes(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.requireAuthenticatedPage(w, r); !ok {
		return
	}
	processID, parts, ok := legacyProcessPath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if len(parts) == 1 && r.Method == http.MethodGet {
		if key, found := s.resolveLegacyProcessWorkflowKey(r.Context(), processID); found {
			http.Redirect(w, r, fmt.Sprintf("/w/%s/process/%s", key, processID), http.StatusSeeOther)
			return
		}
		http.NotFound(w, r)
		return
	}
	if len(parts) == 4 && parts[1] == "substep" && parts[3] == "complete" && r.Method == http.MethodPost {
		http.Error(w, "workflow context required", http.StatusBadRequest)
		return
	}
	s.handleProcessRoutes(w, r)
}

func (s *Server) handleLegacyBackoffice(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.requireAuthenticatedPage(w, r); !ok {
		return
	}
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/backoffice"), "/")
	parts := strings.Split(path, "/")
	if len(parts) == 3 && parts[1] == "process" && r.Method == http.MethodGet {
		processID := strings.TrimSpace(parts[2])
		if processID == "" {
			http.NotFound(w, r)
			return
		}
		if key, found := s.resolveLegacyProcessWorkflowKey(r.Context(), processID); found {
			http.Redirect(w, r, fmt.Sprintf("/w/%s/backoffice/%s/process/%s", key, parts[0], processID), http.StatusSeeOther)
			return
		}
		http.NotFound(w, r)
		return
	}
	s.handleBackoffice(w, r)
}

func (s *Server) handleProcessRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/process/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	processID := parts[0]
	if len(parts) == 1 && r.Method == http.MethodGet {
		s.handleProcessPage(w, r, processID)
		return
	}
	if len(parts) == 2 && parts[1] == "files.zip" && r.Method == http.MethodGet {
		s.handleDownloadAllFiles(w, r, processID)
		return
	}
	if len(parts) == 2 && parts[1] == "notarized.json" && r.Method == http.MethodGet {
		s.handleNotarizedJSON(w, r, processID)
		return
	}
	if len(parts) == 2 && parts[1] == "merkle.json" && r.Method == http.MethodGet {
		s.handleMerkleJSON(w, r, processID)
		return
	}
	if len(parts) == 2 && parts[1] == "timeline" && r.Method == http.MethodGet {
		s.handleTimelinePartial(w, r, processID)
		return
	}
	if len(parts) == 2 && parts[1] == "downloads" && r.Method == http.MethodGet {
		s.handleProcessDownloadsPartial(w, r, processID)
		return
	}
	if len(parts) == 4 && parts[1] == "substep" && parts[3] == "complete" && r.Method == http.MethodPost {
		s.handleCompleteSubstep(w, r, processID, parts[2])
		return
	}
	if len(parts) == 4 && parts[1] == "attachment" && parts[3] == "file" && r.Method == http.MethodGet {
		s.handleDownloadProcessAttachment(w, r, processID, parts[2])
		return
	}
	if len(parts) == 4 && parts[1] == "substep" && parts[3] == "file" && r.Method == http.MethodGet {
		s.handleDownloadSubstepFile(w, r, processID, parts[2])
		return
	}
	http.NotFound(w, r)
}

func (s *Server) handleProcessPage(w http.ResponseWriter, r *http.Request, processID string) {
	user, _, ok := s.requireAuthenticatedPage(w, r)
	if !ok {
		return
	}
	workflowKey, cfg, err := s.selectedWorkflow(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctx := r.Context()
	process, err := s.loadProcess(ctx, processID)
	if err != nil {
		http.Error(w, "process not found", http.StatusNotFound)
		return
	}
	if !s.processBelongsToWorkflow(process, workflowKey) {
		http.Error(w, "process not found", http.StatusNotFound)
		return
	}
	process = s.ensureProcessCompletionArtifacts(ctx, cfg, workflowKey, process)
	timeline := buildTimeline(cfg.Workflow, process, workflowKey, s.roleMetaMap(cfg))
	view := ProcessPageView{PageBase: s.pageBaseForUser(user, "process_body", workflowKey, cfg.Workflow.Name), ProcessID: process.ID.Hex(), Timeline: timeline}
	actor := Actor{
		UserID:      user.UserID,
		OrgSlug:     user.OrgSlug,
		RoleSlugs:   append([]string(nil), user.RoleSlugs...),
		WorkflowKey: workflowKey,
	}
	if len(actor.RoleSlugs) == 0 && !s.enforceAuth {
		actor.RoleSlugs = s.roles(cfg)
	}
	if len(actor.RoleSlugs) > 0 {
		actor.Role = actor.RoleSlugs[0]
	}
	view.ActionList = ActionListView{
		WorkflowKey: workflowKey,
		ProcessID:   process.ID.Hex(),
		CurrentUser: actor,
		Actions:     buildActionList(cfg.Workflow, process, workflowKey, actor, false, s.roleMetaMap(cfg)),
	}
	view.Attachments = buildProcessDownloadAttachments(workflowKey, process, collectProcessAttachments(cfg.Workflow, process))
	if process.DPP != nil {
		view.DPPURL = digitalLinkURL(process.DPP.GTIN, process.DPP.Lot, process.DPP.Serial)
		view.DPPGS1 = gs1ElementString(process.DPP.GTIN, process.DPP.Lot, process.DPP.Serial)
	}
	if err := s.tmpl.ExecuteTemplate(w, "process.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleDigitalLinkDPP(w http.ResponseWriter, r *http.Request) {
	gtin, lot, serial, err := parseDigitalLinkPath(r.URL.Path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	process, err := s.store.LoadProcessByDigitalLink(r.Context(), gtin, lot, serial)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	process.Progress = normalizeProgressKeys(process.Progress)

	workflowKey := strings.TrimSpace(process.WorkflowKey)
	if workflowKey == "" {
		workflowKey = s.defaultWorkflowKey()
	}
	cfg, err := s.workflowByKey(workflowKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	export := buildNotarizedExport(cfg.Workflow, process)
	link := digitalLinkURL(gtin, lot, serial)
	if prefersJSONResponse(r) {
		response := map[string]interface{}{
			"digital_link": link,
			"workflow": map[string]string{
				"key":         workflowKey,
				"name":        cfg.Workflow.Name,
				"description": cfg.Workflow.Description,
			},
			"export": export,
		}
		writeJSON(w, response)
		return
	}

	issuedAt := ""
	if process.DPP != nil && !process.DPP.GeneratedAt.IsZero() {
		issuedAt = process.DPP.GeneratedAt.UTC().Format(time.RFC3339)
	}
	view := DPPPageView{
		PageBase:     s.pageBase("dpp_body", workflowKey, cfg.Workflow.Name),
		ProcessID:    process.ID.Hex(),
		DigitalLink:  link,
		GTIN:         gtin,
		Lot:          lot,
		Serial:       serial,
		IssuedAt:     issuedAt,
		Workflow:     cfg.Workflow,
		Traceability: buildDPPTraceabilityView(cfg.Workflow, process, workflowKey),
		Export:       export,
	}
	if err := s.tmpl.ExecuteTemplate(w, "dpp.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleTimelinePartial(w http.ResponseWriter, r *http.Request, processID string) {
	workflowKey, cfg, err := s.selectedWorkflow(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctx := r.Context()
	process, err := s.loadProcess(ctx, processID)
	if err != nil {
		http.Error(w, "process not found", http.StatusNotFound)
		return
	}
	if !s.processBelongsToWorkflow(process, workflowKey) {
		http.Error(w, "process not found", http.StatusNotFound)
		return
	}
	timeline := buildTimeline(cfg.Workflow, process, workflowKey, s.roleMetaMap(cfg))
	if err := s.tmpl.ExecuteTemplate(w, "timeline.html", timeline); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleProcessDownloadsPartial(w http.ResponseWriter, r *http.Request, processID string) {
	workflowKey, cfg, err := s.selectedWorkflow(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	process, err := s.loadProcess(r.Context(), processID)
	if err != nil {
		http.Error(w, "process not found", http.StatusNotFound)
		return
	}
	if !s.processBelongsToWorkflow(process, workflowKey) {
		http.Error(w, "process not found", http.StatusNotFound)
		return
	}
	process = s.ensureProcessCompletionArtifacts(r.Context(), cfg, workflowKey, process)
	view := ProcessPageView{
		PageBase:  s.pageBase("process_body", workflowKey, cfg.Workflow.Name),
		ProcessID: process.ID.Hex(),
	}
	view.Attachments = buildProcessDownloadAttachments(workflowKey, process, collectProcessAttachments(cfg.Workflow, process))
	if process.DPP != nil {
		view.DPPURL = digitalLinkURL(process.DPP.GTIN, process.DPP.Lot, process.DPP.Serial)
		view.DPPGS1 = gs1ElementString(process.DPP.GTIN, process.DPP.Lot, process.DPP.Serial)
	}
	if err := s.tmpl.ExecuteTemplate(w, "process_downloads", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) ensureProcessCompletionArtifacts(ctx context.Context, cfg RuntimeConfig, workflowKey string, process *Process) *Process {
	if process == nil || !isProcessDone(cfg.Workflow, process) {
		return process
	}

	updated := false
	if strings.TrimSpace(process.Status) != "done" {
		if err := s.store.UpdateProcessStatus(ctx, process.ID, workflowKey, "done"); err != nil {
			log.Printf("failed to persist process status for %s: %v", process.ID.Hex(), err)
		} else {
			updated = true
		}
	}

	if cfg.DPP.Enabled && process.DPP == nil {
		dpp, err := buildProcessDPP(cfg.Workflow, cfg.DPP, process, s.nowUTC())
		if err != nil {
			log.Printf("failed to build dpp for process %s: %v", process.ID.Hex(), err)
		} else if err := s.store.UpdateProcessDPP(ctx, process.ID, workflowKey, dpp); err != nil {
			log.Printf("failed to persist dpp for process %s: %v", process.ID.Hex(), err)
		} else {
			updated = true
		}
	}

	if !updated {
		return process
	}
	reloaded, err := s.store.LoadProcessByID(ctx, process.ID)
	if err != nil {
		log.Printf("failed to reload process %s after completion artifact update: %v", process.ID.Hex(), err)
		return process
	}
	reloaded.Progress = normalizeProgressKeys(reloaded.Progress)
	return reloaded
}

func (s *Server) handleDownloadAllFiles(w http.ResponseWriter, r *http.Request, processID string) {
	workflowKey, cfg, err := s.selectedWorkflow(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	process, err := s.loadProcess(r.Context(), processID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if !s.processBelongsToWorkflow(process, workflowKey) {
		http.NotFound(w, r)
		return
	}

	files := collectProcessAttachments(cfg.Workflow, process)
	filename := fmt.Sprintf("process-%s-files.zip", process.ID.Hex())
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))

	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	manifest := map[string]interface{}{
		"process_id": process.ID.Hex(),
		"generated":  s.nowUTC().Format(time.RFC3339),
		"files":      files,
	}
	if data, err := json.MarshalIndent(manifest, "", "  "); err == nil {
		if entry, err := zipWriter.Create("manifest.json"); err == nil {
			_, _ = entry.Write(data)
		}
	}

	nameCounts := map[string]int{}
	for _, file := range files {
		attachmentID, err := primitive.ObjectIDFromHex(file.AttachmentID)
		if err != nil {
			continue
		}
		download, err := s.store.OpenAttachmentDownload(r.Context(), attachmentID)
		if err != nil {
			continue
		}
		defer download.Close()

		safeName := sanitizeAttachmentFilename(file.Filename)
		baseName := fmt.Sprintf("%s-%s", strings.ReplaceAll(file.SubstepID, ".", "_"), safeName)
		nameCounts[baseName]++
		entryName := baseName
		if nameCounts[baseName] > 1 {
			entryName = fmt.Sprintf("%s-%d", baseName, nameCounts[baseName])
		}
		entry, err := zipWriter.Create(entryName)
		if err != nil {
			continue
		}
		_, _ = io.Copy(entry, download)
	}
}

func (s *Server) handleNotarizedJSON(w http.ResponseWriter, r *http.Request, processID string) {
	workflowKey, cfg, err := s.selectedWorkflow(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	process, err := s.loadProcess(r.Context(), processID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if !s.processBelongsToWorkflow(process, workflowKey) {
		http.NotFound(w, r)
		return
	}
	export := buildNotarizedExport(cfg.Workflow, process)
	writeJSON(w, export)
}

func (s *Server) handleMerkleJSON(w http.ResponseWriter, r *http.Request, processID string) {
	workflowKey, cfg, err := s.selectedWorkflow(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	process, err := s.loadProcess(r.Context(), processID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if !s.processBelongsToWorkflow(process, workflowKey) {
		http.NotFound(w, r)
		return
	}
	export := buildNotarizedExport(cfg.Workflow, process)
	writeJSON(w, export.Merkle)
}

func (s *Server) handleDownloadSubstepFile(w http.ResponseWriter, r *http.Request, processID, substepID string) {
	workflowKey, cfg, err := s.selectedWorkflow(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	process, err := s.loadProcess(r.Context(), processID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if !s.processBelongsToWorkflow(process, workflowKey) {
		http.NotFound(w, r)
		return
	}
	substep, _, err := findSubstep(cfg.Workflow, substepID)
	if err != nil || substep.InputType != "file" {
		http.NotFound(w, r)
		return
	}
	progress, ok := process.Progress[substepID]
	if !ok || progress.State != "done" {
		http.NotFound(w, r)
		return
	}
	attachmentPayload, ok := readAttachmentPayload(progress.Data, substep.InputKey)
	if !ok {
		http.NotFound(w, r)
		return
	}
	attachmentID, err := primitive.ObjectIDFromHex(attachmentPayload.AttachmentID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	attachment, err := s.store.LoadAttachmentByID(r.Context(), attachmentID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	download, err := s.store.OpenAttachmentDownload(r.Context(), attachmentID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer download.Close()

	contentType := strings.TrimSpace(attachment.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	filename := sanitizeAttachmentFilename(attachment.Filename)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	if _, err := io.Copy(w, download); err != nil {
		return
	}
}

func (s *Server) handleDownloadProcessAttachment(w http.ResponseWriter, r *http.Request, processID, attachmentID string) {
	workflowKey, _, err := s.selectedWorkflow(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	process, err := s.loadProcess(r.Context(), processID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if !s.processBelongsToWorkflow(process, workflowKey) {
		http.NotFound(w, r)
		return
	}
	attachmentObjectID, err := primitive.ObjectIDFromHex(strings.TrimSpace(attachmentID))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	attachment, err := s.store.LoadAttachmentByID(r.Context(), attachmentObjectID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if attachment.ProcessID != process.ID {
		http.NotFound(w, r)
		return
	}
	download, err := s.store.OpenAttachmentDownload(r.Context(), attachmentObjectID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer download.Close()

	contentType := strings.TrimSpace(attachment.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	filename := sanitizeAttachmentFilename(attachment.Filename)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	if _, err := io.Copy(w, download); err != nil {
		return
	}
}

func (s *Server) handleBackoffice(w http.ResponseWriter, r *http.Request) {
	if s.enforceAuth {
		workflowKey, _, err := s.selectedWorkflow(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, workflowPath(workflowKey)+"/dashboard", http.StatusSeeOther)
		return
	}
	workflowKey, cfg, err := s.selectedWorkflow(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/backoffice")
	path = strings.Trim(path, "/")
	_, scoped := r.Context().Value(workflowContextKey{}).(workflowContextValue)
	if path == "" {
		if !scoped {
			options, listErr := s.workflowOptions(r.Context())
			if listErr != nil {
				http.Error(w, listErr.Error(), http.StatusInternalServerError)
				return
			}
			view := BackofficeWorkflowPickerView{
				WorkflowPickerView: WorkflowPickerView{
					PageBase:  s.pageBase("backoffice_picker_body", "", ""),
					Workflows: options,
				},
			}
			if err := s.tmpl.ExecuteTemplate(w, "backoffice.html", view); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
		view := BackofficeLandingView{
			PageBase: s.pageBase("backoffice_landing_body", workflowKey, cfg.Workflow.Name),
			Users:    s.userViews(cfg),
		}
		if err := s.tmpl.ExecuteTemplate(w, "backoffice_landing.html", view); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	parts := strings.Split(path, "/")
	role := parts[0]
	if !s.isKnownRole(cfg, role) {
		http.NotFound(w, r)
		return
	}

	if len(parts) == 1 {
		s.handleDepartmentDashboard(w, r, role)
		return
	}
	if len(parts) == 2 && parts[1] == "partial" {
		s.handleDepartmentDashboardPartial(w, r, role)
		return
	}
	if len(parts) == 3 && parts[1] == "process" {
		s.handleDepartmentProcess(w, r, role, parts[2])
		return
	}

	http.NotFound(w, r)
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	user, _, ok := s.requireAuthenticatedPage(w, r)
	if !ok {
		return
	}
	workflowKey, cfg, err := s.selectedWorkflow(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctx := r.Context()
	todoActions, activeProcesses, doneProcesses := s.loadProcessDashboardForRoles(ctx, cfg, workflowKey, user.RoleSlugs)
	view := DashboardView{
		PageBase:        s.pageBaseForUser(user, "dashboard_body", workflowKey, cfg.Workflow.Name),
		UserID:          user.UserID,
		RoleSlugs:       append([]string(nil), user.RoleSlugs...),
		TodoActions:     todoActions,
		ActiveProcesses: activeProcesses,
		DoneProcesses:   doneProcesses,
	}
	templateName := "dashboard.html"
	if strings.HasSuffix(strings.TrimSpace(r.URL.Path), "/partial") {
		templateName = "dashboard_partial.html"
	}
	if err := s.tmpl.ExecuteTemplate(w, templateName, view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleCompleteSubstep(w http.ResponseWriter, r *http.Request, processID, substepID string) {
	user, _, ok := s.requireAuthenticatedPost(w, r)
	if !ok {
		return
	}
	workflowKey, cfg, err := s.selectedWorkflow(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	actor := Actor{
		UserID:      user.UserID,
		OrgSlug:     user.OrgSlug,
		RoleSlugs:   append([]string(nil), user.RoleSlugs...),
		WorkflowKey: workflowKey,
	}
	if len(user.RoleSlugs) > 0 {
		actor.Role = user.RoleSlugs[0]
	}
	if actor.WorkflowKey != "" && actor.WorkflowKey != workflowKey {
		s.renderActionErrorForRequest(w, r, http.StatusForbidden, "Not authorized for this action.", nil, actor)
		return
	}

	ctx := r.Context()
	process, err := s.loadProcess(ctx, processID)
	if err != nil {
		s.renderActionErrorForRequest(w, r, http.StatusNotFound, "Process not found.", process, actor)
		return
	}
	if !s.processBelongsToWorkflow(process, workflowKey) {
		s.renderActionErrorForRequest(w, r, http.StatusNotFound, "Process not found.", process, actor)
		return
	}

	substep, step, err := findSubstep(cfg.Workflow, substepID)
	if err != nil {
		s.renderActionErrorForRequest(w, r, http.StatusNotFound, "Substep not found.", process, actor)
		return
	}
	if len(actor.RoleSlugs) == 0 && strings.TrimSpace(actor.Role) != "" {
		actor.RoleSlugs = []string{strings.TrimSpace(actor.Role)}
	}
	if substep.InputType == "file" {
		_ = r.ParseMultipartForm(attachmentMaxBytes())
	} else {
		_ = r.ParseForm()
	}
	activeRole := strings.TrimSpace(r.FormValue("activeRole"))
	if activeRole == "" && len(actor.RoleSlugs) == 1 {
		activeRole = actor.RoleSlugs[0]
	}
	allowedRoles := substepRoles(substep)
	if !s.enforceAuth && activeRole == "" && len(allowedRoles) > 0 {
		activeRole = allowedRoles[0]
		actor.RoleSlugs = append([]string(nil), allowedRoles...)
	}
	if activeRole == "" || !containsRole(actor.RoleSlugs, activeRole) || !containsRole(allowedRoles, activeRole) {
		s.renderActionErrorForRequest(w, r, http.StatusForbidden, "Not authorized for this action.", process, actor)
		return
	}
	actor.Role = activeRole

	sequenceOK := isSequenceOK(cfg.Workflow, process, substepID)
	if s.authorizer == nil {
		s.renderActionErrorForRequest(w, r, http.StatusBadGateway, "Cerbos check failed.", process, actor)
		return
	}
	allowed, err := s.authorizer.CanComplete(r.Context(), actor, processID, workflowKey, substep, step.Order, step.OrganizationSlug, sequenceOK)
	if err != nil {
		s.renderActionErrorForRequest(w, r, http.StatusBadGateway, "Cerbos check failed.", process, actor)
		return
	}
	if !sequenceOK {
		if progress, ok := process.Progress[substepID]; ok && progress.State == "done" && containsRole(allowedRoles, actor.Role) {
			if isHTMXRequest(r) {
				s.renderActionList(w, r, process, actor, "")
				return
			}
			s.renderDepartmentProcessPage(w, r, process, actor, "")
			return
		}
		s.renderActionErrorForRequest(w, r, http.StatusConflict, "Step is locked: complete previous steps first.", process, actor)
		return
	}
	if !allowed {
		s.renderActionErrorForRequest(w, r, http.StatusForbidden, "Not authorized for this action.", process, actor)
		return
	}

	now := s.nowUTC()
	payload, err := s.parseCompletionPayload(w, r, process.ID, substep, now)
	if err != nil {
		switch {
		case errors.Is(err, ErrAttachmentTooLarge):
			s.renderActionErrorForRequest(w, r, http.StatusRequestEntityTooLarge, "File too large.", process, actor)
		case errors.Is(err, errFileRequired):
			s.renderActionErrorForRequest(w, r, http.StatusBadRequest, "File is required.", process, actor)
		case errors.Is(err, errInvalidForm):
			s.renderActionErrorForRequest(w, r, http.StatusBadRequest, "Invalid form.", process, actor)
		case errors.Is(err, errValueRequired):
			s.renderActionErrorForRequest(w, r, http.StatusBadRequest, "Value is required.", process, actor)
		default:
			s.renderActionErrorForRequest(w, r, http.StatusBadRequest, err.Error(), process, actor)
		}
		return
	}

	progressUpdate := ProcessStep{
		State:  "done",
		DoneAt: &now,
		DoneBy: &actor,
		Data:   payload,
	}

	if err := s.store.UpdateProcessProgress(ctx, process.ID, workflowKey, substepID, progressUpdate); err != nil {
		s.renderActionErrorForRequest(w, r, http.StatusInternalServerError, "Failed to update process.", process, actor)
		return
	}

	notary := Notarization{
		ProcessID: process.ID,
		SubstepID: substepID,
		Payload:   payload,
		Actor:     actor,
		CreatedAt: now,
		FakeNotary: FakeNotary{
			Method: "sha256",
			Digest: digestPayload(payload),
		},
	}
	if err := s.store.InsertNotarization(ctx, notary); err != nil {
		s.renderActionErrorForRequest(w, r, http.StatusInternalServerError, "Failed to notarize payload.", process, actor)
		return
	}

	process, _ = s.loadProcess(ctx, processID)
	if process != nil && isProcessDone(cfg.Workflow, process) {
		_ = s.store.UpdateProcessStatus(ctx, process.ID, workflowKey, "done")
		if cfg.DPP.Enabled && process.DPP == nil {
			dpp, dppErr := buildProcessDPP(cfg.Workflow, cfg.DPP, process, now)
			if dppErr != nil {
				log.Printf("failed to build dpp for process %s: %v", process.ID.Hex(), dppErr)
			} else if updateErr := s.store.UpdateProcessDPP(ctx, process.ID, workflowKey, dpp); updateErr != nil {
				log.Printf("failed to persist dpp for process %s: %v", process.ID.Hex(), updateErr)
			}
		}
		process, _ = s.loadProcess(ctx, processID)
	}

	s.sse.Broadcast("process:"+workflowKey+":"+processID, "process-updated")
	for _, role := range s.roles(cfg) {
		s.sse.Broadcast("role:"+workflowKey+":"+role, "role-updated")
	}
	if isHTMXRequest(r) {
		s.renderActionList(w, r, process, actor, "")
		return
	}
	s.renderDepartmentProcessPage(w, r, process, actor, "")
}

var (
	errInvalidForm   = errors.New("invalid form")
	errValueRequired = errors.New("value required")
	errFileRequired  = errors.New("file required")
)

func (s *Server) parseCompletionPayload(w http.ResponseWriter, r *http.Request, processID primitive.ObjectID, substep WorkflowSub, now time.Time) (map[string]interface{}, error) {
	if substep.InputType == "file" {
		return s.parseFilePayload(w, r, processID, substep, now)
	}
	if substep.InputType == "formata" {
		return s.parseFormataPayload(r, processID, substep, now)
	}
	return parseScalarPayload(r, substep)
}

func parseScalarPayload(r *http.Request, substep WorkflowSub) (map[string]interface{}, error) {
	if err := r.ParseForm(); err != nil {
		return nil, errInvalidForm
	}
	value := strings.TrimSpace(r.FormValue("value"))
	if value == "" {
		return nil, errValueRequired
	}
	payload, err := normalizePayload(substep, value)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func (s *Server) parseFilePayload(w http.ResponseWriter, r *http.Request, processID primitive.ObjectID, substep WorkflowSub, now time.Time) (map[string]interface{}, error) {
	maxBytes := attachmentMaxBytes()
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		if isRequestTooLarge(err) {
			return nil, ErrAttachmentTooLarge
		}
		return nil, errInvalidForm
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	files := r.MultipartForm.File[substep.InputKey]
	if len(files) != 1 {
		return nil, errFileRequired
	}
	part := files[0]
	file, err := part.Open()
	if err != nil {
		return nil, errInvalidForm
	}
	defer file.Close()

	contentType := strings.TrimSpace(part.Header.Get("Content-Type"))
	reader := io.Reader(file)
	if contentType == "" {
		header := make([]byte, 512)
		count, readErr := io.ReadFull(file, header)
		switch {
		case readErr == nil:
		case errors.Is(readErr, io.EOF), errors.Is(readErr, io.ErrUnexpectedEOF):
		default:
			return nil, errInvalidForm
		}
		sniffed := bytes.TrimSpace(header[:count])
		if len(sniffed) > 0 {
			contentType = http.DetectContentType(sniffed)
		}
		reader = io.MultiReader(bytes.NewReader(header[:count]), file)
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	attachment, err := s.store.SaveAttachment(r.Context(), AttachmentUpload{
		ProcessID:   processID,
		SubstepID:   substep.SubstepID,
		Filename:    part.Filename,
		ContentType: contentType,
		MaxBytes:    maxBytes,
		UploadedAt:  now,
	}, reader)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		substep.InputKey: map[string]interface{}{
			"attachmentId": attachment.ID.Hex(),
			"filename":     attachment.Filename,
			"contentType":  attachment.ContentType,
			"size":         attachment.SizeBytes,
			"sha256":       attachment.SHA256,
		},
	}, nil
}

type decodedDataURL struct {
	ContentType string
	Data        []byte
}

func (s *Server) parseFormataPayload(r *http.Request, processID primitive.ObjectID, substep WorkflowSub, now time.Time) (map[string]interface{}, error) {
	payload, err := parseFormataScalarPayload(r, substep)
	if err != nil {
		return nil, err
	}
	raw := payload[substep.InputKey]
	converted, err := s.persistFormataAttachments(r.Context(), processID, substep, raw, now, []string{substep.InputKey})
	if err != nil {
		return nil, err
	}
	payload[substep.InputKey] = converted
	return payload, nil
}

func parseFormataScalarPayload(r *http.Request, substep WorkflowSub) (map[string]interface{}, error) {
	if err := r.ParseForm(); err != nil {
		return nil, errInvalidForm
	}
	rawValue := strings.TrimSpace(r.FormValue("value"))
	if rawValue == "" {
		fallback := formMapWithoutValue(r.PostForm)
		if len(fallback) == 0 {
			rawValue = "{}"
		} else {
			data, err := json.Marshal(fallback)
			if err != nil {
				return nil, errInvalidForm
			}
			rawValue = string(data)
		}
	}
	payload, err := normalizePayload(substep, rawValue)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func formMapWithoutValue(values url.Values) map[string]interface{} {
	if len(values) == 0 {
		return nil
	}
	result := map[string]interface{}{}
	for key, rawValues := range values {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" || trimmedKey == "value" {
			continue
		}
		switch len(rawValues) {
		case 0:
			continue
		case 1:
			result[trimmedKey] = strings.TrimSpace(rawValues[0])
		default:
			items := make([]string, 0, len(rawValues))
			for _, item := range rawValues {
				items = append(items, strings.TrimSpace(item))
			}
			result[trimmedKey] = items
		}
	}
	return result
}

func (s *Server) persistFormataAttachments(ctx context.Context, processID primitive.ObjectID, substep WorkflowSub, raw interface{}, now time.Time, path []string) (interface{}, error) {
	switch typed := raw.(type) {
	case map[string]interface{}:
		normalized := make(map[string]interface{}, len(typed))
		for key, value := range typed {
			nextPath := append(append([]string(nil), path...), key)
			converted, err := s.persistFormataAttachments(ctx, processID, substep, value, now, nextPath)
			if err != nil {
				return nil, err
			}
			normalized[key] = converted
		}
		return normalized, nil
	case primitive.M:
		return s.persistFormataAttachments(ctx, processID, substep, map[string]interface{}(typed), now, path)
	case []interface{}:
		normalized := make([]interface{}, len(typed))
		for index, value := range typed {
			nextPath := append(append([]string(nil), path...), strconv.Itoa(index))
			converted, err := s.persistFormataAttachments(ctx, processID, substep, value, now, nextPath)
			if err != nil {
				return nil, err
			}
			normalized[index] = converted
		}
		return normalized, nil
	case string:
		dataURL, ok := decodeDataURL(typed)
		if !ok {
			return typed, nil
		}
		filename := formataAttachmentFilename(substep.SubstepID, path, dataURL.ContentType)
		attachment, err := s.store.SaveAttachment(ctx, AttachmentUpload{
			ProcessID:   processID,
			SubstepID:   substep.SubstepID,
			Filename:    filename,
			ContentType: dataURL.ContentType,
			MaxBytes:    attachmentMaxBytes(),
			UploadedAt:  now,
		}, bytes.NewReader(dataURL.Data))
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"attachmentId": attachment.ID.Hex(),
			"filename":     attachment.Filename,
			"contentType":  attachment.ContentType,
			"size":         attachment.SizeBytes,
			"sha256":       attachment.SHA256,
		}, nil
	default:
		return raw, nil
	}
}

func decodeDataURL(raw string) (decodedDataURL, bool) {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(strings.ToLower(trimmed), "data:") {
		return decodedDataURL{}, false
	}
	commaIndex := strings.Index(trimmed, ",")
	if commaIndex < 0 {
		return decodedDataURL{}, false
	}
	metadata := strings.TrimSpace(trimmed[len("data:"):commaIndex])
	payload := trimmed[commaIndex+1:]
	if payload == "" {
		return decodedDataURL{}, false
	}

	isBase64 := strings.HasSuffix(strings.ToLower(metadata), ";base64")
	if isBase64 {
		metadata = strings.TrimSpace(metadata[:len(metadata)-len(";base64")])
	}
	contentType := metadata
	if semicolon := strings.Index(contentType, ";"); semicolon >= 0 {
		contentType = contentType[:semicolon]
	}
	contentType = strings.TrimSpace(contentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	if isBase64 {
		cleaned := strings.Map(func(r rune) rune {
			switch r {
			case '\n', '\r', '\t', ' ':
				return -1
			default:
				return r
			}
		}, payload)
		data, err := base64.StdEncoding.DecodeString(cleaned)
		if err != nil {
			return decodedDataURL{}, false
		}
		return decodedDataURL{ContentType: contentType, Data: data}, true
	}

	decoded, err := url.PathUnescape(payload)
	if err != nil {
		return decodedDataURL{}, false
	}
	return decodedDataURL{ContentType: contentType, Data: []byte(decoded)}, true
}

func formataAttachmentFilename(substepID string, path []string, contentType string) string {
	fieldPath := strings.TrimSpace(strings.Join(path, "_"))
	fieldPath = strings.Trim(strings.ReplaceAll(fieldPath, ".", "_"), "_")
	if fieldPath == "" {
		fieldPath = "attachment"
	}
	fieldPath = sanitizeAttachmentFilename(fieldPath)
	if !strings.Contains(fieldPath, ".") {
		mediaType, _, _ := mime.ParseMediaType(contentType)
		extensions, _ := mime.ExtensionsByType(strings.TrimSpace(mediaType))
		if len(extensions) > 0 {
			fieldPath += extensions[0]
		}
	}
	prefix := strings.Trim(strings.ReplaceAll(strings.TrimSpace(substepID), ".", "_"), "_")
	if prefix == "" {
		return fieldPath
	}
	return prefix + "-" + fieldPath
}

func isRequestTooLarge(err error) bool {
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "request body too large") || strings.Contains(msg, "message too large")
}

func readAttachmentPayload(data map[string]interface{}, inputKey string) (AttachmentPayload, bool) {
	if data == nil {
		return AttachmentPayload{}, false
	}
	raw, ok := data[inputKey]
	if !ok {
		return AttachmentPayload{}, false
	}

	values, ok := raw.(map[string]interface{})
	if !ok {
		if typed, ok := raw.(primitive.M); ok {
			values = map[string]interface{}(typed)
		} else {
			return AttachmentPayload{}, false
		}
	}

	attachmentID, _ := asString(values["attachmentId"])
	filename, _ := asString(values["filename"])
	contentType, _ := asString(values["contentType"])
	size, _ := asInt64(values["size"])
	sha256Digest, _ := asString(values["sha256"])
	if attachmentID == "" {
		return AttachmentPayload{}, false
	}
	return AttachmentPayload{
		AttachmentID: attachmentID,
		Filename:     filename,
		ContentType:  contentType,
		Size:         size,
		SHA256:       sha256Digest,
	}, true
}

func asString(value interface{}) (string, bool) {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed), true
	case fmt.Stringer:
		return strings.TrimSpace(typed.String()), true
	case nil:
		return "", false
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", typed)), true
	}
}

func asInt64(value interface{}) (int64, bool) {
	switch typed := value.(type) {
	case int:
		return int64(typed), true
	case int32:
		return int64(typed), true
	case int64:
		return typed, true
	case float32:
		return int64(typed), true
	case float64:
		return int64(typed), true
	case json.Number:
		parsed, err := typed.Int64()
		return parsed, err == nil
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func sanitizeAttachmentFilename(filename string) string {
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return "attachment"
	}
	replacer := strings.NewReplacer(
		"\x00", "",
		"\r", "_",
		"\n", "_",
		"\\", "_",
		"/", "_",
		"\"", "",
	)
	filename = replacer.Replace(filename)
	if filename == "" {
		return "attachment"
	}
	return filename
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.requireAuthenticatedPost(w, r); !ok {
		return
	}
	workflowKey, cfg, err := s.selectedWorkflow(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	queryWorkflow := strings.TrimSpace(r.URL.Query().Get("workflow"))
	if queryWorkflow != "" && queryWorkflow != workflowKey {
		http.Error(w, "workflow mismatch", http.StatusBadRequest)
		return
	}
	processID := r.URL.Query().Get("processId")
	role := r.URL.Query().Get("role")
	if processID == "" && role == "" {
		http.Error(w, "processId or role required", http.StatusBadRequest)
		return
	}
	if role != "" && !s.isKnownRole(cfg, role) {
		http.Error(w, "unknown role", http.StatusBadRequest)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	streamKey := "process:" + workflowKey + ":" + processID
	if role != "" {
		streamKey = "role:" + workflowKey + ":" + role
	}
	ch := s.sse.Subscribe(streamKey)
	defer s.sse.Unsubscribe(streamKey, ch)

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-ch:
			eventName := "process-updated"
			if role != "" {
				eventName = "role-updated"
			}
			fmt.Fprintf(w, "event: %s\n", eventName)
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		}
	}
}

func (s *Server) handleDepartmentDashboard(w http.ResponseWriter, r *http.Request, role string) {
	user, _, ok := s.requireAuthenticatedPage(w, r)
	if !ok {
		return
	}
	workflowKey, cfg, err := s.selectedWorkflow(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if s.enforceAuth && !containsRole(user.RoleSlugs, role) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	actor := Actor{UserID: user.UserID, Role: role, WorkflowKey: workflowKey}
	if strings.TrimSpace(actor.UserID) == "" {
		actor = s.actorForRole(cfg, role, workflowKey)
	}

	ctx := r.Context()
	todoActions, activeProcesses, doneProcesses := s.loadProcessDashboard(ctx, cfg, workflowKey, role)
	view := DepartmentDashboardView{
		PageBase:        s.pageBaseForUser(user, "dept_dashboard_body", workflowKey, cfg.Workflow.Name),
		CurrentUser:     actor,
		RoleLabel:       s.roleLabel(cfg, role),
		TodoActions:     todoActions,
		ActiveProcesses: activeProcesses,
		DoneProcesses:   doneProcesses,
	}
	if err := s.tmpl.ExecuteTemplate(w, "backoffice_department.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleDepartmentProcess(w http.ResponseWriter, r *http.Request, role, processID string) {
	user, _, ok := s.requireAuthenticatedPage(w, r)
	if !ok {
		return
	}
	workflowKey, cfg, err := s.selectedWorkflow(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if s.enforceAuth && !containsRole(user.RoleSlugs, role) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	actor := Actor{UserID: user.UserID, Role: role, WorkflowKey: workflowKey}
	if strings.TrimSpace(actor.UserID) == "" {
		actor = s.actorForRole(cfg, role, workflowKey)
	}

	ctx := r.Context()
	process, err := s.loadProcess(ctx, processID)
	if err != nil {
		http.Error(w, "process not found", http.StatusNotFound)
		return
	}
	if !s.processBelongsToWorkflow(process, workflowKey) {
		http.Error(w, "process not found", http.StatusNotFound)
		return
	}
	actions := buildActionList(cfg.Workflow, process, workflowKey, actor, true, s.roleMetaMap(cfg))
	view := DepartmentProcessView{
		PageBase:    s.pageBaseForUser(user, "dept_process_body", workflowKey, cfg.Workflow.Name),
		CurrentUser: actor,
		RoleLabel:   s.roleLabel(cfg, role),
		ProcessID:   process.ID.Hex(),
		Actions:     actions,
	}
	if err := s.tmpl.ExecuteTemplate(w, "backoffice_process.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleDepartmentDashboardPartial(w http.ResponseWriter, r *http.Request, role string) {
	user, _, ok := s.requireAuthenticatedPage(w, r)
	if !ok {
		return
	}
	workflowKey, cfg, err := s.selectedWorkflow(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if s.enforceAuth && !containsRole(user.RoleSlugs, role) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	actor := Actor{UserID: user.UserID, Role: role, WorkflowKey: workflowKey}

	ctx := r.Context()
	todoActions, activeProcesses, doneProcesses := s.loadProcessDashboard(ctx, cfg, workflowKey, role)
	view := DepartmentDashboardView{
		PageBase:        s.pageBase("", workflowKey, cfg.Workflow.Name),
		CurrentUser:     actor,
		RoleLabel:       s.roleLabel(cfg, role),
		TodoActions:     todoActions,
		ActiveProcesses: activeProcesses,
		DoneProcesses:   doneProcesses,
	}
	if err := s.tmpl.ExecuteTemplate(w, "backoffice_department_partial.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) loadProcess(ctx context.Context, id string) (*Process, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	process, err := s.store.LoadProcessByID(ctx, objectID)
	if err != nil {
		return nil, err
	}
	process.Progress = normalizeProgressKeys(process.Progress)
	return process, nil
}

func (s *Server) loadLatestProcess(ctx context.Context, workflowKey string) (*Process, error) {
	process, err := s.store.LoadLatestProcessByWorkflow(ctx, workflowKey)
	if err != nil {
		return nil, err
	}
	process.Progress = normalizeProgressKeys(process.Progress)
	return process, nil
}

func (s *Server) runtimeConfig() (RuntimeConfig, error) {
	if s.configProvider != nil {
		return s.configProvider()
	}
	key := s.defaultWorkflowKey()
	cfg, err := s.workflowByKey(key)
	if err != nil {
		return RuntimeConfig{}, err
	}
	return cfg, nil
}

func (s *Server) getConfig() (RuntimeConfig, error) {
	return s.runtimeConfig()
}

func sortedWorkflowKeys(catalog map[string]RuntimeConfig) []string {
	keys := make([]string, 0, len(catalog))
	for key := range catalog {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sameCatalogModTimes(a, b map[string]time.Time) bool {
	if len(a) != len(b) {
		return false
	}
	for path, modA := range a {
		modB, ok := b[path]
		if !ok || !modA.Equal(modB) {
			return false
		}
	}
	return true
}

func (s *Server) workflowCatalog() (map[string]RuntimeConfig, error) {
	s.configMu.Lock()
	defer s.configMu.Unlock()

	dir := strings.TrimSpace(s.configDir)
	if dir == "" {
		dir = "config"
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("config dir not found: %w", err)
	}

	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}
		paths = append(paths, filepath.Join(dir, name))
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return nil, errors.New("workflow config catalog is empty")
	}

	modTimes := make(map[string]time.Time, len(paths))
	for _, path := range paths {
		info, statErr := os.Stat(path)
		if statErr != nil {
			return nil, fmt.Errorf("config stat failed for %s: %w", path, statErr)
		}
		modTimes[path] = info.ModTime()
	}
	if s.catalog != nil && sameCatalogModTimes(s.catalogModTime, modTimes) {
		cached := make(map[string]RuntimeConfig, len(s.catalog))
		for key, cfg := range s.catalog {
			cached[key] = cfg
		}
		return cached, nil
	}

	catalog := make(map[string]RuntimeConfig, len(paths))
	for _, path := range paths {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, fmt.Errorf("read config %s: %w", path, readErr)
		}
		var cfg RuntimeConfig
		if unmarshalErr := yaml.Unmarshal(data, &cfg); unmarshalErr != nil {
			return nil, fmt.Errorf("parse config %s: %w", path, unmarshalErr)
		}
		normalizeWorkflowConfig(&cfg)
		if cfg.Workflow.Name == "" || len(cfg.Workflow.Steps) == 0 {
			return nil, fmt.Errorf("workflow config is empty in %s", filepath.Base(path))
		}
		if normalizeErr := normalizeInputTypes(&cfg.Workflow); normalizeErr != nil {
			return nil, fmt.Errorf("%s: %w", filepath.Base(path), normalizeErr)
		}
		if normalizeErr := normalizeDPPConfig(&cfg.DPP); normalizeErr != nil {
			return nil, fmt.Errorf("%s: %w", filepath.Base(path), normalizeErr)
		}
		key := strings.TrimSpace(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
		if key == "" {
			return nil, fmt.Errorf("workflow key is empty for %s", filepath.Base(path))
		}
		if _, exists := catalog[key]; exists {
			return nil, fmt.Errorf("duplicate workflow key %q", key)
		}
		catalog[key] = cfg
	}

	s.catalog = catalog
	s.catalogModTime = modTimes

	cloned := make(map[string]RuntimeConfig, len(catalog))
	for key, cfg := range catalog {
		cloned[key] = cfg
	}
	return cloned, nil
}

func (s *Server) workflowByKey(key string) (RuntimeConfig, error) {
	catalog, err := s.workflowCatalog()
	if err != nil {
		return RuntimeConfig{}, err
	}
	cfg, ok := catalog[key]
	if !ok {
		return RuntimeConfig{}, fmt.Errorf("workflow %q not found", key)
	}
	return cfg, nil
}

func normalizeWorkflowConfig(cfg *RuntimeConfig) {
	if cfg == nil {
		return
	}
	defaultOrg := ""
	if len(cfg.Organizations) > 0 {
		defaultOrg = strings.TrimSpace(cfg.Organizations[0].Slug)
	}
	if defaultOrg == "" && len(cfg.Departments) > 0 {
		defaultOrg = "default-org"
		cfg.Organizations = []WorkflowOrganization{
			{
				Slug:   defaultOrg,
				Name:   "Default organization",
				Color:  "#f0f3ea",
				Border: "#d9e0d0",
			},
		}
	}

	if len(cfg.Roles) == 0 && len(cfg.Departments) > 0 {
		for _, dept := range cfg.Departments {
			cfg.Roles = append(cfg.Roles, WorkflowRole{
				OrgSlug: defaultOrg,
				Slug:    strings.TrimSpace(dept.ID),
				Name:    strings.TrimSpace(dept.Name),
				Color:   strings.TrimSpace(dept.Color),
				Border:  strings.TrimSpace(dept.Border),
			})
		}
	}

	for stepIdx := range cfg.Workflow.Steps {
		step := &cfg.Workflow.Steps[stepIdx]
		if strings.TrimSpace(step.OrganizationSlug) == "" {
			step.OrganizationSlug = defaultOrg
		}
		for subIdx := range step.Substep {
			sub := &step.Substep[subIdx]
			if len(sub.Roles) == 0 && strings.TrimSpace(sub.Role) != "" {
				sub.Roles = []string{strings.TrimSpace(sub.Role)}
			}
			if strings.TrimSpace(sub.Role) == "" && len(sub.Roles) > 0 {
				sub.Role = strings.TrimSpace(sub.Roles[0])
			}
		}
	}
}

func (s *Server) roleMetaMap(cfg RuntimeConfig) map[string]RoleMeta {
	roles := map[string]RoleMeta{}
	for _, role := range cfg.Roles {
		meta := RoleMeta{
			ID:     role.Slug,
			Label:  role.Name,
			Color:  role.Color,
			Border: role.Border,
		}
		if meta.Label == "" {
			meta.Label = role.Slug
		}
		if meta.Color == "" {
			meta.Color = "#f0f3ea"
		}
		if meta.Border == "" {
			meta.Border = "#d9e0d0"
		}
		roles[role.Slug] = meta
	}
	if len(roles) > 0 {
		return roles
	}
	for _, dept := range cfg.Departments {
		meta := RoleMeta{
			ID:     dept.ID,
			Label:  dept.Name,
			Color:  dept.Color,
			Border: dept.Border,
		}
		if meta.Label == "" {
			meta.Label = dept.ID
		}
		if meta.Color == "" {
			meta.Color = "#f0f3ea"
		}
		if meta.Border == "" {
			meta.Border = "#d9e0d0"
		}
		roles[dept.ID] = meta
	}
	return roles
}

func (s *Server) roleLabel(cfg RuntimeConfig, role string) string {
	meta := roleMetaFor(role, s.roleMetaMap(cfg))
	return meta.Label
}

func (s *Server) isKnownRole(cfg RuntimeConfig, role string) bool {
	for _, item := range cfg.Roles {
		if item.Slug == role {
			return true
		}
	}
	for _, dept := range cfg.Departments {
		if dept.ID == role {
			return true
		}
	}
	return false
}

func (s *Server) roles(cfg RuntimeConfig) []string {
	var roles []string
	for _, item := range cfg.Roles {
		roles = append(roles, item.Slug)
	}
	if len(roles) > 0 {
		return roles
	}
	for _, dept := range cfg.Departments {
		roles = append(roles, dept.ID)
	}
	return roles
}

func (s *Server) actorForRole(cfg RuntimeConfig, role, workflowKey string) Actor {
	for _, user := range cfg.Users {
		if user.DepartmentID == role {
			return Actor{UserID: user.ID, Role: role, WorkflowKey: workflowKey}
		}
	}
	if role == "" {
		role = "unknown"
	}
	return Actor{UserID: role, Role: role, WorkflowKey: workflowKey}
}

func (s *Server) defaultRole(cfg RuntimeConfig) string {
	if len(cfg.Roles) > 0 {
		return cfg.Roles[0].Slug
	}
	if len(cfg.Departments) > 0 {
		return cfg.Departments[0].ID
	}
	return ""
}

func (s *Server) userViews(cfg RuntimeConfig) []UserView {
	metaMap := s.roleMetaMap(cfg)
	var views []UserView
	for _, user := range cfg.Users {
		meta := roleMetaFor(user.DepartmentID, metaMap)
		views = append(views, UserView{
			ID:         user.ID,
			Name:       user.Name,
			Role:       user.DepartmentID,
			RoleLabel:  meta.Label,
			RoleColor:  meta.Color,
			RoleBorder: meta.Border,
		})
	}
	return views
}

func (s *Server) loadProcessDashboard(ctx context.Context, cfg RuntimeConfig, workflowKey, role string) ([]ActionTodo, []ProcessSummary, []ProcessSummary) {
	processes, err := s.store.ListRecentProcessesByWorkflow(ctx, workflowKey, 25)
	if err != nil {
		return nil, nil, nil
	}

	var todo []ActionTodo
	var active []ProcessSummary
	var done []ProcessSummary
	for _, process := range processes {
		process.Progress = normalizeProgressKeys(process.Progress)
		status := deriveProcessStatus(cfg.Workflow, &process)
		summary := buildProcessSummaryForRole(cfg.Workflow, &process, status, role)
		if status == "done" {
			done = append(done, summary)
		} else {
			todo = append(todo, buildRoleTodos(cfg.Workflow, &process, role)...)
			if summary.NextSubstep != "" {
				active = append(active, summary)
			}
		}
	}
	return todo, active, done
}

func (s *Server) loadProcessDashboardForRoles(ctx context.Context, cfg RuntimeConfig, workflowKey string, roles []string) ([]ActionTodo, []ProcessSummary, []ProcessSummary) {
	processes, err := s.store.ListRecentProcessesByWorkflow(ctx, workflowKey, 25)
	if err != nil {
		return nil, nil, nil
	}
	roleSet := map[string]struct{}{}
	for _, role := range roles {
		trimmed := strings.TrimSpace(role)
		if trimmed != "" {
			roleSet[trimmed] = struct{}{}
		}
	}
	todoSeen := map[string]struct{}{}
	activeSeen := map[string]struct{}{}
	doneSeen := map[string]struct{}{}
	var todo []ActionTodo
	var active []ProcessSummary
	var done []ProcessSummary
	for _, process := range processes {
		process.Progress = normalizeProgressKeys(process.Progress)
		status := deriveProcessStatus(cfg.Workflow, &process)
		summary := buildProcessSummary(cfg.Workflow, &process, status)
		if status == "done" {
			if _, ok := doneSeen[summary.ID]; !ok {
				doneSeen[summary.ID] = struct{}{}
				done = append(done, summary)
			}
			continue
		}
		availMap := computeAvailability(cfg.Workflow, &process)
		for _, sub := range orderedSubsteps(cfg.Workflow) {
			allowedRoles := substepRoles(sub)
			matched := ""
			for _, role := range allowedRoles {
				if _, ok := roleSet[role]; ok {
					matched = role
					break
				}
			}
			if matched == "" {
				continue
			}
			stepStatus := "locked"
			if progress, ok := process.Progress[sub.SubstepID]; ok && progress.State == "done" {
				stepStatus = "done"
			} else if availMap[sub.SubstepID] {
				stepStatus = "available"
			}
			if stepStatus == "available" {
				key := summary.ID + "|" + sub.SubstepID
				if _, ok := todoSeen[key]; !ok {
					todoSeen[key] = struct{}{}
					todo = append(todo, ActionTodo{
						ProcessID: summary.ID,
						SubstepID: sub.SubstepID,
						Title:     sub.Title,
						Role:      matched,
						Status:    stepStatus,
					})
				}
				if _, ok := activeSeen[summary.ID]; !ok {
					activeSeen[summary.ID] = struct{}{}
					active = append(active, summary)
				}
			}
		}
	}
	return todo, active, done
}

func containsRole(roles []string, role string) bool {
	target := strings.TrimSpace(role)
	for _, item := range roles {
		if strings.TrimSpace(item) == target {
			return true
		}
	}
	return false
}

func substepRoles(sub WorkflowSub) []string {
	if len(sub.Roles) > 0 {
		out := make([]string, 0, len(sub.Roles))
		for _, role := range sub.Roles {
			trimmed := strings.TrimSpace(role)
			if trimmed == "" {
				continue
			}
			out = append(out, trimmed)
		}
		if len(out) > 0 {
			return out
		}
	}
	if strings.TrimSpace(sub.Role) != "" {
		return []string{strings.TrimSpace(sub.Role)}
	}
	return nil
}

func intersectRoles(allowed []string, owned []string) []string {
	ownedSet := map[string]struct{}{}
	for _, role := range owned {
		ownedSet[strings.TrimSpace(role)] = struct{}{}
	}
	var matches []string
	for _, role := range allowed {
		trimmed := strings.TrimSpace(role)
		if _, ok := ownedSet[trimmed]; ok {
			matches = append(matches, trimmed)
		}
	}
	return matches
}

func sortedSteps(def WorkflowDef) []WorkflowStep {
	steps := append([]WorkflowStep(nil), def.Steps...)
	sort.Slice(steps, func(i, j int) bool { return steps[i].Order < steps[j].Order })
	return steps
}

func sortedSubsteps(step WorkflowStep) []WorkflowSub {
	subs := append([]WorkflowSub(nil), step.Substep...)
	sort.Slice(subs, func(i, j int) bool { return subs[i].Order < subs[j].Order })
	return subs
}

func buildTimeline(def WorkflowDef, process *Process, workflowKey string, roleMeta map[string]RoleMeta) []TimelineStep {
	steps := sortedSteps(def)
	availableMap := computeAvailability(def, process)

	var timeline []TimelineStep
	for _, step := range steps {
		row := TimelineStep{StepID: step.StepID, Title: step.Title}
		for _, sub := range sortedSubsteps(step) {
			allowedRoles := substepRoles(sub)
			primaryRole := sub.Role
			if strings.TrimSpace(primaryRole) == "" && len(allowedRoles) > 0 {
				primaryRole = allowedRoles[0]
			}
			meta := roleMetaFor(primaryRole, roleMeta)
			roleBadges := make([]TimelineRoleBadge, 0, len(allowedRoles))
			for _, role := range allowedRoles {
				badgeMeta := roleMetaFor(role, roleMeta)
				roleBadges = append(roleBadges, TimelineRoleBadge{
					ID:     role,
					Label:  badgeMeta.Label,
					Color:  badgeMeta.Color,
					Border: badgeMeta.Border,
				})
			}
			entry := TimelineSubstep{
				SubstepID:  sub.SubstepID,
				Title:      sub.Title,
				Role:       strings.Join(allowedRoles, ", "),
				RoleBadges: roleBadges,
				RoleLabel:  meta.Label,
				RoleColor:  meta.Color,
				RoleBorder: meta.Border,
			}
			if process != nil {
				if progress, ok := process.Progress[sub.SubstepID]; ok && progress.State == "done" {
					entry.Status = "done"
					if progress.DoneBy != nil {
						entry.DoneBy = progress.DoneBy.UserID
						entry.DoneRole = progress.DoneBy.Role
					}
					if progress.DoneAt != nil {
						entry.DoneAt = progress.DoneAt.Format(time.RFC3339)
					}
					if sub.InputType == "file" {
						if attachment, ok := readAttachmentPayload(progress.Data, sub.InputKey); ok {
							entry.FileName = attachment.Filename
							entry.FileSHA256 = attachment.SHA256
							entry.FileURL = fmt.Sprintf("%s/process/%s/substep/%s/file", workflowPath(workflowKey), process.ID.Hex(), sub.SubstepID)
						}
					} else {
						if value, ok := progress.Data[sub.InputKey]; ok {
							entry.DisplayValue = strings.TrimSpace(fmt.Sprintf("%v", value))
						}
					}
				} else if availableMap[sub.SubstepID] {
					entry.Status = "available"
				} else {
					entry.Status = "locked"
				}
			} else {
				entry.Status = "locked"
			}
			row.Substeps = append(row.Substeps, entry)
		}
		timeline = append(timeline, row)
	}
	return timeline
}

type ProcessAttachmentExport struct {
	SubstepID    string `json:"substep_id"`
	AttachmentID string `json:"attachment_id"`
	Filename     string `json:"filename"`
	ContentType  string `json:"content_type,omitempty"`
	SizeBytes    int64  `json:"size_bytes,omitempty"`
	SHA256       string `json:"sha256,omitempty"`
}

func collectProcessAttachments(def WorkflowDef, process *Process) []ProcessAttachmentExport {
	if process == nil {
		return nil
	}
	var files []ProcessAttachmentExport
	seen := map[string]struct{}{}
	for _, sub := range orderedSubsteps(def) {
		progress, ok := process.Progress[sub.SubstepID]
		if !ok || progress.State != "done" {
			continue
		}
		for _, meta := range attachmentsFromValue(progress.Data) {
			if meta.AttachmentID == "" {
				continue
			}
			key := sub.SubstepID + ":" + meta.AttachmentID
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			files = append(files, ProcessAttachmentExport{
				SubstepID:    sub.SubstepID,
				AttachmentID: meta.AttachmentID,
				Filename:     meta.Filename,
				ContentType:  meta.ContentType,
				SizeBytes:    meta.SizeBytes,
				SHA256:       meta.SHA256,
			})
		}
	}
	return files
}

func buildProcessDownloadAttachments(workflowKey string, process *Process, files []ProcessAttachmentExport) []ProcessDownloadAttachment {
	if process == nil || len(files) == 0 {
		return nil
	}
	ordered := append([]ProcessAttachmentExport(nil), files...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].SubstepID != ordered[j].SubstepID {
			return ordered[i].SubstepID < ordered[j].SubstepID
		}
		if ordered[i].Filename != ordered[j].Filename {
			return ordered[i].Filename < ordered[j].Filename
		}
		return ordered[i].AttachmentID < ordered[j].AttachmentID
	})
	views := make([]ProcessDownloadAttachment, 0, len(ordered))
	for _, file := range ordered {
		if strings.TrimSpace(file.AttachmentID) == "" {
			continue
		}
		views = append(views, ProcessDownloadAttachment{
			SubstepID: file.SubstepID,
			Filename:  sanitizeAttachmentFilename(file.Filename),
			URL:       fmt.Sprintf("%s/process/%s/attachment/%s/file", workflowPath(workflowKey), process.ID.Hex(), file.AttachmentID),
		})
	}
	return views
}

func attachmentsFromValue(raw interface{}) []NotarizedAttachment {
	var files []NotarizedAttachment
	collectAttachmentsFromValue(raw, &files)
	return files
}

func collectAttachmentsFromValue(raw interface{}, files *[]NotarizedAttachment) {
	switch typed := raw.(type) {
	case map[string]interface{}:
		if meta := attachmentMetaFromMap(typed); meta != nil {
			*files = append(*files, *meta)
		}
		for _, nested := range typed {
			collectAttachmentsFromValue(nested, files)
		}
	case primitive.M:
		collectAttachmentsFromValue(map[string]interface{}(typed), files)
	case []interface{}:
		for _, nested := range typed {
			collectAttachmentsFromValue(nested, files)
		}
	}
}

func attachmentMetaFromPayload(data map[string]interface{}, inputKey string) *NotarizedAttachment {
	if data == nil {
		return nil
	}
	raw, ok := data[inputKey]
	if !ok {
		return nil
	}
	switch typed := raw.(type) {
	case map[string]interface{}:
		return attachmentMetaFromMap(typed)
	case primitive.M:
		return attachmentMetaFromMap(map[string]interface{}(typed))
	default:
		return nil
	}
}

func attachmentMetaFromMap(payload map[string]interface{}) *NotarizedAttachment {
	if payload == nil {
		return nil
	}
	meta := &NotarizedAttachment{}
	if value, ok := asString(payload["attachmentId"]); ok {
		meta.AttachmentID = value
	}
	if value, ok := asString(payload["filename"]); ok {
		meta.Filename = value
	}
	if value, ok := asString(payload["contentType"]); ok {
		meta.ContentType = value
	}
	if value, ok := asString(payload["sha256"]); ok {
		meta.SHA256 = value
	}
	if value, ok := asInt64(payload["size"]); ok {
		meta.SizeBytes = value
	}
	if meta.AttachmentID == "" && meta.Filename == "" && meta.SHA256 == "" {
		return nil
	}
	return meta
}

func buildNotarizedExport(def WorkflowDef, process *Process) NotarizedProcessExport {
	export := NotarizedProcessExport{}
	if process == nil {
		return export
	}
	status := strings.TrimSpace(process.Status)
	if status == "" {
		status = "active"
	}
	if status != "done" && isProcessDone(def, process) {
		status = "done"
	}
	export.ProcessID = process.ID.Hex()
	export.CreatedAt = process.CreatedAt.Format(time.RFC3339)
	export.Status = status

	availableMap := computeAvailability(def, process)
	var leaves []MerkleLeaf
	for _, step := range sortedSteps(def) {
		stepEntry := NotarizedStep{StepID: step.StepID, Title: step.Title}
		for _, sub := range sortedSubsteps(step) {
			entry := NotarizedSubstep{
				SubstepID: sub.SubstepID,
				Title:     sub.Title,
				Role:      sub.Role,
			}
			state := "locked"
			if progress, ok := process.Progress[sub.SubstepID]; ok && progress.State == "done" {
				state = "done"
				if progress.DoneAt != nil {
					entry.DoneAt = progress.DoneAt.Format(time.RFC3339)
				}
				if progress.DoneBy != nil {
					entry.DoneBy = progress.DoneBy.UserID
					entry.DoneRole = progress.DoneBy.Role
				}
				entry.Payload = progress.Data
				entry.Digest = digestPayload(progress.Data)
				if sub.InputType == "file" {
					entry.Attachment = attachmentMetaFromPayload(progress.Data, sub.InputKey)
				}
			} else if availableMap[sub.SubstepID] {
				state = "available"
			}
			entry.Status = state

			leafHash := hashMerkleLeaf(sub.SubstepID, entry)
			leaves = append(leaves, MerkleLeaf{SubstepID: sub.SubstepID, Hash: leafHash})
			stepEntry.Substeps = append(stepEntry.Substeps, entry)
		}
		export.Steps = append(export.Steps, stepEntry)
	}
	export.Merkle = buildMerkleTree(leaves)
	return export
}

func hashMerkleLeaf(substepID string, entry NotarizedSubstep) string {
	payload := struct {
		SubstepID string                 `json:"substep_id"`
		Status    string                 `json:"status"`
		DoneAt    string                 `json:"done_at,omitempty"`
		DoneBy    string                 `json:"done_by,omitempty"`
		DoneRole  string                 `json:"done_role,omitempty"`
		Payload   map[string]interface{} `json:"payload,omitempty"`
	}{
		SubstepID: substepID,
		Status:    entry.Status,
		DoneAt:    entry.DoneAt,
		DoneBy:    entry.DoneBy,
		DoneRole:  entry.DoneRole,
		Payload:   entry.Payload,
	}
	data, _ := json.Marshal(payload)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func buildMerkleTree(leaves []MerkleLeaf) MerkleTree {
	tree := MerkleTree{Leaves: leaves}
	if len(leaves) == 0 {
		return tree
	}
	level := make([]string, 0, len(leaves))
	for _, leaf := range leaves {
		level = append(level, leaf.Hash)
	}
	tree.Levels = append(tree.Levels, append([]string(nil), level...))
	for len(level) > 1 {
		var next []string
		for i := 0; i < len(level); i += 2 {
			left := level[i]
			right := left
			if i+1 < len(level) {
				right = level[i+1]
			}
			sum := sha256.Sum256([]byte(left + right))
			next = append(next, hex.EncodeToString(sum[:]))
		}
		level = next
		tree.Levels = append(tree.Levels, append([]string(nil), level...))
	}
	tree.Root = level[0]
	return tree
}

func writeJSON(w http.ResponseWriter, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(value)
}

func orderedSubsteps(def WorkflowDef) []WorkflowSub {
	var ordered []WorkflowSub
	for _, step := range sortedSteps(def) {
		for _, sub := range sortedSubsteps(step) {
			ordered = append(ordered, sub)
		}
	}
	return ordered
}

func buildProcessSummary(def WorkflowDef, process *Process, status string) ProcessSummary {
	nextSubstep, ok := nextAvailableSubstep(def, process)
	summary := ProcessSummary{
		ID:        process.ID.Hex(),
		Status:    status,
		CreatedAt: process.CreatedAt.Format(time.RFC3339),
	}
	if ok {
		summary.NextSubstep = nextSubstep.SubstepID
		summary.NextTitle = nextSubstep.Title
		summary.NextRole = nextSubstep.Role
	}
	return summary
}

func buildProcessSummaryForRole(def WorkflowDef, process *Process, status, role string) ProcessSummary {
	nextSubstep, ok := nextAvailableSubstepForRole(def, process, role)
	summary := ProcessSummary{
		ID:        process.ID.Hex(),
		Status:    status,
		CreatedAt: process.CreatedAt.Format(time.RFC3339),
	}
	if ok {
		summary.NextSubstep = nextSubstep.SubstepID
		summary.NextTitle = nextSubstep.Title
		summary.NextRole = nextSubstep.Role
	}
	return summary
}

func nextAvailableSubstep(def WorkflowDef, process *Process) (WorkflowSub, bool) {
	if process == nil {
		return WorkflowSub{}, false
	}
	availMap := computeAvailability(def, process)
	for _, sub := range orderedSubsteps(def) {
		if availMap[sub.SubstepID] {
			return sub, true
		}
	}
	return WorkflowSub{}, false
}

func nextAvailableSubstepForRole(def WorkflowDef, process *Process, role string) (WorkflowSub, bool) {
	if process == nil {
		return WorkflowSub{}, false
	}
	availMap := computeAvailability(def, process)
	for _, sub := range orderedSubsteps(def) {
		if sub.Role != role {
			continue
		}
		if availMap[sub.SubstepID] {
			return sub, true
		}
	}
	return WorkflowSub{}, false
}

func buildRoleTodos(def WorkflowDef, process *Process, role string) []ActionTodo {
	if process == nil {
		return nil
	}
	availMap := computeAvailability(def, process)
	var todos []ActionTodo
	for _, sub := range orderedSubsteps(def) {
		if sub.Role != role {
			continue
		}
		status := "locked"
		if step, ok := process.Progress[sub.SubstepID]; ok && step.State == "done" {
			status = "done"
		} else if availMap[sub.SubstepID] {
			status = "available"
		}
		if status == "available" {
			todos = append(todos, ActionTodo{
				ProcessID: process.ID.Hex(),
				SubstepID: sub.SubstepID,
				Title:     sub.Title,
				Status:    status,
			})
		}
	}
	return todos
}

func buildActionList(def WorkflowDef, process *Process, workflowKey string, actor Actor, onlyRole bool, roleMeta map[string]RoleMeta) []ActionView {
	var actions []ActionView
	ordered := orderedSubsteps(def)
	availMap := computeAvailability(def, process)
	for _, sub := range ordered {
		allowedRoles := substepRoles(sub)
		ownedRoles := append([]string(nil), actor.RoleSlugs...)
		if len(ownedRoles) == 0 && strings.TrimSpace(actor.Role) != "" {
			ownedRoles = []string{strings.TrimSpace(actor.Role)}
		}
		matchingRoles := intersectRoles(allowedRoles, ownedRoles)
		primaryRole := sub.Role
		if primaryRole == "" && len(allowedRoles) > 0 {
			primaryRole = allowedRoles[0]
		}
		roleBadges := make([]ActionRoleBadge, 0, len(allowedRoles))
		for _, role := range allowedRoles {
			meta := roleMetaFor(role, roleMeta)
			roleBadges = append(roleBadges, ActionRoleBadge{
				ID:     role,
				Label:  meta.Label,
				Color:  meta.Color,
				Border: meta.Border,
			})
		}
		if onlyRole && strings.TrimSpace(actor.Role) != "" && !containsRole(allowedRoles, actor.Role) {
			continue
		}
		meta := roleMetaFor(primaryRole, roleMeta)
		status := "locked"
		if process != nil {
			if step, ok := process.Progress[sub.SubstepID]; ok && step.State == "done" {
				status = "done"
			} else if availMap[sub.SubstepID] {
				status = "available"
			}
		}
		disabled := status != "available" || len(matchingRoles) == 0
		reason := ""
		if status == "locked" {
			reason = "Locked by sequence"
		} else if status == "done" {
			reason = "Already completed"
		} else if len(matchingRoles) == 0 {
			reason = "Not authorized"
		}
		formSchema := ""
		formUISchema := ""
		doneAt := ""
		doneBy := ""
		doneRole := ""
		var values []ActionKV
		var attachments []ActionAttachmentView
		if status == "done" && process != nil {
			if progress, ok := process.Progress[sub.SubstepID]; ok {
				if progress.DoneAt != nil {
					doneAt = progress.DoneAt.UTC().Format(time.RFC3339)
				}
				if progress.DoneBy != nil {
					doneBy = strings.TrimSpace(progress.DoneBy.UserID)
					doneRole = strings.TrimSpace(progress.DoneBy.Role)
				}
				if sub.InputType == "formata" {
					values = flattenDisplayValues("", progress.Data[sub.InputKey])
				} else if value, ok := progress.Data[sub.InputKey]; ok && !isAttachmentMetaValue(value) {
					values = flattenDisplayValues(sub.InputKey, value)
				}
				attachments = buildActionAttachments(workflowKey, process, progress.Data)
			}
		}
		if sub.InputType == "formata" {
			formSchema = marshalJSONCompact(sub.Schema)
			formUISchema = marshalJSONCompact(sub.UISchema)
		}
		actions = append(actions, ActionView{
			ProcessID:     processIDString(process),
			SubstepID:     sub.SubstepID,
			Title:         sub.Title,
			Role:          primaryRole,
			AllowedRoles:  allowedRoles,
			RoleBadges:    roleBadges,
			MatchingRoles: matchingRoles,
			RoleLabel:     meta.Label,
			RoleColor:     meta.Color,
			RoleBorder:    meta.Border,
			InputKey:      sub.InputKey,
			InputType:     sub.InputType,
			FormSchema:    formSchema,
			FormUISchema:  formUISchema,
			Status:        status,
			DoneAt:        doneAt,
			DoneBy:        doneBy,
			DoneRole:      doneRole,
			Values:        values,
			Attachments:   attachments,
			Disabled:      disabled,
			Reason:        reason,
		})
	}
	return actions
}

func flattenDisplayValues(inputKey string, raw interface{}) []ActionKV {
	key := strings.TrimSpace(inputKey)
	var out []ActionKV
	collectDisplayValues(key, raw, &out)
	if len(out) > 1 {
		sort.Slice(out, func(i, j int) bool {
			return out[i].Key < out[j].Key
		})
	}
	return out
}

func collectDisplayValues(path string, raw interface{}, out *[]ActionKV) {
	switch typed := raw.(type) {
	case map[string]interface{}:
		if isAttachmentMetaMap(typed) {
			return
		}
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			nextPath := key
			if strings.TrimSpace(path) != "" {
				nextPath = path + "." + key
			}
			collectDisplayValues(nextPath, typed[key], out)
		}
	case primitive.M:
		collectDisplayValues(path, map[string]interface{}(typed), out)
	case []interface{}:
		for idx, nested := range typed {
			nextPath := fmt.Sprintf("[%d]", idx)
			if strings.TrimSpace(path) != "" {
				nextPath = fmt.Sprintf("%s[%d]", path, idx)
			}
			collectDisplayValues(nextPath, nested, out)
		}
	default:
		value := truncateDisplayValue(strings.TrimSpace(fmt.Sprintf("%v", typed)))
		key := path
		if strings.TrimSpace(key) == "" {
			key = "value"
		}
		*out = append(*out, ActionKV{Key: key, Value: value})
	}
}

func truncateDisplayValue(value string) string {
	const maxLen = 200
	if len(value) <= maxLen {
		return value
	}
	return value[:maxLen] + "..."
}

func isAttachmentMetaValue(raw interface{}) bool {
	switch typed := raw.(type) {
	case map[string]interface{}:
		return isAttachmentMetaMap(typed)
	case primitive.M:
		return isAttachmentMetaMap(map[string]interface{}(typed))
	default:
		return false
	}
}

func isAttachmentMetaMap(payload map[string]interface{}) bool {
	return attachmentMetaFromMap(payload) != nil
}

func buildActionAttachments(workflowKey string, process *Process, data map[string]interface{}) []ActionAttachmentView {
	if process == nil {
		return nil
	}
	seen := map[string]struct{}{}
	attachments := make([]ActionAttachmentView, 0)
	for _, meta := range attachmentsFromValue(data) {
		id := strings.TrimSpace(meta.AttachmentID)
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		attachments = append(attachments, ActionAttachmentView{
			Filename: sanitizeAttachmentFilename(meta.Filename),
			URL:      fmt.Sprintf("%s/process/%s/attachment/%s/file", workflowPath(workflowKey), process.ID.Hex(), id),
			SHA256:   strings.TrimSpace(meta.SHA256),
		})
	}
	sort.Slice(attachments, func(i, j int) bool {
		if attachments[i].Filename != attachments[j].Filename {
			return attachments[i].Filename < attachments[j].Filename
		}
		return attachments[i].URL < attachments[j].URL
	})
	return attachments
}

func marshalJSONCompact(value interface{}) string {
	if value == nil {
		return ""
	}
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(data)
}

func processIDString(process *Process) string {
	if process == nil {
		return ""
	}
	return process.ID.Hex()
}

func encodeProgressKey(key string) string {
	return strings.ReplaceAll(key, ".", "_")
}

func normalizeProgressKeys(progress map[string]ProcessStep) map[string]ProcessStep {
	if progress == nil {
		return map[string]ProcessStep{}
	}
	normalized := make(map[string]ProcessStep, len(progress))
	for key, value := range progress {
		decoded := strings.ReplaceAll(key, "_", ".")
		normalized[decoded] = value
	}
	return normalized
}

func isHTMXRequest(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("HX-Request"), "true")
}

func roleMetaFor(role string, roleMeta map[string]RoleMeta) RoleMeta {
	if meta, ok := roleMeta[role]; ok {
		return meta
	}
	return RoleMeta{ID: role, Label: role}
}

func computeAvailability(def WorkflowDef, process *Process) map[string]bool {
	available := map[string]bool{}
	ordered := orderedSubsteps(def)
	allPrevDone := true
	for _, sub := range ordered {
		done := false
		if process != nil {
			if entry, ok := process.Progress[sub.SubstepID]; ok && entry.State == "done" {
				done = true
			}
		}
		if done {
			available[sub.SubstepID] = false
			continue
		}
		if allPrevDone {
			available[sub.SubstepID] = true
			allPrevDone = false
		} else {
			available[sub.SubstepID] = false
		}
	}
	return available
}

func isSequenceOK(def WorkflowDef, process *Process, substepID string) bool {
	ordered := orderedSubsteps(def)
	for _, sub := range ordered {
		if sub.SubstepID == substepID {
			return true
		}
		if process == nil {
			return false
		}
		if entry, ok := process.Progress[sub.SubstepID]; !ok || entry.State != "done" {
			return false
		}
	}
	return false
}

func isProcessDone(def WorkflowDef, process *Process) bool {
	for _, sub := range orderedSubsteps(def) {
		entry, ok := process.Progress[sub.SubstepID]
		if !ok || entry.State != "done" {
			return false
		}
	}
	return true
}

func findSubstep(def WorkflowDef, substepID string) (WorkflowSub, WorkflowStep, error) {
	for _, step := range def.Steps {
		for _, sub := range step.Substep {
			if sub.SubstepID == substepID {
				return sub, step, nil
			}
		}
	}
	return WorkflowSub{}, WorkflowStep{}, errors.New("not found")
}

func normalizeInputTypes(workflow *WorkflowDef) error {
	for stepIndex := range workflow.Steps {
		for substepIndex := range workflow.Steps[stepIndex].Substep {
			substep := &workflow.Steps[stepIndex].Substep[substepIndex]
			inputType, err := normalizeInputType(substep.InputType)
			if err != nil {
				return fmt.Errorf("invalid inputType for substep %s: %w", substep.SubstepID, err)
			}
			substep.InputType = inputType
			if err := normalizeSubstepInputConfig(substep); err != nil {
				return fmt.Errorf("invalid inputType for substep %s: %w", substep.SubstepID, err)
			}
		}
	}
	return nil
}

func normalizeInputType(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "number":
		return "number", nil
	case "string", "text":
		return "string", nil
	case "file":
		return "file", nil
	case "formata", "schema", "jsonschema":
		return "formata", nil
	default:
		return "", fmt.Errorf("unsupported value %q (allowed: number, string, text, file, formata)", value)
	}
}

func normalizeSubstepInputConfig(substep *WorkflowSub) error {
	if substep.InputType != "formata" {
		return nil
	}
	if len(substep.Schema) == 0 {
		return errors.New("schema is required when inputType=formata")
	}
	return nil
}

func normalizeDPPConfig(cfg *DPPConfig) error {
	cfg.GTIN = strings.TrimSpace(cfg.GTIN)
	cfg.LotInputKey = strings.TrimSpace(cfg.LotInputKey)
	cfg.LotDefault = strings.TrimSpace(cfg.LotDefault)
	cfg.SerialInputKey = strings.TrimSpace(cfg.SerialInputKey)
	cfg.SerialStrategy = strings.TrimSpace(cfg.SerialStrategy)
	cfg.ProductName = strings.TrimSpace(cfg.ProductName)
	cfg.ProductDescription = strings.TrimSpace(cfg.ProductDescription)
	cfg.OwnerName = strings.TrimSpace(cfg.OwnerName)

	if cfg.LotInputKey == "" {
		cfg.LotInputKey = "batchId"
	}
	if cfg.LotDefault == "" {
		cfg.LotDefault = "defaultProduct"
	}
	if cfg.SerialStrategy == "" {
		cfg.SerialStrategy = "process_id_hex"
	}
	normalizedStrategy, err := normalizeDPPSerialStrategy(cfg.SerialStrategy)
	if err != nil {
		return err
	}
	cfg.SerialStrategy = normalizedStrategy

	if !cfg.Enabled {
		return nil
	}

	normalizedGTIN, err := normalizeGTIN(cfg.GTIN)
	if err != nil {
		return err
	}
	cfg.GTIN = normalizedGTIN
	return nil
}

func normalizeGTIN(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("dpp.gtin is required when dpp.enabled=true")
	}
	for _, char := range trimmed {
		if char < '0' || char > '9' {
			return "", fmt.Errorf("dpp.gtin must contain only digits: %q", raw)
		}
	}
	if len(trimmed) > 14 {
		return "", fmt.Errorf("dpp.gtin must be at most 14 digits: %q", raw)
	}
	if len(trimmed) < 14 {
		trimmed = strings.Repeat("0", 14-len(trimmed)) + trimmed
	}
	return trimmed, nil
}

func normalizeDPPSerialStrategy(raw string) (string, error) {
	strategy := strings.TrimSpace(raw)
	if strategy == "" {
		strategy = "process_id_hex"
	}
	switch strategy {
	case "process_id_hex":
		return strategy, nil
	default:
		return "", fmt.Errorf("unsupported dpp.serialStrategy %q (allowed: process_id_hex)", raw)
	}
}

func normalizePayload(sub WorkflowSub, value string) (map[string]interface{}, error) {
	payload := map[string]interface{}{}
	switch sub.InputType {
	case "number":
		var number float64
		_, err := fmt.Sscanf(value, "%f", &number)
		if err != nil {
			return nil, errors.New("Value must be a number.")
		}
		payload[sub.InputKey] = number
	case "formata":
		var decoded interface{}
		if err := json.Unmarshal([]byte(value), &decoded); err != nil {
			return nil, errors.New("Value must be a valid JSON object.")
		}
		valueObject, ok := decoded.(map[string]interface{})
		if !ok {
			return nil, errors.New("Value must be a valid JSON object.")
		}
		payload[sub.InputKey] = valueObject
	default:
		payload[sub.InputKey] = value
	}
	return payload, nil
}

func prefersJSONResponse(r *http.Request) bool {
	if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("format")), "json") {
		return true
	}
	accept := strings.ToLower(strings.TrimSpace(r.Header.Get("Accept")))
	return strings.Contains(accept, "application/json")
}

func digestPayload(payload map[string]interface{}) string {
	data, _ := json.Marshal(payload)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (s *Server) nowUTC() time.Time {
	if s.now == nil {
		return time.Now().UTC()
	}
	return s.now().UTC()
}

func (s *Server) renderActionError(w http.ResponseWriter, status int, message string, process *Process, actor Actor) {
	w.WriteHeader(status)
	s.renderActionList(w, nil, process, actor, message)
}

func (s *Server) renderActionErrorForRequest(w http.ResponseWriter, r *http.Request, status int, message string, process *Process, actor Actor) {
	w.WriteHeader(status)
	if isHTMXRequest(r) {
		s.renderActionList(w, r, process, actor, message)
		return
	}
	s.renderDepartmentProcessPage(w, r, process, actor, message)
}

func (s *Server) renderActionList(w http.ResponseWriter, r *http.Request, process *Process, actor Actor, message string) {
	workflowKey := s.defaultWorkflowKey()
	cfg := RuntimeConfig{}
	var err error
	if r != nil {
		workflowKey, cfg, err = s.selectedWorkflow(r)
	} else {
		cfg, err = s.runtimeConfig()
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var timeline []TimelineStep
	if process != nil {
		timeline = buildTimeline(cfg.Workflow, process, workflowKey, s.roleMetaMap(cfg))
	}
	view := ActionListView{
		WorkflowKey: workflowKey,
		ProcessID:   processIDString(process),
		CurrentUser: actor,
		Actions:     buildActionList(cfg.Workflow, process, workflowKey, actor, false, s.roleMetaMap(cfg)),
		Error:       message,
		Timeline:    timeline,
	}
	if err := s.tmpl.ExecuteTemplate(w, "action_list.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) renderDepartmentProcessPage(w http.ResponseWriter, r *http.Request, process *Process, actor Actor, message string) {
	workflowKey := s.defaultWorkflowKey()
	cfg := RuntimeConfig{}
	var err error
	if r != nil {
		workflowKey, cfg, err = s.selectedWorkflow(r)
	} else {
		cfg, err = s.runtimeConfig()
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	processID := ""
	if process != nil {
		processID = process.ID.Hex()
	}
	view := DepartmentProcessView{
		PageBase:    s.pageBase("dept_process_body", workflowKey, cfg.Workflow.Name),
		CurrentUser: actor,
		RoleLabel:   s.roleLabel(cfg, actor.Role),
		ProcessID:   processID,
		Actions:     buildActionList(cfg.Workflow, process, workflowKey, actor, false, s.roleMetaMap(cfg)),
		Error:       message,
	}
	if err := s.tmpl.ExecuteTemplate(w, "backoffice_process.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func newSSEHub() *SSEHub {
	return &SSEHub{stream: map[string]map[chan string]struct{}{}}
}

func (h *SSEHub) Subscribe(processID string) chan string {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.stream[processID] == nil {
		h.stream[processID] = map[chan string]struct{}{}
	}
	ch := make(chan string, 5)
	h.stream[processID][ch] = struct{}{}
	return ch
}

func (h *SSEHub) Unsubscribe(processID string, ch chan string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if subs, ok := h.stream[processID]; ok {
		delete(subs, ch)
		close(ch)
		if len(subs) == 0 {
			delete(h.stream, processID)
		}
	}
}

func (h *SSEHub) Broadcast(processID, message string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.stream[processID] {
		select {
		case ch <- message:
		default:
		}
	}
}
