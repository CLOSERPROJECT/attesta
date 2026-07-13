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
}
