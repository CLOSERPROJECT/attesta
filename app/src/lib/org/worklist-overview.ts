import * as m from '#lib/paraglide/messages.js';

import type {
	TaskStatus,
	TaskSubmit,
	Worklist,
	WorklistGroup,
	WorklistGroupInstance,
	WorklistInstance,
	WorklistJob
} from './types.js';
import { worklistJobFor } from './types.js';

export type OverviewGroupChrome = {
	canSpawn: boolean;
	startLabel: string;
	firstChildTaskId: string;
	canClose: boolean;
	closeLabel: string;
};

export type OverviewChildTaskRow = {
	taskId: string;
	title: string;
	status: TaskStatus;
	linkable: boolean;
	/** Config submit cardinality; many-submit history chrome reads this. */
	submit?: TaskSubmit;
	/** Domain Submission N when `submit === 'many'` (from Worklist counts). */
	submissionCount?: number;
};

export type OverviewGroupInstanceRow = {
	id: string;
	/** Projected lot id when present; else group instance id. */
	displayId: string;
	status: TaskStatus;
	/**
	 * Nested child Task rows when the group has 2+ children.
	 * Empty when `flatChild` is set (single-child flatten).
	 */
	children: OverviewChildTaskRow[];
	/**
	 * When the Task group has exactly one child Task, the occurrence presents
	 * that child's interaction directly (title stays `displayId`).
	 */
	flatChild?: OverviewChildTaskRow;
};

export type OverviewTaskRow = {
	taskId: string;
	title: string;
	status: TaskStatus;
	roles: string[];
	/** `actionable` leaf — link to Task route; else status-only. */
	linkable: boolean;
	/** Config submit cardinality; many-submit history chrome reads this. */
	submit?: TaskSubmit;
	/** Domain Submission N when `submit === 'many'` (from Worklist counts). */
	submissionCount?: number;
	/** Present when this Config node is a Task group (Spawn/Close chrome; not a leaf). */
	group?: OverviewGroupChrome;
	groupInstances?: OverviewGroupInstanceRow[];
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

function groupChrome(
	group: WorklistGroup | undefined,
	chromeTitle: string
): OverviewGroupChrome | undefined {
	if (!group) return undefined;
	const firstChildTaskId = group.childTaskIds?.[0] ?? '';
	return {
		canSpawn: Boolean(group.canSpawn && firstChildTaskId),
		startLabel: group.startLabel?.trim() || chromeTitle || m.fallback_spawn(),
		firstChildTaskId,
		canClose: Boolean(group.canClose),
		closeLabel: group.closeLabel?.trim() || m.fallback_close()
	};
}

export function aggregateGroupInstanceStatus(
	instance: WorklistGroupInstance,
	childTaskIds?: string[]
): TaskStatus {
	const ids =
		childTaskIds && childTaskIds.length > 0
			? childTaskIds
			: Object.keys(instance.tasks);
	const statuses = ids.map((id) => (instance.tasks[id] ?? 'locked') as TaskStatus);
	if (statuses.some((s) => s === 'actionable')) return 'actionable';
	if (statuses.some((s) => s === 'available')) return 'available';
	if (statuses.length > 0 && statuses.every((s) => s === 'done')) return 'done';
	return 'locked';
}

function manySubmitFields(
	chromeSubmit: TaskSubmit | undefined,
	counts: Record<string, number> | undefined,
	taskId: string
): Pick<OverviewChildTaskRow, 'submit' | 'submissionCount'> {
	if (chromeSubmit !== 'many') return {};
	return {
		submit: 'many',
		submissionCount: counts?.[taskId] ?? 0
	};
}

/** Accordion under many-submit leaf when Load already reports N > 0. */
export function manySubmitHistoryVisible(
	submit: TaskSubmit | undefined,
	submissionCount: number | undefined
): boolean {
	return submit === 'many' && (submissionCount ?? 0) > 0;
}

export function formatSubmissionCount(count: number): string {
	if (count === 1) return m.submission_count_one({ count });
	return m.submission_count_many({ count });
}

function projectChildTasks(
	job: WorklistJob,
	instance: WorklistGroupInstance,
	childTaskIds: string[]
): OverviewChildTaskRow[] {
	return childTaskIds.map((taskId) => {
		const chrome = taskChrome(job, taskId);
		const status = (instance.tasks[taskId] ?? 'locked') as TaskStatus;
		return {
			taskId,
			title: chrome?.title || taskId,
			status,
			linkable: status === 'actionable',
			...manySubmitFields(chrome?.submit, instance.submissionCounts, taskId)
		};
	});
}

function projectGroupInstances(
	job: WorklistJob,
	group: WorklistGroup | undefined
): OverviewGroupInstanceRow[] | undefined {
	if (!group) return undefined;
	const childTaskIds = group.childTaskIds ?? [];
	const singleChild = childTaskIds.length === 1;
	return group.instances.map((gi) => {
		const children = projectChildTasks(job, gi, childTaskIds);
		const status = aggregateGroupInstanceStatus(gi, childTaskIds);
		if (singleChild) {
			return {
				id: gi.id,
				displayId: gi.label || gi.id,
				status,
				children: [],
				flatChild: children[0]
			};
		}
		return {
			id: gi.id,
			displayId: gi.label || gi.id,
			status,
			children
		};
	});
}

export function taskStatusLabel(status: TaskStatus): string {
	switch (status) {
		case 'locked':
			return m.status_locked();
		case 'unavailable':
			return m.status_unavailable();
		case 'available':
			return m.status_available();
		case 'actionable':
			return m.status_actionable();
		case 'done':
			return m.status_done();
	}
}

export function formatFactValue(value: unknown): string {
	if (value === null || value === undefined) return '—';
	if (typeof value === 'string') return value;
	if (typeof value === 'number' || typeof value === 'boolean') return String(value);
	try {
		return JSON.stringify(value);
	} catch {
		return String(value);
	}
}

/**
 * When Spawn is offered and the provisional first-child Draft is dirty, overview
 * paints a linkable task row (groupInstanceId `new`) instead of the Spawn CTA.
 */
export function provisionalSpawnDraftRowVisible(
	group: OverviewGroupChrome | undefined,
	provisionalFirstChildDirty: boolean
): boolean {
	return Boolean(
		group?.canSpawn && group.firstChildTaskId && provisionalFirstChildDirty
	);
}

/**
 * Open-group Spawn/Close chrome for post-submit links loop (ticket 05).
 * Null when the group is missing on the Worklist instance.
 */
export function openGroupLoopChrome(
	worklist: Worklist | null | undefined,
	jobId: string,
	jobInstanceId: string,
	groupId: string
): OverviewGroupChrome | null {
	if (!jobId || !jobInstanceId || !groupId) return null;
	const job = worklistJobFor(worklist, jobId);
	if (!job) return null;
	const inst = instanceFor(job, jobInstanceId);
	if (!inst) return null;
	const chrome = taskChrome(job, groupId);
	return (
		groupChrome(inst.groups?.[groupId], chrome?.title || groupId) ?? null
	);
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
	let firstOpen: OverviewTaskRow | null = null;

	for (const taskId of jobConfigTaskIds(job)) {
		const chrome = taskChrome(job, taskId);
		const roles = chrome?.roles ?? [];
		const status = (inst.tasks[taskId] ?? 'locked') as TaskStatus;
		const worklistGroup = inst.groups?.[taskId];
		const group = groupChrome(worklistGroup, chrome?.title || taskId);
		const groupInstances = projectGroupInstances(job, worklistGroup);
		const linkable = group ? false : status === 'actionable';
		const row: OverviewTaskRow = {
			taskId,
			title: chrome?.title || taskId,
			status,
			roles,
			linkable,
			...(group
				? {}
				: manySubmitFields(chrome?.submit, inst.submissionCounts, taskId)),
			...(group ? { group } : {}),
			...(groupInstances ? { groupInstances } : {})
		};
		tasks.push(row);
		if (firstOpen == null && (status === 'available' || status === 'actionable')) {
			firstOpen = row;
		}
	}

	// "Next" = first unlocked leaf/group in Config order; handoff when that Task is not ours.
	const handoffRoles =
		firstOpen &&
		!firstOpen.linkable &&
		!firstOpen.group?.canSpawn &&
		!firstOpen.group?.canClose &&
		firstOpen.roles.length > 0
			? [...firstOpen.roles]
			: null;

	// Completion ignores submit: many (same law as home instanceTaskProgress / Go remaining-work).
	const onceRows = tasks.filter((t) => taskChrome(job, t.taskId)?.submit !== 'many');
	return {
		jobId,
		jobInstanceId,
		label: inst.label || inst.id,
		tasks,
		handoffRoles,
		finished: tasks.length > 0 && onceRows.every((t) => t.status === 'done')
	};
}

export function formatHandoffCopy(roles: string[]): string {
	const named = roles.filter((r) => r.length > 0);
	if (!named.length) return m.handoff_waiting_role_unknown();
	if (named.length === 1) {
		return m.handoff_waiting_role_one({ role: named[0] });
	}
	return m.handoff_waiting_roles({ roles: named.join(', ') });
}
