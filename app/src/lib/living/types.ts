import type { TaskFormBundle } from './form/task-form-bundle.js';

export type { TaskFormBundle };

export type LoadResult = {
	org: string;
	activeConfigVersion: string;
	configId: string;
	/** Viewer OperatorIdentity projection for chrome `$operator` (not Cerbos). */
	userId: string;
	displayName: string;
	/** Viewer Config role slugs (OperatorIdentity; board chrome, not Cerbos). */
	roles: string[];
	worklist: Worklist;
	/** Always empty on home Load (ADR-0006); Task mounts use addressed fetch. */
	taskFormBundles: TaskFormBundle[];
};

/** Server-projected operator board (Unlock / startsWhen; done after Submission only). */
export type Worklist = {
	jobs: WorklistJob[];
};

/** Authored submit cardinality on Worklist Task chrome (default once when omit). */
export type TaskSubmit = 'once' | 'many';

export type WorklistTask = {
	id: string;
	title: string;
	/** Authored Config role slugs for board chrome (slug∩roles; not Cerbos). */
	roles: string[];
	/**
	 * once (default) | many — In progress / Completed ignore many (ADR-0008).
	 * Omit treated as once for older wires.
	 */
	submit?: TaskSubmit;
};

export type WorklistJob = {
	jobId: string;
	canStart: boolean;
	repeatable: boolean;
	/** Authored Job display name (chrome; not the instance-label CEL). */
	title?: string;
	startLabel?: string;
	/** Authored CSS color for board Job card left border (chrome). */
	color?: string;
	/** Authored Task id order for chrome iteration (top-level only). */
	taskIds: string[];
	/**
	 * Config-free Task title + roles chrome: top-level in taskIds order, then
	 * nested group-child chrome (lookup by id; not listed in taskIds).
	 */
	tasks: WorklistTask[];
	instances: WorklistInstance[];
};

export type WorklistInstance = {
	id: string;
	label: string;
	/** Viewer-scoped Unlock/Submission status. */
	tasks: Record<string, TaskStatus>;
	/**
	 * Domain Submission N per flat leaf taskId (no group). Omitted when empty.
	 * Overview many-submit chrome reads this without calling recap.
	 */
	submissionCounts?: Record<string, number>;
	/** Present when the Job authors Task groups; omitted for flat Jobs. */
	groups?: Record<string, WorklistGroup>;
};

/** Server-projected Task group chrome (spawn/close); SPA must not re-gate. */
export type WorklistGroup = {
	groupId: string;
	startLabel?: string;
	closeLabel?: string;
	repeatable: boolean;
	canSpawn: boolean;
	canClose: boolean;
	childTaskIds: string[];
	instances: WorklistGroupInstance[];
};

export type WorklistGroupInstance = {
	id: string;
	/** Overview chrome from Config instanceLabelEntity ∩ nested IDsTouched. */
	label?: string;
	tasks: Record<string, TaskStatus>;
	/** Domain Submission N per child leaf taskId on this group instance. */
	submissionCounts?: Record<string, number>;
};

export type TaskStatus = 'locked' | 'unavailable' | 'available' | 'actionable' | 'done';

/** Wire / URL sentinel for provisional spawn (mint.spawningNew). */
export const PROVISIONAL_GROUP_INSTANCE_ID = 'new';

/** Empty / `new` → '' for draft keys and deep-link gates. */
export function normalizeGroupInstanceId(groupInstanceId: string): string {
	if (!groupInstanceId || groupInstanceId === PROVISIONAL_GROUP_INSTANCE_ID) {
		return '';
	}
	return groupInstanceId;
}

export function emptyWorklist(): Worklist {
	return { jobs: [] };
}

export function worklistJobFor(
	worklist: Worklist | null | undefined,
	jobId: string
): WorklistJob | null {
	if (!worklist?.jobs?.length) return null;
	return worklist.jobs.find((row) => row.jobId === jobId) ?? null;
}

export function orderedJobIdsFromWorklist(
	worklist: Worklist | null | undefined
): string[] {
	if (!worklist?.jobs?.length) return [];
	return worklist.jobs.map((row) => row.jobId);
}

/** Entity Domain shape (server-side / bindings bag); kinds are Org Config vocabulary. */
export type Domain = {
	submissions: Submission[];
	jobInstances: JobInstance[];
} & Record<string, EntityMap | Submission[] | JobInstance[] | undefined>;

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

export function taskPath(jobId: string, taskId: string): string {
	return `${jobId}/${taskId}`;
}

export function groupTaskPath(
	jobId: string,
	groupId: string,
	groupInstanceId: string,
	taskId: string
): string {
	return `${jobId}/${groupId}/${groupInstanceId}/${taskId}`;
}

export function emptyDomain(): Domain {
	return {
		submissions: [],
		jobInstances: []
	};
}

/** Authored Task payload shape used by form fixture helpers (not on Load wire). */
export type TaskConfig = {
	id: string;
	[key: string]: unknown;
};

/** Authored Task order under a Job (array order). */
export function orderedTaskIds(tasks: TaskConfig[] | null | undefined): string[] {
	if (!tasks?.length) return [];
	return tasks.map((t) => t.id);
}

/** Lookup by Task id within an ordered Task array. */
export function taskById(
	tasks: TaskConfig[] | null | undefined,
	id: string
): TaskConfig | undefined {
	return tasks?.find((t) => t.id === id);
}
