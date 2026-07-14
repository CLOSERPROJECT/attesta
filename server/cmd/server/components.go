package main

// PageHeaderView is the view model for templates/components/page_header.html.
type PageHeaderView struct {
	BackHref    string
	BackLabel   string
	Title       string
	Subtitle    string
	Description string
	Meta        string
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
	DoneAt         string
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

// TimelineSubstep is one row in the stream timeline accordion (summary + optional body).
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

// StreamTimelineSubstepView wraps one timeline substep for stream_timeline_substep.
type StreamTimelineSubstepView struct {
	Substep    TimelineSubstep
	HideStatus bool
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
	Termination       *ProcessTerminationView
}

func (v StreamInstanceDetailView) StreamTimeline() StreamTimelineView {
	return StreamTimelineView{
		Timeline:   v.Timeline,
		HideStatus: v.HideStatus,
	}
}
