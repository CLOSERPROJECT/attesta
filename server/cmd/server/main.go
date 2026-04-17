package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"html/template"
	"io"
	"log"
	"mime"
	"net"
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
	ID          string   `bson:"id"`
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
	identity       IdentityStore
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
	Selected     bool
	Action       *ActionView
	RoleBadges   []TimelineRoleBadge
	RoleLabel    string
	RoleColor    template.CSS
	RoleBorder   template.CSS
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
	Color  template.CSS
	Border template.CSS
}

type TimelineStep struct {
	StepID     string
	Title      string
	OrgSlug    string
	OrgName    string
	OrgLogoURL string
	Expanded   bool
	Substeps   []TimelineSubstep
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
	WorkflowKey   string
	ProcessID     string
	SubstepID     string
	Title         string
	Role          string
	AllowedRoles  []string
	RoleBadges    []ActionRoleBadge
	MatchingRoles []string
	RoleLabel     string
	RoleColor     template.CSS
	RoleBorder    template.CSS
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
	ReadOnly      bool
	Reason        string
}

type ActionRoleBadge struct {
	ID     string
	Label  string
	Color  template.CSS
	Border template.CSS
}

type ActionKV struct {
	Key   string
	Value string
}

type ActionAttachmentView struct {
	Key         string
	Filename    string
	URL         string
	PreviewURL  string
	PreviewKind string
	SHA256      string
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

type PageBase struct {
	Body            string
	ViteDevServer   string
	WorkflowKey     string
	WorkflowName    string
	WorkflowPath    string
	UserEmail       string
	IsPlatformAdmin bool
	ShowOrgsLink    bool
	ShowMyOrgLink   bool
	ShowLogout      bool
}

type WorkflowOption struct {
	Key               string
	Name              string
	Description       string
	Counts            WorkflowProcessCounts
	CanClone          bool
	CanEdit           bool
	EditAction        string
	EditRequiresPurge bool
	CanDelete         bool
	DeleteAction      string
}

type WorkflowProcessCounts struct {
	NotStarted int
	Started    int
	Terminated int
}

type PublicCatalogResponse struct {
	Organizations []PublicCatalogOrganization `json:"organizations"`
	Roles         []PublicCatalogRole         `json:"roles"`
}

type PublicCatalogOrganization struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type PublicCatalogRole struct {
	OrgSlug string `json:"orgSlug"`
	Name    string `json:"name"`
	Slug    string `json:"slug"`
	Color   string `json:"color"`
	Border  string `json:"border"`
}

type WorkflowPickerView struct {
	PageBase
	Workflows            []WorkflowOption
	ShowCreateStreamCard bool
	Error                string
	Confirmation         string
}

type HomeWorkflowPickerView struct {
	WorkflowPickerView
}

type ActionListView struct {
	WorkflowKey       string
	WorkflowPath      string
	ProcessID         string
	CurrentUser       Actor
	SelectedSubstepID string
	ProcessDone       bool
	Action            *ActionView
	Error             string
	Timeline          []TimelineStep
	DPPURL            string
	DPPGS1            string
	Attachments       []ProcessDownloadAttachment
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
	WorkflowDescription string
	Error               string
	CanStartProcess     bool
	Sort                string
	StatusFilter        string
	CurrentPage         int
	TotalPages          int
	PageNumbers         []int
	HasPreviousPage     bool
	HasNextPage         bool
	PreviousPage        int
	NextPage            int
	Processes           []ProcessListItem
	Preview             ActionListView
}

type LoginView struct {
	PageBase
	Next         string
	Email        string
	Error        string
	Confirmation string
	ShowSignup   bool
}

type SignupView struct {
	PageBase
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
	Error        string
}

type ResetSetView struct {
	PageBase
	Token       string
	Error       string
	ActionPath  string
	Title       string
	SubmitLabel string
}

type AboutView struct {
	PageBase
}

type PlatformAdminView struct {
	PageBase
	SearchQuery              string
	CurrentPage              int
	TotalPages               int
	PageNumbers              []int
	HasPreviousPage          bool
	HasNextPage              bool
	PreviousPage             int
	NextPage                 int
	MatchedOrganizations     int
	Organizations            []PlatformAdminOrganizationRow
	InviteLink               string
	Confirmation             string
	OrganizationError        string
	OrganizationDialogAction string
	OrganizationDialogSlug   string
	OrganizationDialogName   string
	InviteError              string
	InviteDialogEmail        string
	Error                    string
}

type PlatformAdminOrganizationRow struct {
	Name                    string
	Slug                    string
	LogoAttachmentID        string
	OrgAdminEmails          []string
	PendingOrgAdminEmails   []string
	OrgAdminStatus          string
	OrgAdminStatusClassName string
}

type PlatformAdminErrors struct {
	Organization string
	Invite       string
	DialogAction string
	OrgSlug      string
	OrgName      string
	InviteEmail  string
	SearchQuery  string
	Page         int
}

type OrgAdminView struct {
	PageBase
	Organization           Organization
	OrganizationLogoURL    string
	NeedsOrganizationSetup bool
	OrganizationError      string
	RoleError              string
	RoleDialogAction       string
	RoleDialogSlug         string
	RoleDialogName         string
	RoleDialogPalette      string
	InviteError            string
	UsersError             string
	Roles                  []Role
	RolePills              []OrgAdminRoleOption
	RoleRows               []OrgAdminRoleRow
	Users                  []OrgAdminUserRow
	Invites                []OrgAdminInviteRow
	InviteLink             string
	Error                  string
}

type OrgAdminErrors struct {
	Organization string
	Role         string
	RoleAction   string
	RoleSlug     string
	RoleName     string
	RolePalette  string
	Invite       string
	Users        string
}

type OrgAdminRoleOption struct {
	Slug       string
	Name       string
	RoleColor  template.CSS
	RoleBorder template.CSS
	Selected   bool
}

type OrgAdminRoleRow struct {
	Slug       string
	Name       string
	Palette    string
	RoleColor  template.CSS
	RoleBorder template.CSS
	InUse      bool
}

type OrgAdminUserRow struct {
	UserID      string
	Email       string
	Status      string
	Activated   bool
	IsOrgAdmin  bool
	RoleOptions []OrgAdminRoleOption
}

type OrgAdminInviteRow struct {
	Email      string
	RoleSlugs  []string
	InviteLink string
	CreatedAt  time.Time
	ExpiresAt  time.Time
	UsedAt     *time.Time
	Status     string
}

type organizationLogoUpload struct {
	Filename    string
	ContentType string
	Data        []byte
}

func roleSlugsKey(roleSlugs []string) string {
	canon := canonifyRoleSlugs(roleSlugs)
	sorted := append([]string(nil), canon...)
	sort.Strings(sorted)
	return strings.Join(sorted, ",")
}

type WorkflowRefValidationError struct {
	Messages []string
}

func (e *WorkflowRefValidationError) Error() string {
	if e == nil || len(e.Messages) == 0 {
		return "workflow references are invalid"
	}
	return "workflow references are invalid:\n- " + strings.Join(e.Messages, "\n- ")
}

type ProcessPageView struct {
	PageBase
	ProcessID   string
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
	Integrity    DPPIntegrityView
	Export       NotarizedProcessExport
}

type DPPIntegrityView struct {
	Root   DPPIntegrityHashView
	Leaves []DPPIntegrityLeafView
}

type DPPIntegrityHashView struct {
	Full  string
	Short string
}

type DPPIntegrityLeafView struct {
	SubstepID string
	Hash      DPPIntegrityHashView
}

type DPPTraceabilityStep struct {
	StepID           string
	Title            string
	OrganizationName string
	CompletedAt      string
	CompletedAtHuman string
	DetailsDialogID  string
	Substeps         []DPPTraceabilitySubstep
}

type DPPTraceabilitySubstep struct {
	SubstepID   string
	Title       string
	Role        string
	RoleBadges  []DPPTraceabilityRoleBadge
	RoleColor   template.CSS
	RoleBorder  template.CSS
	Status      string
	DoneAt      string
	DoneAtHuman string
	DoneBy      string
	Digest      string
	Values      []DPPTraceabilityValue
	Attachments []ActionAttachmentView
}

type DPPTraceabilityRoleBadge struct {
	ID     string
	Label  string
	Color  template.CSS
	Border template.CSS
}

type DPPTraceabilityValue struct {
	Key   string
	Value string
}

type rolePaletteStyle struct {
	Color  string
	Border string
}

var rolePaletteStyles = map[string]rolePaletteStyle{
	"red":     {Color: "var(--role-red-bg)", Border: "var(--role-red-border)"},
	"orange":  {Color: "var(--role-orange-bg)", Border: "var(--role-orange-border)"},
	"amber":   {Color: "var(--role-amber-bg)", Border: "var(--role-amber-border)"},
	"yellow":  {Color: "var(--role-yellow-bg)", Border: "var(--role-yellow-border)"},
	"lime":    {Color: "var(--role-lime-bg)", Border: "var(--role-lime-border)"},
	"green":   {Color: "var(--role-green-bg)", Border: "var(--role-green-border)"},
	"emerald": {Color: "var(--role-emerald-bg)", Border: "var(--role-emerald-border)"},
	"teal":    {Color: "var(--role-teal-bg)", Border: "var(--role-teal-border)"},
	"cyan":    {Color: "var(--role-cyan-bg)", Border: "var(--role-cyan-border)"},
	"sky":     {Color: "var(--role-sky-bg)", Border: "var(--role-sky-border)"},
	"blue":    {Color: "var(--role-blue-bg)", Border: "var(--role-blue-border)"},
	"indigo":  {Color: "var(--role-indigo-bg)", Border: "var(--role-indigo-border)"},
	"violet":  {Color: "var(--role-violet-bg)", Border: "var(--role-violet-border)"},
	"purple":  {Color: "var(--role-purple-bg)", Border: "var(--role-purple-border)"},
	"fuchsia": {Color: "var(--role-fuchsia-bg)", Border: "var(--role-fuchsia-border)"},
	"pink":    {Color: "var(--role-pink-bg)", Border: "var(--role-pink-border)"},
	"rose":    {Color: "var(--role-rose-bg)", Border: "var(--role-rose-border)"},
}

var rolePaletteKeys = []string{
	"red", "orange", "amber", "yellow", "lime", "green", "emerald", "teal", "cyan",
	"sky", "blue", "indigo", "violet", "purple", "fuchsia", "pink", "rose",
}

func resolveRolePaletteStyle(palette string) rolePaletteStyle {
	key := canonifySlug(palette)
	if style, ok := rolePaletteStyles[key]; ok {
		return style
	}
	return rolePaletteStyles["red"]
}

func defaultRolePaletteFromInput(raw string) string {
	normalized := strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(raw)), " "))
	if normalized == "" || len(rolePaletteKeys) == 0 {
		return "red"
	}
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(normalized))
	return rolePaletteKeys[int(hasher.Sum32()%uint32(len(rolePaletteKeys)))]
}

func rolePaletteKeyFromStyle(color, border, fallbackName string) string {
	trimmedColor := strings.TrimSpace(color)
	trimmedBorder := strings.TrimSpace(border)
	for key, style := range rolePaletteStyles {
		if style.Color == trimmedColor && style.Border == trimmedBorder {
			return key
		}
	}
	return defaultRolePaletteFromInput(fallbackName)
}

func resolveRoleBadgeStyle(color, border string) rolePaletteStyle {
	resolvedColor := strings.TrimSpace(color)
	if resolvedColor == "" {
		resolvedColor = "var(--role-fallback)"
	}
	resolvedBorder := strings.TrimSpace(border)
	if resolvedBorder == "" {
		resolvedBorder = "var(--border)"
	}
	return rolePaletteStyle{
		Color:  resolvedColor,
		Border: resolvedBorder,
	}
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
		identity:      NewAppwriteIdentity(envOr("APPWRITE_ENDPOINT", "http://appwrite/v1"), strings.TrimSpace(os.Getenv("APPWRITE_PROJECT_ID")), strings.TrimSpace(os.Getenv("APPWRITE_API_KEY")), http.DefaultClient),
		tmpl:          tmpl,
		authorizer:    NewCerbosAuthorizer(envOr("CERBOS_URL", "http://localhost:3592"), http.DefaultClient, time.Now),
		sse:           newSSEHub(),
		now:           time.Now,
		workflowDefID: primitive.NewObjectID(),
		configDir:     configDir,
		viteDevServer: strings.TrimRight(strings.TrimSpace(os.Getenv("VITE_DEV_SERVER")), "/"),
		enforceAuth:   true,
	}
	if err := bootstrapFormataBuilderStreams(ctx, server.store, configDir, server.now); err != nil {
		log.Fatal(err)
	}
	if err := server.bootstrapPlatformAdminIdentity(ctx); err != nil {
		log.Fatal(err)
	}

	mux := server.newMux()

	addr := ":3000"
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("server listening on %s", addr)
	if err := http.Serve(listener, logRequests(mux)); err != nil {
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

const (
	noticePasswordResetSuccess = "password_reset_success"
	noticeResetRequestSent     = "reset_request_sent"
)

func clearCookie(w http.ResponseWriter, r *http.Request, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     strings.TrimSpace(name),
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   shouldSecureCookie(r),
	})
}

func requestNotice(r *http.Request) string {
	if r == nil || r.URL == nil {
		return ""
	}
	return strings.TrimSpace(r.URL.Query().Get("notice"))
}

func loginNoticeMessage(code string) string {
	switch strings.TrimSpace(code) {
	case noticePasswordResetSuccess:
		return "Password reset successfully. Now you can enter with your new credentials."
	default:
		return ""
	}
}

func resetRequestNoticeMessage(code string) string {
	switch strings.TrimSpace(code) {
	case noticeResetRequestSent:
		return "If the account exists, a reset link has been sent."
	default:
		return ""
	}
}

func anyoneCanCreateAccount() bool {
	return boolEnvOr("ANYONE_CAN_CREATE_ACCOUNT", true)
}

func newSessionID() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

const platformAdminSessionPrefix = "platform-admin:"

func platformAdminCredentials() (string, string, bool) {
	email := strings.ToLower(strings.TrimSpace(os.Getenv("ADMIN_EMAIL")))
	password := strings.TrimSpace(os.Getenv("ADMIN_PASSWORD"))
	if email == "" || password == "" {
		return "", "", false
	}
	return email, password, true
}

func isPlatformAdminEmail(email string) bool {
	adminEmail, _, ok := platformAdminCredentials()
	if !ok {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(email), adminEmail)
}

func isPlatformAdminIdentityUser(user IdentityUser) bool {
	return isPlatformAdminEmail(user.Email)
}

func isPlatformAdminMembership(membership IdentityMembership) bool {
	return isPlatformAdminEmail(membership.Email)
}

func platformAdminSessionValue() string {
	email, password, ok := platformAdminCredentials()
	if !ok {
		return ""
	}
	sum := sha256.Sum256([]byte(email + "\n" + password))
	return platformAdminSessionPrefix + hex.EncodeToString(sum[:])
}

func isPlatformAdminSessionValue(value string) bool {
	expected := platformAdminSessionValue()
	actual := strings.TrimSpace(value)
	if expected == "" || actual == "" || len(expected) != len(actual) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) == 1
}

func platformAdminAccountUser() *AccountUser {
	email, _, ok := platformAdminCredentials()
	if !ok {
		return nil
	}
	return &AccountUser{
		ID:              stableObjectID("platform-admin:" + email),
		Email:           email,
		IsPlatformAdmin: true,
		Status:          "active",
	}
}

func platformAdminStreamUserID() string {
	if user := platformAdminAccountUser(); user != nil {
		email := strings.TrimSpace(user.Email)
		if email != "" {
			return "platform-admin:" + email
		}
	}
	return "platform-admin"
}

func formataStreamUserID(user *AccountUser) string {
	if user == nil {
		return ""
	}
	if trimmed := strings.TrimSpace(user.IdentityUserID); trimmed != "" {
		return trimmed
	}
	if user.IsPlatformAdmin {
		return platformAdminStreamUserID()
	}
	return ""
}

func (s *Server) platformAdminSession() (*IdentitySession, *AccountUser, bool) {
	user := platformAdminAccountUser()
	if user == nil {
		return nil, nil, false
	}
	session := &IdentitySession{
		Secret:    platformAdminSessionValue(),
		UserID:    platformAdminStreamUserID(),
		ExpiresAt: s.nowUTC().Add(time.Duration(sessionTTLDays()) * 24 * time.Hour),
	}
	return session, user, true
}

func (s *Server) platformAdminIdentitySession(ctx context.Context) (*IdentitySession, error) {
	if s.identity == nil {
		return nil, ErrIdentityUnauthorized
	}
	email, password, ok := platformAdminCredentials()
	if !ok {
		return nil, ErrIdentityUnauthorized
	}
	session, err := s.identity.CreateEmailPasswordSession(ctx, email, password)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *Server) bootstrapPlatformAdminIdentity(ctx context.Context) error {
	if s.identity == nil {
		return nil
	}
	email, password, ok := platformAdminCredentials()
	if !ok {
		return nil
	}
	return s.identity.EnsurePlatformAdminAccount(ctx, email, password)
}

func (s *Server) ensurePlatformAdminOwnsOrganization(ctx context.Context, orgSlug, redirectURL string) (*IdentitySession, error) {
	session, err := s.platformAdminIdentitySession(ctx)
	if err != nil {
		return nil, err
	}
	currentUser, err := s.identity.GetCurrentUser(ctx, session.Secret)
	if err != nil {
		_ = s.identity.DeleteSession(ctx, session.Secret)
		return nil, err
	}
	memberships, err := s.identity.ListOrganizationMemberships(ctx, orgSlug)
	if err != nil {
		_ = s.identity.DeleteSession(ctx, session.Secret)
		return nil, err
	}
	for _, membership := range memberships {
		if currentUser.ID != "" && strings.TrimSpace(membership.UserID) == strings.TrimSpace(currentUser.ID) {
			if membership.IsOrgAdmin {
				return session, nil
			}
			if _, err := s.identity.UpdateOrganizationMembershipAsAdmin(ctx, orgSlug, membership.ID, membership.RoleSlugs, true); err != nil {
				_ = s.identity.DeleteSession(ctx, session.Secret)
				return nil, err
			}
			return session, nil
		}
		if strings.EqualFold(strings.TrimSpace(membership.Email), strings.TrimSpace(currentUser.Email)) {
			if membership.IsOrgAdmin {
				return session, nil
			}
			if _, err := s.identity.UpdateOrganizationMembershipAsAdmin(ctx, orgSlug, membership.ID, membership.RoleSlugs, true); err != nil {
				_ = s.identity.DeleteSession(ctx, session.Secret)
				return nil, err
			}
			return session, nil
		}
	}
	switch {
	case strings.TrimSpace(currentUser.ID) != "":
		if _, err := s.identity.AddOrganizationUserByIDAsAdmin(ctx, orgSlug, currentUser.ID, nil, true); err != nil {
			_ = s.identity.DeleteSession(ctx, session.Secret)
			return nil, err
		}
	case strings.TrimSpace(currentUser.Email) != "":
		if _, err := s.identity.InviteOrganizationUserAsAdmin(ctx, orgSlug, currentUser.Email, redirectURL, nil, true); err != nil {
			_ = s.identity.DeleteSession(ctx, session.Secret)
			return nil, err
		}
	default:
		_ = s.identity.DeleteSession(ctx, session.Secret)
		return nil, ErrIdentityUnauthorized
	}
	return session, nil
}

func (s *Server) readSession(r *http.Request) (*IdentitySession, error) {
	cookie, err := r.Cookie("attesta_session")
	if err != nil {
		return nil, err
	}
	sessionID := strings.TrimSpace(cookie.Value)
	if sessionID == "" {
		return nil, ErrIdentityUnauthorized
	}
	if isPlatformAdminSessionValue(sessionID) {
		session, _, ok := s.platformAdminSession()
		if ok {
			return session, nil
		}
		return nil, ErrIdentityUnauthorized
	}
	if s.identity == nil {
		return nil, ErrIdentityUnauthorized
	}
	session, err := s.identity.GetSession(r.Context(), sessionID)
	if err != nil {
		return nil, err
	}
	if session.ExpiresAt.Before(s.nowUTC()) {
		_ = s.identity.DeleteSession(r.Context(), sessionID)
		return nil, ErrIdentityUnauthorized
	}
	return &session, nil
}

func sessionSecretFromRequest(r *http.Request) (string, error) {
	cookie, err := r.Cookie("attesta_session")
	if err != nil {
		return "", err
	}
	secret := strings.TrimSpace(cookie.Value)
	if secret == "" {
		return "", ErrIdentityUnauthorized
	}
	return secret, nil
}

func (s *Server) currentUser(r *http.Request) (*AccountUser, *IdentitySession, error) {
	session, err := s.readSession(r)
	if err != nil {
		return nil, nil, err
	}
	if isPlatformAdminSessionValue(session.Secret) {
		if _, user, ok := s.platformAdminSession(); ok {
			return user, session, nil
		}
		return nil, nil, ErrIdentityUnauthorized
	}
	if s.identity == nil {
		return nil, nil, ErrIdentityUnauthorized
	}
	identityUser, err := s.identity.GetCurrentUser(r.Context(), session.Secret)
	if err != nil {
		return nil, nil, err
	}
	return s.accountUserFromIdentity(r.Context(), identityUser), session, nil
}

func (s *Server) requireAuthenticatedPage(w http.ResponseWriter, r *http.Request) (*AccountUser, *IdentitySession, bool) {
	if !s.enforceAuth {
		return &AccountUser{}, nil, true
	}
	user, session, err := s.currentUser(r)
	if err == nil {
		return user, session, true
	}
	target := "/login?next=" + url.QueryEscape(r.URL.RequestURI())
	http.Redirect(w, r, target, http.StatusSeeOther)
	return nil, nil, false
}

func (s *Server) requireAuthenticatedPost(w http.ResponseWriter, r *http.Request) (*AccountUser, *IdentitySession, bool) {
	if !s.enforceAuth {
		return &AccountUser{}, nil, true
	}
	user, session, err := s.currentUser(r)
	if err == nil {
		return user, session, true
	}
	http.Error(w, "unauthorized", http.StatusUnauthorized)
	return nil, nil, false
}

func (s *Server) accountUserFromIdentity(ctx context.Context, identityUser IdentityUser) *AccountUser {
	_ = ctx
	roleSlugs := decodeIdentityRoleLabels(identityUser.Labels)
	if identityUser.IsOrgAdmin {
		roleSlugs = canonifyRoleSlugs(append(roleSlugs, "org-admin"))
	}
	user := &AccountUser{
		IdentityUserID: strings.TrimSpace(identityUser.ID),
		Email:          strings.TrimSpace(identityUser.Email),
		OrgSlug:        strings.TrimSpace(identityUser.OrgSlug),
		RoleSlugs:      roleSlugs,
		Status:         strings.TrimSpace(identityUser.Status),
	}
	if user.OrgSlug != "" {
		orgID := stableOrgObjectID(user.OrgSlug)
		user.OrgID = &orgID
	}
	if user.Status == "" {
		user.Status = "active"
	}
	return user
}

func bootstrapFormataBuilderStreams(ctx context.Context, store Store, configDir string, now func() time.Time) error {
	if store == nil {
		return nil
	}
	existing, err := store.ListFormataBuilderStreams(ctx)
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		return nil
	}

	dir := strings.TrimSpace(configDir)
	if dir == "" {
		dir = "config"
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("config dir not found: %w", err)
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
		return errors.New("workflow config catalog is empty")
	}

	updatedAt := time.Now().UTC()
	if now != nil {
		updatedAt = now().UTC()
	}
	for _, path := range paths {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read config %s: %w", path, readErr)
		}
		creatorID := platformAdminStreamUserID()
		if _, saveErr := store.SaveFormataBuilderStream(ctx, FormataBuilderStream{
			Stream:          string(data),
			UpdatedAt:       updatedAt,
			CreatedByUserID: creatorID,
			UpdatedByUserID: creatorID,
		}); saveErr != nil {
			return fmt.Errorf("seed formata stream %s: %w", filepath.Base(path), saveErr)
		}
	}
	return nil
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

func organizationLogoMaxBytes() int64 {
	const defaultMaxBytes = int64(5 * 1024 * 1024)
	raw := strings.TrimSpace(os.Getenv("ORG_LOGO_MAX_BYTES"))
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

func logRequestError(r *http.Request, err error, message string, args ...interface{}) {
	if err == nil {
		return
	}
	summary := fmt.Sprintf(message, args...)
	if r == nil {
		log.Printf("%s: %v", summary, err)
		return
	}
	path := ""
	if r.URL != nil {
		path = r.URL.Path
	}
	log.Printf("%s %s: %s: %v", r.Method, path, summary, err)
}

func logAndHTTPError(w http.ResponseWriter, r *http.Request, status int, userMessage string, err error, message string, args ...interface{}) {
	logRequestError(r, err, message, args...)
	http.Error(w, userMessage, status)
}

func (s *Server) logAndRenderPlatformAdminError(w http.ResponseWriter, r *http.Request, user *AccountUser, confirmation string, errs PlatformAdminErrors, err error, message string, args ...interface{}) {
	logRequestError(r, err, message, args...)
	s.renderPlatformAdmin(w, user, confirmation, errs)
}

func (s *Server) logAndRenderOrgAdminError(w http.ResponseWriter, r *http.Request, user *AccountUser, orgSlug, inviteLink string, errs OrgAdminErrors, err error, message string, args ...interface{}) {
	logRequestError(r, err, message, args...)
	s.renderOrgAdminWithErrors(w, user, orgSlug, inviteLink, errs)
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
	return true
}

func userHasOrganizationContext(user *AccountUser) bool {
	if user == nil {
		return false
	}
	if user.OrgID == nil {
		return false
	}
	return strings.TrimSpace(user.OrgSlug) != ""
}

const (
	cerbosResourceCatalog              = "catalog"
	cerbosResourceFormataBuilder       = "formata_builder"
	cerbosResourceOrgAdminConsole      = "org_admin_console"
	cerbosResourcePlatformAdminConsole = "platform_admin_console"

	cerbosActionAccess = "access"
	cerbosActionEdit   = "edit"
	cerbosActionPurge  = "purge_history"
	cerbosActionSave   = "save"
	cerbosActionView   = "view"
)

func (s *Server) authorizeUserAction(ctx context.Context, user *AccountUser, resourceKind, resourceID string, resourceAttr map[string]interface{}, action string) (bool, error) {
	if user == nil {
		return false, nil
	}
	if s == nil || s.authorizer == nil {
		return false, errors.New("authorizer unavailable")
	}
	return s.authorizer.CanAccess(ctx, user, resourceKind, resourceID, resourceAttr, action)
}

func (s *Server) canAccessPlatformAdminConsole(ctx context.Context, user *AccountUser) (bool, error) {
	return s.authorizeUserAction(ctx, user, cerbosResourcePlatformAdminConsole, "platform-admin-console", nil, cerbosActionAccess)
}

func (s *Server) canAccessOrgAdminConsole(ctx context.Context, user *AccountUser) (bool, error) {
	return s.authorizeUserAction(ctx, user, cerbosResourceOrgAdminConsole, "org-admin-console", map[string]interface{}{
		"orgSlug": strings.TrimSpace(user.OrgSlug),
	}, cerbosActionAccess)
}

func (s *Server) canViewCatalog(ctx context.Context, user *AccountUser) (bool, error) {
	return s.authorizeUserAction(ctx, user, cerbosResourceCatalog, "public-catalog", nil, cerbosActionView)
}

func (s *Server) canViewFormataBuilder(ctx context.Context, user *AccountUser) (bool, error) {
	return s.authorizeUserAction(ctx, user, cerbosResourceFormataBuilder, "formata-builder", nil, cerbosActionView)
}

func (s *Server) canSaveFormataBuilder(ctx context.Context, user *AccountUser) (bool, error) {
	return s.authorizeUserAction(ctx, user, cerbosResourceFormataBuilder, "formata-builder", nil, cerbosActionSave)
}

func (s *Server) canEditStream(ctx context.Context, user *AccountUser, workflowKey string, createdByUserID string, hasProcesses bool) (bool, error) {
	return s.authorizeUserAction(ctx, user, "stream", strings.TrimSpace(workflowKey), map[string]interface{}{
		"workflowKey":     strings.TrimSpace(workflowKey),
		"createdByUserId": strings.TrimSpace(createdByUserID),
		"hasProcesses":    hasProcesses,
	}, cerbosActionEdit)
}

func (s *Server) canPurgeWorkflowData(ctx context.Context, user *AccountUser, workflowKey string) (bool, error) {
	return s.authorizeUserAction(ctx, user, "stream", strings.TrimSpace(workflowKey), map[string]interface{}{
		"workflowKey": strings.TrimSpace(workflowKey),
	}, cerbosActionPurge)
}

func logCapabilityCheckError(err error, message string, args ...interface{}) {
	if err == nil {
		return
	}
	log.Printf(message+": %v", append(args, err)...)
}

func (s *Server) pageBaseForUser(user *AccountUser, body, workflowKey, workflowName string) PageBase {
	base := s.pageBase(body, workflowKey, workflowName)
	if user == nil {
		return base
	}
	base.UserEmail = strings.TrimSpace(user.Email)
	base.IsPlatformAdmin = user.IsPlatformAdmin
	base.ShowLogout = s.enforceAuth
	showOrgsLink, err := s.canAccessPlatformAdminConsole(context.Background(), user)
	if err != nil {
		logCapabilityCheckError(err, "cerbos check failed for platform admin navigation")
	}
	base.ShowOrgsLink = showOrgsLink
	showMyOrgLink, err := s.canAccessOrgAdminConsole(context.Background(), user)
	if err != nil {
		logCapabilityCheckError(err, "cerbos check failed for org admin navigation")
	}
	base.ShowMyOrgLink = showMyOrgLink
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

func formataStreamCreatorID(stream FormataBuilderStream) string {
	if trimmed := strings.TrimSpace(stream.CreatedByUserID); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(stream.UpdatedByUserID)
}

func homePickerMessage(r *http.Request, key string) string {
	if r == nil || r.URL == nil {
		return ""
	}
	return strings.TrimSpace(r.URL.Query().Get(key))
}

func (s *Server) workflowOptions(ctx context.Context, user *AccountUser) ([]WorkflowOption, error) {
	catalog, err := s.workflowCatalog()
	if err != nil {
		return nil, err
	}
	canEditSavedStreams := false
	if user != nil {
		if allowed, err := s.canViewFormataBuilder(ctx, user); err == nil {
			canEditSavedStreams = allowed
		}
	}
	streamsByKey := map[string]FormataBuilderStream{}
	if s.store != nil {
		streams, err := s.store.ListFormataBuilderStreams(ctx)
		if err != nil {
			return nil, err
		}
		for _, stream := range streams {
			if stream.ID.IsZero() {
				continue
			}
			streamsByKey[stream.ID.Hex()] = stream
		}
	}
	keys := sortedWorkflowKeys(catalog)
	options := make([]WorkflowOption, 0, len(keys))
	for _, key := range keys {
		cfg := catalog[key]
		option := WorkflowOption{
			Key:          key,
			Name:         cfg.Workflow.Name,
			Description:  strings.TrimSpace(cfg.Workflow.Description),
			Counts:       WorkflowProcessCounts{},
			EditAction:   "/org-admin/formata-builder?stream=" + key,
			DeleteAction: workflowPath(key) + "/delete",
		}
		if s.store == nil {
			options = append(options, option)
			continue
		}
		processes, listErr := s.store.ListRecentProcessesByWorkflow(ctx, key, 0)
		if listErr != nil {
			return nil, listErr
		}
		option.Counts = workflowProcessCounts(cfg.Workflow, processes)
		stream, ok := streamsByKey[key]
		if ok && s.authorizer != nil && user != nil {
			hasProcesses := option.Counts.NotStarted+option.Counts.Started+option.Counts.Terminated > 0
			option.CanClone = canEditSavedStreams
			if canEditSavedStreams {
				allowed, err := s.canEditStream(ctx, user, key, formataStreamCreatorID(stream), hasProcesses)
				if err == nil {
					option.CanEdit = allowed
					option.EditRequiresPurge = allowed && hasProcesses
				}
			}
			allowed, err := s.authorizer.CanDeleteStream(ctx, user, key, formataStreamCreatorID(stream), hasProcesses)
			if err == nil {
				option.CanDelete = allowed
			}
		}
		options = append(options, option)
	}
	return options, nil
}

func (s *Server) selectedWorkflowUnvalidated(r *http.Request) (string, RuntimeConfig, error) {
	if value := r.Context().Value(workflowContextKey{}); value != nil {
		if selected, ok := value.(workflowContextValue); ok {
			return selected.Key, selected.Cfg, nil
		}
	}
	cfg, err := s.runtimeConfig()
	if err != nil {
		return "", RuntimeConfig{}, err
	}
	return s.defaultWorkflowKey(), cfg, nil
}

func (s *Server) selectedWorkflow(r *http.Request) (string, RuntimeConfig, error) {
	key, cfg, err := s.selectedWorkflowUnvalidated(r)
	if err != nil {
		return "", RuntimeConfig{}, err
	}
	if err := s.validateWorkflowRefs(r.Context(), cfg); err != nil {
		return "", RuntimeConfig{}, err
	}
	return key, cfg, nil
}

func (s *Server) selectedWorkflowOrRedirectHome(w http.ResponseWriter, r *http.Request) (string, RuntimeConfig, bool) {
	workflowKey, cfg, err := s.selectedWorkflow(r)
	if err == nil {
		return workflowKey, cfg, true
	}

	var validationErr *WorkflowRefValidationError
	if errors.As(err, &validationErr) {
		if fallbackKey, _, fallbackErr := s.selectedWorkflowUnvalidated(r); fallbackErr == nil && strings.TrimSpace(fallbackKey) != "" {
			redirectWorkflowHomeWithMessage(w, r, fallbackKey, validationErr.Error())
			return "", RuntimeConfig{}, false
		}
	}

	http.Error(w, err.Error(), http.StatusInternalServerError)
	return "", RuntimeConfig{}, false
}

func (s *Server) validateWorkflowRefs(ctx context.Context, cfg RuntimeConfig) error {
	if s == nil || s.identity == nil {
		return nil
	}
	if !s.enforceAuth {
		return nil
	}
	if len(cfg.Organizations) == 0 && len(cfg.Roles) == 0 {
		return nil
	}

	messages := []string{}
	orgs, err := s.identity.ListOrganizations(ctx)
	if err != nil {
		return err
	}
	orgsBySlug := make(map[string]IdentityOrg, len(orgs))
	for _, org := range orgs {
		orgsBySlug[strings.TrimSpace(org.Slug)] = org
	}
	yamlOrgs := map[string]struct{}{}
	for _, org := range cfg.Organizations {
		slug := strings.TrimSpace(org.Slug)
		if slug == "" {
			continue
		}
		yamlOrgs[slug] = struct{}{}
		if _, ok := orgsBySlug[slug]; !ok {
			messages = append(messages, "missing organization slug "+slug)
		}
	}

	yamlRolesByOrg := map[string]map[string]struct{}{}
	yamlRoleOrgs := map[string][]string{}
	for _, role := range cfg.Roles {
		orgSlug := strings.TrimSpace(role.OrgSlug)
		roleSlug := strings.TrimSpace(role.Slug)
		if orgSlug == "" || roleSlug == "" {
			continue
		}
		if _, ok := yamlRolesByOrg[orgSlug]; !ok {
			yamlRolesByOrg[orgSlug] = map[string]struct{}{}
		}
		yamlRolesByOrg[orgSlug][roleSlug] = struct{}{}
		orgs := yamlRoleOrgs[roleSlug]
		foundOrg := false
		for _, existingOrg := range orgs {
			if existingOrg == orgSlug {
				foundOrg = true
				break
			}
		}
		if !foundOrg {
			yamlRoleOrgs[roleSlug] = append(orgs, orgSlug)
		}
		org, ok := orgsBySlug[orgSlug]
		if !ok || !identityOrgHasRole(org, roleSlug) {
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
				if stepOrg != "" {
					if _, ok := yamlRolesByOrg[stepOrg][trimmedRole]; !ok {
						if len(yamlRoleOrgs[trimmedRole]) == 0 {
							messages = append(messages, "substep "+sub.SubstepID+" references role not in yaml: "+trimmedRole)
						} else {
							messages = append(messages, "substep "+sub.SubstepID+" role "+trimmedRole+" not in step organization "+stepOrg)
						}
						continue
					}
					if !identityOrgHasRole(orgsBySlug[stepOrg], trimmedRole) {
						messages = append(messages, "missing role slug "+stepOrg+"/"+trimmedRole)
					}
					continue
				}

				roleOrgs := yamlRoleOrgs[trimmedRole]
				if len(roleOrgs) == 0 {
					messages = append(messages, "substep "+sub.SubstepID+" references role not in yaml: "+trimmedRole)
					continue
				}
				foundRole := false
				for _, orgSlug := range roleOrgs {
					if identityOrgHasRole(orgsBySlug[orgSlug], trimmedRole) {
						foundRole = true
						break
					}
				}
				if !foundRole {
					messages = append(messages, "missing role slug "+roleOrgs[0]+"/"+trimmedRole)
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

func normalizeHomeStatusFilter(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "active", "done":
		return strings.TrimSpace(strings.ToLower(value))
	default:
		return "all"
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
		lastAt = humanReadableTraceabilityTime(latest)
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

func redirectHomeWithMessage(w http.ResponseWriter, r *http.Request, key, message string) {
	target := "/"
	trimmedKey := strings.TrimSpace(key)
	trimmedMessage := strings.TrimSpace(message)
	if trimmedKey != "" && trimmedMessage != "" {
		target += "?" + url.Values{trimmedKey: []string{trimmedMessage}}.Encode()
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func redirectWorkflowHomeWithMessage(w http.ResponseWriter, r *http.Request, workflowKey, message string) {
	target := workflowPath(workflowKey) + "/"
	trimmedMessage := strings.TrimSpace(message)
	if trimmedMessage != "" {
		target += "?" + url.Values{"error": []string{trimmedMessage}}.Encode()
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
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
	options, err := s.workflowOptions(r.Context(), user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	showCreateStreamCard, authErr := s.canViewFormataBuilder(r.Context(), user)
	if authErr != nil {
		logRequestError(r, authErr, "cerbos check failed for formata builder card")
	}
	view := HomeWorkflowPickerView{
		WorkflowPickerView: WorkflowPickerView{
			PageBase:             s.pageBaseForUser(user, "home_picker_body", "", ""),
			Workflows:            options,
			ShowCreateStreamCard: showCreateStreamCard,
			Error:                homePickerMessage(r, "error"),
			Confirmation:         homePickerMessage(r, "confirmation"),
		},
	}
	if err := s.tmpl.ExecuteTemplate(w, "home.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleAbout(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
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

func (s *Server) handlePublicCatalog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, ok := s.requireCatalogAccessAPI(w, r); !ok {
		return
	}
	if s == nil || s.identity == nil {
		http.Error(w, "catalog not configured", http.StatusInternalServerError)
		return
	}

	organizations, err := s.identity.ListOrganizations(r.Context())
	if err != nil {
		http.Error(w, "failed to list organizations", http.StatusInternalServerError)
		return
	}

	response := PublicCatalogResponse{
		Organizations: make([]PublicCatalogOrganization, 0, len(organizations)),
		Roles:         []PublicCatalogRole{},
	}
	for _, org := range organizations {
		orgSlug := strings.TrimSpace(org.Slug)
		if orgSlug == "" {
			continue
		}
		response.Organizations = append(response.Organizations, PublicCatalogOrganization{
			Name: strings.TrimSpace(org.Name),
			Slug: orgSlug,
		})

		for _, role := range org.Roles {
			if canonifySlug(role.Slug) == "org-admin" {
				continue
			}
			response.Roles = append(response.Roles, PublicCatalogRole{
				OrgSlug: orgSlug,
				Name:    strings.TrimSpace(role.Name),
				Slug:    strings.TrimSpace(role.Slug),
				Color:   strings.TrimSpace(role.Color),
				Border:  strings.TrimSpace(role.Border),
			})
		}
	}
	sort.Slice(response.Organizations, func(i, j int) bool {
		if response.Organizations[i].Name == response.Organizations[j].Name {
			return response.Organizations[i].Slug < response.Organizations[j].Slug
		}
		return response.Organizations[i].Name < response.Organizations[j].Name
	})
	sort.Slice(response.Roles, func(i, j int) bool {
		if response.Roles[i].OrgSlug != response.Roles[j].OrgSlug {
			return response.Roles[i].OrgSlug < response.Roles[j].OrgSlug
		}
		if response.Roles[i].Name != response.Roles[j].Name {
			return response.Roles[i].Name < response.Roles[j].Name
		}
		return response.Roles[i].Slug < response.Roles[j].Slug
	})

	writeJSON(w, response)
}

func (s *Server) requireCatalogAccessAPI(w http.ResponseWriter, r *http.Request) (*AccountUser, bool) {
	user, _, ok := s.requireAuthenticatedPost(w, r)
	if !ok {
		return nil, false
	}
	allowed, err := s.canViewCatalog(r.Context(), user)
	if err != nil {
		logAndHTTPError(w, r, http.StatusBadGateway, "cerbos check failed", err, "cerbos check failed for public catalog")
		return nil, false
	}
	if !allowed {
		http.Error(w, "forbidden", http.StatusForbidden)
		return nil, false
	}
	return user, true
}

func (s *Server) newMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("../web/dist"))))
	mux.HandleFunc("/docs", s.handleDocs)
	mux.HandleFunc("/docs/", s.handleDocs)
	mux.HandleFunc("/about", s.handleAbout)
	mux.HandleFunc("/api/catalog", s.handlePublicCatalog)
	mux.HandleFunc("/01/", s.handleDigitalLinkDPP)
	mux.HandleFunc("/login", s.handleLogin)
	mux.HandleFunc("/signup", s.handleSignup)
	mux.HandleFunc("/logout", s.handleLogout)
	mux.HandleFunc("/admin/orgs", s.handleAdminOrgs)
	mux.HandleFunc("/admin/orgs/", s.handleAdminOrgs)
	mux.HandleFunc("/invite/", s.handleInvite)
	mux.HandleFunc("/reset", s.handleResetRequest)
	mux.HandleFunc("/reset/", s.handleResetSet)
	mux.HandleFunc("/org-admin/roles", s.handleOrgAdminRoles)
	mux.HandleFunc("/org-admin/logo/", s.handleOrgAdminLogo)
	mux.HandleFunc("/org-admin/users", s.handleOrgAdminUsers)
	mux.HandleFunc("/org-admin/formata-builder", s.handleOrgAdminFormataBuilder)
	mux.HandleFunc("/org-admin/formata-builder/", s.handleOrgAdminFormataBuilder)
	mux.HandleFunc("/organization/logo/", s.handleOrganizationLogo)
	mux.HandleFunc("/w/", s.handleWorkflowRoutes)
	mux.HandleFunc("/", s.handleHome)
	mux.HandleFunc("/events", s.handleEvents)
	return mux
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

func (s *Server) writeSessionCookie(w http.ResponseWriter, r *http.Request, session IdentitySession) error {
	if strings.TrimSpace(session.Secret) == "" {
		return errors.New("session secret required")
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "attesta_session",
		Value:    session.Secret,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   shouldSecureCookie(r),
	})
	return nil
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if s.enforceAuth {
			if _, _, err := s.currentUser(r); err == nil {
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
		}
		view := LoginView{
			PageBase:     s.pageBase("login_body", "", ""),
			Next:         safeNextPath(r, "/"),
			Confirmation: loginNoticeMessage(requestNotice(r)),
			ShowSignup:   anyoneCanCreateAccount(),
		}
		if err := s.tmpl.ExecuteTemplate(w, "login.html", view); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			logAndHTTPError(w, r, http.StatusBadRequest, "invalid form", err, "failed to parse login form")
			return
		}
		email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
		password := strings.TrimSpace(r.FormValue("password"))
		next := safeNextPath(r, "/")

		if adminEmail, adminPassword, ok := platformAdminCredentials(); ok && strings.EqualFold(email, adminEmail) {
			if subtle.ConstantTimeCompare([]byte(password), []byte(adminPassword)) != 1 {
				view := LoginView{
					PageBase:   s.pageBase("login_body", "", ""),
					Email:      email,
					Next:       next,
					Error:      "Invalid email or password.",
					ShowSignup: anyoneCanCreateAccount(),
				}
				w.WriteHeader(http.StatusUnauthorized)
				_ = s.tmpl.ExecuteTemplate(w, "login.html", view)
				return
			}
			session, _, ok := s.platformAdminSession()
			if !ok {
				http.Error(w, "login failed", http.StatusInternalServerError)
				return
			}
			if err := s.writeSessionCookie(w, r, *session); err != nil {
				logAndHTTPError(w, r, http.StatusInternalServerError, "login failed", err, "failed to write platform admin session cookie")
				return
			}
			http.Redirect(w, r, next, http.StatusSeeOther)
			return
		}

		if s.identity == nil {
			http.Error(w, "login unavailable", http.StatusServiceUnavailable)
			return
		}
		session, err := s.identity.CreateEmailPasswordSession(r.Context(), email, password)
		if isLoginCredentialError(err) {
			view := LoginView{
				PageBase:   s.pageBase("login_body", "", ""),
				Email:      email,
				Next:       next,
				Error:      "Invalid email or password.",
				ShowSignup: anyoneCanCreateAccount(),
			}
			w.WriteHeader(http.StatusUnauthorized)
			_ = s.tmpl.ExecuteTemplate(w, "login.html", view)
			return
		}
		if err != nil {
			logAndHTTPError(w, r, http.StatusInternalServerError, "login failed", err, "failed to create email/password session for %s", email)
			return
		}
		if err := s.writeSessionCookie(w, r, session); err != nil {
			logAndHTTPError(w, r, http.StatusInternalServerError, "login failed", err, "failed to write session cookie for %s", email)
			return
		}
		http.Redirect(w, r, next, http.StatusSeeOther)
		return
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleSignup(w http.ResponseWriter, r *http.Request) {
	if !anyoneCanCreateAccount() {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		if s.enforceAuth {
			if _, _, err := s.currentUser(r); err == nil {
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
		}
		view := SignupView{PageBase: s.pageBase("signup_body", "", "")}
		if err := s.tmpl.ExecuteTemplate(w, "signup.html", view); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			logAndHTTPError(w, r, http.StatusBadRequest, "invalid form", err, "failed to parse signup form")
			return
		}
		if s.identity == nil {
			http.Error(w, "signup unavailable", http.StatusServiceUnavailable)
			return
		}
		email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
		password := strings.TrimSpace(r.FormValue("password"))
		if err := validatePassword(password); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = s.tmpl.ExecuteTemplate(w, "signup.html", SignupView{
				PageBase: s.pageBase("signup_body", "", ""),
				Email:    email,
				Error:    err.Error(),
			})
			return
		}
		if _, err := s.identity.CreateAccount(r.Context(), email, password, ""); err != nil && !errors.Is(err, ErrIdentityUnauthorized) {
			logAndHTTPError(w, r, http.StatusInternalServerError, "signup failed", err, "failed to create account for %s", email)
			return
		}
		session, err := s.identity.CreateEmailPasswordSession(r.Context(), email, password)
		if err != nil {
			logAndHTTPError(w, r, http.StatusInternalServerError, "signup failed", err, "failed to create session after signup for %s", email)
			return
		}
		if err := s.writeSessionCookie(w, r, session); err != nil {
			logAndHTTPError(w, r, http.StatusInternalServerError, "signup failed", err, "failed to write signup session cookie for %s", email)
			return
		}
		identityUser, err := s.identity.GetCurrentUser(r.Context(), session.Secret)
		if err != nil {
			logAndHTTPError(w, r, http.StatusInternalServerError, "signup failed", err, "failed to load signed up user %s", email)
			return
		}
		redirectTarget := "/"
		if strings.TrimSpace(identityUser.OrgSlug) == "" {
			redirectTarget = "/org-admin/users"
		}
		http.Redirect(w, r, redirectTarget, http.StatusSeeOther)
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
		if s.identity != nil && !isPlatformAdminSessionValue(cookie.Value) {
			if deleteErr := s.identity.DeleteSession(r.Context(), strings.TrimSpace(cookie.Value)); deleteErr != nil {
				logRequestError(r, deleteErr, "failed to delete session during logout")
			}
		}
	}
	clearCookie(w, r, "attesta_session")
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func validatePassword(value string) error {
	password := strings.TrimSpace(value)
	if len(password) < 12 {
		return errors.New("password must be at least 12 characters")
	}
	return nil
}

func (s *Server) handleInvite(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/invite/accept") {
		s.handleInviteAccept(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/invite/password") {
		s.handleInvitePassword(w, r)
		return
	}
	http.NotFound(w, r)
}

func inviteAcceptParams(r *http.Request) (string, string, string, string) {
	query := r.URL.Query()
	return strings.TrimSpace(query.Get("teamId")),
		strings.TrimSpace(query.Get("membershipId")),
		strings.TrimSpace(query.Get("userId")),
		strings.TrimSpace(query.Get("secret"))
}

func (s *Server) handleInviteAccept(w http.ResponseWriter, r *http.Request) {
	if s.identity == nil {
		http.NotFound(w, r)
		return
	}
	teamID, membershipID, userID, secret := inviteAcceptParams(r)
	if teamID == "" || membershipID == "" || userID == "" || secret == "" {
		http.Error(w, "invalid invite link", http.StatusBadRequest)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	session, err := s.identity.AcceptInvite(r.Context(), teamID, membershipID, userID, secret)
	if err != nil {
		logAndHTTPError(w, r, http.StatusBadRequest, "failed to accept invite", err, "failed to accept invite team=%s membership=%s user=%s", teamID, membershipID, userID)
		return
	}
	if err := s.writeSessionCookie(w, r, session); err != nil {
		logAndHTTPError(w, r, http.StatusInternalServerError, "failed to login", err, "failed to write invite session cookie for user %s", userID)
		return
	}
	if identityUser, err := s.identity.GetCurrentUser(r.Context(), session.Secret); err == nil && !identityUser.PasswordSet {
		http.Redirect(w, r, "/invite/password", http.StatusSeeOther)
		return
	} else if err != nil {
		logRequestError(r, err, "failed to load invited user after accepting invite")
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handleInvitePassword(w http.ResponseWriter, r *http.Request) {
	if s.identity == nil {
		http.NotFound(w, r)
		return
	}
	user, session, ok := s.requireAuthenticatedPage(w, r)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		view := InviteView{
			PageBase: s.pageBaseForUser(user, "invite_body", "", ""),
			Token:    "password",
			Email:    strings.TrimSpace(user.Email),
			Org:      strings.TrimSpace(user.OrgSlug),
			Roles:    append([]string(nil), user.RoleSlugs...),
		}
		if err := s.tmpl.ExecuteTemplate(w, "invite.html", view); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			logAndHTTPError(w, r, http.StatusBadRequest, "invalid form", err, "failed to parse invite password form")
			return
		}
		password := strings.TrimSpace(r.FormValue("password"))
		confirmPassword := strings.TrimSpace(r.FormValue("confirm_password"))
		if password != confirmPassword {
			w.WriteHeader(http.StatusBadRequest)
			_ = s.tmpl.ExecuteTemplate(w, "invite.html", InviteView{
				PageBase: s.pageBaseForUser(user, "invite_body", "", ""),
				Token:    "password",
				Email:    strings.TrimSpace(user.Email),
				Org:      strings.TrimSpace(user.OrgSlug),
				Roles:    append([]string(nil), user.RoleSlugs...),
				Error:    "passwords do not match",
			})
			return
		}
		if err := validatePassword(password); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = s.tmpl.ExecuteTemplate(w, "invite.html", InviteView{
				PageBase: s.pageBaseForUser(user, "invite_body", "", ""),
				Token:    "password",
				Email:    strings.TrimSpace(user.Email),
				Org:      strings.TrimSpace(user.OrgSlug),
				Roles:    append([]string(nil), user.RoleSlugs...),
				Error:    err.Error(),
			})
			return
		}
		if err := s.identity.UpdateCurrentPassword(r.Context(), session.Secret, password); err != nil {
			logAndHTTPError(w, r, http.StatusInternalServerError, "failed to update password", err, "failed to update invited user password for %s", user.Email)
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

func requestBaseURL(r *http.Request) string {
	scheme := "http"
	if shouldSecureCookie(r) {
		scheme = "https"
	}
	host := strings.TrimSpace(r.Host)
	if host == "" {
		host = "localhost:3000"
	}
	return scheme + "://" + host
}

func resetRedirectURL(r *http.Request) string {
	if configured := strings.TrimSpace(os.Getenv("APPWRITE_RESET_REDIRECT_URL")); configured != "" {
		return configured
	}
	return requestBaseURL(r) + "/reset/confirm"
}

func inviteRedirectURL(r *http.Request) string {
	if configured := strings.TrimSpace(os.Getenv("APPWRITE_INVITE_REDIRECT_URL")); configured != "" {
		return configured
	}
	return requestBaseURL(r) + "/invite/accept"
}

func resetConfirmParams(r *http.Request) (string, string) {
	query := r.URL.Query()
	return strings.TrimSpace(query.Get("userId")), strings.TrimSpace(query.Get("secret"))
}

func (s *Server) handleResetRequest(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		view := ResetRequestView{
			PageBase:     s.pageBase("reset_request_body", "", ""),
			Confirmation: resetRequestNoticeMessage(requestNotice(r)),
		}
		if err := s.tmpl.ExecuteTemplate(w, "reset_request.html", view); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			logAndHTTPError(w, r, http.StatusBadRequest, "invalid form", err, "failed to parse reset request form")
			return
		}
		email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))

		if s.identity != nil {
			if err := s.identity.CreateRecovery(r.Context(), email, resetRedirectURL(r)); err != nil {
				logRequestError(r, err, "failed to create password recovery for %s", email)
				w.WriteHeader(http.StatusBadGateway)
				_ = s.tmpl.ExecuteTemplate(w, "reset_request.html", ResetRequestView{
					PageBase: s.pageBase("reset_request_body", "", ""),
					Email:    email,
					Error:    "Unable to send reset email right now. Please try again.",
				})
				return
			}
		}
		http.Redirect(w, r, "/reset?notice="+url.QueryEscape(noticeResetRequestSent), http.StatusSeeOther)
		return
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleResetSet(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/reset/confirm" {
		s.handleResetConfirm(w, r)
		return
	}
	http.NotFound(w, r)
}

func (s *Server) handleResetConfirm(w http.ResponseWriter, r *http.Request) {
	if s.identity == nil {
		http.NotFound(w, r)
		return
	}
	userID, secret := resetConfirmParams(r)
	if userID == "" || secret == "" {
		http.Error(w, "invalid or expired reset link", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		view := ResetSetView{
			PageBase:    s.pageBase("reset_set_body", "", ""),
			Token:       "confirm?userId=" + url.QueryEscape(userID) + "&secret=" + url.QueryEscape(secret),
			Title:       "Set New Password",
			SubmitLabel: "Update password",
		}
		if err := s.tmpl.ExecuteTemplate(w, "reset_set.html", view); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			logAndHTTPError(w, r, http.StatusBadRequest, "invalid form", err, "failed to parse reset confirmation form")
			return
		}
		password := strings.TrimSpace(r.FormValue("password"))
		confirmPassword := strings.TrimSpace(r.FormValue("confirm_password"))
		if password != confirmPassword {
			w.WriteHeader(http.StatusBadRequest)
			_ = s.tmpl.ExecuteTemplate(w, "reset_set.html", ResetSetView{
				PageBase:    s.pageBase("reset_set_body", "", ""),
				Token:       "confirm?userId=" + url.QueryEscape(userID) + "&secret=" + url.QueryEscape(secret),
				Error:       "passwords do not match",
				Title:       "Set New Password",
				SubmitLabel: "Update password",
			})
			return
		}
		if err := validatePassword(password); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = s.tmpl.ExecuteTemplate(w, "reset_set.html", ResetSetView{
				PageBase:    s.pageBase("reset_set_body", "", ""),
				Token:       "confirm?userId=" + url.QueryEscape(userID) + "&secret=" + url.QueryEscape(secret),
				Error:       err.Error(),
				Title:       "Set New Password",
				SubmitLabel: "Update password",
			})
			return
		}
		if err := s.identity.CompleteRecovery(r.Context(), userID, secret, password); err != nil {
			logAndHTTPError(w, r, http.StatusInternalServerError, "failed to reset password", err, "failed to complete password recovery for user %s", userID)
			return
		}
		http.Redirect(w, r, "/login?notice="+url.QueryEscape(noticePasswordResetSuccess), http.StatusSeeOther)
		return
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
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

func isLoginCredentialError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrIdentityUnauthorized) || errors.Is(err, ErrIdentityNotFound) {
		return true
	}
	type statusCoder interface {
		GetStatusCode() int
	}
	var appErr statusCoder
	if errors.As(err, &appErr) {
		switch appErr.GetStatusCode() {
		case http.StatusBadRequest, http.StatusUnauthorized, http.StatusNotFound:
			return true
		}
	}
	return false
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

func isSameAccount(a, b *AccountUser) bool {
	if a == nil || b == nil {
		return false
	}
	if strings.TrimSpace(a.IdentityUserID) != "" && strings.TrimSpace(b.IdentityUserID) != "" {
		return strings.TrimSpace(a.IdentityUserID) == strings.TrimSpace(b.IdentityUserID)
	}
	return !a.ID.IsZero() && !b.ID.IsZero() && a.ID == b.ID
}

func accountActorID(user *AccountUser) string {
	if user == nil {
		return "legacy-user"
	}
	if actorID := appwriteActorID(user.IdentityUserID); actorID != "" {
		return actorID
	}
	if user.ID.IsZero() {
		return "legacy-user"
	}
	return user.ID.Hex()
}

func stableObjectID(seed string) primitive.ObjectID {
	sum := sha1.Sum([]byte(strings.TrimSpace(seed)))
	var id primitive.ObjectID
	copy(id[:], sum[:len(id)])
	return id
}

func stableOrgObjectID(slug string) primitive.ObjectID {
	return stableObjectID("org:" + canonifySlug(slug))
}

const appwriteActorPrefix = "appwrite:"

func appwriteActorID(userID string) string {
	trimmed := strings.TrimSpace(userID)
	if trimmed == "" {
		return ""
	}
	return appwriteActorPrefix + trimmed
}

func parseAppwriteActorID(actorID string) (string, bool) {
	trimmed := strings.TrimSpace(actorID)
	if !strings.HasPrefix(trimmed, appwriteActorPrefix) {
		return "", false
	}
	userID := strings.TrimSpace(strings.TrimPrefix(trimmed, appwriteActorPrefix))
	if userID == "" {
		return "", false
	}
	return userID, true
}

func organizationFromIdentityOrg(org IdentityOrg) Organization {
	return Organization{
		ID:               stableOrgObjectID(org.Slug),
		Slug:             strings.TrimSpace(org.Slug),
		Name:             strings.TrimSpace(org.Name),
		LogoAttachmentID: strings.TrimSpace(org.LogoFileID),
	}
}

func rolesFromIdentityOrg(org IdentityOrg) []Role {
	orgID := stableOrgObjectID(org.Slug)
	roles := make([]Role, 0, len(org.Roles))
	for _, role := range org.Roles {
		roles = append(roles, Role{
			OrgID:   orgID,
			OrgSlug: strings.TrimSpace(org.Slug),
			Slug:    strings.TrimSpace(role.Slug),
			Name:    strings.TrimSpace(role.Name),
			Color:   strings.TrimSpace(role.Color),
			Border:  strings.TrimSpace(role.Border),
		})
	}
	return roles
}

func identityOrgHasRole(org IdentityOrg, roleSlug string) bool {
	canonRole := canonifySlug(roleSlug)
	for _, role := range org.Roles {
		if canonifySlug(role.Slug) == canonRole {
			return true
		}
	}
	return false
}

func ensureOrgAdminRoleOption(roles []Role) []Role {
	for _, role := range roles {
		if containsRole([]string{role.Slug}, "org-admin") || containsRole([]string{role.Slug}, "org_admin") {
			return roles
		}
	}
	withAdmin := make([]Role, 0, len(roles)+1)
	withAdmin = append(withAdmin, Role{Slug: "org-admin", Name: "Org Admin"})
	withAdmin = append(withAdmin, roles...)
	return withAdmin
}

func (s *Server) requireOrgAdmin(w http.ResponseWriter, r *http.Request) (*AccountUser, bool) {
	user, _, ok := s.requireAuthenticatedPage(w, r)
	if !ok {
		return nil, false
	}
	allowed, err := s.canAccessOrgAdminConsole(r.Context(), user)
	if err != nil {
		logAndHTTPError(w, r, http.StatusBadGateway, "cerbos check failed", err, "cerbos check failed for org admin console")
		return nil, false
	}
	if !allowed {
		http.Error(w, "forbidden", http.StatusForbidden)
		return nil, false
	}
	return user, true
}

func (s *Server) requirePlatformAdmin(w http.ResponseWriter, r *http.Request) (*AccountUser, bool) {
	user, _, ok := s.requireAuthenticatedPage(w, r)
	if !ok {
		return nil, false
	}
	allowed, err := s.canAccessPlatformAdminConsole(r.Context(), user)
	if err != nil {
		logAndHTTPError(w, r, http.StatusBadGateway, "cerbos check failed", err, "cerbos check failed for platform admin console")
		return nil, false
	}
	if !allowed {
		http.Error(w, "forbidden", http.StatusForbidden)
		return nil, false
	}
	return user, true
}

func (s *Server) readOrganizationLogoUpload(r *http.Request) (*organizationLogoUpload, string) {
	if r.MultipartForm == nil {
		return nil, ""
	}
	files := r.MultipartForm.File["logo"]
	if len(files) == 0 {
		return nil, ""
	}
	if len(files) != 1 {
		return nil, "upload a single logo file"
	}
	part := files[0]
	if part.Size <= 0 {
		return nil, "logo file is empty"
	}
	file, err := part.Open()
	if err != nil {
		return nil, "invalid logo upload"
	}
	defer file.Close()

	contentType := strings.TrimSpace(part.Header.Get("Content-Type"))
	header := make([]byte, 512)
	count, readErr := io.ReadFull(file, header)
	switch {
	case readErr == nil:
	case errors.Is(readErr, io.EOF), errors.Is(readErr, io.ErrUnexpectedEOF):
	default:
		return nil, "invalid logo upload"
	}
	sniffed := bytes.TrimSpace(header[:count])
	if len(sniffed) > 0 {
		detected := strings.TrimSpace(http.DetectContentType(sniffed))
		if detected != "" && detected != "application/octet-stream" {
			contentType = detected
		}
	}
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = detectAttachmentContentType(part.Filename)
	}
	if !isAllowedOrganizationLogo(contentType, part.Filename) {
		return nil, "logo must be a PNG, JPG, WEBP, or SVG image"
	}
	data, err := io.ReadAll(io.MultiReader(bytes.NewReader(header[:count]), file))
	if err != nil {
		return nil, "invalid logo upload"
	}
	return &organizationLogoUpload{
		Filename:    strings.TrimSpace(part.Filename),
		ContentType: contentType,
		Data:        data,
	}, ""
}

func (s *Server) parseOrganizationLogoUpload(ctx context.Context, r *http.Request, orgSlug string) (string, string) {
	upload, errMsg := s.readOrganizationLogoUpload(r)
	if errMsg != "" || upload == nil {
		return "", errMsg
	}
	attachment, saveErr := s.store.SaveAttachment(ctx, AttachmentUpload{
		ProcessID:   primitive.NilObjectID,
		SubstepID:   "organization-logo:" + canonifySlug(orgSlug),
		Filename:    upload.Filename,
		ContentType: upload.ContentType,
		MaxBytes:    organizationLogoMaxBytes(),
		UploadedAt:  s.nowUTC(),
	}, bytes.NewReader(upload.Data))
	if saveErr != nil {
		switch {
		case errors.Is(saveErr, ErrAttachmentTooLarge):
			return "", "logo file too large"
		default:
			return "", "failed to upload logo"
		}
	}
	return attachment.ID.Hex(), ""
}

func isAllowedOrganizationLogo(contentType, filename string) bool {
	normalizedContentType := strings.ToLower(strings.TrimSpace(contentType))
	if idx := strings.Index(normalizedContentType, ";"); idx >= 0 {
		normalizedContentType = strings.TrimSpace(normalizedContentType[:idx])
	}
	switch normalizedContentType {
	case "image/png", "image/jpeg", "image/webp", "image/svg+xml":
		return true
	}
	switch strings.ToLower(strings.TrimSpace(filepath.Ext(filename))) {
	case ".png", ".jpg", ".jpeg", ".webp", ".svg":
		return true
	}
	return false
}

func (s *Server) platformOrganizations(ctx context.Context) []Organization {
	if s.identity == nil {
		return nil
	}
	orgs, err := s.identity.ListOrganizations(ctx)
	if err != nil {
		log.Printf("failed to list platform organizations: %v", err)
		return nil
	}
	organizations := make([]Organization, 0, len(orgs))
	for _, org := range orgs {
		organizations = append(organizations, organizationFromIdentityOrg(org))
	}
	sort.Slice(organizations, func(i, j int) bool {
		if organizations[i].Name != organizations[j].Name {
			return organizations[i].Name < organizations[j].Name
		}
		return organizations[i].Slug < organizations[j].Slug
	})
	return organizations
}

const platformAdminOrganizationsPerPage = 12
const homeProcessesPerPage = 10

func filterPlatformOrganizations(organizations []Organization, query string) []Organization {
	trimmedQuery := strings.ToLower(strings.TrimSpace(query))
	if trimmedQuery == "" {
		return append([]Organization(nil), organizations...)
	}
	filtered := make([]Organization, 0, len(organizations))
	for _, organization := range organizations {
		name := strings.ToLower(strings.TrimSpace(organization.Name))
		if strings.Contains(name, trimmedQuery) {
			filtered = append(filtered, organization)
		}
	}
	return filtered
}

func normalizePlatformAdminPage(raw int, totalItems int) int {
	totalPages := 1
	if totalItems > 0 {
		totalPages = (totalItems + platformAdminOrganizationsPerPage - 1) / platformAdminOrganizationsPerPage
	}
	if raw < 1 {
		return 1
	}
	if raw > totalPages {
		return totalPages
	}
	return raw
}

func normalizeHomePage(raw int, totalItems int) int {
	totalPages := 1
	if totalItems > 0 {
		totalPages = (totalItems + homeProcessesPerPage - 1) / homeProcessesPerPage
	}
	if raw < 1 {
		return 1
	}
	if raw > totalPages {
		return totalPages
	}
	return raw
}

func parsePositiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < 1 {
		return fallback
	}
	return value
}

func platformAdminListStateFromRequest(r *http.Request) (string, int) {
	if r == nil {
		return "", 1
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	page := parsePositiveInt(r.URL.Query().Get("page"), 1)
	if r.Method == http.MethodPost {
		if value := strings.TrimSpace(r.FormValue("q")); value != "" || query == "" {
			query = value
		}
		page = parsePositiveInt(firstNonEmpty(strings.TrimSpace(r.FormValue("page")), strconv.Itoa(page)), page)
	}
	return query, page
}

func platformAdminPath(query string, page int) string {
	values := url.Values{}
	if trimmedQuery := strings.TrimSpace(query); trimmedQuery != "" {
		values.Set("q", trimmedQuery)
	}
	if page > 1 {
		values.Set("page", strconv.Itoa(page))
	}
	if encoded := values.Encode(); encoded != "" {
		return "/admin/orgs?" + encoded
	}
	return "/admin/orgs"
}

func redirectPlatformAdminWithMessage(w http.ResponseWriter, r *http.Request, query string, page int, confirmation string) {
	target := platformAdminPath(query, page)
	trimmed := strings.TrimSpace(confirmation)
	if trimmed == "" {
		http.Redirect(w, r, target, http.StatusSeeOther)
		return
	}
	values := url.Values{}
	if parsed, err := url.Parse(target); err == nil {
		values = parsed.Query()
		target = parsed.Path
	}
	values.Set("confirmation", trimmed)
	http.Redirect(w, r, target+"?"+values.Encode(), http.StatusSeeOther)
}

var errPlatformAdminInviteCrossOrg = errors.New("platform admin invite email belongs to another organization")

func platformAdminOrganizationRows(ctx context.Context, organizations []Organization, identity IdentityStore) []PlatformAdminOrganizationRow {
	rows := make([]PlatformAdminOrganizationRow, 0, len(organizations))
	for _, organization := range organizations {
		row := PlatformAdminOrganizationRow{
			Name:             organization.Name,
			Slug:             organization.Slug,
			LogoAttachmentID: organization.LogoAttachmentID,
		}
		if identity != nil && strings.TrimSpace(organization.Slug) != "" {
			memberships, err := identity.ListOrganizationMemberships(ctx, organization.Slug)
			if err != nil {
				log.Printf("failed to list organization memberships for %s: %v", organization.Slug, err)
			} else {
				row.OrgAdminEmails, row.PendingOrgAdminEmails = summarizePlatformOrgAdminMemberships(memberships)
			}
		}
		row.OrgAdminStatus, row.OrgAdminStatusClassName = platformOrgAdminStatus(row.OrgAdminEmails, row.PendingOrgAdminEmails)
		rows = append(rows, row)
	}
	return rows
}

func summarizePlatformOrgAdminMemberships(memberships []IdentityMembership) ([]string, []string) {
	accepted := map[string]struct{}{}
	pending := map[string]struct{}{}
	for _, membership := range memberships {
		if !membership.IsOrgAdmin || isPlatformAdminMembership(membership) {
			continue
		}
		email := strings.TrimSpace(membership.Email)
		if email == "" {
			continue
		}
		if membership.Confirmed {
			accepted[email] = struct{}{}
			continue
		}
		pending[email] = struct{}{}
	}
	return sortedStringKeys(accepted), sortedStringKeys(pending)
}

func sortedStringKeys(items map[string]struct{}) []string {
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	for item := range items {
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func platformOrgAdminStatus(accepted []string, pending []string) (string, string) {
	switch {
	case len(accepted) > 0:
		return "At least one org admin accepted", "accepted"
	case len(pending) > 0:
		return "All org admin invites pending", "pending"
	default:
		return "No org admin", "missing"
	}
}

func (s *Server) inviteOrganizationAdminWithSession(ctx context.Context, sessionSecret string, org IdentityOrg, email, redirectURL string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return "", nil
	}
	memberships, err := s.identity.ListOrganizationMemberships(ctx, org.Slug)
	if err != nil {
		return "", err
	}
	for _, membership := range memberships {
		if !strings.EqualFold(strings.TrimSpace(membership.Email), email) {
			continue
		}
		if membership.IsOrgAdmin {
			return "org admin access already assigned", nil
		}
		if _, err := s.identity.UpdateOrganizationMembershipAsAdmin(ctx, org.Slug, membership.ID, nil, true); err != nil {
			return "", err
		}
		return "org admin access updated", nil
	}
	existingUser, err := s.identity.GetUserByEmail(ctx, email)
	switch {
	case err == nil:
		if existingUser.OrgSlug != "" && !strings.EqualFold(strings.TrimSpace(existingUser.OrgSlug), strings.TrimSpace(org.Slug)) {
			return "", errPlatformAdminInviteCrossOrg
		}
	case err != nil && !errors.Is(err, ErrIdentityNotFound):
		return "", err
	}
	if _, err := s.identity.InviteOrganizationUser(ctx, sessionSecret, org.Slug, email, redirectURL, nil, true); err != nil {
		return "", err
	}
	return "invite sent", nil
}

func (s *Server) platformAdminView(user *AccountUser, confirmation string, errs PlatformAdminErrors) PlatformAdminView {
	errs.Organization = strings.TrimSpace(errs.Organization)
	errs.Invite = strings.TrimSpace(errs.Invite)
	errs.DialogAction = strings.TrimSpace(errs.DialogAction)
	errs.OrgSlug = strings.TrimSpace(errs.OrgSlug)
	errs.OrgName = strings.TrimSpace(errs.OrgName)
	errs.InviteEmail = strings.TrimSpace(errs.InviteEmail)
	errs.SearchQuery = strings.TrimSpace(errs.SearchQuery)

	organizations := s.platformOrganizations(context.Background())
	filteredOrganizations := filterPlatformOrganizations(organizations, errs.SearchQuery)
	currentPage := normalizePlatformAdminPage(errs.Page, len(filteredOrganizations))
	start := (currentPage - 1) * platformAdminOrganizationsPerPage
	end := min(start+platformAdminOrganizationsPerPage, len(filteredOrganizations))
	pagedOrganizations := filteredOrganizations
	if start < len(filteredOrganizations) {
		pagedOrganizations = filteredOrganizations[start:end]
	} else if len(filteredOrganizations) > 0 {
		pagedOrganizations = filteredOrganizations[:0]
	}
	totalPages := 1
	if len(filteredOrganizations) > 0 {
		totalPages = (len(filteredOrganizations) + platformAdminOrganizationsPerPage - 1) / platformAdminOrganizationsPerPage
	}
	pageNumbers := make([]int, 0, totalPages)
	for page := 1; page <= totalPages; page++ {
		pageNumbers = append(pageNumbers, page)
	}
	rows := platformAdminOrganizationRows(context.Background(), pagedOrganizations, s.identity)
	return PlatformAdminView{
		PageBase:                 s.pageBaseForUser(user, "platform_admin_body", "", ""),
		SearchQuery:              errs.SearchQuery,
		CurrentPage:              currentPage,
		TotalPages:               totalPages,
		PageNumbers:              pageNumbers,
		HasPreviousPage:          currentPage > 1,
		HasNextPage:              currentPage < totalPages,
		PreviousPage:             max(currentPage-1, 1),
		NextPage:                 min(currentPage+1, totalPages),
		MatchedOrganizations:     len(filteredOrganizations),
		Organizations:            rows,
		Confirmation:             strings.TrimSpace(confirmation),
		OrganizationError:        errs.Organization,
		OrganizationDialogAction: errs.DialogAction,
		OrganizationDialogSlug:   errs.OrgSlug,
		OrganizationDialogName:   errs.OrgName,
		InviteError:              errs.Invite,
		InviteDialogEmail:        errs.InviteEmail,
		Error:                    firstNonEmpty(errs.Organization, errs.Invite),
	}
}

func (s *Server) renderPlatformAdmin(w http.ResponseWriter, user *AccountUser, confirmation string, errs PlatformAdminErrors) {
	view := s.platformAdminView(user, confirmation, errs)
	if err := s.tmpl.ExecuteTemplate(w, "platform_admin.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) renderPlatformAdminResults(w http.ResponseWriter, user *AccountUser, confirmation string, errs PlatformAdminErrors) {
	view := s.platformAdminView(user, confirmation, errs)
	if err := s.tmpl.ExecuteTemplate(w, "platform_admin_results", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleAdminOrgs(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.requirePlatformAdmin(w, r)
	if !ok {
		return
	}
	if s.identity == nil {
		http.Error(w, "identity unavailable", http.StatusServiceUnavailable)
		return
	}
	path := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/admin/orgs"))
	if strings.HasPrefix(path, "/logo/") {
		s.handlePlatformAdminLogo(w, r)
		return
	}
	if path != "" && path != "/" {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		searchQuery, page := platformAdminListStateFromRequest(r)
		confirmation := homePickerMessage(r, "confirmation")
		if isHTMXRequest(r) {
			s.renderPlatformAdminResults(w, admin, confirmation, PlatformAdminErrors{SearchQuery: searchQuery, Page: page})
			return
		}
		s.renderPlatformAdmin(w, admin, confirmation, PlatformAdminErrors{SearchQuery: searchQuery, Page: page})
		return
	case http.MethodPost:
		contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
		if strings.HasPrefix(contentType, "multipart/form-data") {
			r.Body = http.MaxBytesReader(w, r.Body, organizationLogoMaxBytes())
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				if isRequestTooLarge(err) {
					s.renderPlatformAdmin(w, admin, "", PlatformAdminErrors{Organization: "logo file too large"})
					return
				}
				logAndHTTPError(w, r, http.StatusBadRequest, "invalid form", err, "failed to parse platform admin multipart form")
				return
			}
			if r.MultipartForm != nil {
				defer r.MultipartForm.RemoveAll()
			}
		} else if err := r.ParseForm(); err != nil {
			logAndHTTPError(w, r, http.StatusBadRequest, "invalid form", err, "failed to parse platform admin form")
			return
		}
		searchQuery, page := platformAdminListStateFromRequest(r)
		intent := strings.TrimSpace(r.FormValue("intent"))
		if intent == "" {
			intent = "create_org"
		}
		if intent == "invite_org_admin" {
			orgSlug := strings.TrimSpace(r.FormValue("org_slug"))
			email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
			if email == "" {
				s.renderPlatformAdmin(w, admin, "", PlatformAdminErrors{Invite: "email is required", DialogAction: "invite", OrgSlug: orgSlug, InviteEmail: email, SearchQuery: searchQuery, Page: page})
				return
			}
			if orgSlug == "" {
				s.renderPlatformAdmin(w, admin, "", PlatformAdminErrors{Invite: "organization is required", DialogAction: "invite", OrgSlug: orgSlug, InviteEmail: email, SearchQuery: searchQuery, Page: page})
				return
			}
			redirectURL := inviteRedirectURL(r)
			org, err := s.identity.GetOrganizationBySlug(r.Context(), orgSlug)
			if err != nil || org == nil {
				if err != nil {
					logRequestError(r, err, "failed to load organization %s for platform admin invite", orgSlug)
				}
				s.renderPlatformAdmin(w, admin, "", PlatformAdminErrors{Invite: "organization not found", DialogAction: "invite", OrgSlug: orgSlug, InviteEmail: email, SearchQuery: searchQuery, Page: page})
				return
			}
			platformSession, err := s.ensurePlatformAdminOwnsOrganization(r.Context(), org.Slug, redirectURL)
			if err != nil {
				s.logAndRenderPlatformAdminError(w, r, admin, "", PlatformAdminErrors{Invite: "failed to create invite", DialogAction: "invite", OrgSlug: orgSlug, InviteEmail: email, SearchQuery: searchQuery, Page: page}, err, "failed to ensure platform admin owns organization %s for invite %s", org.Slug, email)
				return
			}
			defer func() {
				_ = s.identity.DeleteSession(r.Context(), platformSession.Secret)
			}()
			message, err := s.inviteOrganizationAdminWithSession(r.Context(), platformSession.Secret, *org, email, redirectURL)
			if errors.Is(err, errPlatformAdminInviteCrossOrg) {
				s.renderPlatformAdmin(w, admin, "", PlatformAdminErrors{Invite: "email already belongs to another organization", DialogAction: "invite", OrgSlug: orgSlug, InviteEmail: email, SearchQuery: searchQuery, Page: page})
				return
			}
			if err != nil {
				s.logAndRenderPlatformAdminError(w, r, admin, "", PlatformAdminErrors{Invite: "failed to create invite", DialogAction: "invite", OrgSlug: orgSlug, InviteEmail: email, SearchQuery: searchQuery, Page: page}, err, "failed to create org admin invite for %s in %s", email, org.Slug)
				return
			}
			s.renderPlatformAdmin(w, admin, message, PlatformAdminErrors{SearchQuery: searchQuery, Page: page})
			return
		}
		switch intent {
		case "create_org":
			name := strings.TrimSpace(r.FormValue("name"))
			inviteEmail := strings.ToLower(strings.TrimSpace(r.FormValue("invite_email")))
			if name == "" {
				s.renderPlatformAdmin(w, admin, "", PlatformAdminErrors{Organization: "organization name is required", DialogAction: "create", OrgName: name, InviteEmail: inviteEmail, SearchQuery: searchQuery, Page: page})
				return
			}
			orgSlug := canonifySlug(name)
			if existing, err := s.identity.GetOrganizationBySlug(r.Context(), orgSlug); err == nil && existing != nil {
				s.renderPlatformAdmin(w, admin, "", PlatformAdminErrors{Organization: "organization slug already exists", DialogAction: "create", OrgName: name, InviteEmail: inviteEmail, SearchQuery: searchQuery, Page: page})
				return
			}
			logoUpload, logoErrMsg := s.readOrganizationLogoUpload(r)
			if logoErrMsg != "" {
				s.renderPlatformAdmin(w, admin, "", PlatformAdminErrors{Organization: logoErrMsg, DialogAction: "create", OrgName: name, InviteEmail: inviteEmail, SearchQuery: searchQuery, Page: page})
				return
			}
			platformSession, err := s.platformAdminIdentitySession(r.Context())
			if err != nil {
				s.logAndRenderPlatformAdminError(w, r, admin, "", PlatformAdminErrors{Organization: "failed to create organization", DialogAction: "create", OrgName: name, InviteEmail: inviteEmail, SearchQuery: searchQuery, Page: page}, err, "failed to create platform admin session for organization creation")
				return
			}
			defer func() {
				_ = s.identity.DeleteSession(r.Context(), platformSession.Secret)
			}()
			createdOrg, err := s.identity.CreateOrganization(r.Context(), platformSession.Secret, name)
			if err != nil {
				if isDuplicateSlugError(err) {
					s.renderPlatformAdmin(w, admin, "", PlatformAdminErrors{Organization: "organization slug already exists", DialogAction: "create", OrgName: name, InviteEmail: inviteEmail, SearchQuery: searchQuery, Page: page})
					return
				}
				s.logAndRenderPlatformAdminError(w, r, admin, "", PlatformAdminErrors{Organization: "failed to create organization", DialogAction: "create", OrgName: name, InviteEmail: inviteEmail, SearchQuery: searchQuery, Page: page}, err, "failed to create organization %s", name)
				return
			}
			if logoUpload != nil {
				logoFile, err := s.identity.UploadOrganizationLogo(r.Context(), createdOrg.Slug, IdentityFile{
					Filename:    logoUpload.Filename,
					ContentType: logoUpload.ContentType,
					Data:        logoUpload.Data,
				})
				if err != nil {
					s.logAndRenderPlatformAdminError(w, r, admin, "", PlatformAdminErrors{Organization: "failed to upload logo", DialogAction: "create", OrgName: name, InviteEmail: inviteEmail, SearchQuery: searchQuery, Page: page}, err, "failed to upload logo for organization %s", createdOrg.Slug)
					return
				}
				if _, err := s.identity.UpdateOrganizationAsAdmin(r.Context(), createdOrg.Slug, createdOrg.Name, logoFile.ID, createdOrg.Roles); err != nil {
					s.logAndRenderPlatformAdminError(w, r, admin, "", PlatformAdminErrors{Organization: "failed to update organization", DialogAction: "create", OrgName: name, InviteEmail: inviteEmail, SearchQuery: searchQuery, Page: page}, err, "failed to attach logo to organization %s", createdOrg.Slug)
					return
				}
			}
			if inviteEmail != "" {
				message, err := s.inviteOrganizationAdminWithSession(r.Context(), platformSession.Secret, createdOrg, inviteEmail, inviteRedirectURL(r))
				if errors.Is(err, errPlatformAdminInviteCrossOrg) {
					s.renderPlatformAdmin(w, admin, "organization created", PlatformAdminErrors{Invite: "email already belongs to another organization", DialogAction: "invite", OrgSlug: createdOrg.Slug, InviteEmail: inviteEmail, SearchQuery: searchQuery, Page: page})
					return
				}
				if err != nil {
					s.logAndRenderPlatformAdminError(w, r, admin, "organization created", PlatformAdminErrors{Invite: "failed to create invite", DialogAction: "invite", OrgSlug: createdOrg.Slug, InviteEmail: inviteEmail, SearchQuery: searchQuery, Page: page}, err, "failed to create org admin invite for new organization %s", createdOrg.Slug)
					return
				}
				redirectPlatformAdminWithMessage(w, r, searchQuery, page, "organization created and "+message)
				return
			}
			redirectPlatformAdminWithMessage(w, r, searchQuery, page, "organization created")
			return
		case "set_org":
			currentSlug := strings.TrimSpace(r.FormValue("org_slug"))
			name := strings.TrimSpace(r.FormValue("name"))
			if currentSlug == "" {
				s.renderPlatformAdmin(w, admin, "", PlatformAdminErrors{Organization: "organization not found", DialogAction: "edit", SearchQuery: searchQuery, Page: page})
				return
			}
			if name == "" {
				s.renderPlatformAdmin(w, admin, "", PlatformAdminErrors{Organization: "organization name is required", DialogAction: "edit", OrgSlug: currentSlug, OrgName: name, SearchQuery: searchQuery, Page: page})
				return
			}
			org, err := s.identity.GetOrganizationBySlug(r.Context(), currentSlug)
			if err != nil || org == nil {
				if err != nil {
					logRequestError(r, err, "failed to load organization %s for platform admin update", currentSlug)
				}
				s.renderPlatformAdmin(w, admin, "", PlatformAdminErrors{Organization: "organization not found", DialogAction: "edit", OrgSlug: currentSlug, OrgName: name, SearchQuery: searchQuery, Page: page})
				return
			}
			targetOrgSlug := canonifySlug(name)
			if targetOrgSlug != strings.TrimSpace(org.Slug) {
				if existing, err := s.identity.GetOrganizationBySlug(r.Context(), targetOrgSlug); err == nil && existing != nil && strings.TrimSpace(existing.ID) != strings.TrimSpace(org.ID) {
					s.renderPlatformAdmin(w, admin, "", PlatformAdminErrors{Organization: "organization slug already exists", DialogAction: "edit", OrgSlug: currentSlug, OrgName: name, SearchQuery: searchQuery, Page: page})
					return
				}
			}
			logoUpload, logoErrMsg := s.readOrganizationLogoUpload(r)
			if logoErrMsg != "" {
				s.renderPlatformAdmin(w, admin, "", PlatformAdminErrors{Organization: logoErrMsg, DialogAction: "edit", OrgSlug: currentSlug, OrgName: name, SearchQuery: searchQuery, Page: page})
				return
			}
			previousLogoFileID := strings.TrimSpace(org.LogoFileID)
			logoFileID := previousLogoFileID
			if logoUpload != nil {
				logoFile, err := s.identity.UploadOrganizationLogo(r.Context(), targetOrgSlug, IdentityFile{
					Filename:    logoUpload.Filename,
					ContentType: logoUpload.ContentType,
					Data:        logoUpload.Data,
				})
				if err != nil {
					s.logAndRenderPlatformAdminError(w, r, admin, "", PlatformAdminErrors{Organization: "failed to upload logo", DialogAction: "edit", OrgSlug: currentSlug, OrgName: name, SearchQuery: searchQuery, Page: page}, err, "failed to upload updated logo for organization %s", targetOrgSlug)
					return
				}
				logoFileID = logoFile.ID
			}
			updatedOrg, err := s.identity.UpdateOrganizationAsAdmin(r.Context(), currentSlug, name, logoFileID, append([]IdentityRole(nil), org.Roles...))
			if err != nil {
				if isDuplicateSlugError(err) {
					s.renderPlatformAdmin(w, admin, "", PlatformAdminErrors{Organization: "organization slug already exists", DialogAction: "edit", OrgSlug: currentSlug, OrgName: name, SearchQuery: searchQuery, Page: page})
					return
				}
				s.logAndRenderPlatformAdminError(w, r, admin, "", PlatformAdminErrors{Organization: "failed to update organization", DialogAction: "edit", OrgSlug: currentSlug, OrgName: name, SearchQuery: searchQuery, Page: page}, err, "failed to update organization %s", currentSlug)
				return
			}
			if logoUpload != nil && previousLogoFileID != "" && previousLogoFileID != strings.TrimSpace(updatedOrg.LogoFileID) {
				if err := s.identity.DeleteOrganizationLogo(r.Context(), previousLogoFileID); err != nil && !errors.Is(err, ErrIdentityNotFound) {
					log.Printf("failed to delete previous organization logo %q: %v", previousLogoFileID, err)
				}
			}
			s.renderPlatformAdmin(w, admin, "organization updated", PlatformAdminErrors{SearchQuery: searchQuery, Page: page})
			return
		case "delete_org":
			currentSlug := strings.TrimSpace(r.FormValue("org_slug"))
			if currentSlug == "" {
				s.renderPlatformAdmin(w, admin, "", PlatformAdminErrors{Organization: "organization not found", DialogAction: "delete", SearchQuery: searchQuery, Page: page})
				return
			}
			org, err := s.identity.GetOrganizationBySlug(r.Context(), currentSlug)
			if err != nil || org == nil {
				if err != nil {
					logRequestError(r, err, "failed to load organization %s for platform admin deletion", currentSlug)
				}
				s.renderPlatformAdmin(w, admin, "", PlatformAdminErrors{Organization: "organization not found", DialogAction: "delete", OrgSlug: currentSlug, SearchQuery: searchQuery, Page: page})
				return
			}
			previousLogoFileID := strings.TrimSpace(org.LogoFileID)
			if err := s.identity.DeleteOrganizationAsAdmin(r.Context(), currentSlug); err != nil {
				s.logAndRenderPlatformAdminError(w, r, admin, "", PlatformAdminErrors{Organization: "failed to delete organization", DialogAction: "delete", OrgSlug: currentSlug, OrgName: org.Name, SearchQuery: searchQuery, Page: page}, err, "failed to delete organization %s", currentSlug)
				return
			}
			if previousLogoFileID != "" {
				if err := s.identity.DeleteOrganizationLogo(r.Context(), previousLogoFileID); err != nil && !errors.Is(err, ErrIdentityNotFound) {
					log.Printf("failed to delete organization logo %q after deleting org %s: %v", previousLogoFileID, currentSlug, err)
				}
			}
			s.renderPlatformAdmin(w, admin, "organization deleted", PlatformAdminErrors{SearchQuery: searchQuery, Page: page})
			return
		default:
			s.renderPlatformAdmin(w, admin, "", PlatformAdminErrors{Organization: "unsupported action", SearchQuery: searchQuery, Page: page})
			return
		}
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) renderOrgAdmin(w http.ResponseWriter, user *AccountUser, orgSlug, inviteLink, errMsg string) {
	errs := OrgAdminErrors{
		Organization: strings.TrimSpace(errMsg),
	}
	s.renderOrgAdminWithErrors(w, user, orgSlug, inviteLink, errs)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func buildOrgAdminRolePills(roles []Role) []OrgAdminRoleOption {
	rolePills := make([]OrgAdminRoleOption, 0, len(roles))
	for _, role := range roles {
		roleStyle := resolveRoleBadgeStyle(role.Color, role.Border)
		rolePills = append(rolePills, OrgAdminRoleOption{
			Slug:       role.Slug,
			Name:       role.Name,
			RoleColor:  cssValue(roleStyle.Color, "var(--role-fallback)"),
			RoleBorder: cssValue(roleStyle.Border, "var(--border)"),
		})
	}
	return rolePills
}

func organizationRoleInUse(roleSlug string, users []OrgAdminUserRow, invites []OrgAdminInviteRow) bool {
	trimmedRoleSlug := strings.TrimSpace(roleSlug)
	if trimmedRoleSlug == "" {
		return false
	}
	for _, user := range users {
		for _, role := range user.RoleOptions {
			if role.Selected && containsRole([]string{role.Slug}, trimmedRoleSlug) {
				return true
			}
		}
	}
	for _, invite := range invites {
		if containsRole(invite.RoleSlugs, trimmedRoleSlug) {
			return true
		}
	}
	return false
}

func buildOrgAdminRoleRows(roles []Role, users []OrgAdminUserRow, invites []OrgAdminInviteRow) []OrgAdminRoleRow {
	rows := make([]OrgAdminRoleRow, 0, len(roles))
	for _, role := range roles {
		if containsRole([]string{role.Slug}, "org-admin") || containsRole([]string{role.Slug}, "org_admin") {
			continue
		}
		roleStyle := resolveRoleBadgeStyle(role.Color, role.Border)
		rows = append(rows, OrgAdminRoleRow{
			Slug:       strings.TrimSpace(role.Slug),
			Name:       strings.TrimSpace(role.Name),
			Palette:    rolePaletteKeyFromStyle(role.Color, role.Border, role.Name),
			RoleColor:  cssValue(roleStyle.Color, "var(--role-fallback)"),
			RoleBorder: cssValue(roleStyle.Border, "var(--border)"),
			InUse:      organizationRoleInUse(role.Slug, users, invites),
		})
	}
	return rows
}

func buildOrgAdminUserRowsFromIdentity(rolePills []OrgAdminRoleOption, users []IdentityUser) []OrgAdminUserRow {
	orgUsers := make([]OrgAdminUserRow, 0, len(users))
	for _, orgUser := range users {
		if isPlatformAdminIdentityUser(orgUser) {
			continue
		}
		roleSlugs := decodeIdentityRoleLabels(orgUser.Labels)
		roleOptions := make([]OrgAdminRoleOption, 0, len(rolePills))
		for _, role := range rolePills {
			selected := containsRole(roleSlugs, role.Slug)
			if containsRole([]string{role.Slug}, "org-admin") || containsRole([]string{role.Slug}, "org_admin") {
				selected = orgUser.IsOrgAdmin
			}
			roleOptions = append(roleOptions, OrgAdminRoleOption{
				Slug:       role.Slug,
				Name:       role.Name,
				RoleColor:  role.RoleColor,
				RoleBorder: role.RoleBorder,
				Selected:   selected,
			})
		}
		userID := strings.TrimSpace(orgUser.ID)
		if userID == "" {
			userID = strings.TrimSpace(orgUser.Email)
		}
		orgUsers = append(orgUsers, OrgAdminUserRow{
			UserID:      userID,
			Email:       orgUser.Email,
			Status:      orgUser.Status,
			Activated:   !strings.EqualFold(strings.TrimSpace(orgUser.Status), "pending") && !strings.EqualFold(strings.TrimSpace(orgUser.Status), "invited"),
			IsOrgAdmin:  orgUser.IsOrgAdmin,
			RoleOptions: roleOptions,
		})
	}
	return orgUsers
}

func buildOrgAdminInviteRowsFromMemberships(memberships []IdentityMembership, now time.Time) []OrgAdminInviteRow {
	orgInvites := make([]OrgAdminInviteRow, 0, len(memberships))
	for _, membership := range memberships {
		status := "accepted"
		if !membership.Confirmed {
			status = "pending"
			if !membership.InvitedAt.IsZero() && membership.InvitedAt.Add(7*24*time.Hour).Before(now) {
				status = "expired"
			}
		}
		roleSlugs := append([]string(nil), membership.RoleSlugs...)
		if membership.IsOrgAdmin {
			roleSlugs = canonifyRoleSlugs(append(roleSlugs, "org-admin"))
		}
		expiresAt := time.Time{}
		if !membership.InvitedAt.IsZero() {
			expiresAt = membership.InvitedAt.Add(7 * 24 * time.Hour)
		}
		var usedAt *time.Time
		if !membership.JoinedAt.IsZero() {
			joinedAt := membership.JoinedAt
			usedAt = &joinedAt
		}
		orgInvites = append(orgInvites, OrgAdminInviteRow{
			Email:     membership.Email,
			RoleSlugs: roleSlugs,
			CreatedAt: membership.InvitedAt,
			ExpiresAt: expiresAt,
			UsedAt:    usedAt,
			Status:    status,
		})
	}
	return orgInvites
}

func (s *Server) loadOrgAdminState(ctx context.Context, user *AccountUser, orgSlug string) (Organization, []Role, []OrgAdminUserRow, []OrgAdminInviteRow, error) {
	_ = user
	if s.identity == nil {
		return Organization{}, nil, nil, nil, ErrIdentityNotFound
	}
	orgIdentity, err := s.identity.GetOrganizationBySlug(ctx, orgSlug)
	if err != nil || orgIdentity == nil {
		return Organization{}, nil, nil, nil, ErrIdentityNotFound
	}
	org := organizationFromIdentityOrg(*orgIdentity)
	roles := rolesFromIdentityOrg(*orgIdentity)
	roles = ensureOrgAdminRoleOption(roles)
	rolePills := buildOrgAdminRolePills(roles)

	identityUsers, identityUsersErr := s.identity.ListOrganizationUsers(ctx, org.Slug)
	if identityUsersErr != nil {
		return Organization{}, nil, nil, nil, identityUsersErr
	}
	orgUsers := buildOrgAdminUserRowsFromIdentity(rolePills, identityUsers)

	if memberships, membershipsErr := s.identity.ListOrganizationMemberships(ctx, org.Slug); membershipsErr == nil {
		return org, roles, orgUsers, buildOrgAdminInviteRowsFromMemberships(memberships, s.nowUTC()), nil
	}
	return org, roles, orgUsers, nil, nil
}

func (s *Server) renderOrgAdminWithErrors(w http.ResponseWriter, user *AccountUser, orgSlug, inviteLink string, errs OrgAdminErrors) {
	errs.Organization = strings.TrimSpace(errs.Organization)
	errs.Role = strings.TrimSpace(errs.Role)
	errs.Invite = strings.TrimSpace(errs.Invite)
	errs.Users = strings.TrimSpace(errs.Users)

	if !userHasOrganizationContext(user) || strings.TrimSpace(orgSlug) == "" {
		view := OrgAdminView{
			PageBase:               s.pageBaseForUser(user, "org_admin_body", "", ""),
			NeedsOrganizationSetup: true,
			OrganizationError:      errs.Organization,
			RoleError:              errs.Role,
			RoleDialogAction:       strings.TrimSpace(errs.RoleAction),
			RoleDialogSlug:         strings.TrimSpace(errs.RoleSlug),
			RoleDialogName:         strings.TrimSpace(errs.RoleName),
			RoleDialogPalette:      strings.TrimSpace(errs.RolePalette),
			InviteError:            errs.Invite,
			UsersError:             errs.Users,
			InviteLink:             strings.TrimSpace(inviteLink),
			Error:                  firstNonEmpty(errs.Organization, errs.Role, errs.Invite, errs.Users),
		}
		if err := s.tmpl.ExecuteTemplate(w, "org_admin.html", view); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	org, roles, orgUsers, orgInvites, err := s.loadOrgAdminState(context.Background(), user, orgSlug)
	if err != nil {
		if !errors.Is(err, ErrIdentityNotFound) {
			log.Printf("failed to load org-admin state for org %s: %v", orgSlug, err)
		}
		http.Error(w, "organization not found", http.StatusNotFound)
		return
	}
	rolePills := buildOrgAdminRolePills(roles)
	roleRows := buildOrgAdminRoleRows(roles, orgUsers, orgInvites)

	view := OrgAdminView{
		PageBase:               s.pageBaseForUser(user, "org_admin_body", "", ""),
		Organization:           org,
		OrganizationLogoURL:    "/org-admin/logo/" + strings.TrimSpace(org.LogoAttachmentID),
		NeedsOrganizationSetup: false,
		OrganizationError:      errs.Organization,
		RoleError:              errs.Role,
		RoleDialogAction:       strings.TrimSpace(errs.RoleAction),
		RoleDialogSlug:         strings.TrimSpace(errs.RoleSlug),
		RoleDialogName:         strings.TrimSpace(errs.RoleName),
		RoleDialogPalette:      strings.TrimSpace(errs.RolePalette),
		InviteError:            errs.Invite,
		UsersError:             errs.Users,
		Roles:                  roles,
		RolePills:              rolePills,
		RoleRows:               roleRows,
		Users:                  orgUsers,
		Invites:                orgInvites,
		InviteLink:             strings.TrimSpace(inviteLink),
		Error:                  firstNonEmpty(errs.Organization, errs.Role, errs.Invite, errs.Users),
	}
	if strings.TrimSpace(org.LogoAttachmentID) == "" {
		view.OrganizationLogoURL = ""
	}
	if err := s.tmpl.ExecuteTemplate(w, "org_admin.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleOrgAdminLogo(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.requireOrgAdmin(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	logoID := strings.Trim(strings.TrimPrefix(r.URL.Path, "/org-admin/logo/"), "/")
	if logoID == "" {
		http.NotFound(w, r)
		return
	}
	if s.identity == nil {
		http.NotFound(w, r)
		return
	}
	org, err := s.identity.GetOrganizationBySlug(r.Context(), admin.OrgSlug)
	if err != nil || org == nil {
		if err != nil {
			logRequestError(r, err, "failed to load organization %s logo metadata", admin.OrgSlug)
		}
		http.NotFound(w, r)
		return
	}
	if strings.TrimSpace(org.LogoFileID) == "" || strings.TrimSpace(org.LogoFileID) != logoID {
		http.NotFound(w, r)
		return
	}
	logo, err := s.identity.GetOrganizationLogo(r.Context(), logoID)
	if err != nil {
		if !errors.Is(err, ErrIdentityNotFound) {
			logRequestError(r, err, "failed to load organization logo %s", logoID)
		}
		http.NotFound(w, r)
		return
	}
	contentType := strings.TrimSpace(logo.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	filename := sanitizeAttachmentFilename(logo.Filename)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", filename))
	_, _ = w.Write(logo.Data)
}

func (s *Server) handleOrganizationLogo(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.requireAuthenticatedPage(w, r); !ok {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	orgSlug := strings.Trim(strings.TrimPrefix(r.URL.Path, "/organization/logo/"), "/")
	if orgSlug == "" || s.identity == nil {
		http.NotFound(w, r)
		return
	}
	org, err := s.identity.GetOrganizationBySlug(r.Context(), orgSlug)
	if err != nil || org == nil || strings.TrimSpace(org.LogoFileID) == "" {
		if err != nil && !errors.Is(err, ErrIdentityNotFound) {
			logRequestError(r, err, "failed to load organization %s logo metadata", orgSlug)
		}
		http.NotFound(w, r)
		return
	}
	logo, err := s.identity.GetOrganizationLogo(r.Context(), strings.TrimSpace(org.LogoFileID))
	if err != nil {
		if !errors.Is(err, ErrIdentityNotFound) {
			logRequestError(r, err, "failed to load organization %s logo", orgSlug)
		}
		http.NotFound(w, r)
		return
	}
	contentType := strings.TrimSpace(logo.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	filename := sanitizeAttachmentFilename(logo.Filename)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", filename))
	_, _ = w.Write(logo.Data)
}

func (s *Server) handlePlatformAdminLogo(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requirePlatformAdmin(w, r); !ok {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	logoID := strings.Trim(strings.TrimPrefix(r.URL.Path, "/admin/orgs/logo/"), "/")
	if logoID == "" || s.identity == nil {
		http.NotFound(w, r)
		return
	}
	logo, err := s.identity.GetOrganizationLogo(r.Context(), logoID)
	if err != nil {
		if !errors.Is(err, ErrIdentityNotFound) {
			logRequestError(r, err, "failed to load organization logo %s for platform admin", logoID)
		}
		http.NotFound(w, r)
		return
	}
	contentType := strings.TrimSpace(logo.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	filename := sanitizeAttachmentFilename(logo.Filename)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", filename))
	_, _ = w.Write(logo.Data)
}

func (s *Server) handleOrgAdminRoles(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireOrgAdmin(w, r)
	if !ok {
		return
	}
	if !userHasOrganizationContext(user) {
		s.renderOrgAdminWithErrors(w, user, "", "", OrgAdminErrors{Organization: "create organization first"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.renderOrgAdmin(w, user, user.OrgSlug, "", "")
		return
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			logAndHTTPError(w, r, http.StatusBadRequest, "invalid form", err, "failed to parse org-admin roles form")
			return
		}
		intent := strings.TrimSpace(r.FormValue("intent"))
		if intent == "" {
			intent = "create_role"
		}
		if s.identity == nil {
			http.Error(w, "identity unavailable", http.StatusServiceUnavailable)
			return
		}
		org, err := s.identity.GetOrganizationBySlug(r.Context(), user.OrgSlug)
		if err != nil || org == nil {
			if err != nil {
				logRequestError(r, err, "failed to load organization %s for role creation", user.OrgSlug)
			}
			http.NotFound(w, r)
			return
		}
		orgUsers, listUsersErr := s.identity.ListOrganizationUsers(r.Context(), user.OrgSlug)
		if listUsersErr != nil {
			s.logAndRenderOrgAdminError(w, r, user, user.OrgSlug, "", OrgAdminErrors{Role: "failed to load organization users"}, listUsersErr, "failed to list organization users for role action in %s", user.OrgSlug)
			return
		}
		memberships, membershipsErr := s.identity.ListOrganizationMemberships(r.Context(), user.OrgSlug)
		if membershipsErr != nil {
			s.logAndRenderOrgAdminError(w, r, user, user.OrgSlug, "", OrgAdminErrors{Role: "failed to load organization users"}, membershipsErr, "failed to list organization memberships for role action in %s", user.OrgSlug)
			return
		}
		roleRows := buildOrgAdminRoleRows(rolesFromIdentityOrg(*org), buildOrgAdminUserRowsFromIdentity(buildOrgAdminRolePills(rolesFromIdentityOrg(*org)), orgUsers), buildOrgAdminInviteRowsFromMemberships(memberships, s.nowUTC()))

		findRoleRow := func(roleSlug string) *OrgAdminRoleRow {
			for idx := range roleRows {
				if containsRole([]string{roleRows[idx].Slug}, roleSlug) {
					return &roleRows[idx]
				}
			}
			return nil
		}

		sessionSecret, err := sessionSecretFromRequest(r)
		if err != nil {
			logAndHTTPError(w, r, http.StatusUnauthorized, "unauthorized", err, "failed to read session secret for org-admin role action")
			return
		}

		switch intent {
		case "create_role":
			name := strings.TrimSpace(r.FormValue("name"))
			palette := strings.TrimSpace(r.FormValue("palette"))
			if name == "" {
				s.renderOrgAdminWithErrors(w, user, user.OrgSlug, "", OrgAdminErrors{Role: "role name is required", RoleAction: "create", RolePalette: palette})
				return
			}
			roleSlug := canonifyIdentityRoleSlug(name)
			if roleSlug == "" {
				s.renderOrgAdminWithErrors(w, user, user.OrgSlug, "", OrgAdminErrors{Role: invalidRoleNameMessage, RoleAction: "create", RoleName: name, RolePalette: palette})
				return
			}
			if palette == "" {
				palette = defaultRolePaletteFromInput(name)
			}
			paletteStyle := resolveRolePaletteStyle(palette)
			for _, existingRole := range ensureOrgAdminRoleOption(rolesFromIdentityOrg(*org)) {
				if strings.EqualFold(canonifyIdentityRoleSlug(existingRole.Slug), roleSlug) {
					s.renderOrgAdminWithErrors(w, user, user.OrgSlug, "", OrgAdminErrors{Role: "role slug already exists", RoleAction: "create", RoleName: name, RolePalette: palette})
					return
				}
			}
			updatedRoles := append(append([]IdentityRole(nil), org.Roles...), IdentityRole{
				Slug:   roleSlug,
				Name:   name,
				Color:  paletteStyle.Color,
				Border: paletteStyle.Border,
			})
			if _, err := s.identity.UpdateOrganization(r.Context(), sessionSecret, user.OrgSlug, org.Name, org.LogoFileID, updatedRoles); err != nil {
				if isDuplicateSlugError(err) {
					s.renderOrgAdminWithErrors(w, user, user.OrgSlug, "", OrgAdminErrors{Role: "role slug already exists", RoleAction: "create", RoleName: name, RolePalette: palette})
					return
				}
				s.logAndRenderOrgAdminError(w, r, user, user.OrgSlug, "", OrgAdminErrors{Role: "failed to create role", RoleAction: "create", RoleName: name, RolePalette: palette}, err, "failed to update organization %s with new role %s", user.OrgSlug, roleSlug)
				return
			}
		case "set_role":
			currentSlug := strings.TrimSpace(r.FormValue("role_slug"))
			name := strings.TrimSpace(r.FormValue("name"))
			palette := strings.TrimSpace(r.FormValue("palette"))
			if currentSlug == "" {
				s.renderOrgAdminWithErrors(w, user, user.OrgSlug, "", OrgAdminErrors{Role: "role not found", RoleAction: "edit"})
				return
			}
			if name == "" {
				s.renderOrgAdminWithErrors(w, user, user.OrgSlug, "", OrgAdminErrors{Role: "role name is required", RoleAction: "edit", RoleSlug: currentSlug, RolePalette: palette})
				return
			}
			targetRow := findRoleRow(currentSlug)
			if targetRow == nil {
				s.renderOrgAdminWithErrors(w, user, user.OrgSlug, "", OrgAdminErrors{Role: "role not found", RoleAction: "edit", RoleSlug: currentSlug, RoleName: name, RolePalette: palette})
				return
			}
			if targetRow.InUse {
				s.renderOrgAdminWithErrors(w, user, user.OrgSlug, "", OrgAdminErrors{Role: "remove the role from the users that have it before continuing with the action", RoleAction: "edit", RoleSlug: currentSlug, RoleName: targetRow.Name, RolePalette: targetRow.Palette})
				return
			}
			roleSlug := canonifyIdentityRoleSlug(name)
			if roleSlug == "" {
				s.renderOrgAdminWithErrors(w, user, user.OrgSlug, "", OrgAdminErrors{Role: invalidRoleNameMessage, RoleAction: "edit", RoleSlug: currentSlug, RoleName: name, RolePalette: palette})
				return
			}
			if palette == "" {
				palette = defaultRolePaletteFromInput(name)
			}
			paletteStyle := resolveRolePaletteStyle(palette)
			updatedRoles := append([]IdentityRole(nil), org.Roles...)
			found := false
			for idx := range updatedRoles {
				if !containsRole([]string{updatedRoles[idx].Slug}, currentSlug) {
					if strings.EqualFold(canonifyIdentityRoleSlug(updatedRoles[idx].Slug), roleSlug) {
						s.renderOrgAdminWithErrors(w, user, user.OrgSlug, "", OrgAdminErrors{Role: "role slug already exists", RoleAction: "edit", RoleSlug: currentSlug, RoleName: name, RolePalette: palette})
						return
					}
					continue
				}
				found = true
				updatedRoles[idx].Slug = roleSlug
				updatedRoles[idx].Name = name
				updatedRoles[idx].Color = paletteStyle.Color
				updatedRoles[idx].Border = paletteStyle.Border
			}
			if !found {
				s.renderOrgAdminWithErrors(w, user, user.OrgSlug, "", OrgAdminErrors{Role: "role not found", RoleAction: "edit", RoleSlug: currentSlug, RoleName: name, RolePalette: palette})
				return
			}
			if _, err := s.identity.UpdateOrganization(r.Context(), sessionSecret, user.OrgSlug, org.Name, org.LogoFileID, updatedRoles); err != nil {
				if isDuplicateSlugError(err) {
					s.renderOrgAdminWithErrors(w, user, user.OrgSlug, "", OrgAdminErrors{Role: "role slug already exists", RoleAction: "edit", RoleSlug: currentSlug, RoleName: name, RolePalette: palette})
					return
				}
				s.logAndRenderOrgAdminError(w, r, user, user.OrgSlug, "", OrgAdminErrors{Role: "failed to update role", RoleAction: "edit", RoleSlug: currentSlug, RoleName: name, RolePalette: palette}, err, "failed to update role %s in organization %s", currentSlug, user.OrgSlug)
				return
			}
		case "delete_role":
			currentSlug := strings.TrimSpace(r.FormValue("role_slug"))
			if currentSlug == "" {
				s.renderOrgAdminWithErrors(w, user, user.OrgSlug, "", OrgAdminErrors{Role: "role not found", RoleAction: "delete"})
				return
			}
			targetRow := findRoleRow(currentSlug)
			if targetRow == nil {
				s.renderOrgAdminWithErrors(w, user, user.OrgSlug, "", OrgAdminErrors{Role: "role not found", RoleAction: "delete", RoleSlug: currentSlug})
				return
			}
			if targetRow.InUse {
				s.renderOrgAdminWithErrors(w, user, user.OrgSlug, "", OrgAdminErrors{Role: "remove the role from the users that have it before continuing with the action", RoleAction: "delete", RoleSlug: currentSlug, RoleName: targetRow.Name})
				return
			}
			updatedRoles := make([]IdentityRole, 0, len(org.Roles))
			found := false
			for _, role := range org.Roles {
				if containsRole([]string{role.Slug}, currentSlug) {
					found = true
					continue
				}
				updatedRoles = append(updatedRoles, role)
			}
			if !found {
				s.renderOrgAdminWithErrors(w, user, user.OrgSlug, "", OrgAdminErrors{Role: "role not found", RoleAction: "delete", RoleSlug: currentSlug})
				return
			}
			if _, err := s.identity.UpdateOrganization(r.Context(), sessionSecret, user.OrgSlug, org.Name, org.LogoFileID, updatedRoles); err != nil {
				s.logAndRenderOrgAdminError(w, r, user, user.OrgSlug, "", OrgAdminErrors{Role: "failed to delete role", RoleAction: "delete", RoleSlug: currentSlug, RoleName: targetRow.Name}, err, "failed to delete role %s from organization %s", currentSlug, user.OrgSlug)
				return
			}
		default:
			s.renderOrgAdminWithErrors(w, user, user.OrgSlug, "", OrgAdminErrors{Role: "unsupported action"})
			return
		}
		http.Redirect(w, r, "/org-admin/users", http.StatusSeeOther)
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
	if s.identity == nil {
		http.Error(w, "identity unavailable", http.StatusServiceUnavailable)
		return
	}
	if r.Method != http.MethodPost {
		s.renderOrgAdmin(w, admin, admin.OrgSlug, "", "")
		return
	}
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.HasPrefix(contentType, "multipart/form-data") {
		r.Body = http.MaxBytesReader(w, r.Body, organizationLogoMaxBytes())
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			if isRequestTooLarge(err) {
				s.renderOrgAdminWithErrors(w, admin, admin.OrgSlug, "", OrgAdminErrors{Organization: "logo file too large"})
				return
			}
			logAndHTTPError(w, r, http.StatusBadRequest, "invalid form", err, "failed to parse org-admin multipart form")
			return
		}
		if r.MultipartForm != nil {
			defer r.MultipartForm.RemoveAll()
		}
	} else if err := r.ParseForm(); err != nil {
		logAndHTTPError(w, r, http.StatusBadRequest, "invalid form", err, "failed to parse org-admin users form")
		return
	}
	intent := strings.TrimSpace(r.FormValue("intent"))
	if intent == "" {
		intent = "invite"
	}
	if !userHasOrganizationContext(admin) {
		if intent != "create_org" {
			s.renderOrgAdminWithErrors(w, admin, "", "", OrgAdminErrors{Organization: "create organization first"})
			return
		}
		name := strings.TrimSpace(r.FormValue("name"))
		if name == "" {
			s.renderOrgAdminWithErrors(w, admin, "", "", OrgAdminErrors{Organization: "organization name is required"})
			return
		}
		orgSlug := canonifySlug(name)
		if existing, err := s.identity.GetOrganizationBySlug(r.Context(), orgSlug); err == nil && existing != nil {
			s.renderOrgAdminWithErrors(w, admin, "", "", OrgAdminErrors{Organization: "organization slug already exists"})
			return
		}
		logoUpload, logoErrMsg := s.readOrganizationLogoUpload(r)
		if logoErrMsg != "" {
			s.renderOrgAdminWithErrors(w, admin, "", "", OrgAdminErrors{Organization: logoErrMsg})
			return
		}
		sessionSecret, err := sessionSecretFromRequest(r)
		if err != nil {
			logAndHTTPError(w, r, http.StatusUnauthorized, "unauthorized", err, "failed to read session secret for organization creation")
			return
		}
		createdOrg, err := s.identity.CreateOrganization(r.Context(), sessionSecret, name)
		if err != nil {
			if isDuplicateSlugError(err) {
				s.renderOrgAdminWithErrors(w, admin, "", "", OrgAdminErrors{Organization: "organization slug already exists"})
				return
			}
			s.logAndRenderOrgAdminError(w, r, admin, "", "", OrgAdminErrors{Organization: "failed to create organization"}, err, "failed to create organization %s", name)
			return
		}
		if logoUpload != nil {
			logoFile, err := s.identity.UploadOrganizationLogo(r.Context(), createdOrg.Slug, IdentityFile{
				Filename:    logoUpload.Filename,
				ContentType: logoUpload.ContentType,
				Data:        logoUpload.Data,
			})
			if err != nil {
				s.logAndRenderOrgAdminError(w, r, admin, "", "", OrgAdminErrors{Organization: "failed to upload logo"}, err, "failed to upload logo for organization %s", createdOrg.Slug)
				return
			}
			createdOrg, err = s.identity.UpdateOrganization(r.Context(), sessionSecret, createdOrg.Slug, createdOrg.Name, logoFile.ID, createdOrg.Roles)
			if err != nil {
				s.logAndRenderOrgAdminError(w, r, admin, createdOrg.Slug, "", OrgAdminErrors{Organization: "failed to update organization"}, err, "failed to attach logo to organization %s", createdOrg.Slug)
				return
			}
		}
		http.Redirect(w, r, "/org-admin/users", http.StatusSeeOther)
		return
	}

	switch intent {
	case "create_org":
		s.renderOrgAdminWithErrors(w, admin, admin.OrgSlug, "", OrgAdminErrors{Organization: "organization already exists for your account"})
	case "invite":
		email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
		if email == "" {
			s.renderOrgAdminWithErrors(w, admin, admin.OrgSlug, "", OrgAdminErrors{Invite: "email is required"})
			return
		}
		org, err := s.identity.GetOrganizationBySlug(r.Context(), admin.OrgSlug)
		if err != nil || org == nil {
			if err != nil {
				logRequestError(r, err, "failed to load organization %s for invite", admin.OrgSlug)
			}
			http.NotFound(w, r)
			return
		}
		selectedRoles := requestedRoleSlugs(r.Form)
		allowedRoles := ensureOrgAdminRoleOption(rolesFromIdentityOrg(*org))
		allowed := make(map[string]struct{}, len(allowedRoles))
		for _, role := range allowedRoles {
			allowed[strings.TrimSpace(role.Slug)] = struct{}{}
		}
		for _, roleSlug := range selectedRoles {
			if _, ok := allowed[strings.TrimSpace(roleSlug)]; !ok {
				s.renderOrgAdminWithErrors(w, admin, admin.OrgSlug, "", OrgAdminErrors{Invite: "role not found"})
				return
			}
		}
		isOrgAdmin := containsRole(selectedRoles, "org-admin")
		businessRoles := make([]string, 0, len(selectedRoles))
		for _, roleSlug := range selectedRoles {
			if containsRole([]string{roleSlug}, "org-admin") || containsRole([]string{roleSlug}, "org_admin") {
				isOrgAdmin = true
				continue
			}
			businessRoles = append(businessRoles, roleSlug)
		}
		memberships, err := s.identity.ListOrganizationMemberships(r.Context(), admin.OrgSlug)
		if err != nil {
			s.logAndRenderOrgAdminError(w, r, admin, admin.OrgSlug, "", OrgAdminErrors{Invite: "failed to create invite"}, err, "failed to list memberships for organization %s during invite", admin.OrgSlug)
			return
		}
		for _, membership := range memberships {
			if !strings.EqualFold(strings.TrimSpace(membership.Email), email) {
				continue
			}
			if membership.Confirmed {
				labels := make([]string, 0, len(businessRoles)+1)
				for _, roleSlug := range businessRoles {
					labels = append(labels, encodeIdentityRoleLabel(roleSlug))
				}
				if isOrgAdmin {
					labels = append(labels, identityOrgAdminLabel)
				}
				if _, err := s.identity.UpdateUserLabels(r.Context(), membership.UserID, labels); err != nil {
					s.logAndRenderOrgAdminError(w, r, admin, admin.OrgSlug, "", OrgAdminErrors{Invite: "failed to update user roles"}, err, "failed to update labels for invited member %s in organization %s", membership.UserID, admin.OrgSlug)
					return
				}
				s.renderOrgAdmin(w, admin, admin.OrgSlug, "", "")
				return
			}
			if roleSlugsKey(append(append([]string{}, membership.RoleSlugs...), func() []string {
				if membership.IsOrgAdmin {
					return []string{"org-admin"}
				}
				return nil
			}()...)) == roleSlugsKey(selectedRoles) {
				s.renderOrgAdmin(w, admin, admin.OrgSlug, "", "")
				return
			}
			sessionSecret, err := sessionSecretFromRequest(r)
			if err != nil {
				logAndHTTPError(w, r, http.StatusUnauthorized, "unauthorized", err, "failed to read session secret for membership update in %s", admin.OrgSlug)
				return
			}
			if _, err := s.identity.UpdateOrganizationMembership(r.Context(), sessionSecret, admin.OrgSlug, membership.ID, businessRoles, isOrgAdmin); err != nil {
				s.logAndRenderOrgAdminError(w, r, admin, admin.OrgSlug, "", OrgAdminErrors{Invite: "failed to create invite"}, err, "failed to update membership %s in organization %s", membership.ID, admin.OrgSlug)
				return
			}
			s.renderOrgAdmin(w, admin, admin.OrgSlug, "", "")
			return
		}
		existingUser, err := s.identity.GetUserByEmail(r.Context(), email)
		switch {
		case err == nil && existingUser.OrgSlug != "" && !strings.EqualFold(strings.TrimSpace(existingUser.OrgSlug), strings.TrimSpace(admin.OrgSlug)):
			s.renderOrgAdminWithErrors(w, admin, admin.OrgSlug, "", OrgAdminErrors{Invite: "email already belongs to another organization"})
			return
		case err == nil && strings.EqualFold(strings.TrimSpace(existingUser.OrgSlug), strings.TrimSpace(admin.OrgSlug)):
			labels := make([]string, 0, len(businessRoles)+1)
			for _, roleSlug := range businessRoles {
				labels = append(labels, encodeIdentityRoleLabel(roleSlug))
			}
			if isOrgAdmin {
				labels = append(labels, identityOrgAdminLabel)
			}
			if _, err := s.identity.UpdateUserLabels(r.Context(), existingUser.ID, labels); err != nil {
				s.logAndRenderOrgAdminError(w, r, admin, admin.OrgSlug, "", OrgAdminErrors{Invite: "failed to update user roles"}, err, "failed to update labels for existing user %s in organization %s", existingUser.ID, admin.OrgSlug)
				return
			}
			s.renderOrgAdmin(w, admin, admin.OrgSlug, "", "")
			return
		case err != nil && !errors.Is(err, ErrIdentityNotFound):
			s.logAndRenderOrgAdminError(w, r, admin, admin.OrgSlug, "", OrgAdminErrors{Invite: "failed to load existing user"}, err, "failed to look up existing user %s during invite", email)
			return
		}
		sessionSecret, err := sessionSecretFromRequest(r)
		if err != nil {
			logAndHTTPError(w, r, http.StatusUnauthorized, "unauthorized", err, "failed to read session secret for invite creation in %s", admin.OrgSlug)
			return
		}
		if _, err := s.identity.InviteOrganizationUser(r.Context(), sessionSecret, admin.OrgSlug, email, inviteRedirectURL(r), businessRoles, isOrgAdmin); err != nil {
			s.logAndRenderOrgAdminError(w, r, admin, admin.OrgSlug, "", OrgAdminErrors{Invite: "failed to create invite"}, err, "failed to create invite for %s in organization %s", email, admin.OrgSlug)
			return
		}
		s.renderOrgAdmin(w, admin, admin.OrgSlug, "", "")
	case "update_org":
		name := strings.TrimSpace(r.FormValue("name"))
		if name == "" {
			s.renderOrgAdminWithErrors(w, admin, admin.OrgSlug, "", OrgAdminErrors{Organization: "organization name is required"})
			return
		}
		org, err := s.identity.GetOrganizationBySlug(r.Context(), admin.OrgSlug)
		if err != nil || org == nil {
			if err != nil {
				logRequestError(r, err, "failed to load organization %s for update", admin.OrgSlug)
			}
			http.NotFound(w, r)
			return
		}
		targetOrgSlug := canonifySlug(name)
		if targetOrgSlug != strings.TrimSpace(org.Slug) {
			if existing, err := s.identity.GetOrganizationBySlug(r.Context(), targetOrgSlug); err == nil && existing != nil && strings.TrimSpace(existing.ID) != strings.TrimSpace(org.ID) {
				s.renderOrgAdminWithErrors(w, admin, admin.OrgSlug, "", OrgAdminErrors{Organization: "organization slug already exists"})
				return
			}
		}
		logoUpload, logoErrMsg := s.readOrganizationLogoUpload(r)
		if logoErrMsg != "" {
			s.renderOrgAdminWithErrors(w, admin, admin.OrgSlug, "", OrgAdminErrors{Organization: logoErrMsg})
			return
		}
		previousLogoFileID := strings.TrimSpace(org.LogoFileID)
		logoFileID := previousLogoFileID
		if logoUpload != nil {
			logoFile, err := s.identity.UploadOrganizationLogo(r.Context(), targetOrgSlug, IdentityFile{
				Filename:    logoUpload.Filename,
				ContentType: logoUpload.ContentType,
				Data:        logoUpload.Data,
			})
			if err != nil {
				s.logAndRenderOrgAdminError(w, r, admin, admin.OrgSlug, "", OrgAdminErrors{Organization: "failed to upload logo"}, err, "failed to upload updated logo for organization %s", targetOrgSlug)
				return
			}
			logoFileID = logoFile.ID
		}
		sessionSecret, err := sessionSecretFromRequest(r)
		if err != nil {
			logAndHTTPError(w, r, http.StatusUnauthorized, "unauthorized", err, "failed to read session secret for organization update in %s", admin.OrgSlug)
			return
		}
		updatedOrg, err := s.identity.UpdateOrganization(r.Context(), sessionSecret, admin.OrgSlug, name, logoFileID, append([]IdentityRole(nil), org.Roles...))
		if err != nil {
			if isDuplicateSlugError(err) {
				s.renderOrgAdminWithErrors(w, admin, admin.OrgSlug, "", OrgAdminErrors{Organization: "organization slug already exists"})
				return
			}
			s.logAndRenderOrgAdminError(w, r, admin, admin.OrgSlug, "", OrgAdminErrors{Organization: "failed to update organization"}, err, "failed to update organization %s", admin.OrgSlug)
			return
		}
		if logoUpload != nil && previousLogoFileID != "" && previousLogoFileID != strings.TrimSpace(updatedOrg.LogoFileID) {
			if err := s.identity.DeleteOrganizationLogo(r.Context(), previousLogoFileID); err != nil && !errors.Is(err, ErrIdentityNotFound) {
				log.Printf("failed to delete previous organization logo %q: %v", previousLogoFileID, err)
			}
		}
		http.Redirect(w, r, "/org-admin/users", http.StatusSeeOther)
	case "set_roles":
		userID := strings.TrimSpace(r.FormValue("userId"))
		if userID == "" {
			s.renderOrgAdminWithErrors(w, admin, admin.OrgSlug, "", OrgAdminErrors{Users: "user is required"})
			return
		}
		org, err := s.identity.GetOrganizationBySlug(r.Context(), admin.OrgSlug)
		if err != nil || org == nil {
			if err != nil {
				logRequestError(r, err, "failed to load organization %s for role update", admin.OrgSlug)
			}
			http.NotFound(w, r)
			return
		}
		targetUsers, err := s.identity.ListOrganizationUsers(r.Context(), admin.OrgSlug)
		if err != nil {
			s.logAndRenderOrgAdminError(w, r, admin, admin.OrgSlug, "", OrgAdminErrors{Users: "user not found"}, err, "failed to list organization users for %s", admin.OrgSlug)
			return
		}
		var target *IdentityUser
		for idx := range targetUsers {
			targetKey := firstNonEmpty(targetUsers[idx].ID, targetUsers[idx].Email)
			if strings.TrimSpace(targetKey) == userID {
				target = &targetUsers[idx]
				break
			}
		}
		if target == nil {
			s.renderOrgAdminWithErrors(w, admin, admin.OrgSlug, "", OrgAdminErrors{Users: "user not found"})
			return
		}
		if isPlatformAdminIdentityUser(*target) {
			s.renderOrgAdminWithErrors(w, admin, admin.OrgSlug, "", OrgAdminErrors{Users: "user not found"})
			return
		}
		selectedRoles := requestedRoleSlugs(r.Form)
		allowedRoles := ensureOrgAdminRoleOption(rolesFromIdentityOrg(*org))
		allowed := make(map[string]struct{}, len(allowedRoles))
		for _, role := range allowedRoles {
			allowed[strings.TrimSpace(role.Slug)] = struct{}{}
		}
		for _, roleSlug := range selectedRoles {
			if _, ok := allowed[strings.TrimSpace(roleSlug)]; !ok {
				s.renderOrgAdminWithErrors(w, admin, admin.OrgSlug, "", OrgAdminErrors{Users: "role not found"})
				return
			}
		}
		if firstNonEmpty(target.ID, target.Email) == firstNonEmpty(admin.IdentityUserID, admin.Email) && !containsRole(selectedRoles, "org-admin") {
			s.renderOrgAdminWithErrors(w, admin, admin.OrgSlug, "", OrgAdminErrors{Users: "cannot remove org-admin from your own account"})
			return
		}
		labels := make([]string, 0, len(target.Labels)+len(selectedRoles)+1)
		for _, label := range target.Labels {
			if isManagedIdentityLabel(label) {
				continue
			}
			labels = append(labels, strings.TrimSpace(label))
		}
		isOrgAdmin := containsRole(selectedRoles, "org-admin")
		for _, roleSlug := range selectedRoles {
			if containsRole([]string{roleSlug}, "org-admin") || containsRole([]string{roleSlug}, "org_admin") {
				isOrgAdmin = true
				continue
			}
			labels = append(labels, encodeIdentityRoleLabel(roleSlug))
		}
		if isOrgAdmin {
			labels = append(labels, identityOrgAdminLabel)
		}
		if _, err := s.identity.UpdateUserLabels(r.Context(), target.ID, labels); err != nil {
			s.logAndRenderOrgAdminError(w, r, admin, admin.OrgSlug, "", OrgAdminErrors{Users: "failed to update user roles"}, err, "failed to update labels for user %s in organization %s", target.ID, admin.OrgSlug)
			return
		}
		s.renderOrgAdmin(w, admin, admin.OrgSlug, "", "")
	case "delete_user":
		userID := strings.TrimSpace(r.FormValue("userId"))
		if userID == "" {
			s.renderOrgAdminWithErrors(w, admin, admin.OrgSlug, "", OrgAdminErrors{Users: "user is required"})
			return
		}
		memberships, err := s.identity.ListOrganizationMemberships(r.Context(), admin.OrgSlug)
		if err != nil {
			s.logAndRenderOrgAdminError(w, r, admin, admin.OrgSlug, "", OrgAdminErrors{Users: "user not found"}, err, "failed to list memberships for organization %s during delete", admin.OrgSlug)
			return
		}
		var target *IdentityMembership
		for idx := range memberships {
			targetKey := firstNonEmpty(memberships[idx].UserID, memberships[idx].Email)
			if strings.TrimSpace(targetKey) == userID {
				target = &memberships[idx]
				break
			}
		}
		if target == nil {
			s.renderOrgAdminWithErrors(w, admin, admin.OrgSlug, "", OrgAdminErrors{Users: "user not found"})
			return
		}
		if isPlatformAdminMembership(*target) {
			s.renderOrgAdminWithErrors(w, admin, admin.OrgSlug, "", OrgAdminErrors{Users: "user not found"})
			return
		}
		if firstNonEmpty(target.UserID, target.Email) == firstNonEmpty(admin.IdentityUserID, admin.Email) {
			s.renderOrgAdminWithErrors(w, admin, admin.OrgSlug, "", OrgAdminErrors{Users: "cannot delete yourself"})
			return
		}
		sessionSecret, err := sessionSecretFromRequest(r)
		if err != nil {
			logAndHTTPError(w, r, http.StatusUnauthorized, "unauthorized", err, "failed to read session secret for membership delete in %s", admin.OrgSlug)
			return
		}
		if err := s.identity.DeleteOrganizationMembership(r.Context(), sessionSecret, admin.OrgSlug, target.ID); err != nil {
			s.logAndRenderOrgAdminError(w, r, admin, admin.OrgSlug, "", OrgAdminErrors{Users: "failed to delete user"}, err, "failed to delete membership %s in organization %s", target.ID, admin.OrgSlug)
			return
		}
		if strings.TrimSpace(target.UserID) != "" {
			targetUser, getErr := s.identity.GetUserByID(r.Context(), target.UserID)
			if getErr != nil && !errors.Is(getErr, ErrIdentityNotFound) {
				s.logAndRenderOrgAdminError(w, r, admin, admin.OrgSlug, "", OrgAdminErrors{Users: "failed to delete user"}, getErr, "failed to load deleted membership user %s in organization %s", target.UserID, admin.OrgSlug)
				return
			}
			labels := make([]string, 0, len(targetUser.Labels))
			for _, label := range targetUser.Labels {
				if isManagedIdentityLabel(label) {
					continue
				}
				labels = append(labels, strings.TrimSpace(label))
			}
			if _, err := s.identity.UpdateUserLabels(r.Context(), target.UserID, labels); err != nil {
				s.logAndRenderOrgAdminError(w, r, admin, admin.OrgSlug, "", OrgAdminErrors{Users: "failed to delete user"}, err, "failed to clear labels for deleted user %s in organization %s", target.UserID, admin.OrgSlug)
				return
			}
		}
		s.renderOrgAdmin(w, admin, admin.OrgSlug, "", "")
	default:
		s.renderOrgAdminWithErrors(w, admin, admin.OrgSlug, "", OrgAdminErrors{Users: "unsupported action"})
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

func cloneRequestWithSelectedSubstep(r *http.Request, substepID string) *http.Request {
	clone := r.Clone(r.Context())
	if clone.URL != nil {
		copied := *clone.URL
		query := copied.Query()
		substepID = strings.TrimSpace(substepID)
		if substepID == "" {
			query.Del("substep")
		} else {
			query.Set("substep", substepID)
		}
		copied.RawQuery = query.Encode()
		clone.URL = &copied
	}
	if clone.URL != nil {
		clone.RequestURI = clone.URL.RequestURI()
	}
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
	case rest == "/delete":
		s.handleDeleteWorkflow(w, cloneRequestWithPath(scopedReq, rest))
		return
	case strings.HasPrefix(rest, "/process/"):
		s.handleProcessRoutes(w, cloneRequestWithPath(scopedReq, rest))
		return
	case rest == "/events":
		s.handleEvents(w, cloneRequestWithPath(scopedReq, rest))
		return
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleDeleteWorkflow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user, _, ok := s.requireAuthenticatedPage(w, r)
	if !ok {
		return
	}
	workflowKey, cfg, err := s.selectedWorkflowUnvalidated(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if s.store == nil {
		http.Error(w, "store not configured", http.StatusInternalServerError)
		return
	}

	streamID, err := primitive.ObjectIDFromHex(workflowKey)
	if err != nil {
		redirectHomeWithMessage(w, r, "error", "Only saved streams can be deleted.")
		return
	}
	stream, err := s.store.LoadFormataBuilderStreamByID(r.Context(), streamID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			redirectHomeWithMessage(w, r, "error", "Stream not found.")
			return
		}
		http.Error(w, "failed to load stream", http.StatusInternalServerError)
		return
	}

	hasProcesses, err := s.store.HasProcessesByWorkflow(r.Context(), workflowKey)
	if err != nil {
		http.Error(w, "failed to check stream processes", http.StatusInternalServerError)
		return
	}
	if s.authorizer == nil {
		http.Error(w, "cerbos check failed", http.StatusBadGateway)
		return
	}
	allowed, err := s.authorizer.CanDeleteStream(r.Context(), user, workflowKey, formataStreamCreatorID(*stream), hasProcesses)
	if err != nil {
		logRequestError(r, err, "cerbos check failed for stream %s delete", workflowKey)
		http.Error(w, "cerbos check failed", http.StatusBadGateway)
		return
	}
	canPurgeWorkflowData, err := s.canPurgeWorkflowData(r.Context(), user, workflowKey)
	if err != nil {
		logRequestError(r, err, "cerbos check failed for stream %s purge_history", workflowKey)
		http.Error(w, "cerbos check failed", http.StatusBadGateway)
		return
	}
	if !allowed {
		switch {
		case hasProcesses && !canPurgeWorkflowData:
			redirectHomeWithMessage(w, r, "error", cfg.Workflow.Name+" cannot be deleted because one or more processes have already been started.")
		default:
			redirectHomeWithMessage(w, r, "error", "Only the stream creator or a platform admin can delete "+cfg.Workflow.Name+".")
		}
		return
	}

	if canPurgeWorkflowData {
		if err := s.store.DeleteWorkflowData(r.Context(), workflowKey); err != nil {
			http.Error(w, "failed to delete stream data", http.StatusInternalServerError)
			return
		}
	}
	if err := s.store.DeleteFormataBuilderStream(r.Context(), streamID); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			redirectHomeWithMessage(w, r, "error", "Stream not found.")
			return
		}
		http.Error(w, "failed to delete stream", http.StatusInternalServerError)
		return
	}

	redirectHomeWithMessage(w, r, "confirmation", cfg.Workflow.Name+" was deleted.")
}

func (s *Server) handleWorkflowHome(w http.ResponseWriter, r *http.Request) {
	user, _, ok := s.requireAuthenticatedPage(w, r)
	if !ok {
		return
	}
	workflowKey, cfg, err := s.selectedWorkflowUnvalidated(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctx := r.Context()
	workflowError := homePickerMessage(r, "error")
	if validationErr := s.validateWorkflowRefs(ctx, cfg); validationErr != nil {
		var refErr *WorkflowRefValidationError
		if !errors.As(validationErr, &refErr) {
			http.Error(w, validationErr.Error(), http.StatusInternalServerError)
			return
		}
		if workflowError == "" {
			workflowError = refErr.Error()
		}
	}
	sortKey := normalizeHomeSortKey(strings.TrimSpace(r.URL.Query().Get("sort")))
	statusFilter := normalizeHomeStatusFilter(r.URL.Query().Get("filter"))
	page := parsePositiveInt(r.URL.Query().Get("page"), 1)
	processesRaw, err := s.store.ListRecentProcessesByWorkflow(ctx, workflowKey, 0)
	if err != nil {
		logRequestError(r, err, "failed to list recent processes for workflow %s", workflowKey)
		processesRaw = nil
	}

	totalSubsteps := countWorkflowSubsteps(cfg.Workflow)
	var processes []ProcessListItem
	for _, process := range processesRaw {
		process.Progress = normalizeProgressKeys(process.Progress)
		status := deriveProcessStatus(cfg.Workflow, &process)
		if statusFilter != "all" && status != statusFilter {
			continue
		}
		doneCount, lastAt, lastDigest := processProgressStats(cfg.Workflow, &process)
		percent := 0
		if totalSubsteps > 0 {
			percent = int(float64(doneCount) / float64(totalSubsteps) * 100)
		}
		item := ProcessListItem{
			ID:              process.ID.Hex(),
			Status:          status,
			CreatedAt:       humanReadableTraceabilityTime(process.CreatedAt),
			CreatedAtTime:   process.CreatedAt,
			DoneSubsteps:    doneCount,
			TotalSubsteps:   totalSubsteps,
			Percent:         percent,
			LastNotarizedAt: lastAt,
			LastDigestShort: lastDigest,
		}
		processes = append(processes, item)
	}

	sortHomeProcessList(processes, sortKey)
	currentPage := normalizeHomePage(page, len(processes))
	start := (currentPage - 1) * homeProcessesPerPage
	end := min(start+homeProcessesPerPage, len(processes))
	pagedProcesses := processes
	if start < len(processes) {
		pagedProcesses = processes[start:end]
	} else if len(processes) > 0 {
		pagedProcesses = processes[:0]
	}
	totalPages := 1
	if len(processes) > 0 {
		totalPages = (len(processes) + homeProcessesPerPage - 1) / homeProcessesPerPage
	}
	pageNumbers := make([]int, 0, totalPages)
	for current := 1; current <= totalPages; current++ {
		pageNumbers = append(pageNumbers, current)
	}

	actor := actorFromAccountUser(user, workflowKey)
	if len(actor.RoleSlugs) == 0 && !s.enforceAuth {
		actor.RoleSlugs = s.roles(cfg)
		if len(actor.RoleSlugs) > 0 {
			actor.Role = actor.RoleSlugs[0]
		}
	}
	preview := makeActionListReadOnly(
		s.buildProcessActionListView(ctx, cfg, workflowKey, buildWorkflowPreviewProcess(cfg.Workflow, workflowKey), actor, "", "", false),
		"Preview only. Start an instance to submit data.",
	)

	view := HomeView{
		PageBase:            s.pageBaseForUser(user, "home_body", workflowKey, cfg.Workflow.Name),
		WorkflowDescription: strings.TrimSpace(cfg.Workflow.Description),
		Error:               workflowError,
		CanStartProcess:     workflowError == "",
		Sort:                sortKey,
		StatusFilter:        statusFilter,
		CurrentPage:         currentPage,
		TotalPages:          totalPages,
		PageNumbers:         pageNumbers,
		HasPreviousPage:     currentPage > 1,
		HasNextPage:         currentPage < totalPages,
		PreviousPage:        max(currentPage-1, 1),
		NextPage:            min(currentPage+1, totalPages),
		Processes:           pagedProcesses,
		Preview:             preview,
	}
	if err := s.tmpl.ExecuteTemplate(w, "stream.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleStartProcess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	workflowKey, cfg, ok := s.selectedWorkflowOrRedirectHome(w, r)
	if !ok {
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
	if len(parts) == 2 && parts[1] == "content" && r.Method == http.MethodGet {
		s.handleProcessContentPartial(w, r, processID)
		return
	}
	if len(parts) == 2 && parts[1] == "actions" && r.Method == http.MethodGet {
		s.handleProcessActionsPartial(w, r, processID)
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
	workflowKey, cfg, ok := s.selectedWorkflowOrRedirectHome(w, r)
	if !ok {
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
	actor := Actor{
		ID:          accountActorID(user),
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
	selectedSubstepID := strings.TrimSpace(r.URL.Query().Get("substep"))
	view := s.buildProcessPageView(
		ctx,
		s.pageBaseForUser(user, "process_body", workflowKey, cfg.Workflow.Name),
		cfg,
		workflowKey,
		process,
		actor,
		selectedSubstepID,
		"",
		false,
	)
	if err := s.tmpl.ExecuteTemplate(w, "process.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) buildProcessActionListView(ctx context.Context, cfg RuntimeConfig, workflowKey string, process *Process, actor Actor, selectedSubstepID, message string, onlyRole bool) ActionListView {
	actions := buildActionList(cfg.Workflow, process, workflowKey, actor, onlyRole, s.roleMetaMap(cfg))
	processDone := process != nil && isProcessDone(cfg.Workflow, process)
	selected := resolveSelectedSubstepID(actions, selectedSubstepID, processDone)
	timeline := decorateTimelineSelection(buildTimeline(cfg.Workflow, process, workflowKey, s.roleMetaMap(cfg), organizationNameMap(cfg)), selected)
	timeline = decorateTimelineOrganizationLogos(timeline, organizationLogoURLMap(ctx, s.identity))
	actions = s.applyDoneByEmailToActions(ctx, cfg.Workflow, actor, actions)
	timeline = decorateTimelineActions(timeline, actions)

	view := ActionListView{
		WorkflowKey:       workflowKey,
		WorkflowPath:      workflowPath(workflowKey),
		ProcessID:         processIDString(process),
		CurrentUser:       actor,
		SelectedSubstepID: selected,
		ProcessDone:       processDone,
		Error:             message,
		Timeline:          timeline,
	}
	if action, ok := selectedActionBySubstep(actions, selected, processDone); ok {
		view.Action = &action
	}

	if processDone {
		view.Attachments = buildProcessDownloadAttachments(workflowKey, process, collectProcessAttachments(cfg.Workflow, process))
		if process != nil && process.DPP != nil {
			view.DPPURL = digitalLinkURL(process.DPP.GTIN, process.DPP.Lot, process.DPP.Serial)
			view.DPPGS1 = gs1ElementString(process.DPP.GTIN, process.DPP.Lot, process.DPP.Serial)
		}
	}
	if view.Action != nil {
		action := *view.Action
		view.Action = &action
	}
	view.Timeline = s.applyDoneByEmailToTimeline(ctx, cfg.Workflow, actor, view.Timeline)
	return view
}

func (s *Server) buildProcessPageView(ctx context.Context, pageBase PageBase, cfg RuntimeConfig, workflowKey string, process *Process, actor Actor, selectedSubstepID, message string, onlyRole bool) ProcessPageView {
	actionList := s.buildProcessActionListView(ctx, cfg, workflowKey, process, actor, selectedSubstepID, message, onlyRole)
	processID := ""
	if process != nil {
		processID = process.ID.Hex()
	}
	return ProcessPageView{
		PageBase:    pageBase,
		ProcessID:   processID,
		ActionList:  actionList,
		DPPURL:      actionList.DPPURL,
		DPPGS1:      actionList.DPPGS1,
		Attachments: actionList.Attachments,
	}
}

func buildWorkflowPreviewProcess(def WorkflowDef, workflowKey string) *Process {
	process := &Process{
		WorkflowKey: workflowKey,
		Status:      "active",
		Progress:    map[string]ProcessStep{},
	}
	for _, step := range sortedSteps(def) {
		for _, sub := range sortedSubsteps(step) {
			process.Progress[sub.SubstepID] = ProcessStep{State: "pending"}
		}
	}
	return process
}

func makeActionListReadOnly(view ActionListView, reason string) ActionListView {
	reason = strings.TrimSpace(reason)
	if view.Action != nil {
		action := *view.Action
		action.ReadOnly = true
		action.Reason = reason
		view.Action = &action
	}
	for stepIndex := range view.Timeline {
		for substepIndex := range view.Timeline[stepIndex].Substeps {
			action := view.Timeline[stepIndex].Substeps[substepIndex].Action
			if action == nil {
				continue
			}
			actionCopy := *action
			actionCopy.ReadOnly = true
			actionCopy.Reason = reason
			view.Timeline[stepIndex].Substeps[substepIndex].Action = &actionCopy
		}
	}
	return view
}

func actorFromAccountUser(user *AccountUser, workflowKey string) Actor {
	actor := Actor{
		WorkflowKey: workflowKey,
	}
	if user == nil {
		return actor
	}
	actor.ID = accountActorID(user)
	actor.OrgSlug = strings.TrimSpace(user.OrgSlug)
	actor.RoleSlugs = append([]string(nil), user.RoleSlugs...)
	if len(actor.RoleSlugs) > 0 {
		actor.Role = actor.RoleSlugs[0]
	}
	return actor
}

func viewerCanSeeDoneByEmail(def WorkflowDef, viewer Actor) bool {
	orgSlug := strings.TrimSpace(viewer.OrgSlug)
	if orgSlug == "" {
		return false
	}
	for _, step := range def.Steps {
		if strings.TrimSpace(step.OrganizationSlug) == orgSlug {
			return true
		}
	}
	return false
}

type userIdentityView struct {
	email      string
	fallbackID string
}

func (s *Server) lookupUserIdentityByActorID(ctx context.Context, actorID string, cache map[string]userIdentityView) (userIdentityView, bool) {
	id := strings.TrimSpace(actorID)
	if id == "" {
		return userIdentityView{}, false
	}
	if identity, ok := cache[id]; ok {
		return identity, strings.TrimSpace(identity.email) != "" || strings.TrimSpace(identity.fallbackID) != ""
	}
	if appwriteUserID, ok := parseAppwriteActorID(id); ok {
		if s.identity == nil {
			cache[id] = userIdentityView{}
			return userIdentityView{}, false
		}
		user, err := s.identity.GetUserByID(ctx, appwriteUserID)
		if err != nil {
			cache[id] = userIdentityView{}
			return userIdentityView{}, false
		}
		identity := userIdentityView{
			email:      strings.TrimSpace(user.Email),
			fallbackID: appwriteActorID(firstNonEmpty(user.ID, appwriteUserID)),
		}
		cache[id] = identity
		return identity, identity.email != "" || identity.fallbackID != ""
	}
	cache[id] = userIdentityView{}
	return userIdentityView{}, false
}

func (s *Server) applyDoneByEmailToActions(ctx context.Context, def WorkflowDef, viewer Actor, actions []ActionView) []ActionView {
	if len(actions) == 0 {
		return actions
	}
	canSeeEmail := viewerCanSeeDoneByEmail(def, viewer)
	cache := map[string]userIdentityView{}
	for idx := range actions {
		identity, ok := s.lookupUserIdentityByActorID(ctx, actions[idx].DoneBy, cache)
		if !ok {
			continue
		}
		if canSeeEmail {
			if strings.TrimSpace(identity.email) != "" {
				actions[idx].DoneBy = identity.email
			} else if strings.TrimSpace(identity.fallbackID) != "" {
				actions[idx].DoneBy = identity.fallbackID
			}
			continue
		}
		if strings.TrimSpace(identity.fallbackID) != "" {
			actions[idx].DoneBy = identity.fallbackID
		}
	}
	return actions
}

func (s *Server) applyDoneByEmailToTimeline(ctx context.Context, def WorkflowDef, viewer Actor, timeline []TimelineStep) []TimelineStep {
	if len(timeline) == 0 {
		return timeline
	}
	canSeeEmail := viewerCanSeeDoneByEmail(def, viewer)
	cache := map[string]userIdentityView{}
	for stepIdx := range timeline {
		for subIdx := range timeline[stepIdx].Substeps {
			identity, ok := s.lookupUserIdentityByActorID(ctx, timeline[stepIdx].Substeps[subIdx].DoneBy, cache)
			if !ok {
				continue
			}
			if canSeeEmail {
				if strings.TrimSpace(identity.email) != "" {
					timeline[stepIdx].Substeps[subIdx].DoneBy = identity.email
				} else if strings.TrimSpace(identity.fallbackID) != "" {
					timeline[stepIdx].Substeps[subIdx].DoneBy = identity.fallbackID
				}
				continue
			}
			if strings.TrimSpace(identity.fallbackID) != "" {
				timeline[stepIdx].Substeps[subIdx].DoneBy = identity.fallbackID
			}
		}
	}
	return timeline
}

func (s *Server) applyDoneByIdentityFallbackToDPPTraceability(ctx context.Context, traceability []DPPTraceabilityStep) []DPPTraceabilityStep {
	if len(traceability) == 0 {
		return traceability
	}
	cache := map[string]userIdentityView{}
	for stepIdx := range traceability {
		for subIdx := range traceability[stepIdx].Substeps {
			identity, ok := s.lookupUserIdentityByActorID(ctx, traceability[stepIdx].Substeps[subIdx].DoneBy, cache)
			if !ok {
				continue
			}
			if strings.TrimSpace(identity.fallbackID) != "" {
				traceability[stepIdx].Substeps[subIdx].DoneBy = identity.fallbackID
			}
		}
	}
	return traceability
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
	traceability := buildDPPTraceabilityView(cfg.Workflow, process, workflowKey, s.roleMetaMap(cfg), organizationNameMap(cfg))
	traceability = s.applyDoneByIdentityFallbackToDPPTraceability(r.Context(), traceability)
	view := DPPPageView{
		PageBase:     s.pageBase("dpp_body", workflowKey, cfg.Workflow.Name),
		ProcessID:    process.ID.Hex(),
		DigitalLink:  link,
		GTIN:         gtin,
		Lot:          lot,
		Serial:       serial,
		IssuedAt:     issuedAt,
		Workflow:     cfg.Workflow,
		Traceability: traceability,
		Integrity:    buildDPPIntegrityView(export.Merkle),
		Export:       export,
	}
	if err := s.tmpl.ExecuteTemplate(w, "dpp.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleProcessContentPartial(w http.ResponseWriter, r *http.Request, processID string) {
	user, _, ok := s.requireAuthenticatedPage(w, r)
	if !ok {
		return
	}
	workflowKey, cfg, ok := s.selectedWorkflowOrRedirectHome(w, r)
	if !ok {
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
	actor := Actor{
		ID:          accountActorID(user),
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
	view := s.buildProcessPageView(
		ctx,
		s.pageBaseForUser(user, "process_body", workflowKey, cfg.Workflow.Name),
		cfg,
		workflowKey,
		process,
		actor,
		strings.TrimSpace(r.URL.Query().Get("substep")),
		"",
		false,
	)
	if err := s.tmpl.ExecuteTemplate(w, "process_content.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleProcessActionsPartial(w http.ResponseWriter, r *http.Request, processID string) {
	user, _, ok := s.requireAuthenticatedPage(w, r)
	if !ok {
		return
	}
	workflowKey, cfg, ok := s.selectedWorkflowOrRedirectHome(w, r)
	if !ok {
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
	actor := Actor{
		ID:          accountActorID(user),
		OrgSlug:     user.OrgSlug,
		RoleSlugs:   append([]string(nil), user.RoleSlugs...),
		WorkflowKey: workflowKey,
	}
	if len(actor.RoleSlugs) > 0 {
		actor.Role = actor.RoleSlugs[0]
	}
	if len(actor.RoleSlugs) == 0 && !s.enforceAuth {
		actor.RoleSlugs = s.roles(cfg)
	}
	view := s.buildProcessActionListView(r.Context(), cfg, workflowKey, process, actor, strings.TrimSpace(r.URL.Query().Get("substep")), "", false)
	if err := s.tmpl.ExecuteTemplate(w, "action_list.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleProcessDownloadsPartial(w http.ResponseWriter, r *http.Request, processID string) {
	workflowKey, cfg, ok := s.selectedWorkflowOrRedirectHome(w, r)
	if !ok {
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
	workflowKey, cfg, ok := s.selectedWorkflowOrRedirectHome(w, r)
	if !ok {
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
	workflowKey, cfg, ok := s.selectedWorkflowOrRedirectHome(w, r)
	if !ok {
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
	workflowKey, cfg, ok := s.selectedWorkflowOrRedirectHome(w, r)
	if !ok {
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
	workflowKey, cfg, ok := s.selectedWorkflowOrRedirectHome(w, r)
	if !ok {
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
	disposition := "attachment"
	if strings.TrimSpace(r.URL.Query().Get("inline")) != "" {
		disposition = "inline"
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("%s; filename=%q", disposition, filename))
	if _, err := io.Copy(w, download); err != nil {
		return
	}
}

func (s *Server) handleDownloadProcessAttachment(w http.ResponseWriter, r *http.Request, processID, attachmentID string) {
	workflowKey, _, ok := s.selectedWorkflowOrRedirectHome(w, r)
	if !ok {
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
	disposition := "attachment"
	if strings.TrimSpace(r.URL.Query().Get("inline")) != "" {
		disposition = "inline"
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("%s; filename=%q", disposition, filename))
	if _, err := io.Copy(w, download); err != nil {
		return
	}
}

func (s *Server) handleCompleteSubstep(w http.ResponseWriter, r *http.Request, processID, substepID string) {
	user, _, ok := s.requireAuthenticatedPost(w, r)
	if !ok {
		return
	}
	workflowKey, cfg, selected := s.selectedWorkflowOrRedirectHome(w, r)
	if !selected {
		return
	}
	actor := Actor{
		ID:          accountActorID(user),
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
		if !errors.Is(err, mongo.ErrNoDocuments) {
			logRequestError(r, err, "failed to load process %s for substep %s completion", processID, substepID)
		}
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
	_ = r.ParseForm()
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
		logRequestError(r, err, "cerbos check failed for process %s substep %s", processID, substepID)
		s.renderActionErrorForRequest(w, r, http.StatusBadGateway, "Cerbos check failed.", process, actor)
		return
	}
	if !sequenceOK {
		if progress, ok := process.Progress[substepID]; ok && progress.State == "done" && containsRole(allowedRoles, actor.Role) {
			nextReq := cloneRequestWithSelectedSubstep(r, "")
			if isProcessContentTargetRequest(r) {
				s.renderProcessContent(w, nextReq, process, actor, "")
				return
			}
			if isHTMXRequest(r) {
				s.renderActionList(w, nextReq, process, actor, "")
				return
			}
			s.renderDepartmentProcessPage(w, nextReq, process, actor, "")
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
	payload, err := s.parseCompletionPayload(r, process.ID, substep, now)
	if err != nil {
		switch {
		case errors.Is(err, ErrAttachmentTooLarge):
			s.renderActionErrorForRequest(w, r, http.StatusRequestEntityTooLarge, "File too large.", process, actor)
		case errors.Is(err, errInvalidForm):
			s.renderActionErrorForRequest(w, r, http.StatusBadRequest, "Invalid form.", process, actor)
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
		logRequestError(r, err, "failed to update process %s substep %s", process.ID.Hex(), substepID)
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
		logRequestError(r, err, "failed to notarize process %s substep %s", process.ID.Hex(), substepID)
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
	nextReq := cloneRequestWithSelectedSubstep(r, "")
	if isProcessContentTargetRequest(r) {
		s.renderProcessContent(w, nextReq, process, actor, "")
		return
	}
	if isHTMXRequest(r) {
		s.renderActionList(w, nextReq, process, actor, "")
		return
	}
	s.renderDepartmentProcessPage(w, nextReq, process, actor, "")
}

var (
	errInvalidForm = errors.New("invalid form")
)

func (s *Server) parseCompletionPayload(r *http.Request, processID primitive.ObjectID, substep WorkflowSub, now time.Time) (map[string]interface{}, error) {
	if substep.InputType != "formata" {
		return nil, errors.New("Only formata substeps are supported.")
	}
	return s.parseFormataPayload(r, processID, substep, now)
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
	normalizedPath := make([]string, 0, len(path))
	for _, segment := range path {
		normalizedPath = append(normalizedPath, normalizeFormataPathSegment(segment))
	}
	fieldPath := strings.TrimSpace(strings.Join(normalizedPath, "_"))
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

func normalizeFormataPathSegment(value string) string {
	trimmed := strings.TrimSpace(value)
	return strings.TrimSuffix(trimmed, "[]")
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

func cloneWorkflowCatalog(catalog map[string]RuntimeConfig) map[string]RuntimeConfig {
	cloned := make(map[string]RuntimeConfig, len(catalog))
	for key, cfg := range catalog {
		cloned[key] = cfg
	}
	return cloned
}

func parseRuntimeConfigData(source string, data []byte) (RuntimeConfig, error) {
	var cfg RuntimeConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return RuntimeConfig{}, fmt.Errorf("parse config %s: %w", source, err)
	}
	normalizeWorkflowConfig(&cfg)
	if cfg.Workflow.Name == "" || len(cfg.Workflow.Steps) == 0 {
		return RuntimeConfig{}, fmt.Errorf("workflow config is empty in %s", source)
	}
	if err := normalizeInputTypes(&cfg.Workflow); err != nil {
		return RuntimeConfig{}, fmt.Errorf("%s: %w", source, err)
	}
	if err := normalizeDPPConfig(&cfg.DPP); err != nil {
		return RuntimeConfig{}, fmt.Errorf("%s: %w", source, err)
	}
	return cfg, nil
}

func workflowCatalogModTime(stream FormataBuilderStream) time.Time {
	if !stream.UpdatedAt.IsZero() {
		return stream.UpdatedAt
	}
	if !stream.ID.IsZero() {
		return stream.ID.Timestamp()
	}
	return time.Time{}
}

func (s *Server) workflowCatalog() (map[string]RuntimeConfig, error) {
	s.configMu.Lock()
	defer s.configMu.Unlock()

	if s.store != nil {
		streams, err := s.store.ListFormataBuilderStreams(context.Background())
		if err != nil {
			return nil, fmt.Errorf("list formata streams: %w", err)
		}
		if len(streams) > 0 {
			modTimes := make(map[string]time.Time, len(streams))
			for _, stream := range streams {
				if stream.ID.IsZero() {
					return nil, errors.New("formata stream id is empty")
				}
				key := stream.ID.Hex()
				modTimes[key] = workflowCatalogModTime(stream)
			}
			if s.catalog != nil && sameCatalogModTimes(s.catalogModTime, modTimes) {
				return cloneWorkflowCatalog(s.catalog), nil
			}

			catalog := make(map[string]RuntimeConfig, len(streams))
			for _, stream := range streams {
				if stream.ID.IsZero() {
					return nil, errors.New("formata stream id is empty")
				}
				key := stream.ID.Hex()
				cfg, parseErr := parseRuntimeConfigData("stream "+key, []byte(stream.Stream))
				if parseErr != nil {
					return nil, parseErr
				}
				catalog[key] = cfg
			}
			s.catalog = catalog
			s.catalogModTime = modTimes
			return cloneWorkflowCatalog(catalog), nil
		}
	}

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
		return cloneWorkflowCatalog(s.catalog), nil
	}

	catalog := make(map[string]RuntimeConfig, len(paths))
	for _, path := range paths {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, fmt.Errorf("read config %s: %w", path, readErr)
		}
		cfg, parseErr := parseRuntimeConfigData(filepath.Base(path), data)
		if parseErr != nil {
			return nil, parseErr
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

	return cloneWorkflowCatalog(catalog), nil
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

func organizationNameMap(cfg RuntimeConfig) map[string]string {
	out := map[string]string{}
	for _, org := range cfg.Organizations {
		slug := strings.TrimSpace(org.Slug)
		if slug == "" {
			continue
		}
		name := strings.TrimSpace(org.Name)
		if name == "" {
			name = slug
		}
		out[slug] = name
	}
	return out
}

func organizationLogoURLMap(ctx context.Context, identity IdentityStore) map[string]string {
	if identity == nil {
		return map[string]string{}
	}
	orgs, err := identity.ListOrganizations(ctx)
	if err != nil {
		return map[string]string{}
	}
	out := map[string]string{}
	for _, org := range orgs {
		slug := strings.TrimSpace(org.Slug)
		if slug == "" || strings.TrimSpace(org.LogoFileID) == "" {
			continue
		}
		out[slug] = "/organization/logo/" + url.PathEscape(slug)
	}
	return out
}

func organizationDisplayName(slug string, orgNames map[string]string) string {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return ""
	}
	if name, ok := orgNames[slug]; ok && strings.TrimSpace(name) != "" {
		return strings.TrimSpace(name)
	}
	return slug
}

func buildTimeline(def WorkflowDef, process *Process, workflowKey string, roleMeta map[string]RoleMeta, orgNames map[string]string) []TimelineStep {
	steps := sortedSteps(def)
	availableMap := computeAvailability(def, process)

	var timeline []TimelineStep
	for _, step := range steps {
		row := TimelineStep{
			StepID:  step.StepID,
			Title:   step.Title,
			OrgSlug: strings.TrimSpace(step.OrganizationSlug),
			OrgName: organizationDisplayName(step.OrganizationSlug, orgNames),
		}
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
					Color:  cssValue(badgeMeta.Color, "var(--role-fallback)"),
					Border: cssValue(badgeMeta.Border, "var(--border)"),
				})
			}
			entry := TimelineSubstep{
				SubstepID:  sub.SubstepID,
				Title:      sub.Title,
				Role:       strings.Join(allowedRoles, ", "),
				RoleBadges: roleBadges,
				RoleLabel:  meta.Label,
				RoleColor:  cssValue(meta.Color, "var(--role-fallback)"),
				RoleBorder: cssValue(meta.Border, "var(--border)"),
			}
			if process != nil {
				if progress, ok := process.Progress[sub.SubstepID]; ok && progress.State == "done" {
					entry.Status = "done"
					if progress.DoneBy != nil {
						entry.DoneBy = progress.DoneBy.ID
						entry.DoneRole = progress.DoneBy.Role
						selectedRole := strings.TrimSpace(progress.DoneBy.Role)
						if selectedRole != "" {
							selectedMeta := roleMetaFor(selectedRole, roleMeta)
							entry.Role = selectedRole
							entry.RoleBadges = []TimelineRoleBadge{
								{
									ID:     selectedRole,
									Label:  selectedMeta.Label,
									Color:  cssValue(selectedMeta.Color, "var(--role-fallback)"),
									Border: cssValue(selectedMeta.Border, "var(--border)"),
								},
							}
							entry.RoleLabel = selectedMeta.Label
							entry.RoleColor = cssValue(selectedMeta.Color, "var(--role-fallback)")
							entry.RoleBorder = cssValue(selectedMeta.Border, "var(--border)")
						}
					}
					if progress.DoneAt != nil {
						entry.DoneAt = humanReadableTraceabilityTime(*progress.DoneAt)
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

type keyedAttachmentView struct {
	Key  string
	Meta NotarizedAttachment
}

func attachmentViewsFromValue(raw interface{}) []keyedAttachmentView {
	if typed, ok := raw.(map[string]interface{}); ok {
		var files []keyedAttachmentView
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			value := typed[key]
			if isAttachmentMetaValue(value) {
				collectAttachmentViews(normalizeFormataPathSegment(key), value, &files)
				continue
			}
			collectAttachmentViews("", value, &files)
		}
		return files
	}
	if typed, ok := raw.(primitive.M); ok {
		return attachmentViewsFromValue(map[string]interface{}(typed))
	}
	var files []keyedAttachmentView
	collectAttachmentViews("", raw, &files)
	return files
}

func collectAttachmentViews(path string, raw interface{}, files *[]keyedAttachmentView) {
	switch typed := raw.(type) {
	case map[string]interface{}:
		if meta := attachmentMetaFromMap(typed); meta != nil {
			key := strings.TrimSpace(path)
			if key == "" {
				key = "value"
			}
			*files = append(*files, keyedAttachmentView{Key: key, Meta: *meta})
			return
		}
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			nextPath := normalizeFormataPathSegment(key)
			if strings.TrimSpace(path) != "" {
				nextPath = path + "." + normalizeFormataPathSegment(key)
			}
			collectAttachmentViews(nextPath, typed[key], files)
		}
	case primitive.M:
		collectAttachmentViews(path, map[string]interface{}(typed), files)
	case []interface{}:
		for idx, nested := range typed {
			nextPath := fmt.Sprintf("[%d]", idx)
			if strings.TrimSpace(path) != "" {
				nextPath = fmt.Sprintf("%s[%d]", path, idx)
			}
			collectAttachmentViews(nextPath, nested, files)
		}
	case primitive.A:
		for idx, nested := range typed {
			nextPath := fmt.Sprintf("[%d]", idx)
			if strings.TrimSpace(path) != "" {
				nextPath = fmt.Sprintf("%s[%d]", path, idx)
			}
			collectAttachmentViews(nextPath, nested, files)
		}
	}
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
	case primitive.A:
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
					entry.DoneBy = progress.DoneBy.ID
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

func substepOrganizationMap(def WorkflowDef) map[string]string {
	orgs := map[string]string{}
	for _, step := range sortedSteps(def) {
		orgSlug := strings.TrimSpace(step.OrganizationSlug)
		for _, sub := range sortedSubsteps(step) {
			orgs[sub.SubstepID] = orgSlug
		}
	}
	return orgs
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

func buildActionList(def WorkflowDef, process *Process, workflowKey string, actor Actor, onlyRole bool, roleMeta map[string]RoleMeta) []ActionView {
	var actions []ActionView
	ordered := orderedSubsteps(def)
	availMap := computeAvailability(def, process)
	substepOrgs := substepOrganizationMap(def)
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
				Color:  cssValue(meta.Color, "var(--role-fallback)"),
				Border: cssValue(meta.Border, "var(--border)"),
			})
		}
		if onlyRole && strings.TrimSpace(actor.Role) != "" && !containsRole(allowedRoles, actor.Role) {
			continue
		}
		meta := roleMetaFor(primaryRole, roleMeta)
		role := primaryRole
		roleLabel := meta.Label
		roleColor := cssValue(meta.Color, "var(--role-fallback)")
		roleBorder := cssValue(meta.Border, "var(--border)")
		status := "locked"
		if process != nil {
			if step, ok := process.Progress[sub.SubstepID]; ok && step.State == "done" {
				status = "done"
			} else if availMap[sub.SubstepID] {
				status = "available"
			}
		}
		stepOrgSlug := substepOrgs[sub.SubstepID]
		orgAuthorized := stepOrgSlug == "" || strings.TrimSpace(actor.OrgSlug) == stepOrgSlug
		disabled := status != "available" || len(matchingRoles) == 0 || !orgAuthorized
		reason := ""
		if status == "locked" {
			reason = "Locked by sequence"
		} else if status == "done" {
			reason = "Already completed"
		} else if !orgAuthorized {
			reason = "Not authorized for organization"
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
					doneAt = humanReadableTraceabilityTime(*progress.DoneAt)
				}
				if progress.DoneBy != nil {
					doneBy = strings.TrimSpace(progress.DoneBy.ID)
					doneRole = strings.TrimSpace(progress.DoneBy.Role)
					if doneRole != "" {
						selectedMeta := roleMetaFor(doneRole, roleMeta)
						role = doneRole
						roleBadges = []ActionRoleBadge{
							{
								ID:     doneRole,
								Label:  selectedMeta.Label,
								Color:  cssValue(selectedMeta.Color, "var(--role-fallback)"),
								Border: cssValue(selectedMeta.Border, "var(--border)"),
							},
						}
						roleLabel = selectedMeta.Label
						roleColor = cssValue(selectedMeta.Color, "var(--role-fallback)")
						roleBorder = cssValue(selectedMeta.Border, "var(--border)")
					}
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
			WorkflowKey:   workflowKey,
			ProcessID:     processIDString(process),
			SubstepID:     sub.SubstepID,
			Title:         sub.Title,
			Role:          role,
			AllowedRoles:  allowedRoles,
			RoleBadges:    roleBadges,
			MatchingRoles: matchingRoles,
			RoleLabel:     roleLabel,
			RoleColor:     roleColor,
			RoleBorder:    roleBorder,
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

func resolveSelectedSubstepID(actions []ActionView, requested string, processDone bool) string {
	if processDone || len(actions) == 0 {
		return ""
	}
	requested = strings.TrimSpace(requested)
	if requested != "" {
		for _, action := range actions {
			if action.SubstepID == requested {
				return requested
			}
		}
	}
	for _, action := range actions {
		if action.Status == "available" {
			return action.SubstepID
		}
	}
	return actions[0].SubstepID
}

func selectedActionBySubstep(actions []ActionView, selectedSubstepID string, processDone bool) (ActionView, bool) {
	if processDone {
		return ActionView{}, false
	}
	selectedSubstepID = strings.TrimSpace(selectedSubstepID)
	if selectedSubstepID == "" {
		if len(actions) == 0 {
			return ActionView{}, false
		}
		return actions[0], true
	}
	for _, action := range actions {
		if action.SubstepID == selectedSubstepID {
			return action, true
		}
	}
	return ActionView{}, false
}

func decorateTimelineSelection(timeline []TimelineStep, selectedSubstepID string) []TimelineStep {
	selectedSubstepID = strings.TrimSpace(selectedSubstepID)
	for stepIndex := range timeline {
		expanded := false
		for substepIndex := range timeline[stepIndex].Substeps {
			selected := selectedSubstepID != "" && timeline[stepIndex].Substeps[substepIndex].SubstepID == selectedSubstepID
			timeline[stepIndex].Substeps[substepIndex].Selected = selected
			if selected {
				expanded = true
			}
		}
		timeline[stepIndex].Expanded = expanded
	}
	return timeline
}

func decorateTimelineActions(timeline []TimelineStep, actions []ActionView) []TimelineStep {
	if len(timeline) == 0 || len(actions) == 0 {
		return timeline
	}
	actionsBySubstep := make(map[string]ActionView, len(actions))
	for _, action := range actions {
		actionsBySubstep[strings.TrimSpace(action.SubstepID)] = action
	}
	for stepIndex := range timeline {
		for substepIndex := range timeline[stepIndex].Substeps {
			substepID := strings.TrimSpace(timeline[stepIndex].Substeps[substepIndex].SubstepID)
			action, ok := actionsBySubstep[substepID]
			if !ok {
				continue
			}
			actionCopy := action
			timeline[stepIndex].Substeps[substepIndex].Action = &actionCopy
		}
	}
	return timeline
}

func decorateTimelineOrganizationLogos(timeline []TimelineStep, logoURLs map[string]string) []TimelineStep {
	if len(timeline) == 0 || len(logoURLs) == 0 {
		return timeline
	}
	for stepIndex := range timeline {
		timeline[stepIndex].OrgLogoURL = strings.TrimSpace(logoURLs[strings.TrimSpace(timeline[stepIndex].OrgSlug)])
	}
	return timeline
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
			nextPath := normalizeFormataPathSegment(key)
			if strings.TrimSpace(path) != "" {
				nextPath = path + "." + normalizeFormataPathSegment(key)
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
	case primitive.A:
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

func buildDPPIntegrityView(tree MerkleTree) DPPIntegrityView {
	view := DPPIntegrityView{
		Root: DPPIntegrityHashView{
			Full:  tree.Root,
			Short: shortHashLabel(tree.Root),
		},
		Leaves: make([]DPPIntegrityLeafView, 0, len(tree.Leaves)),
	}
	for _, leaf := range tree.Leaves {
		view.Leaves = append(view.Leaves, DPPIntegrityLeafView{
			SubstepID: leaf.SubstepID,
			Hash: DPPIntegrityHashView{
				Full:  leaf.Hash,
				Short: shortHashLabel(leaf.Hash),
			},
		})
	}
	return view
}

func shortHashLabel(hash string) string {
	hash = strings.TrimSpace(hash)
	const shortLen = 7
	if len(hash) <= shortLen {
		return hash
	}
	return hash[:shortLen]
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
	for _, item := range attachmentViewsFromValue(data) {
		meta := item.Meta
		id := strings.TrimSpace(meta.AttachmentID)
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		downloadURL := fmt.Sprintf("%s/process/%s/attachment/%s/file", workflowPath(workflowKey), process.ID.Hex(), id)
		previewKind := actionAttachmentPreviewKind(meta)
		attachments = append(attachments, ActionAttachmentView{
			Key:         item.Key,
			Filename:    sanitizeAttachmentFilename(meta.Filename),
			URL:         downloadURL,
			PreviewURL:  actionAttachmentPreviewURL(downloadURL, previewKind),
			PreviewKind: previewKind,
			SHA256:      strings.TrimSpace(meta.SHA256),
		})
	}
	sort.Slice(attachments, func(i, j int) bool {
		if attachments[i].Key != attachments[j].Key {
			return attachments[i].Key < attachments[j].Key
		}
		if attachments[i].Filename != attachments[j].Filename {
			return attachments[i].Filename < attachments[j].Filename
		}
		return attachments[i].URL < attachments[j].URL
	})
	return attachments
}

func actionAttachmentPreviewKind(meta NotarizedAttachment) string {
	contentType := strings.ToLower(strings.TrimSpace(meta.ContentType))
	filename := strings.ToLower(strings.TrimSpace(meta.Filename))
	switch {
	case strings.HasPrefix(contentType, "image/"):
		return "image"
	case contentType == "application/pdf":
		return "document"
	}
	switch filepath.Ext(filename) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp", ".svg":
		return "image"
	case ".pdf":
		return "document"
	default:
		return ""
	}
}

func actionAttachmentPreviewURL(downloadURL, previewKind string) string {
	if previewKind == "" {
		return ""
	}
	inlineURL := downloadURL + "?inline=1"
	if previewKind == "document" {
		return inlineURL + "#page=1&toolbar=0&navpanes=0&view=FitH"
	}
	return inlineURL
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
	return RoleMeta{
		ID:     role,
		Label:  role,
		Color:  "var(--role-fallback)",
		Border: "var(--border)",
	}
}

func cssValue(value, fallback string) template.CSS {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		trimmed = strings.TrimSpace(fallback)
	}
	return template.CSS(trimmed)
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
	if sub.InputType != "formata" {
		return nil, errors.New("Value must be a valid JSON object.")
	}
	var decoded interface{}
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		return nil, errors.New("Value must be a valid JSON object.")
	}
	valueObject, ok := decoded.(map[string]interface{})
	if !ok {
		return nil, errors.New("Value must be a valid JSON object.")
	}
	return map[string]interface{}{sub.InputKey: valueObject}, nil
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

func (s *Server) renderActionErrorForRequest(w http.ResponseWriter, r *http.Request, status int, message string, process *Process, actor Actor) {
	w.WriteHeader(status)
	if isProcessContentTargetRequest(r) {
		s.renderProcessContent(w, r, process, actor, message)
		return
	}
	if isHTMXRequest(r) {
		s.renderActionList(w, r, process, actor, message)
		return
	}
	s.renderDepartmentProcessPage(w, r, process, actor, message)
}

func isProcessContentTargetRequest(r *http.Request) bool {
	if r == nil || !isHTMXRequest(r) {
		return false
	}
	return strings.TrimSpace(r.Header.Get("HX-Target")) == "process-page-content"
}

func (s *Server) renderActionList(w http.ResponseWriter, r *http.Request, process *Process, actor Actor, message string) {
	workflowKey := s.defaultWorkflowKey()
	cfg := RuntimeConfig{}
	var err error
	selectedSubstepID := ""
	if r != nil {
		workflowKey, cfg, err = s.selectedWorkflow(r)
		selectedSubstepID = strings.TrimSpace(r.URL.Query().Get("substep"))
	} else {
		cfg, err = s.runtimeConfig()
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctx := context.Background()
	if r != nil {
		ctx = r.Context()
	}
	view := s.buildProcessActionListView(ctx, cfg, workflowKey, process, actor, selectedSubstepID, message, false)
	if err := s.tmpl.ExecuteTemplate(w, "action_list.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) renderDepartmentProcessPage(w http.ResponseWriter, r *http.Request, process *Process, actor Actor, message string) {
	workflowKey := s.defaultWorkflowKey()
	cfg := RuntimeConfig{}
	var err error
	selectedSubstepID := ""
	if r != nil {
		workflowKey, cfg, err = s.selectedWorkflow(r)
		selectedSubstepID = strings.TrimSpace(r.URL.Query().Get("substep"))
	} else {
		cfg, err = s.runtimeConfig()
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctx := context.Background()
	if r != nil {
		ctx = r.Context()
	}
	view := s.buildProcessPageView(
		ctx,
		s.pageBase("process_body", workflowKey, cfg.Workflow.Name),
		cfg,
		workflowKey,
		process,
		actor,
		selectedSubstepID,
		message,
		false,
	)
	if err := s.tmpl.ExecuteTemplate(w, "process.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) renderProcessContent(w http.ResponseWriter, r *http.Request, process *Process, actor Actor, message string) {
	workflowKey := s.defaultWorkflowKey()
	cfg := RuntimeConfig{}
	var err error
	selectedSubstepID := ""
	if r != nil {
		workflowKey, cfg, err = s.selectedWorkflow(r)
		selectedSubstepID = strings.TrimSpace(r.URL.Query().Get("substep"))
	} else {
		cfg, err = s.runtimeConfig()
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctx := context.Background()
	if r != nil {
		ctx = r.Context()
	}
	view := s.buildProcessPageView(
		ctx,
		s.pageBase("process_body", workflowKey, cfg.Workflow.Name),
		cfg,
		workflowKey,
		process,
		actor,
		selectedSubstepID,
		message,
		false,
	)
	if err := s.tmpl.ExecuteTemplate(w, "process_content.html", view); err != nil {
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
