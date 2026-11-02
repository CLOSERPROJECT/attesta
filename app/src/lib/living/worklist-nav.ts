import type { CommitNext } from './commit-port.js';
import type { TaskFormBundle } from './form/task-form-bundle.js';
import type { Worklist, WorklistInstance, WorklistJob } from './types.js';
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

function instanceFor(
	job: WorklistJob,
	jobInstanceId: string
): WorklistInstance | null {
	return job.instances.find((inst) => inst.id === jobInstanceId) ?? null;
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
