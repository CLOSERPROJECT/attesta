import * as m from '#lib/paraglide/messages.js';

import type { PostCommitTarget } from './worklist-nav.js';
import {
	PROVISIONAL_GROUP_INSTANCE_ID,
	type Worklist,
	type WorklistInstance,
	type WorklistJob,
	type WorklistTask
} from './types.js';

export type AvailableRow = {
	jobId: string;
	jobInstanceId: string;
	taskId: string;
	/** Nested leaf only — open group instance coordinates. */
	groupId?: string;
	groupInstanceId?: string;
	/** Job instance label (product-facing name); falls back to instance id. */
	instanceLabel: string;
	/** Job display title; falls back to jobId when absent. */
	jobTitle: string;
	/** Authored Job color for card left border; omit when unset. */
	jobColor?: string;
	title: string;
	roles: string[];
	tasksDone: number;
	tasksTotal: number;
};

export type CanStartRow = {
	jobId: string;
	/** Job display title; falls back to jobId when absent. */
	jobTitle: string;
	/** Authored Job color for card left border; omit when unset. */
	jobColor?: string;
	startLabel: string;
	firstTaskId: string;
};

export type InProgressRow = {
	jobId: string;
	jobInstanceId: string;
	/** Job instance label (product-facing name); falls back to instance id. */
	instanceLabel: string;
	/** Job display title; falls back to jobId when absent. */
	jobTitle: string;
	/** Authored Job color for card left border; omit when unset. */
	jobColor?: string;
	tasksDone: number;
	tasksTotal: number;
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

function jobColorOf(job: WorklistJob): string | undefined {
	const color = job.color?.trim();
	return color || undefined;
}

function pushAvailableRow(
	out: AvailableRow[],
	job: WorklistJob,
	inst: WorklistInstance,
	taskId: string,
	progress: { done: number; total: number },
	group?: { groupId: string; groupInstanceId: string }
): void {
	const chrome = taskChrome(job, taskId);
	const jobColor = jobColorOf(job);
	out.push({
		jobId: job.jobId,
		jobInstanceId: inst.id,
		taskId,
		...(group
			? { groupId: group.groupId, groupInstanceId: group.groupInstanceId }
			: {}),
		instanceLabel: inst.label || inst.id,
		jobTitle: job.title?.trim() || job.jobId,
		...(jobColor ? { jobColor } : {}),
		title: chrome?.title || taskId,
		roles: chrome?.roles ?? [],
		tasksDone: progress.done,
		tasksTotal: progress.total
	});
}

/**
 * Sort: Job → instance → Config taskIds (flat actionable + nested group
 * children). When group.canSpawn, also a provisional first-leaf row
 * (`groupInstanceId` = `new`) — Available chrome for Spawn-via-first-leaf;
 * no separate home Spawn band. Close stays overview-only.
 */
export function availableRowsFromWorklist(
	worklist: Worklist | null | undefined
): AvailableRow[] {
	const jobs = worklist?.jobs;
	if (!jobs?.length) return [];
	const out: AvailableRow[] = [];
	for (const job of jobs) {
		for (const inst of job.instances ?? []) {
			const progress = instanceTaskProgress(job, inst);
			for (const taskId of jobConfigTaskIds(job)) {
				const group = inst.groups?.[taskId];
				if (group) {
					const childIds = group.childTaskIds ?? [];
					for (const gi of group.instances ?? []) {
						for (const childId of childIds) {
							if (gi.tasks[childId] !== 'actionable') continue;
							pushAvailableRow(out, job, inst, childId, progress, {
								groupId: taskId,
								groupInstanceId: gi.id
							});
						}
					}
					const firstChildId = childIds[0];
					if (group.canSpawn && firstChildId) {
						pushAvailableRow(out, job, inst, firstChildId, progress, {
							groupId: taskId,
							groupInstanceId: PROVISIONAL_GROUP_INSTANCE_ID
						});
					}
					continue;
				}
				if (inst.tasks[taskId] !== 'actionable') continue;
				pushAvailableRow(out, job, inst, taskId, progress);
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
		const jobColor = jobColorOf(job);
		out.push({
			jobId: job.jobId,
			jobTitle: job.title?.trim() || job.jobId,
			...(jobColor ? { jobColor } : {}),
			startLabel: job.startLabel?.trim() || m.fallback_start(),
			firstTaskId
		});
	}
	return out;
}

/**
 * Unfinished Job instances (Config taskIds via instanceTaskProgress:
 * include when done < total). Order: Worklist Job → instance.
 * Overlaps Available when an instance still has actionable Tasks.
 */
export function inProgressRowsFromWorklist(
	worklist: Worklist | null | undefined
): InProgressRow[] {
	const jobs = worklist?.jobs;
	if (!jobs?.length) return [];
	const out: InProgressRow[] = [];
	for (const job of jobs) {
		for (const inst of job.instances ?? []) {
			const progress = instanceTaskProgress(job, inst);
			if (progress.done >= progress.total) continue;
			const jobColor = jobColorOf(job);
			out.push({
				jobId: job.jobId,
				jobInstanceId: inst.id,
				instanceLabel: inst.label || inst.id,
				jobTitle: job.title?.trim() || job.jobId,
				...(jobColor ? { jobColor } : {}),
				tasksDone: progress.done,
				tasksTotal: progress.total
			});
		}
	}
	return out;
}

/** Same shape as In progress; finished instances (done >= total). */
export type CompletedRow = InProgressRow;

/**
 * Finished Job instances (Config taskIds via instanceTaskProgress:
 * include when done >= total). Order: Worklist Job → instance.
 * Inverse of In progress; no overlap with that band.
 */
export function completedRowsFromWorklist(
	worklist: Worklist | null | undefined
): CompletedRow[] {
	const jobs = worklist?.jobs;
	if (!jobs?.length) return [];
	const out: CompletedRow[] = [];
	for (const job of jobs) {
		for (const inst of job.instances ?? []) {
			const progress = instanceTaskProgress(job, inst);
			if (progress.done < progress.total) continue;
			const jobColor = jobColorOf(job);
			out.push({
				jobId: job.jobId,
				jobInstanceId: inst.id,
				instanceLabel: inst.label || inst.id,
				jobTitle: job.title?.trim() || job.jobId,
				...(jobColor ? { jobColor } : {}),
				tasksDone: progress.done,
				tasksTotal: progress.total
			});
		}
	}
	return out;
}

/** SPA Task path segment: provisional Start uses `new`. */
export function taskRouteInstanceSegment(jobInstanceId: string): string {
	return jobInstanceId === '' ? 'new' : jobInstanceId;
}

export function availableRowKey(row: AvailableRow): string {
	if (row.groupId && row.groupInstanceId) {
		return `${row.jobId}/${row.jobInstanceId}/${row.groupId}/${row.groupInstanceId}/${row.taskId}`;
	}
	return `${row.jobId}/${row.jobInstanceId}/${row.taskId}`;
}

/** Deep-link target for a home Available row (nested includes group coords). */
export function availableRowTarget(row: AvailableRow): Extract<PostCommitTarget, { kind: 'task' }> {
	if (row.groupId && row.groupInstanceId) {
		return {
			kind: 'task',
			jobId: row.jobId,
			jobInstanceId: row.jobInstanceId,
			groupId: row.groupId,
			groupInstanceId: row.groupInstanceId,
			taskId: row.taskId
		};
	}
	return {
		kind: 'task',
		jobId: row.jobId,
		jobInstanceId: row.jobInstanceId,
		taskId: row.taskId
	};
}

export function inProgressRowKey(row: InProgressRow): string {
	return `${row.jobId}/${row.jobInstanceId}`;
}

/** Deep-link target for a home In progress row → Job instance overview. */
export function inProgressRowTarget(
	row: InProgressRow
): Extract<PostCommitTarget, { kind: 'overview' }> {
	return {
		kind: 'overview',
		jobId: row.jobId,
		jobInstanceId: row.jobInstanceId
	};
}

export function completedRowKey(row: CompletedRow): string {
	return inProgressRowKey(row);
}

/** Deep-link target for a home Completed row → Job instance overview. */
export function completedRowTarget(
	row: CompletedRow
): Extract<PostCommitTarget, { kind: 'overview' }> {
	return inProgressRowTarget(row);
}
