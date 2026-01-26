package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type WorkflowDef struct {
	ID    primitive.ObjectID `bson:"_id,omitempty"`
	Name  string             `bson:"name"`
	Steps []WorkflowStep     `bson:"steps"`
}

type WorkflowStep struct {
	StepID  string        `bson:"stepId"`
	Title   string        `bson:"title"`
	Order   int           `bson:"order"`
	Substep []WorkflowSub `bson:"substeps"`
}

type WorkflowSub struct {
	SubstepID string `bson:"substepId"`
	Title     string `bson:"title"`
	Order     int    `bson:"order"`
	Role      string `bson:"role"`
	InputKey  string `bson:"inputKey"`
	InputType string `bson:"inputType"`
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

type Server struct {
	mongo         *mongo.Client
	db            *mongo.Database
	tmpl          *template.Template
	cerbosURL     string
	sse           *SSEHub
	workflowDef   WorkflowDef
	workflowDefID primitive.ObjectID
}

type SSEHub struct {
	mu     sync.Mutex
	stream map[string]map[chan string]struct{}
}

type TimelineSubstep struct {
	SubstepID string
	Title     string
	Role      string
	RoleLabel string
	Status    string
	DoneBy    string
	DoneRole  string
	DoneAt    string
	Data      map[string]interface{}
}

type TimelineStep struct {
	StepID   string
	Title    string
	Substeps []TimelineSubstep
}

type ActionView struct {
	ProcessID string
	SubstepID string
	Title     string
	Role      string
	InputKey  string
	InputType string
	Status    string
	Disabled  bool
	Reason    string
}

type ActionTodo struct {
	ProcessID string
	SubstepID string
	Title     string
	Status    string
}

type ProcessSummary struct {
	ID          string
	Status      string
	CreatedAt   string
	NextSubstep string
	NextTitle   string
	NextRole    string
}

type BackofficeLandingView struct {
	Body string
}

type DepartmentDashboardView struct {
	Body            string
	CurrentUser     Actor
	RoleLabel       string
	TodoActions     []ActionTodo
	ActiveProcesses []ProcessSummary
	DoneProcesses   []ProcessSummary
}

type DepartmentProcessView struct {
	Body        string
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

type HomeView struct {
	Body            string
	LatestProcessID string
}

type ProcessPageView struct {
	Body      string
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
	workflowDef, err := ensureWorkflowDef(ctx, db)
	if err != nil {
		log.Fatal(err)
	}

	tmpl, err := template.ParseGlob("templates/*.html")
	if err != nil {
		log.Fatal(err)
	}

	server := &Server{
		mongo:         client,
		db:            db,
		tmpl:          tmpl,
		cerbosURL:     envOr("CERBOS_URL", "http://localhost:3592"),
		sse:           newSSEHub(),
		workflowDef:   workflowDef,
		workflowDefID: workflowDef.ID,
	}

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("../web/dist"))))
	mux.HandleFunc("/", server.handleHome)
	mux.HandleFunc("/process/start", server.handleStartProcess)
	mux.HandleFunc("/process/", server.handleProcessRoutes)
	mux.HandleFunc("/backoffice", server.handleBackoffice)
	mux.HandleFunc("/backoffice/", server.handleBackoffice)
	mux.HandleFunc("/impersonate", server.handleImpersonate)
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

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func ensureWorkflowDef(ctx context.Context, db *mongo.Database) (WorkflowDef, error) {
	collection := db.Collection("workflow_defs")
	var existing WorkflowDef
	if err := collection.FindOne(ctx, bson.M{"name": "Gallium Recycling Notarization"}).Decode(&existing); err == nil {
		return existing, nil
	}

	seed := WorkflowDef{
		Name: "Gallium Recycling Notarization",
		Steps: []WorkflowStep{
			{
				StepID: "1",
				Title:  "Incoming intake",
				Order:  1,
				Substep: []WorkflowSub{
					{
						SubstepID: "1.1",
						Title:     "Record incoming gallium (kg)",
						Order:     1,
						Role:      "dep1",
						InputKey:  "incomingKg",
						InputType: "number",
					},
					{
						SubstepID: "1.2",
						Title:     "Attach batch ID",
						Order:     2,
						Role:      "dep1",
						InputKey:  "batchId",
						InputType: "string",
					},
				},
			},
			{
				StepID: "2",
				Title:  "Refinement",
				Order:  2,
				Substep: []WorkflowSub{
					{
						SubstepID: "2.1",
						Title:     "Record refined gallium output (kg)",
						Order:     1,
						Role:      "dep2",
						InputKey:  "refinedKg",
						InputType: "number",
					},
					{
						SubstepID: "2.2",
						Title:     "Record purity (%)",
						Order:     2,
						Role:      "dep2",
						InputKey:  "purityPct",
						InputType: "number",
					},
				},
			},
			{
				StepID: "3",
				Title:  "QA + Notarize",
				Order:  3,
				Substep: []WorkflowSub{
					{
						SubstepID: "3.1",
						Title:     "QA sign-off",
						Order:     1,
						Role:      "dep3",
						InputKey:  "qaNote",
						InputType: "string",
					},
					{
						SubstepID: "3.2",
						Title:     "Finalize notarization",
						Order:     2,
						Role:      "dep3",
						InputKey:  "finalHash",
						InputType: "string",
					},
				},
			},
		},
	}

	result, err := collection.InsertOne(ctx, seed)
	if err != nil {
		return WorkflowDef{}, err
	}
	seed.ID = result.InsertedID.(primitive.ObjectID)
	return seed, nil
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	ctx := r.Context()
	collection := s.db.Collection("processes")
	opts := options.FindOne().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	var latest Process
	latestID := ""
	if err := collection.FindOne(ctx, bson.M{}, opts).Decode(&latest); err == nil {
		latestID = latest.ID.Hex()
	}

	if err := s.tmpl.ExecuteTemplate(w, "home.html", HomeView{Body: "home_body", LatestProcessID: latestID}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleStartProcess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()
	process := Process{
		WorkflowDefID: s.workflowDefID,
		CreatedAt:     time.Now().UTC(),
		CreatedBy:     "demo",
		Status:        "active",
		Progress:      map[string]ProcessStep{},
	}
	for _, step := range sortedSteps(s.workflowDef) {
		for _, sub := range sortedSubsteps(step) {
			process.Progress[encodeProgressKey(sub.SubstepID)] = ProcessStep{State: "pending"}
		}
	}
	result, err := s.db.Collection("processes").InsertOne(ctx, process)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	id := result.InsertedID.(primitive.ObjectID)
	for _, role := range []string{"dep1", "dep2", "dep3"} {
		s.sse.Broadcast("role:"+role, "role-updated")
	}
	http.Redirect(w, r, fmt.Sprintf("/process/%s", id.Hex()), http.StatusSeeOther)
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
	if len(parts) == 2 && parts[1] == "timeline" && r.Method == http.MethodGet {
		s.handleTimelinePartial(w, r, processID)
		return
	}
	if len(parts) == 4 && parts[1] == "substep" && parts[3] == "complete" && r.Method == http.MethodPost {
		s.handleCompleteSubstep(w, r, processID, parts[2])
		return
	}
	http.NotFound(w, r)
}

func (s *Server) handleProcessPage(w http.ResponseWriter, r *http.Request, processID string) {
	ctx := r.Context()
	process, err := s.loadProcess(ctx, processID)
	if err != nil {
		http.Error(w, "process not found", http.StatusNotFound)
		return
	}
	timeline := buildTimeline(s.workflowDef, process)
	view := ProcessPageView{Body: "process_body", ProcessID: process.ID.Hex(), Timeline: timeline}
	if err := s.tmpl.ExecuteTemplate(w, "process.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleTimelinePartial(w http.ResponseWriter, r *http.Request, processID string) {
	ctx := r.Context()
	process, err := s.loadProcess(ctx, processID)
	if err != nil {
		http.Error(w, "process not found", http.StatusNotFound)
		return
	}
	timeline := buildTimeline(s.workflowDef, process)
	if err := s.tmpl.ExecuteTemplate(w, "timeline.html", timeline); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleBackoffice(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/backoffice")
	path = strings.Trim(path, "/")
	if path == "" {
		view := BackofficeLandingView{Body: "backoffice_landing_body"}
		if err := s.tmpl.ExecuteTemplate(w, "backoffice_landing.html", view); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	parts := strings.Split(path, "/")
	role := parts[0]
	if !isKnownRole(role) {
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
	cookie := &http.Cookie{
		Name:  "demo_user",
		Value: fmt.Sprintf("%s|%s", userID, role),
		Path:  "/",
	}
	http.SetCookie(w, cookie)
	http.Redirect(w, r, fmt.Sprintf("/backoffice/%s", role), http.StatusSeeOther)
}

func (s *Server) handleCompleteSubstep(w http.ResponseWriter, r *http.Request, processID, substepID string) {
	actor := readActor(r)
	if actor.UserID == "" {
		actor = Actor{UserID: "u1", Role: "dep1"}
	}

	ctx := r.Context()
	process, err := s.loadProcess(ctx, processID)
	if err != nil {
		s.renderActionErrorForRequest(w, r, http.StatusNotFound, "Process not found.", process, actor)
		return
	}

	substep, step, err := findSubstep(s.workflowDef, substepID)
	if err != nil {
		s.renderActionErrorForRequest(w, r, http.StatusNotFound, "Substep not found.", process, actor)
		return
	}

	sequenceOK := isSequenceOK(s.workflowDef, process, substepID)
	allowed, err := s.checkCerbos(r.Context(), actor, processID, substep, step.Order, sequenceOK)
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

	if err := r.ParseForm(); err != nil {
		s.renderActionErrorForRequest(w, r, http.StatusBadRequest, "Invalid form.", process, actor)
		return
	}
	value := strings.TrimSpace(r.FormValue("value"))
	if value == "" {
		s.renderActionErrorForRequest(w, r, http.StatusBadRequest, "Value is required.", process, actor)
		return
	}
	payload, err := normalizePayload(substep, value)
	if err != nil {
		s.renderActionErrorForRequest(w, r, http.StatusBadRequest, err.Error(), process, actor)
		return
	}

	now := time.Now().UTC()
	progressUpdate := ProcessStep{
		State:  "done",
		DoneAt: &now,
		DoneBy: &actor,
		Data:   payload,
	}

	update := bson.M{
		"$set": bson.M{
			"progress." + encodeProgressKey(substepID): progressUpdate,
		},
	}

	if err := s.db.Collection("processes").FindOneAndUpdate(ctx, bson.M{"_id": process.ID}, update).Err(); err != nil {
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
	if _, err := s.db.Collection("notarizations").InsertOne(ctx, notary); err != nil {
		s.renderActionErrorForRequest(w, r, http.StatusInternalServerError, "Failed to notarize payload.", process, actor)
		return
	}

	process, _ = s.loadProcess(ctx, processID)
	if process != nil && isProcessDone(s.workflowDef, process) {
		_, _ = s.db.Collection("processes").UpdateOne(ctx, bson.M{"_id": process.ID}, bson.M{"$set": bson.M{"status": "done"}})
	}

	s.sse.Broadcast(processID, "process-updated")
	for _, role := range []string{"dep1", "dep2", "dep3"} {
		s.sse.Broadcast("role:"+role, "role-updated")
	}
	if isHTMXRequest(r) {
		s.renderActionList(w, process, actor, "")
		return
	}
	s.renderDepartmentProcessPage(w, process, actor, "")
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	processID := r.URL.Query().Get("processId")
	role := r.URL.Query().Get("role")
	if processID == "" && role == "" {
		http.Error(w, "processId or role required", http.StatusBadRequest)
		return
	}
	if role != "" && !isKnownRole(role) {
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
	actor := readActor(r)
	if actor.Role != role || actor.UserID == "" {
		actor = actorForRole(role)
		cookie := &http.Cookie{
			Name:  "demo_user",
			Value: fmt.Sprintf("%s|%s", actor.UserID, actor.Role),
			Path:  "/",
		}
		http.SetCookie(w, cookie)
	}

	ctx := r.Context()
	todoActions, activeProcesses, doneProcesses := s.loadProcessDashboard(ctx, role)
	view := DepartmentDashboardView{
		Body:            "dept_dashboard_body",
		CurrentUser:     actor,
		RoleLabel:       roleLabel(role),
		TodoActions:     todoActions,
		ActiveProcesses: activeProcesses,
		DoneProcesses:   doneProcesses,
	}
	if err := s.tmpl.ExecuteTemplate(w, "backoffice_department.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleDepartmentProcess(w http.ResponseWriter, r *http.Request, role, processID string) {
	actor := readActor(r)
	if actor.Role != role || actor.UserID == "" {
		actor = actorForRole(role)
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
	actions := buildActionList(s.workflowDef, process, actor, true)
	view := DepartmentProcessView{
		Body:        "dept_process_body",
		CurrentUser: actor,
		RoleLabel:   roleLabel(role),
		ProcessID:   process.ID.Hex(),
		Actions:     actions,
	}
	if err := s.tmpl.ExecuteTemplate(w, "backoffice_process.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleDepartmentDashboardPartial(w http.ResponseWriter, r *http.Request, role string) {
	actor := readActor(r)
	if actor.Role != role || actor.UserID == "" {
		actor = actorForRole(role)
	}

	ctx := r.Context()
	todoActions, activeProcesses, doneProcesses := s.loadProcessDashboard(ctx, role)
	view := DepartmentDashboardView{
		CurrentUser:     actor,
		RoleLabel:       roleLabel(role),
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
	var process Process
	if err := s.db.Collection("processes").FindOne(ctx, bson.M{"_id": objectID}).Decode(&process); err != nil {
		return nil, err
	}
	process.Progress = normalizeProgressKeys(process.Progress)
	return &process, nil
}

func (s *Server) loadLatestProcess(ctx context.Context) (*Process, error) {
	opts := options.FindOne().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	var process Process
	if err := s.db.Collection("processes").FindOne(ctx, bson.M{}, opts).Decode(&process); err != nil {
		return nil, err
	}
	process.Progress = normalizeProgressKeys(process.Progress)
	return &process, nil
}

func (s *Server) loadProcessDashboard(ctx context.Context, role string) ([]ActionTodo, []ProcessSummary, []ProcessSummary) {
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(25)
	cursor, err := s.db.Collection("processes").Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, nil, nil
	}
	defer cursor.Close(ctx)

	var todo []ActionTodo
	var active []ProcessSummary
	var done []ProcessSummary
	for cursor.Next(ctx) {
		var process Process
		if err := cursor.Decode(&process); err != nil {
			continue
		}
		process.Progress = normalizeProgressKeys(process.Progress)
		status := process.Status
		if status == "" {
			status = "active"
		}
		if status != "done" && isProcessDone(s.workflowDef, &process) {
			status = "done"
		}
		summary := buildProcessSummaryForRole(s.workflowDef, &process, status, role)
		if status == "done" {
			done = append(done, summary)
		} else {
			todo = append(todo, buildRoleTodos(s.workflowDef, &process, role)...)
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

func buildTimeline(def WorkflowDef, process *Process) []TimelineStep {
	steps := sortedSteps(def)
	availableMap := computeAvailability(def, process)

	var timeline []TimelineStep
	for _, step := range steps {
		row := TimelineStep{StepID: step.StepID, Title: step.Title}
		for _, sub := range sortedSubsteps(step) {
			entry := TimelineSubstep{
				SubstepID: sub.SubstepID,
				Title:     sub.Title,
				Role:      sub.Role,
				RoleLabel: roleLabel(sub.Role),
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
					entry.Data = progress.Data
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

func buildActionList(def WorkflowDef, process *Process, actor Actor, onlyRole bool) []ActionView {
	var actions []ActionView
	ordered := orderedSubsteps(def)
	availMap := computeAvailability(def, process)
	for _, sub := range ordered {
		if onlyRole && sub.Role != actor.Role {
			continue
		}
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
			ProcessID: processIDString(process),
			SubstepID: sub.SubstepID,
			Title:     sub.Title,
			Role:      sub.Role,
			InputKey:  sub.InputKey,
			InputType: sub.InputType,
			Status:    status,
			Disabled:  disabled,
			Reason:    reason,
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

func roleLabel(role string) string {
	if strings.HasPrefix(role, "dep") && len(role) > 3 {
		return "Department " + strings.TrimPrefix(role, "dep")
	}
	return role
}

func isHTMXRequest(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("HX-Request"), "true")
}

func actorForRole(role string) Actor {
	switch role {
	case "dep2":
		return Actor{UserID: "u2", Role: "dep2"}
	case "dep3":
		return Actor{UserID: "u3", Role: "dep3"}
	default:
		return Actor{UserID: "u1", Role: "dep1"}
	}
}

func isKnownRole(role string) bool {
	return role == "dep1" || role == "dep2" || role == "dep3"
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

func (s *Server) checkCerbos(ctx context.Context, actor Actor, processID string, sub WorkflowSub, stepOrder int, sequenceOK bool) (bool, error) {
	request := map[string]interface{}{
		"requestId": fmt.Sprintf("req-%d", time.Now().UnixNano()),
		"principal": map[string]interface{}{
			"id":    actor.UserID,
			"roles": []string{actor.Role},
		},
		"resource": map[string]interface{}{
			"kind": "substep",
			"instances": map[string]interface{}{
				sub.SubstepID: map[string]interface{}{
					"attr": map[string]interface{}{
						"roleRequired": sub.Role,
						"stepOrder":    stepOrder,
						"substepOrder": sub.Order,
						"substepId":    sub.SubstepID,
						"processId":    processID,
						"sequenceOk":   sequenceOK,
					},
				},
			},
		},
		"actions": []string{"complete"},
	}

	body, _ := json.Marshal(request)
	endpoint := strings.TrimSuffix(s.cerbosURL, "/") + "/api/check"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(body)))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("cerbos status %d", resp.StatusCode)
	}
	var result struct {
		ResourceInstances map[string]struct {
			Actions map[string]string `json:"actions"`
		} `json:"resourceInstances"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}
	if res, ok := result.ResourceInstances[sub.SubstepID]; ok {
		if effect, ok := res.Actions["complete"]; ok {
			return strings.EqualFold(effect, "EFFECT_ALLOW"), nil
		}
	}
	return false, nil
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
	var timeline []TimelineStep
	if process != nil {
		timeline = buildTimeline(s.workflowDef, process)
	}
	view := ActionListView{
		ProcessID:   processIDString(process),
		CurrentUser: actor,
		Actions:     buildActionList(s.workflowDef, process, actor, true),
		Error:       message,
		Timeline:    timeline,
	}
	if err := s.tmpl.ExecuteTemplate(w, "action_list.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) renderDepartmentProcessPage(w http.ResponseWriter, process *Process, actor Actor, message string) {
	processID := ""
	if process != nil {
		processID = process.ID.Hex()
	}
	view := DepartmentProcessView{
		Body:        "dept_process_body",
		CurrentUser: actor,
		RoleLabel:   roleLabel(actor.Role),
		ProcessID:   processID,
		Actions:     buildActionList(s.workflowDef, process, actor, true),
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
