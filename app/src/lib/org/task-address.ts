import { PROVISIONAL_JOB_INSTANCE_ID } from './types.js';

/**
 * Normalized Task path address under a stream instance.
 * `jobInstanceId` is empty when provisional; wire/URL uses `new` via helpers below.
 */
export type TaskAddress = {
	stream: string;
	instance: string;
	jobId: string;
	taskId: string;
	jobInstanceId: string;
	provisionalJob: boolean;
};

/** Route/wire segment is provisional when empty or the `new` sentinel. */
export function isProvisionalJobInstance(routeInstance: string): boolean {
	return routeInstance === '' || routeInstance === PROVISIONAL_JOB_INSTANCE_ID;
}

/** Collapse route/wire instance segment to Domain empty id when provisional. */
export function normalizeJobInstanceId(routeInstance: string): string {
	return isProvisionalJobInstance(routeInstance) ? '' : routeInstance;
}

/** Normalize stream + Job + instance + Task segments into one Task address. */
export function normalizeTaskAddress(input: {
	stream: string;
	instance: string;
	jobId: string;
	jobInstanceId: string;
	taskId: string;
}): TaskAddress {
	const provisionalJob = isProvisionalJobInstance(input.jobInstanceId);
	return {
		stream: input.stream,
		instance: input.instance,
		jobId: input.jobId,
		taskId: input.taskId,
		jobInstanceId: provisionalJob ? '' : input.jobInstanceId,
		provisionalJob
	};
}

/** SPA / wire Task path segment: provisional Start uses `new`. */
export function taskRouteInstanceSegment(jobInstanceId: string): string {
	return jobInstanceId === '' ? PROVISIONAL_JOB_INSTANCE_ID : jobInstanceId;
}

/** Wire `jobInstanceId` for TaskForm fetch (provisional → `new`). */
export function wireJobInstanceId(
	addr: Pick<TaskAddress, 'provisionalJob' | 'jobInstanceId'>
): string {
	return addr.provisionalJob ? PROVISIONAL_JOB_INSTANCE_ID : addr.jobInstanceId;
}

/** Commit mint.startingNew when provisional job start or Play (no stream instance). */
export function needsStartingNewMint(
	addr: Pick<TaskAddress, 'provisionalJob' | 'instance'>
): boolean {
	return addr.provisionalJob || !addr.instance;
}

/** Stable address key (`pv-recovery|261102-1|intake|new|register`); provisional always keyed as `new`. */
export function taskAddressKey(
	addr: Pick<
		TaskAddress,
		'stream' | 'instance' | 'jobId' | 'taskId' | 'jobInstanceId' | 'provisionalJob'
	>
): string {
	return `${addr.stream}|${addr.instance}|${addr.jobId}|${wireJobInstanceId(addr)}|${addr.taskId}`;
}
