import type { Worklist } from './types.js';
import { worklistJobFor } from './types.js';

/**
 * When a Task leaf is unavailable, return authored blockedReason (pass-through)
 * or null when absent. Used so deep links stay on the Task page instead of
 * bouncing to instance home.
 */
export function taskBlockedReason(
	worklist: Worklist | null | undefined,
	jobId: string,
	jobInstanceId: string,
	taskId: string
): string | null {
	const job = worklistJobFor(worklist, jobId);
	if (!job) return null;
	const inst = job.instances.find((row) => row.id === jobInstanceId);
	if (!inst) return null;
	if (inst.tasks[taskId] !== 'unavailable') return null;
	const reason = inst.blockedReasons?.[taskId]?.trim();
	return reason || null;
}

/** True when the addressed leaf is projected unavailable (show blocked chrome). */
export function taskIsUnavailable(
	worklist: Worklist | null | undefined,
	jobId: string,
	jobInstanceId: string,
	taskId: string
): boolean {
	const job = worklistJobFor(worklist, jobId);
	const inst = job?.instances.find((row) => row.id === jobInstanceId);
	return inst?.tasks[taskId] === 'unavailable';
}
