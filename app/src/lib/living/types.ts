import type { TaskFormBundle } from './form/task-form-bundle.js';

export type { TaskFormBundle };

/** Org hydrate from GET /api/living/domain — Worklist + bundles + config metadata. */
export type LoadResult = {
	org: string;
	activeConfigVersion: string;
	configId: string;
	worklist: Worklist;
	/** Worklist-available + provisional Task form mount payloads (replaced on Commit). */
	taskFormBundles: TaskFormBundle[];
};

/** Server-projected operator board (Unlock / startsWhen / leftover openness). */
export type Worklist = {
	jobs: WorklistJob[];
};

export type WorklistJob = {
	jobId: string;
	canStart: boolean;
	repeatable: boolean;
	startLabel?: string;
	/** Authored Task id order for chrome iteration. */
	taskIds: string[];
	instances: WorklistInstance[];
};

export type WorklistInstance = {
	id: string;
	label: string;
	tasks: Record<string, TaskStatus>;
};

export type TaskStatus = 'locked' | 'available' | 'done';

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

/** Entity Domain shape (server-side / bindings bag); not kept on SPA Load/Commit. */
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
