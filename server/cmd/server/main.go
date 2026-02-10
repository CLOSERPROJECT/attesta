package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
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
	ID    primitive.ObjectID `bson:"_id,omitempty" yaml:"-"`
	Name  string             `bson:"name" yaml:"name"`
	Steps []WorkflowStep     `bson:"steps" yaml:"steps"`
}

type WorkflowStep struct {
	StepID  string        `bson:"stepId" yaml:"id"`
	Title   string        `bson:"title" yaml:"title"`
	Order   int           `bson:"order" yaml:"order"`
	Substep []WorkflowSub `bson:"substeps" yaml:"substeps"`
}

type WorkflowSub struct {
	SubstepID string `bson:"substepId" yaml:"id"`
	Title     string `bson:"title" yaml:"title"`
	Order     int    `bson:"order" yaml:"order"`
	Role      string `bson:"role" yaml:"role"`
	InputKey  string `bson:"inputKey" yaml:"inputKey"`
	InputType string `bson:"inputType" yaml:"inputType"`
}

type Process struct {
	ID            primitive.ObjectID     `bson:"_id,omitempty"`
	WorkflowDefID primitive.ObjectID     `bson:"workflowDefId"`
	CreatedAt     time.Time              `bson:"createdAt"`
	CreatedBy     string                 `bson:"createdBy"`
	Status        string                 `bson:"status"`
	Progress      map[string]ProcessStep `bson:"progress"`
}

type ProcessStep struct {
	State  string                 `bson:"state"`
	DoneAt *time.Time             `bson:"doneAt,omitempty"`
	DoneBy *Actor                 `bson:"doneBy,omitempty"`
	Data   map[string]interface{} `bson:"data,omitempty"`
}

type Actor struct {
	UserID string `bson:"userId"`
	Role   string `bson:"role"`
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
}

type SSEHub struct {
	mu     sync.Mutex
	stream map[string]map[chan string]struct{}
}

type TimelineSubstep struct {
	SubstepID    string
	Title        string
	Role         string
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
	ProcessID  string
	SubstepID  string
	Title      string
	Role       string
	RoleLabel  string
	RoleColor  string
	RoleBorder string
	InputKey   string
	InputType  string
	Status     string
	Disabled   bool
	Reason     string
}

type ActionTodo struct {
	ProcessID string
	SubstepID string
	Title     string
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
	Workflow    WorkflowDef  `yaml:"workflow"`
	Departments []Department `yaml:"departments"`
	Users       []User       `yaml:"users"`
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
}

type BackofficeLandingView struct {
	PageBase
	Users []UserView
}

type DepartmentDashboardView struct {
	PageBase
	CurrentUser     Actor
	RoleLabel       string
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

type ProcessPageView struct {
	PageBase
	ProcessID string
	Timeline  []TimelineStep
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
	}

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("../web/dist"))))
	mux.HandleFunc("/", server.handleHome)
	mux.HandleFunc("/process/start", server.handleLegacyStartProcess)
	mux.HandleFunc("/process/", server.handleLegacyProcessRoutes)
	mux.HandleFunc("/backoffice", server.handleLegacyBackoffice)
	mux.HandleFunc("/backoffice/", server.handleLegacyBackoffice)
	mux.HandleFunc("/impersonate", server.handleLegacyImpersonate)
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

func (s *Server) pageBase(body string) PageBase {
	return PageBase{Body: body, ViteDevServer: s.viteDevServer}
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
	cfg, err := s.runtimeConfig()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctx := r.Context()
	latestID := ""
	if latest, err := s.loadLatestProcess(ctx); err == nil {
		latestID = latest.ID.Hex()
	}
	sortKey := normalizeHomeSortKey(strings.TrimSpace(r.URL.Query().Get("sort")))
	processesRaw, err := s.store.ListRecentProcesses(ctx, 0)
	if err != nil {
		processesRaw = nil
	}

	totalSubsteps := countWorkflowSubsteps(cfg.Workflow)
	var processes []ProcessListItem
	var history []ProcessListItem
	for _, process := range processesRaw {
		process.Progress = normalizeProgressKeys(process.Progress)
		status := strings.TrimSpace(process.Status)
		if status == "" {
			status = "active"
		}
		if status != "done" && isProcessDone(cfg.Workflow, &process) {
			status = "done"
		}
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
		PageBase:        s.pageBase("home_body"),
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
	cfg, err := s.runtimeConfig()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctx := r.Context()
	process := Process{
		WorkflowDefID: s.workflowDefID,
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
		s.sse.Broadcast("role:"+role, "role-updated")
	}
	http.Redirect(w, r, fmt.Sprintf("/process/%s", id.Hex()), http.StatusSeeOther)
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
	if _, err := s.loadProcess(ctx, processID); err != nil {
		return "", false
	}
	return s.defaultWorkflowKey(), true
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

func (s *Server) handleLegacyImpersonate(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		http.Error(w, "workflow context required", http.StatusBadRequest)
		return
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
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
	if len(parts) == 4 && parts[1] == "substep" && parts[3] == "complete" && r.Method == http.MethodPost {
		s.handleCompleteSubstep(w, r, processID, parts[2])
		return
	}
	if len(parts) == 4 && parts[1] == "substep" && parts[3] == "file" && r.Method == http.MethodGet {
		s.handleDownloadSubstepFile(w, r, processID, parts[2])
		return
	}
	http.NotFound(w, r)
}

func (s *Server) handleProcessPage(w http.ResponseWriter, r *http.Request, processID string) {
	cfg, err := s.runtimeConfig()
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
	timeline := buildTimeline(cfg.Workflow, process, s.roleMetaMap(cfg))
	view := ProcessPageView{PageBase: s.pageBase("process_body"), ProcessID: process.ID.Hex(), Timeline: timeline}
	if err := s.tmpl.ExecuteTemplate(w, "process.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleTimelinePartial(w http.ResponseWriter, r *http.Request, processID string) {
	cfg, err := s.runtimeConfig()
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
	timeline := buildTimeline(cfg.Workflow, process, s.roleMetaMap(cfg))
	if err := s.tmpl.ExecuteTemplate(w, "timeline.html", timeline); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleDownloadAllFiles(w http.ResponseWriter, r *http.Request, processID string) {
	cfg, err := s.runtimeConfig()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	process, err := s.loadProcess(r.Context(), processID)
	if err != nil {
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
	cfg, err := s.runtimeConfig()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	process, err := s.loadProcess(r.Context(), processID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	export := buildNotarizedExport(cfg.Workflow, process)
	writeJSON(w, export)
}

func (s *Server) handleMerkleJSON(w http.ResponseWriter, r *http.Request, processID string) {
	cfg, err := s.runtimeConfig()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	process, err := s.loadProcess(r.Context(), processID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	export := buildNotarizedExport(cfg.Workflow, process)
	writeJSON(w, export.Merkle)
}

func (s *Server) handleDownloadSubstepFile(w http.ResponseWriter, r *http.Request, processID, substepID string) {
	cfg, err := s.runtimeConfig()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	process, err := s.loadProcess(r.Context(), processID)
	if err != nil {
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

func (s *Server) handleBackoffice(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.runtimeConfig()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/backoffice")
	path = strings.Trim(path, "/")
	if path == "" {
		view := BackofficeLandingView{
			PageBase: s.pageBase("backoffice_landing_body"),
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

func (s *Server) handleImpersonate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	userID := r.FormValue("userId")
	role := r.FormValue("role")
	if userID == "" || role == "" {
		http.Error(w, "invalid impersonation", http.StatusBadRequest)
		return
	}
	if cfg, err := s.runtimeConfig(); err == nil && !s.isKnownRole(cfg, role) {
		http.Error(w, "unknown role", http.StatusBadRequest)
		return
	}
	cookie := &http.Cookie{
		Name:  "demo_user",
		Value: fmt.Sprintf("%s|%s", userID, role),
		Path:  "/",
	}
	http.SetCookie(w, cookie)
	http.Redirect(w, r, fmt.Sprintf("/backoffice/%s", role), http.StatusSeeOther)
}

func (s *Server) handleCompleteSubstep(w http.ResponseWriter, r *http.Request, processID, substepID string) {
	cfg, err := s.runtimeConfig()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	actor := readActor(r)
	if actor.UserID == "" {
		actor = s.actorForRole(cfg, s.defaultRole(cfg))
	}

	ctx := r.Context()
	process, err := s.loadProcess(ctx, processID)
	if err != nil {
		s.renderActionErrorForRequest(w, r, http.StatusNotFound, "Process not found.", process, actor)
		return
	}

	substep, step, err := findSubstep(cfg.Workflow, substepID)
	if err != nil {
		s.renderActionErrorForRequest(w, r, http.StatusNotFound, "Substep not found.", process, actor)
		return
	}

	sequenceOK := isSequenceOK(cfg.Workflow, process, substepID)
	if s.authorizer == nil {
		s.renderActionErrorForRequest(w, r, http.StatusBadGateway, "Cerbos check failed.", process, actor)
		return
	}
	allowed, err := s.authorizer.CanComplete(r.Context(), actor, processID, substep, step.Order, sequenceOK)
	if err != nil {
		s.renderActionErrorForRequest(w, r, http.StatusBadGateway, "Cerbos check failed.", process, actor)
		return
	}
	if !sequenceOK {
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

	if err := s.store.UpdateProcessProgress(ctx, process.ID, substepID, progressUpdate); err != nil {
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
		_ = s.store.UpdateProcessStatus(ctx, process.ID, "done")
	}

	s.sse.Broadcast(processID, "process-updated")
	for _, role := range s.roles(cfg) {
		s.sse.Broadcast("role:"+role, "role-updated")
	}
	if isHTMXRequest(r) {
		s.renderActionList(w, process, actor, "")
		return
	}
	s.renderDepartmentProcessPage(w, process, actor, "")
}

var (
	errInvalidForm   = errors.New("invalid form")
	errValueRequired = errors.New("value required")
	errFileRequired  = errors.New("file required")
)

func (s *Server) parseCompletionPayload(w http.ResponseWriter, r *http.Request, processID primitive.ObjectID, substep WorkflowSub, now time.Time) (map[string]interface{}, error) {
	if substep.InputType != "file" {
		return parseScalarPayload(r, substep)
	}
	return s.parseFilePayload(w, r, processID, substep, now)
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
	cfg, err := s.runtimeConfig()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

	streamKey := processID
	if role != "" {
		streamKey = "role:" + role
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
	cfg, err := s.runtimeConfig()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	actor := readActor(r)
	if actor.Role != role || actor.UserID == "" {
		actor = s.actorForRole(cfg, role)
		cookie := &http.Cookie{
			Name:  "demo_user",
			Value: fmt.Sprintf("%s|%s", actor.UserID, actor.Role),
			Path:  "/",
		}
		http.SetCookie(w, cookie)
	}

	ctx := r.Context()
	todoActions, activeProcesses, doneProcesses := s.loadProcessDashboard(ctx, cfg, role)
	view := DepartmentDashboardView{
		PageBase:        s.pageBase("dept_dashboard_body"),
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
	cfg, err := s.runtimeConfig()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	actor := readActor(r)
	if actor.Role != role || actor.UserID == "" {
		actor = s.actorForRole(cfg, role)
		cookie := &http.Cookie{
			Name:  "demo_user",
			Value: fmt.Sprintf("%s|%s", actor.UserID, actor.Role),
			Path:  "/",
		}
		http.SetCookie(w, cookie)
	}

	ctx := r.Context()
	process, err := s.loadProcess(ctx, processID)
	if err != nil {
		http.Error(w, "process not found", http.StatusNotFound)
		return
	}
	actions := buildActionList(cfg.Workflow, process, actor, true, s.roleMetaMap(cfg))
	view := DepartmentProcessView{
		PageBase:    s.pageBase("dept_process_body"),
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
	cfg, err := s.runtimeConfig()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	actor := readActor(r)
	if actor.Role != role || actor.UserID == "" {
		actor = s.actorForRole(cfg, role)
	}

	ctx := r.Context()
	todoActions, activeProcesses, doneProcesses := s.loadProcessDashboard(ctx, cfg, role)
	view := DepartmentDashboardView{
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

func (s *Server) loadLatestProcess(ctx context.Context) (*Process, error) {
	process, err := s.store.LoadLatestProcess(ctx)
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
		if cfg.Workflow.Name == "" || len(cfg.Workflow.Steps) == 0 {
			return nil, fmt.Errorf("workflow config is empty in %s", filepath.Base(path))
		}
		if normalizeErr := normalizeInputTypes(&cfg.Workflow); normalizeErr != nil {
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

func (s *Server) roleMetaMap(cfg RuntimeConfig) map[string]RoleMeta {
	roles := map[string]RoleMeta{}
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
	for _, dept := range cfg.Departments {
		if dept.ID == role {
			return true
		}
	}
	return false
}

func (s *Server) roles(cfg RuntimeConfig) []string {
	var roles []string
	for _, dept := range cfg.Departments {
		roles = append(roles, dept.ID)
	}
	return roles
}

func (s *Server) actorForRole(cfg RuntimeConfig, role string) Actor {
	for _, user := range cfg.Users {
		if user.DepartmentID == role {
			return Actor{UserID: user.ID, Role: role}
		}
	}
	if role == "" {
		role = "unknown"
	}
	return Actor{UserID: role, Role: role}
}

func (s *Server) defaultRole(cfg RuntimeConfig) string {
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

func (s *Server) loadProcessDashboard(ctx context.Context, cfg RuntimeConfig, role string) ([]ActionTodo, []ProcessSummary, []ProcessSummary) {
	processes, err := s.store.ListRecentProcesses(ctx, 25)
	if err != nil {
		return nil, nil, nil
	}

	var todo []ActionTodo
	var active []ProcessSummary
	var done []ProcessSummary
	for _, process := range processes {
		process.Progress = normalizeProgressKeys(process.Progress)
		status := process.Status
		if status == "" {
			status = "active"
		}
		if status != "done" && isProcessDone(cfg.Workflow, &process) {
			status = "done"
		}
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

func readActor(r *http.Request) Actor {
	cookie, err := r.Cookie("demo_user")
	if err != nil {
		return Actor{}
	}
	parts := strings.Split(cookie.Value, "|")
	if len(parts) != 2 {
		return Actor{}
	}
	return Actor{UserID: parts[0], Role: parts[1]}
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

func buildTimeline(def WorkflowDef, process *Process, roleMeta map[string]RoleMeta) []TimelineStep {
	steps := sortedSteps(def)
	availableMap := computeAvailability(def, process)

	var timeline []TimelineStep
	for _, step := range steps {
		row := TimelineStep{StepID: step.StepID, Title: step.Title}
		for _, sub := range sortedSubsteps(step) {
			meta := roleMetaFor(sub.Role, roleMeta)
			entry := TimelineSubstep{
				SubstepID:  sub.SubstepID,
				Title:      sub.Title,
				Role:       sub.Role,
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
							entry.FileURL = fmt.Sprintf("/process/%s/substep/%s/file", process.ID.Hex(), sub.SubstepID)
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
	for _, sub := range orderedSubsteps(def) {
		if sub.InputType != "file" {
			continue
		}
		progress, ok := process.Progress[sub.SubstepID]
		if !ok || progress.State != "done" {
			continue
		}
		meta := attachmentMetaFromPayload(progress.Data, sub.InputKey)
		if meta == nil || meta.AttachmentID == "" {
			continue
		}
		files = append(files, ProcessAttachmentExport{
			SubstepID:    sub.SubstepID,
			AttachmentID: meta.AttachmentID,
			Filename:     meta.Filename,
			ContentType:  meta.ContentType,
			SizeBytes:    meta.SizeBytes,
			SHA256:       meta.SHA256,
		})
	}
	return files
}

func attachmentMetaFromPayload(data map[string]interface{}, inputKey string) *NotarizedAttachment {
	if data == nil {
		return nil
	}
	raw, ok := data[inputKey]
	if !ok {
		return nil
	}
	payload, ok := raw.(map[string]interface{})
	if !ok {
		return nil
	}
	meta := &NotarizedAttachment{}
	if value, ok := payload["attachmentId"].(string); ok {
		meta.AttachmentID = value
	}
	if value, ok := payload["filename"].(string); ok {
		meta.Filename = value
	}
	if value, ok := payload["contentType"].(string); ok {
		meta.ContentType = value
	}
	if value, ok := payload["sha256"].(string); ok {
		meta.SHA256 = value
	}
	if value, ok := payload["size"].(int64); ok {
		meta.SizeBytes = value
	} else if value, ok := payload["size"].(float64); ok {
		meta.SizeBytes = int64(value)
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

func buildActionList(def WorkflowDef, process *Process, actor Actor, onlyRole bool, roleMeta map[string]RoleMeta) []ActionView {
	var actions []ActionView
	ordered := orderedSubsteps(def)
	availMap := computeAvailability(def, process)
	for _, sub := range ordered {
		if onlyRole && sub.Role != actor.Role {
			continue
		}
		meta := roleMetaFor(sub.Role, roleMeta)
		status := "locked"
		if process != nil {
			if step, ok := process.Progress[sub.SubstepID]; ok && step.State == "done" {
				status = "done"
			} else if availMap[sub.SubstepID] {
				status = "available"
			}
		}
		disabled := status != "available" || sub.Role != actor.Role
		reason := ""
		if status == "locked" {
			reason = "Locked by sequence"
		} else if status == "done" {
			reason = "Already completed"
		} else if sub.Role != actor.Role {
			reason = "Role mismatch"
		}
		actions = append(actions, ActionView{
			ProcessID:  processIDString(process),
			SubstepID:  sub.SubstepID,
			Title:      sub.Title,
			Role:       sub.Role,
			RoleLabel:  meta.Label,
			RoleColor:  meta.Color,
			RoleBorder: meta.Border,
			InputKey:   sub.InputKey,
			InputType:  sub.InputType,
			Status:     status,
			Disabled:   disabled,
			Reason:     reason,
		})
	}
	return actions
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
			inputType, err := normalizeInputType(workflow.Steps[stepIndex].Substep[substepIndex].InputType)
			if err != nil {
				substep := workflow.Steps[stepIndex].Substep[substepIndex]
				return fmt.Errorf("invalid inputType for substep %s: %w", substep.SubstepID, err)
			}
			workflow.Steps[stepIndex].Substep[substepIndex].InputType = inputType
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
	default:
		return "", fmt.Errorf("unsupported value %q (allowed: number, string, text, file)", value)
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
	default:
		payload[sub.InputKey] = value
	}
	return payload, nil
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
	s.renderActionList(w, process, actor, message)
}

func (s *Server) renderActionErrorForRequest(w http.ResponseWriter, r *http.Request, status int, message string, process *Process, actor Actor) {
	w.WriteHeader(status)
	if isHTMXRequest(r) {
		s.renderActionList(w, process, actor, message)
		return
	}
	s.renderDepartmentProcessPage(w, process, actor, message)
}

func (s *Server) renderActionList(w http.ResponseWriter, process *Process, actor Actor, message string) {
	cfg, err := s.runtimeConfig()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var timeline []TimelineStep
	if process != nil {
		timeline = buildTimeline(cfg.Workflow, process, s.roleMetaMap(cfg))
	}
	view := ActionListView{
		ProcessID:   processIDString(process),
		CurrentUser: actor,
		Actions:     buildActionList(cfg.Workflow, process, actor, true, s.roleMetaMap(cfg)),
		Error:       message,
		Timeline:    timeline,
	}
	if err := s.tmpl.ExecuteTemplate(w, "action_list.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) renderDepartmentProcessPage(w http.ResponseWriter, process *Process, actor Actor, message string) {
	cfg, err := s.runtimeConfig()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	processID := ""
	if process != nil {
		processID = process.ID.Hex()
	}
	view := DepartmentProcessView{
		PageBase:    s.pageBase("dept_process_body"),
		CurrentUser: actor,
		RoleLabel:   s.roleLabel(cfg, actor.Role),
		ProcessID:   processID,
		Actions:     buildActionList(cfg.Workflow, process, actor, true, s.roleMetaMap(cfg)),
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
