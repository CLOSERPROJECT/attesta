import type { CommitNext } from './commit-port.js';
import type { TaskFormBundle } from './form/task-form-bundle.js';
import type { TaskStatus, Worklist, WorklistInstance, WorklistJob } from './types.js';
import { worklistJobFor } from './types.js';

/** Post-Commit / exit / deep-link SPA target (ADR-0006). */
export type PostCommitTarget =
	| { kind: 'task'; jobId: string; jobInstanceId: string; taskId: string }
	| { kind: 'overview'; jobId: string; jobInstanceId: string }
	| { kind: 'home' };

export type TaskDeepLinkTarget =
	| { kind: 'task' }
	| { kind: 'overview'; jobId: string; jobInstanceId: string }
	| { kind: 'home' };

export type OverviewTaskRow = {
	taskId: string;
	title: string;
	status: TaskStatus;
	roles: string[];
	/** Projected viewerAvailable membership — link to Task route; else status-only. */
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

/** Config/`taskIds` order for a Worklist Job (not `types.orderedTaskIds`). */
function jobConfigTaskIds(job: WorklistJob): string[] {
	if (job.taskIds?.length) return job.taskIds;
	return (job.tasks ?? []).map((t) => t.id);
}

function taskChrome(job: WorklistJob, taskId: string) {
	return job.tasks.find((t) => t.id === taskId) ?? null;
}

function viewerMayAct(inst: WorklistInstance, taskId: string): boolean {
	return (inst.viewerAvailable ?? []).includes(taskId);
}

/**
 * When a Task cannot be mounted (deep-link gate miss, task-form 403/404):
 * overview if the instance is known on the Worklist, else Home.
 */
export function resolveUnmountableTaskTarget(
	jobId: string,
	jobInstanceId: string,
	provisional: boolean,
	worklist: Worklist | null | undefined
): Extract<PostCommitTarget, { kind: 'overview' | 'home' }> {
	if (provisional || !jobId || !jobInstanceId || jobInstanceId === 'new') {
		return { kind: 'home' };
	}
	const job = worklistJobFor(worklist, jobId);
	if (!job || !instanceFor(job, jobInstanceId)) {
		return { kind: 'home' };
	}
	return { kind: 'overview', jobId, jobInstanceId };
}

/**
 * Thin adapter over Commit `next` (ADR-0006). Does not re-walk Worklist —
 * destination is server-evaluated.
 */
export function resolvePostCommitNavigation(
	next: CommitNext,
	jobId: string,
	jobInstanceId: string,
	nextTaskFormBundle: TaskFormBundle | null | undefined
): PostCommitTarget {
	if (next === 'task') {
		const bundle = nextTaskFormBundle;
		if (
			bundle &&
			bundle.jobId &&
			bundle.taskId &&
			bundle.jobInstanceId &&
			bundle.jobId === jobId
		) {
			return {
				kind: 'task',
				jobId: bundle.jobId,
				jobInstanceId: bundle.jobInstanceId,
				taskId: bundle.taskId
			};
		}
		return { kind: 'home' };
	}
	if (next === 'overview' && jobId && jobInstanceId) {
		return { kind: 'overview', jobId, jobInstanceId };
	}
	return { kind: 'home' };
}

/**
 * Deep-link gate: provisional Start always mounts; existing instance must be
 * projected viewerAvailable or redirect to overview/Home.
 */
export function resolveTaskDeepLink(
	jobId: string,
	jobInstanceId: string,
	taskId: string,
	provisional: boolean,
	worklist: Worklist | null | undefined
): TaskDeepLinkTarget {
	if (!jobId || !taskId) return { kind: 'home' };
	if (provisional) return { kind: 'task' };

	const job = worklistJobFor(worklist, jobId);
	const inst = job ? instanceFor(job, jobInstanceId) : null;
	if (job && inst && viewerMayAct(inst, taskId)) {
		return { kind: 'task' };
	}
	return resolveUnmountableTaskTarget(jobId, jobInstanceId, false, worklist);
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
	jobInstanceId: string
): JobInstanceOverviewModel | null {
	if (!jobId || !jobInstanceId || jobInstanceId === 'new') return null;
	const job = worklistJobFor(worklist, jobId);
	if (!job) return null;
	const inst = instanceFor(job, jobInstanceId);
	if (!inst) return null;

	const tasks: OverviewTaskRow[] = [];
	let firstAvailable: OverviewTaskRow | null = null;

	for (const taskId of jobConfigTaskIds(job)) {
		const chrome = taskChrome(job, taskId);
		const roles = chrome?.roles ?? [];
		const status = (inst.tasks[taskId] ?? 'locked') as TaskStatus;
		const linkable = viewerMayAct(inst, taskId);
		const row: OverviewTaskRow = {
			taskId,
			title: chrome?.title || taskId,
			status,
			roles,
			linkable
		};
		tasks.push(row);
		if (firstAvailable == null && status === 'available') {
			firstAvailable = row;
		}
	}

	// "Next" = first available in Config order; handoff only when that Task is not ours.
	const handoffRoles =
		firstAvailable && !firstAvailable.linkable && firstAvailable.roles.length > 0
			? [...firstAvailable.roles]
			: null;

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
