import type { TaskFormBundle } from './form/task-form-bundle.js';
import type { TaskStatus, Worklist, WorklistInstance, WorklistJob } from './types.js';
import { worklistJobFor } from './types.js';
import { rolesIntersect } from './worklist-home.js';

/** Post-Commit SPA target (ADR-0006). */
export type PostCommitTarget =
	| { kind: 'task'; jobId: string; jobInstanceId: string; taskId: string }
	| { kind: 'overview'; jobId: string; jobInstanceId: string }
	| { kind: 'home' };

/** Where a deep-linked Task URL should land when it is not mountable. */
export type TaskDeepLinkTarget =
	| { kind: 'task' }
	| { kind: 'overview'; jobId: string; jobInstanceId: string }
	| { kind: 'home' };

export type OverviewTaskRow = {
	taskId: string;
	title: string;
	status: TaskStatus;
	roles: string[];
	/** Role-eligible `available` — link to Task route; else status-only. */
	linkable: boolean;
};

export type JobInstanceOverviewModel = {
	jobId: string;
	jobInstanceId: string;
	label: string;
	tasks: OverviewTaskRow[];
	/** Roles for the first available Task the viewer cannot act on; null if none. */
	handoffRoles: string[] | null;
	finished: boolean;
};

function instanceFor(
	job: WorklistJob,
	jobInstanceId: string
): WorklistInstance | null {
	return job.instances.find((inst) => inst.id === jobInstanceId) ?? null;
}

function orderedTaskIds(job: WorklistJob): string[] {
	if (job.taskIds?.length) return job.taskIds;
	return (job.tasks ?? []).map((t) => t.id);
}

function taskChrome(job: WorklistJob, taskId: string) {
	return job.tasks.find((t) => t.id === taskId) ?? null;
}

/** True when every authored Task on the instance is `done`. */
export function instanceIsFinished(
	worklist: Worklist | null | undefined,
	jobId: string,
	jobInstanceId: string
): boolean {
	const job = worklistJobFor(worklist, jobId);
	if (!job) return false;
	const inst = instanceFor(job, jobInstanceId);
	if (!inst) return false;
	const ids = orderedTaskIds(job);
	if (!ids.length) return false;
	return ids.every((tid) => inst.tasks[tid] === 'done');
}

/** True when any Task on the instance is not yet `done` (overview still useful). */
export function instanceHasRemainingWork(
	worklist: Worklist | null | undefined,
	jobId: string,
	jobInstanceId: string
): boolean {
	const job = worklistJobFor(worklist, jobId);
	if (!job) return false;
	const inst = instanceFor(job, jobInstanceId);
	if (!inst) return false;
	return orderedTaskIds(job).some((tid) => inst.tasks[tid] !== 'done');
}

/**
 * ADR-0006: nextTaskFormBundle → Task; else overview while instance has work;
 * else Home when finished / unknown.
 */
export function resolvePostCommitNavigation(
	jobId: string,
	jobInstanceId: string,
	worklist: Worklist | null | undefined,
	nextTaskFormBundle: TaskFormBundle | null | undefined
): PostCommitTarget {
	const next = nextTaskFormBundle;
	if (
		next &&
		next.jobId &&
		next.taskId &&
		next.jobInstanceId &&
		next.jobId === jobId
	) {
		return {
			kind: 'task',
			jobId: next.jobId,
			jobInstanceId: next.jobInstanceId,
			taskId: next.taskId
		};
	}
	if (jobInstanceId && instanceHasRemainingWork(worklist, jobId, jobInstanceId)) {
		return { kind: 'overview', jobId, jobInstanceId };
	}
	return { kind: 'home' };
}

/**
 * Deep-link gate: provisional Start always mounts; existing instance must be
 * role-eligible `available` or redirect to overview/Home.
 */
export function resolveTaskDeepLink(
	jobId: string,
	jobInstanceId: string,
	taskId: string,
	provisional: boolean,
	worklist: Worklist | null | undefined,
	viewerRoles: string[] | null | undefined
): TaskDeepLinkTarget {
	if (!jobId || !taskId) return { kind: 'home' };
	if (provisional) return { kind: 'task' };

	const job = worklistJobFor(worklist, jobId);
	if (!job) return { kind: 'home' };
	const inst = instanceFor(job, jobInstanceId);
	if (!inst) return { kind: 'home' };

	const status = inst.tasks[taskId];
	if (status == null) return { kind: 'overview', jobId, jobInstanceId };

	const chrome = taskChrome(job, taskId);
	const roles = chrome?.roles ?? [];
	if (status === 'available' && rolesIntersect(viewerRoles, roles)) {
		return { kind: 'task' };
	}
	return { kind: 'overview', jobId, jobInstanceId };
}

/** Exit from Task without Commit: provisional → Home; else overview. */
export function resolveTaskExit(
	jobId: string,
	jobInstanceId: string,
	provisional: boolean
): PostCommitTarget {
	if (provisional || !jobInstanceId || jobInstanceId === 'new') {
		return { kind: 'home' };
	}
	return { kind: 'overview', jobId, jobInstanceId };
}

export function jobInstanceOverview(
	worklist: Worklist | null | undefined,
	jobId: string,
	jobInstanceId: string,
	viewerRoles: string[] | null | undefined
): JobInstanceOverviewModel | null {
	if (!jobId || !jobInstanceId || jobInstanceId === 'new') return null;
	const job = worklistJobFor(worklist, jobId);
	if (!job) return null;
	const inst = instanceFor(job, jobInstanceId);
	if (!inst) return null;

	const tasks: OverviewTaskRow[] = [];
	let handoffRoles: string[] | null = null;

	for (const taskId of orderedTaskIds(job)) {
		const chrome = taskChrome(job, taskId);
		const roles = chrome?.roles ?? [];
		const status = (inst.tasks[taskId] ?? 'locked') as TaskStatus;
		const eligible = rolesIntersect(viewerRoles, roles);
		const linkable = status === 'available' && eligible;
		tasks.push({
			taskId,
			title: chrome?.title || taskId,
			status,
			roles,
			linkable
		});
		if (
			handoffRoles == null &&
			status === 'available' &&
			!eligible &&
			roles.length > 0
		) {
			handoffRoles = [...roles];
		}
	}

	return {
		jobId,
		jobInstanceId,
		label: inst.label || inst.id,
		tasks,
		handoffRoles,
		finished: tasks.length > 0 && tasks.every((t) => t.status === 'done')
	};
}

export function formatHandoffCopy(roles: string[]): string {
	const named = roles.filter((r) => r.length > 0);
	if (!named.length) return 'Next work is waiting for another role.';
	if (named.length === 1) {
		return `Next work waits for role: ${named[0]}.`;
	}
	return `Next work waits for roles: ${named.join(', ')}.`;
}
