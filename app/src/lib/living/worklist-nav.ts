import type { CommitNext, HttpLinkItem } from './commit-port.js';
import type { TaskFormBundle } from './form/task-form-bundle.js';
import type { Worklist, WorklistInstance, WorklistJob } from './types.js';
import {
	normalizeGroupInstanceId,
	PROVISIONAL_GROUP_INSTANCE_ID,
	worklistJobFor
} from './types.js';
import { taskRouteInstanceSegment } from './worklist-home.js';

/** Post-Commit / exit / deep-link SPA target (ADR-0006). */
export type PostCommitTarget =
	| {
			kind: 'task';
			jobId: string;
			jobInstanceId: string;
			taskId: string;
			groupId?: string;
			groupInstanceId?: string;
	  }
	| { kind: 'overview'; jobId: string; jobInstanceId: string }
	| { kind: 'home' };

/** Commit follow-up: stay when links present, else ADR-0006 target. */
export type CommitFollowUp = { kind: 'stay' } | PostCommitTarget;

/** True when Commit returned at least one HttpLink (link and/or download). */
export function commitHasLinks(
	links: readonly HttpLinkItem[] | null | undefined
): boolean {
	return (links ?? []).length > 0;
}

/**
 * Post-Commit SPA decision: stay on read-only form when any links are present;
 * otherwise thin-adapt server `next` (ADR-0006 Task groups amend).
 */
export function resolveCommitFollowUp(
	next: CommitNext,
	jobId: string,
	jobInstanceId: string,
	nextTaskFormBundle: TaskFormBundle | null | undefined,
	links: readonly HttpLinkItem[] | null | undefined
): CommitFollowUp {
	if (commitHasLinks(links)) return { kind: 'stay' };
	return resolvePostCommitNavigation(
		next,
		jobId,
		jobInstanceId,
		nextTaskFormBundle
	);
}

/** Provisional first-leaf Spawn target after post-submit "Start another". */
export function resolveGroupStartAnotherTarget(
	jobId: string,
	jobInstanceId: string,
	groupId: string,
	firstChildTaskId: string
): PostCommitTarget {
	return {
		kind: 'task',
		jobId,
		jobInstanceId,
		groupId,
		groupInstanceId: PROVISIONAL_GROUP_INSTANCE_ID,
		taskId: firstChildTaskId
	};
}

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
	return inst.tasks[taskId] === 'actionable';
}

/**
 * When a Task cannot be mounted (deep-link gate miss, task-form 403/404):
 * overview if the instance is known on the Worklist, else Home.
 */
export function resolveUnmountableTaskTarget(
	jobId: string,
	jobInstanceId: string,
	provisionalJob: boolean,
	worklist: Worklist | null | undefined
): Extract<PostCommitTarget, { kind: 'overview' | 'home' }> {
	if (provisionalJob || !jobId || !jobInstanceId || jobInstanceId === 'new') {
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
			const groupId = bundle.groupId ?? '';
			const groupInstanceId = bundle.groupInstanceId ?? '';
			if (groupId) {
				return {
					kind: 'task',
					jobId: bundle.jobId,
					jobInstanceId: bundle.jobInstanceId,
					taskId: bundle.taskId,
					groupId,
					groupInstanceId: taskRouteInstanceSegment(groupInstanceId)
				};
			}
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
 * Deep-link gate: provisional Start/Spawn always mounts; existing instance must
 * be projected actionable (flat or nested) or redirect to overview/Home.
 */
export function resolveTaskDeepLink(
	jobId: string,
	jobInstanceId: string,
	taskId: string,
	provisionalJob: boolean,
	worklist: Worklist | null | undefined,
	groupId = '',
	groupInstanceId = ''
): TaskDeepLinkTarget {
	if (!jobId || !taskId) return { kind: 'home' };
	if (provisionalJob) return { kind: 'task' };

	const job = worklistJobFor(worklist, jobId);
	const inst = job ? instanceFor(job, jobInstanceId) : null;
	if (!job || !inst) {
		return resolveUnmountableTaskTarget(jobId, jobInstanceId, false, worklist);
	}

	if (groupId) {
		const provisionalSpawn = normalizeGroupInstanceId(groupInstanceId) === '';
		if (provisionalSpawn) return { kind: 'task' };
		const group = inst.groups?.[groupId];
		const gi = group?.instances.find((row) => row.id === groupInstanceId);
		if (gi && gi.tasks[taskId] === 'actionable') {
			return { kind: 'task' };
		}
		return resolveUnmountableTaskTarget(jobId, jobInstanceId, false, worklist);
	}

	if (viewerMayAct(inst, taskId)) {
		return { kind: 'task' };
	}
	return resolveUnmountableTaskTarget(jobId, jobInstanceId, false, worklist);
}

/** Exit from Task without Commit: provisional Job → Home; else overview. */
export function resolveTaskExit(
	jobId: string,
	jobInstanceId: string,
	provisionalJob: boolean
): PostCommitTarget {
	if (provisionalJob || !jobInstanceId || jobInstanceId === 'new') {
		return { kind: 'home' };
	}
	return { kind: 'overview', jobId, jobInstanceId };
}
