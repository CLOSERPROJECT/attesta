import type { CommitNavigation, ContinueNavigation } from './commit-port.js';
import type { TaskFormBundle } from './form/task-form-bundle.js';
import { taskIsUnavailable } from './task-blocked.js';
import type { Worklist, WorklistInstance, WorklistJob } from './types.js';
import { worklistJobFor } from './types.js';

/** Post-Commit / exit / deep-link SPA target (ADR-0006 / ADR-0014). */
export type PostCommitTarget =
	| {
			kind: 'task';
			stream: string;
			instance: string;
			jobId: string;
			jobInstanceId: string;
			taskId: string;
	  }
	| { kind: 'home'; stream: string; instance: string }
	| { kind: 'myStream'; stream: string }
	| { kind: 'my' };

/** Commit follow-up: stay only when server `navigation` is stay. */
export type CommitFollowUp = { kind: 'stay' } | PostCommitTarget;

export type StreamNavScope = {
	stream: string;
	/** Org stream instance; empty on Play until Commit mints one. */
	instance: string;
};

function asScope(stream: string, instance: string): StreamNavScope {
	return { stream, instance };
}

/**
 * Post-Commit SPA decision from server `navigation` only (ADR-0006).
 * Stay is never inferred from HTTP links. `home` is SPA instance timeline
 * (`/:stream/i/:instance`) via navigate-target.
 */
export function resolveCommitFollowUp(
	navigation: CommitNavigation,
	stream: string,
	instance: string,
	jobId: string,
	jobInstanceId: string,
	nextTaskFormBundle: TaskFormBundle | null | undefined
): CommitFollowUp {
	if (navigation === 'stay') return { kind: 'stay' };
	return resolvePostCommitNavigation(
		navigation,
		stream,
		instance,
		jobId,
		jobInstanceId,
		nextTaskFormBundle
	);
}

/** Continue after stay: server `continueNavigation` only (never re-derive). */
export function resolveContinueNavigation(
	continueNavigation: ContinueNavigation | null | undefined,
	stream: string,
	instance: string,
	jobId: string,
	jobInstanceId: string,
	nextTaskFormBundle: TaskFormBundle | null | undefined
): PostCommitTarget {
	return resolvePostCommitNavigation(
		continueNavigation ?? 'home',
		stream,
		instance,
		jobId,
		jobInstanceId,
		nextTaskFormBundle
	);
}

export type TaskDeepLinkTarget =
	| { kind: 'task' }
	| { kind: 'home'; stream: string; instance: string }
	| { kind: 'myStream'; stream: string }
	| { kind: 'my' };

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
 * `home` when scoped (timeline via navigate-target); Play → stream dashboard.
 */
export function resolveUnmountableTaskTarget(
	stream: string,
	instance: string,
	_jobId: string,
	_jobInstanceId: string,
	provisionalJob: boolean,
	_worklist: Worklist | null | undefined
): Exclude<PostCommitTarget, { kind: 'task' }> {
	if (instance) {
		return { kind: 'home', stream, instance };
	}
	if (provisionalJob && stream) {
		return { kind: 'myStream', stream };
	}
	return { kind: 'my' };
}

/**
 * Thin adapter over Commit `navigation` / Continue (ADR-0006). Does not
 * re-walk Worklist — destination is server-evaluated. Callers must not pass
 * `stay` (use resolveCommitFollowUp / resolveContinueNavigation).
 */
export function resolvePostCommitNavigation(
	navigation: Exclude<CommitNavigation, 'stay'> | ContinueNavigation,
	stream: string,
	instance: string,
	jobId: string,
	jobInstanceId: string,
	nextTaskFormBundle: TaskFormBundle | null | undefined
): PostCommitTarget {
	if (navigation === 'task') {
		const bundle = nextTaskFormBundle;
		const homeInstance = instance || jobInstanceId;
		if (
			bundle &&
			bundle.jobId &&
			bundle.taskId &&
			bundle.jobInstanceId &&
			bundle.jobId === jobId &&
			stream &&
			homeInstance
		) {
			return {
				kind: 'task',
				stream,
				instance: homeInstance,
				jobId: bundle.jobId,
				jobInstanceId: bundle.jobInstanceId,
				taskId: bundle.taskId
			};
		}
		return resolveInstanceHome(asScope(stream, instance), jobInstanceId);
	}
	// Legacy wire/Config overview → home (ADR-0006 hard cut).
	return resolveInstanceHome(asScope(stream, instance), jobInstanceId);
}

function resolveInstanceHome(
	scope: StreamNavScope,
	fallbackInstance: string
): PostCommitTarget {
	const instance = scope.instance || fallbackInstance;
	if (scope.stream && instance) {
		return { kind: 'home', stream: scope.stream, instance };
	}
	if (scope.stream) {
		return { kind: 'myStream', stream: scope.stream };
	}
	return { kind: 'my' };
}

/**
 * Deep-link gate: provisional Start only when Worklist canStart + first Task
 * (same CollectActionableTaskRefs / ViewerMayMountTask law as task-form).
 * Play (`play: true`) skips the Worklist canStart check — task-form is the gate.
 * Existing instance must be projected actionable or redirect Home.
 */
export function resolveTaskDeepLink(
	stream: string,
	instance: string,
	jobId: string,
	jobInstanceId: string,
	taskId: string,
	provisionalJob: boolean,
	worklist: Worklist | null | undefined,
	opts: { play?: boolean } = {}
): TaskDeepLinkTarget {
	if (!jobId || !taskId) {
		return resolveUnmountableTaskTarget(
			stream,
			instance,
			jobId,
			jobInstanceId,
			provisionalJob,
			worklist
		);
	}

	if (opts.play && provisionalJob) {
		return { kind: 'task' };
	}

	const job = worklistJobFor(worklist, jobId);
	if (provisionalJob) {
		if (!job?.canStart || job.taskIds[0] !== taskId) {
			return resolveUnmountableTaskTarget(
				stream,
				instance,
				jobId,
				jobInstanceId,
				true,
				worklist
			);
		}
		return { kind: 'task' };
	}

	const inst = job ? instanceFor(job, jobInstanceId) : null;
	if (!job || !inst) {
		return resolveUnmountableTaskTarget(
			stream,
			instance,
			jobId,
			jobInstanceId,
			false,
			worklist
		);
	}

	if (viewerMayAct(inst, taskId)) {
		return { kind: 'task' };
	}
	// Unavailable leaves stay mounted so blockedReason can surface (§6).
	if (taskIsUnavailable(worklist, jobId, jobInstanceId, taskId)) {
		return { kind: 'task' };
	}
	return resolveUnmountableTaskTarget(
		stream,
		instance,
		jobId,
		jobInstanceId,
		false,
		worklist
	);
}

/** Exit from Task without Commit: `home` when scoped (Play → stream dashboard). */
export function resolveTaskExit(
	stream: string,
	instance: string,
	provisionalJob: boolean
): PostCommitTarget {
	if (instance) {
		return { kind: 'home', stream, instance };
	}
	if (provisionalJob && stream) {
		return { kind: 'myStream', stream };
	}
	return { kind: 'my' };
}
