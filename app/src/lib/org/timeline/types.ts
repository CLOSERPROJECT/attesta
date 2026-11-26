/** GET /api/org/timeline — enriched Org instance timeline (server projection). */

export type TimelineSubmission = {
	timestamp?: string;
	timestampHuman?: string;
	userId?: string;
};

/** Allowed-role pill on a timeline substep (Config RoleDefs / Identity catalog). */
export type TimelineRoleBadge = {
	id: string;
	label: string;
	palette: string;
};

export type TimelineSubstep = {
	substepId: string;
	title: string;
	status: string;
	statusLabel?: string;
	orgActionable?: boolean;
	/** Operator has a Submission on this done Task — stays visible when Show all is off. */
	doneByMe?: boolean;
	submitMany?: boolean;
	blockedReason?: string;
	href?: string;
	roleBadges?: TimelineRoleBadge[];
	submissions?: TimelineSubmission[];
};

export type TimelineRow = {
	id: string;
	label: string;
	expanded: boolean;
	status?: string;
	statusLabel?: string;
	substeps: TimelineSubstep[];
};

export type TimelineStep = {
	stepId: string;
	title: string;
	substepCount: number;
	expanded: boolean;
	status?: string;
	statusLabel?: string;
	substeps?: TimelineSubstep[];
	rows?: TimelineRow[];
};

export type TimelineMode = 'boot' | 'instance';

export type TimelineView = {
	stream: string;
	instance: string;
	mode?: TimelineMode;
	label: string;
	status: string;
	steps: TimelineStep[];
};
