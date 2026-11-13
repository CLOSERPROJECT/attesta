import type { CommitNavigation, ContinueNavigation } from './commit-port.js';
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

/** Commit follow-up: stay only when server `navigation` is stay. */
export type CommitFollowUp = { kind: 'stay' } | PostCommitTarget;

/**
 * Post-Commit SPA decision from server `navigation` only (ADR-0006).
 * Stay is never inferred from HTTP links.
 */
export function resolveCommitFollowUp(
	navigation: CommitNavigation,
	jobId: string,
	jobInstanceId: string,
	nextTaskFormBundle: TaskFormBundle | null | undefined
): CommitFollowUp {
	if (navigation === 'stay') return { kind: 'stay' };
	return resolvePostCommitNavigation(
		navigation,
		jobId,
		jobInstanceId,
		nextTaskFormBundle
	);
}

/** Continue after stay: server `continueNavigation` only (never re-derive). */
export function resolveContinueNavigation(
	continueNavigation: ContinueNavigation | null | undefined,
	jobId: string,
	jobInstanceId: string,
	nextTaskFormBundle: TaskFormBundle | null | undefined
): PostCommitTarget {
	return resolvePostCommitNavigation(
		continueNavigation ?? 'home',
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
 * always Home (overview unlinked as a product entry).
 */
export function resolveUnmountableTaskTarget(
	_jobId: string,
	_jobInstanceId: string,
	_provisionalJob: boolean,
	_worklist: Worklist | null | undefined
): Extract<PostCommitTarget, { kind: 'home' }> {
	return { kind: 'home' };
}

/**
 * Thin adapter over Commit `navigation` / Continue (ADR-0006). Does not
 * re-walk Worklist — destination is server-evaluated. Callers must not pass
 * `stay` (use resolveCommitFollowUp / resolveContinueNavigation).
 */
export function resolvePostCommitNavigation(
	navigation: Exclude<CommitNavigation, 'stay'> | ContinueNavigation,
	jobId: string,
	jobInstanceId: string,
	nextTaskFormBundle: TaskFormBundle | null | undefined
): PostCommitTarget {
	if (navigation === 'task') {
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
	// Legacy wire/Config overview → home (ADR-0006 hard cut).
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

/** Exit from Task without Commit: always Home (overview unlinked). */
export function resolveTaskExit(
	_jobId: string,
	_jobInstanceId: string,
	_provisionalJob: boolean
): Extract<PostCommitTarget, { kind: 'home' }> {
	return { kind: 'home' };
}
