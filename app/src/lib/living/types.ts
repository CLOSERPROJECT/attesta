/** Org Domain + Active Config hydrate from GET /api/living/domain. */
export type LoadResult = {
	org: string;
	activeConfigVersion: string;
	domain: Domain;
	config: OrgConfig;
};

export type Domain = {
	load: EntityMap;
	pallet: EntityMap;
	container: EntityMap;
	submissions: Submission[];
	jobInstances: JobInstance[];
};

export type EntityMap = Record<string, Record<string, unknown>>;

export type Submission = {
	jobId: string;
	taskId: string;
	jobInstanceId?: string;
	timestamp: string;
	userId: string;
	facts: Record<string, unknown>;
	idsTouched: string[];
};

export type JobInstance = {
	jobId: string;
	id: string;
	label: string;
};

export type OrgConfig = {
	id: string;
	org?: string;
	version: string;
	domain?: Record<string, unknown>;
	roles?: string[];
	/** Job id → authored Job; key order is authoring order. */
	jobs: Record<string, JobConfig>;
};

/** One authored Task: required unique `id` plus payload fields (roles, schema, …). */
export type TaskConfig = {
	id: string;
	[key: string]: unknown;
};

/** Job chrome + ordered Task array (Unlock order = array order). */
export type JobConfig = {
	repeatable: boolean;
	instanceId?: string;
	label: string;
	startLabel?: string;
	startsWhen?: {
		task: { jobId: string; taskId: string };
		quantifier: string;
	} | null;
	tasks: TaskConfig[];
};

export function taskPath(jobId: string, taskId: string): string {
	return `${jobId}/${taskId}`;
}

export function emptyDomain(): Domain {
	return {
		load: {},
		pallet: {},
		container: {},
		submissions: [],
		jobInstances: []
	};
}

export function orderedJobIds(config: OrgConfig | null | undefined): string[] {
	if (!config?.jobs) return [];
	return Object.keys(config.jobs);
}

export function jobInstancesFor(
	domain: Domain | null | undefined,
	jobId: string
): JobInstance[] {
	if (!domain?.jobInstances?.length) return [];
	return domain.jobInstances.filter((row) => row.jobId === jobId);
}
