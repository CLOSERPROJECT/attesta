import type { Worklist, WorklistInstance, WorklistJob, WorklistTask } from './types.js';

export type AvailableRow = {
	jobId: string;
	jobInstanceId: string;
	taskId: string;
	/** Job instance label (product-facing name); falls back to instance id. */
	instanceLabel: string;
	/** Job display title; falls back to jobId when absent. */
	jobTitle: string;
	title: string;
	roles: string[];
	tasksDone: number;
	tasksTotal: number;
};

export type CanStartRow = {
	jobId: string;
	/** Job display title; falls back to jobId when absent. */
	jobTitle: string;
	startLabel: string;
	firstTaskId: string;
};

function taskChrome(job: WorklistJob, taskId: string): WorklistTask | null {
	return job.tasks.find((t) => t.id === taskId) ?? null;
}

function jobConfigTaskIds(job: WorklistJob): string[] {
	if (job.taskIds?.length) return job.taskIds;
	return (job.tasks ?? []).map((t) => t.id);
}

/** done/total Task counts for one Job instance (Config taskIds order). */
export function instanceTaskProgress(
	job: WorklistJob,
	inst: WorklistInstance
): { done: number; total: number } {
	const ids = jobConfigTaskIds(job);
	let done = 0;
	for (const taskId of ids) {
		if (inst.tasks[taskId] === 'done') done += 1;
	}
	return { done, total: ids.length };
}

/** Fraction for ring stroke; 0 when total is 0. */
export function progressFraction(done: number, total: number): number {
	if (total <= 0) return 0;
	return Math.min(1, Math.max(0, done / total));
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
			const { done, total } = instanceTaskProgress(job, inst);
			for (const taskId of inst.viewerAvailable ?? []) {
				const chrome = taskChrome(job, taskId);
				out.push({
					jobId: job.jobId,
					jobInstanceId: inst.id,
					taskId,
					instanceLabel: inst.label || inst.id,
					jobTitle: job.title?.trim() || job.jobId,
					title: chrome?.title || taskId,
					roles: chrome?.roles ?? [],
					tasksDone: done,
					tasksTotal: total
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
			jobTitle: job.title?.trim() || job.jobId,
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
