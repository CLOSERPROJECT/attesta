package main

import (
	"strings"
	"time"
)

// StreamCardView is the view model for templates/components/stream_card.html.
type StreamCardView struct {
	Key               string
	Name              string
	Description       string
	Counts            WorkflowProcessCounts
	HasUserTurn       bool
	CanClone          bool
	CanEdit           bool
	EditAction        string
	EditRequiresPurge bool
	CanDelete         bool
	DeleteAction      string
}

// PublicStreamCardOrgView is one participating organization avatar on a public stream card.
type PublicStreamCardOrgView struct {
	Name     string
	LogoURL  string
	Initials string
}

// ManagedPublicStreamCardView wraps a public stream card with optional /my management actions.
type ManagedPublicStreamCardView struct {
	Card              PublicStreamCardView
	Key               string
	CanClone          bool
	CanEdit           bool
	EditAction        string
	EditRequiresPurge bool
	CanDelete         bool
	DeleteAction      string
}

// MyHomeStreamGroupView is one taxonomy (or Uncategorized) block on /my.
// CategoryName/CategoryIconURL are always set; ShowCategoryHeader is true only
// on the first non-empty subcategory under a category so the h2 is not repeated.
type MyHomeStreamGroupView struct {
	CategoryName           string
	CategoryIconURL        string
	CategorySlug           string
	SubCategorySlug        string
	SubCategoryName        string
	SubCategoryIconURL     string
	SubCategoryDescription string
	AnchorID               string
	ShowCategoryHeader     bool
	Uncategorized          bool
	Streams                []ManagedPublicStreamCardView
}

// PublicStreamCardView is the view model for templates/components/public_stream_card.html.
type PublicStreamCardView struct {
	Name                  string
	Description           string
	Href                  string // /streams/:key when set; empty keeps a non-link article
	PassportEnabled       bool
	InstanceCount         int
	ActiveCount           int  // active instances (dashboard Active: not done, not terminated)
	AllCompleted          bool // true when T>=1 and no active instances (settled)
	StepCount             int
	RoleCount             int
	OrganizationCount     int // total orgs before avatar truncation
	Organizations         []PublicStreamCardOrgView
	OrganizationsOverflow int // count beyond the first four avatars; 0 when none
}

// PublicStreamRunView is one recent completed DPP run on the public stream page.
type PublicStreamRunView struct {
	CompletedAt string
	DigitalLink string
}

// PublicStreamPageView is the view model for templates/pages/public_stream.html.
type PublicStreamPageView struct {
	PageBase
	HomeHref      string // /?category=…&subCategory=… or /
	Header        PublicStreamCardView
	Organizations []PublicStreamCardOrgView // step-order unique orgs; full list
	RecentRuns    []PublicStreamRunView
	Blueprint     StreamInstanceDetailView
}

// CategorySidebarLeafView is one subcategory row in category_sidebar.
// If Href is set, render an anchor; otherwise render an HTMX button using PartialURL/PushURL.
type CategorySidebarLeafView struct {
	Slug       string
	Name       string
	IconURL    string
	Active     bool
	Href       string // /my anchors, e.g. #cat-supply-chain--procurement
	PartialURL string // public home HTMX get
	PushURL    string // public home hx-push-url
}

// CategorySidebarCategoryView is one accordion category in category_sidebar.
type CategorySidebarCategoryView struct {
	Slug          string
	Name          string
	IconURL       string
	Expanded      bool
	SubCategories []CategorySidebarLeafView
}

// CategorySidebarView is the root for templates/components/category_sidebar.html.
type CategorySidebarView struct {
	Title         string
	CloseTargetID string // popover id for mobile drawer close; empty = no close control
	Categories    []CategorySidebarCategoryView
}

// PublicHomeStreamResultsView is the HTMX swap target for landing stream cards.
type PublicHomeStreamResultsView struct {
	Streams                []PublicStreamCardView
	CreateStreamHref       string
	CategoryName           string
	CategoryIconURL        string
	SubCategoryName        string
	SubCategoryIconURL     string
	SubCategoryDescription string
}

// WorkflowProcessCounts holds process status totals shown on a stream card.
type WorkflowProcessCounts struct {
	NotStarted int
	Started    int
	Terminated int
}

// StreamInstanceCard is the view model for templates/components/stream_instance_card.html.
type StreamInstanceCard struct {
	ID                 string
	Name               string
	Status             string
	StatusLabel        string
	DetailHref         string
	CreatedAt          string
	CreatedAtISO       string
	CreatedAtTime      time.Time
	DoneSubsteps       int
	TotalSubsteps      int
	Percent            int
	LastNotarizedAt    string
	LastNotarizedAtISO string
	LastDigestShort    string
}

// SubstepRoleBadge is a role pill on a substep body (preview/result modes).
type SubstepRoleBadge struct {
	ID      string
	Label   string
	Palette string
}

// SubstepRoleOption is a selectable role in termination / multi-role forms.
type SubstepRoleOption struct {
	Slug  string
	Label string
}

// SubstepKV is a flattened submitted value row on a completed substep body.
type SubstepKV struct {
	Key   string
	Value string
}

// SubstepAttachmentView is a file attachment on a substep body.
type SubstepAttachmentView struct {
	AttachmentID string
	Key          string
	Filename     string
	URL          string
	PreviewURL   string
	PreviewKind  string
	SHA256       string
}

// StepSummaryView is the view model for templates/components/stream_step_summary.html.
type StepSummaryView struct {
	StepID           string
	Title            string
	OrganizationName string
	OrgLogoURL       string
	HideOrgMark      bool
	CompletedAt      string
	CompletedAtHuman string
	SubstepCount     int
}

// SubstepBodyMode is the render mode for templates/components/substep_body.html.
type SubstepBodyMode string

const (
	SubstepBodyModePreview    SubstepBodyMode = "preview"
	SubstepBodyModeActionable SubstepBodyMode = "actionable"
	SubstepBodyModeResult     SubstepBodyMode = "result"
	SubstepBodyModeMessage    SubstepBodyMode = "message"
)

// SubstepBodyView is the view model for templates/components/substep_body.html.
type SubstepBodyView struct {
	WorkflowKey    string
	ProcessID      string
	SubstepID      string
	Title          string
	Description    string
	Role           string
	RoleBadges     []SubstepRoleBadge
	MatchingRoles  []SubstepRoleOption
	RoleLabel      string
	Palette        string
	InputKey       string
	InputType      string
	FormSchema     string
	FormUISchema   string
	Status         string
	Mode           SubstepBodyMode
	DoneAt         string
	DoneAtISO      string
	DoneBy         string
	DoneRole       string
	Values         []SubstepKV
	Attachments    []SubstepAttachmentView
	Disabled       bool
	ReadOnly       bool
	Reason         string
	DetailMessage  string
	CanAdaptForm   bool
	AdaptURL       string
	FormataArchURL string
	OverrideReason string
	HasOverride    bool
	Digest         string
}

func resolveSubstepBodyMode(v SubstepBodyView) SubstepBodyMode {
	if strings.TrimSpace(v.DetailMessage) != "" {
		return SubstepBodyModeMessage
	}
	if v.Status == "done" {
		return SubstepBodyModeResult
	}
	if v.Status == "available" && !v.Disabled && !v.ReadOnly {
		return SubstepBodyModeActionable
	}
	return SubstepBodyModePreview
}

func effectiveSubstepBodyMode(v SubstepBodyView) SubstepBodyMode {
	if v.Mode != "" {
		return v.Mode
	}
	return resolveSubstepBodyMode(v)
}

// SubstepShellDisplay is resolved shell chrome for substep_shell (body-first, summary fallback).
type SubstepShellDisplay struct {
	Status      string
	StatusLabel string
	Palette     string
	DoneAt      string
	DoneAtISO   string
	DoneBy      string
}

func substepShellDisplay(sub TimelineSubstep) SubstepShellDisplay {
	if sub.Body != nil {
		status := sub.Body.Status
		if status == "available" && sub.Body.Disabled {
			status = "active"
		}
		return SubstepShellDisplay{
			Status:      status,
			StatusLabel: processStatusLabel(status),
			Palette:     sub.Body.Palette,
			DoneAt:      sub.Body.DoneAt,
			DoneAtISO:   sub.Body.DoneAtISO,
			DoneBy:      sub.Body.DoneBy,
		}
	}
	label := sub.StatusLabel
	if label == "" {
		label = processStatusLabel(sub.Status)
	}
	return SubstepShellDisplay{
		Status:      sub.Status,
		StatusLabel: label,
		Palette:     sub.Palette,
		DoneAt:      sub.DoneAt,
		DoneAtISO:   sub.DoneAtISO,
		DoneBy:      sub.DoneBy,
	}
}

// TimelineSubstep is one row in the stream timeline accordion.
// Shell chrome reads from Body via substepShellDisplay; summary Status/Done* fields
// remain for nil-body fallbacks and legacy builders until fully removed.
type TimelineSubstep struct {
	SubstepID   string
	Title       string
	Selected    bool
	Body        *SubstepBodyView
	Palette     string
	Status      string
	StatusLabel string
	DoneBy      string
	DoneRole    string
	DoneAt      string
	DoneAtISO   string
}

// TimelineStep groups substeps under a blueprint step in the stream timeline.
type TimelineStep struct {
	Summary  StepSummaryView
	OrgSlug  string
	Expanded bool
	Substeps []TimelineSubstep
}

// StreamTimelineView is the view model for templates/components/stream_timeline.html.
type StreamTimelineView struct {
	Timeline   []TimelineStep
	HideStatus bool
}

// StreamTimelineStepView wraps one timeline step for stream_timeline_step.
type StreamTimelineStepView struct {
	Step       TimelineStep
	HideStatus bool
}

// StreamTimelineSubstepView wraps one timeline substep for substep_shell.
type StreamTimelineSubstepView struct {
	Substep    TimelineSubstep
	HideStatus bool
}

// StreamTerminationDetailsView is the view model for templates/components/stream_termination_details.html.
type StreamTerminationDetailsView struct {
	EndedAt      string
	EndedAtHuman string
	EndedBy      string
	SubstepID    string
	Reason       string
}

// StreamInstanceDetailView is the HTMX/SSE partial payload for stream instance detail content.
type StreamInstanceDetailView struct {
	WorkflowKey       string
	WorkflowPath      string
	ProcessID         string
	CurrentUser       Actor
	SelectedSubstepID string
	ProcessDone       bool
	SelectedBody      *SubstepBodyView
	Error             string
	Timeline          []TimelineStep
	HideStatus        bool
	DPPURL            string
	DPPGS1            string
	Attachments       []ProcessDownloadAttachment
	CanTerminate      bool
	TerminateAction   string
	TerminateSubstep  string
	TerminateRoles    []SubstepRoleOption
	Termination       *StreamTerminationDetailsView
}

func (v StreamInstanceDetailView) StreamTimeline() StreamTimelineView {
	return StreamTimelineView{
		Timeline:   v.Timeline,
		HideStatus: v.HideStatus,
	}
}

// BreadcrumbItem is one crumb in templates/components/breadcrumbs.html.
type BreadcrumbItem struct {
	Label   string
	Href    string
	Current bool // true => last crumb; still a link, with aria-current="page"
}

// BreadcrumbsView is the view model for templates/components/breadcrumbs.html.
type BreadcrumbsView struct {
	Items []BreadcrumbItem
}

// AdminConsoleNavItem is one soft-nav link in templates/components/admin_console.html.
type AdminConsoleNavItem struct {
	Href   string
	Title  string
	Copy   string
	Active bool
}

// AdminConsoleView is the view model for templates/components/admin_console.html.
// MainTemplate is executed via the render func with MainData as the dot.
type AdminConsoleView struct {
	ID           string // default "admin-console" when empty
	NavLabel     string
	Title        string
	Subtitle     string
	Breadcrumbs  BreadcrumbsView
	NavItems     []AdminConsoleNavItem
	MainTemplate string
	MainData     any
}
