import type { TaskStatus, Worklist, WorklistInstance, WorklistJob } from './types.js';
import { worklistJobFor } from './types.js';

export type OverviewTaskRow = {
	taskId: string;
	title: string;
	status: TaskStatus;
	roles: string[];
	/** Projected viewerAvailable membership — link to Task route; else status-only. */
	linkable: boolean;
};

export type JobInstanceOverviewModel = {
	jobId: string;
	jobInstanceId: string;
	label: string;
	tasks: OverviewTaskRow[];
	/** Roles for the first available Task the viewer cannot act on; null if none. */
	handoffRoles: string[] | null;
	finished: boolean;
};

function instanceFor(
	job: WorklistJob,
	jobInstanceId: string
): WorklistInstance | null {
	return job.instances.find((inst) => inst.id === jobInstanceId) ?? null;
}

/** Config/`taskIds` order for a Worklist Job (not `types.orderedTaskIds`). */
function jobConfigTaskIds(job: WorklistJob): string[] {
	if (job.taskIds?.length) return job.taskIds;
	return (job.tasks ?? []).map((t) => t.id);
}

function taskChrome(job: WorklistJob, taskId: string) {
	return job.tasks.find((t) => t.id === taskId) ?? null;
}

function viewerMayAct(inst: WorklistInstance, taskId: string): boolean {
	return (inst.viewerAvailable ?? []).includes(taskId);
}

export function jobInstanceOverview(
	worklist: Worklist | null | undefined,
	jobId: string,
	jobInstanceId: string
): JobInstanceOverviewModel | null {
	if (!jobId || !jobInstanceId || jobInstanceId === 'new') return null;
	const job = worklistJobFor(worklist, jobId);
	if (!job) return null;
	const inst = instanceFor(job, jobInstanceId);
	if (!inst) return null;

	const tasks: OverviewTaskRow[] = [];
	let firstAvailable: OverviewTaskRow | null = null;

	for (const taskId of jobConfigTaskIds(job)) {
		const chrome = taskChrome(job, taskId);
		const roles = chrome?.roles ?? [];
		const status = (inst.tasks[taskId] ?? 'locked') as TaskStatus;
		const linkable = viewerMayAct(inst, taskId);
		const row: OverviewTaskRow = {
			taskId,
			title: chrome?.title || taskId,
			status,
			roles,
			linkable
		};
		tasks.push(row);
		if (firstAvailable == null && status === 'available') {
			firstAvailable = row;
		}
	}

	// "Next" = first available in Config order; handoff only when that Task is not ours.
	const handoffRoles =
		firstAvailable && !firstAvailable.linkable && firstAvailable.roles.length > 0
			? [...firstAvailable.roles]
			: null;

	return {
		jobId,
		jobInstanceId,
		label: inst.label || inst.id,
		tasks,
		handoffRoles,
		finished: tasks.length > 0 && tasks.every((t) => t.status === 'done')
	};
}

export function formatHandoffCopy(roles: string[]): string {
	const named = roles.filter((r) => r.length > 0);
	if (!named.length) return 'Next work is waiting for another role.';
	if (named.length === 1) {
		return `Next work waits for role: ${named[0]}.`;
	}
	return `Next work waits for roles: ${named.join(', ')}.`;
}
