import type { Worklist, WorklistJob, WorklistTask } from './types.js';

export type AvailableRow = {
	jobId: string;
	jobInstanceId: string;
	taskId: string;
	instanceLabel: string;
	title: string;
	roles: string[];
};

export type CanStartRow = {
	jobId: string;
	startLabel: string;
	firstTaskId: string;
};

function taskChrome(job: WorklistJob, taskId: string): WorklistTask | null {
	return job.tasks.find((t) => t.id === taskId) ?? null;
}

/** Sort: Job → instance → Task (Worklist / Org Config order). Reads projected viewerAvailable. */
export function availableRowsFromWorklist(
	worklist: Worklist | null | undefined
): AvailableRow[] {
	const jobs = worklist?.jobs;
	if (!jobs?.length) return [];
	const out: AvailableRow[] = [];
	for (const job of jobs) {
		for (const inst of job.instances ?? []) {
			for (const taskId of inst.viewerAvailable ?? []) {
				const chrome = taskChrome(job, taskId);
				out.push({
					jobId: job.jobId,
					jobInstanceId: inst.id,
					taskId,
					instanceLabel: inst.label || inst.id,
					title: chrome?.title || taskId,
					roles: chrome?.roles ?? []
				});
			}
		}
	}
	return out;
}

/** `canStart` is already viewer-scoped on the wire. */
export function canStartRowsFromWorklist(
	worklist: Worklist | null | undefined
): CanStartRow[] {
	const jobs = worklist?.jobs;
	if (!jobs?.length) return [];
	const out: CanStartRow[] = [];
	for (const job of jobs) {
		if (!job.canStart) continue;
		const firstTaskId = job.taskIds?.[0] ?? job.tasks?.[0]?.id;
		if (!firstTaskId) continue;
		out.push({
			jobId: job.jobId,
			startLabel: job.startLabel?.trim() || 'Start',
			firstTaskId
		});
	}
	return out;
}

/** SPA Task path segment: provisional Start uses `new`. */
export function taskRouteInstanceSegment(jobInstanceId: string): string {
	return jobInstanceId === '' ? 'new' : jobInstanceId;
}

export function availableRowKey(row: AvailableRow): string {
	return `${row.jobId}/${row.jobInstanceId}/${row.taskId}`;
}
