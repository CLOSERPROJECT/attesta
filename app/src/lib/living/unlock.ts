import type { Domain, Submission } from './types.js';

/** Authored Task order under a Job (Unlock order). */
export function orderedTaskIds(tasks: Record<string, unknown> | null | undefined): string[] {
	if (!tasks) return [];
	return Object.keys(tasks);
}

function hasSubmissionOnInstance(
	subs: Submission[],
	jobId: string,
	taskId: string,
	jobInstanceId: string
): boolean {
	return subs.some(
		(s) =>
			s.jobId === jobId &&
			s.taskId === taskId &&
			s.jobInstanceId === jobInstanceId
	);
}

/** Predecessor with Leftovers completeness "some" must not unlock the next Task. */
export function taskMayBeUnlockGate(task: unknown): boolean {
	if (!task || typeof task !== 'object' || Array.isArray(task)) return true;
	const dyn = (task as { dynamicProperties?: unknown }).dynamicProperties;
	if (!dyn || typeof dyn !== 'object' || Array.isArray(dyn)) return true;
	for (const raw of Object.values(dyn as Record<string, unknown>)) {
		if (!raw || typeof raw !== 'object' || Array.isArray(raw)) continue;
		if ((raw as { completeness?: unknown }).completeness === 'some') return false;
	}
	return true;
}

export function taskUnlocksWithInstance(task: unknown): boolean {
	if (!task || typeof task !== 'object' || Array.isArray(task)) return false;
	return (task as { unlockFrom?: unknown }).unlockFrom === 'instance';
}

export type UnlockQuery = {
	orderedIds: string[];
	jobId: string;
	taskId: string;
	jobInstanceId: string;
	submissions: Submission[];
	tasks: Record<string, unknown>;
};

/**
 * Task k+1 unlocks when predecessor k has a Submission on this Job instance.
 * First Task is always unlocked. Client chrome only — Cerbos re-gates on Commit.
 */
export function isTaskUnlocked(q: UnlockQuery): boolean {
	const index = q.orderedIds.indexOf(q.taskId);
	if (index < 0) return false;
	if (index === 0) return true;
	if (!q.jobInstanceId) return false;

	const previousId = q.orderedIds[index - 1]!;
	const previous = q.tasks[previousId];
	if (!taskMayBeUnlockGate(previous)) {
		const current = q.tasks[q.taskId];
		return taskUnlocksWithInstance(current) && q.jobInstanceId !== '';
	}
	return hasSubmissionOnInstance(q.submissions, q.jobId, previousId, q.jobInstanceId);
}

export function taskSubmitted(
	domain: Domain | null | undefined,
	jobId: string,
	taskId: string,
	jobInstanceId: string
): boolean {
	if (!domain) return false;
	return hasSubmissionOnInstance(domain.submissions, jobId, taskId, jobInstanceId);
}
