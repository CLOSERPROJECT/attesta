import type { TaskFormBundle } from './form/task-form-bundle.js';

export type { TaskFormBundle };

export type LoadResult = {
	org: string;
	activeConfigVersion: string;
	configId: string;
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

export type WorklistTask = {
	id: string;
	title: string;
	/** Authored Config role slugs for board chrome (slug∩roles; not Cerbos). */
	roles: string[];
};

export type WorklistJob = {
	jobId: string;
	canStart: boolean;
	repeatable: boolean;
	startLabel?: string;
	/** Authored Task id order for chrome iteration. */
	taskIds: string[];
	/** Config-free Task title + roles chrome (alongside taskIds). */
	tasks: WorklistTask[];
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
