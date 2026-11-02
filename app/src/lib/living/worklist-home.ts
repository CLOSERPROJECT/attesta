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

/** slug∩roles board chrome (mirrors Go rolesIntersect; not Cerbos). */
export function rolesIntersect(viewer: string[] | null | undefined, allowed: string[]): boolean {
	if (!viewer?.length || !allowed.length) return false;
	const have = new Set(viewer.filter((r) => r.length > 0));
	return allowed.some((r) => r.length > 0 && have.has(r));
}

function taskChrome(job: WorklistJob, taskId: string): WorklistTask | null {
	return job.tasks.find((t) => t.id === taskId) ?? null;
}

/** Sort: Job → instance → Task (Worklist / Org Config order). */
export function availableRowsFromWorklist(
	worklist: Worklist | null | undefined,
	viewerRoles: string[] | null | undefined
): AvailableRow[] {
	const jobs = worklist?.jobs;
	if (!jobs?.length) return [];
	const out: AvailableRow[] = [];
	for (const job of jobs) {
		const ordered = job.taskIds?.length
			? job.taskIds
			: (job.tasks ?? []).map((t) => t.id);
		for (const inst of job.instances ?? []) {
			for (const taskId of ordered) {
				if (inst.tasks[taskId] !== 'available') continue;
				const chrome = taskChrome(job, taskId);
				const roles = chrome?.roles ?? [];
				if (!rolesIntersect(viewerRoles, roles)) continue;
				out.push({
					jobId: job.jobId,
					jobInstanceId: inst.id,
					taskId,
					instanceLabel: inst.label || inst.id,
					title: chrome?.title || taskId,
					roles
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
