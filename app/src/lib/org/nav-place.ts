import * as m from '#lib/paraglide/messages.js';

import type { Worklist } from './types.js';
import { worklistJobFor } from './types.js';
import { resolveTaskExit, type PostCommitTarget } from './worklist-nav.js';
import { jobInstanceOverview } from './worklist-overview.js';

export type NavPlaceRouteKind = 'home' | 'archive' | 'overview' | 'task';

/** Org path segments after base strip (trailing slash normalized). */
export function orgPathSegments(pathname: string): string[] {
	let path = pathname;
	if (path.length > 1 && path.endsWith('/')) {
		path = path.slice(0, -1);
	}
	if (path === '/' || path === '') {
		return [];
	}
	return path.replace(/^\//, '').split('/').filter(Boolean);
}

const ARCHIVE_SEGMENT = 'archive';

/**
 * Task/overview coords from pathname only.
 * Nested group paths (5+ segments) are rejected — flat leaf routes only.
 */
export type OrgRouteCoords = {
	jobId: string;
	jobInstanceId: string;
	taskId: string;
};

export function orgRouteCoordsFromPath(input: {
	pathname: string;
}): OrgRouteCoords {
	const segments = orgPathSegments(input.pathname);
	if (segments.length > 3) {
		return {
			jobId: '',
			jobInstanceId: '',
			taskId: ''
		};
	}
	if (segments.length === 3) {
		return {
			jobId: segments[0] ?? '',
			jobInstanceId: segments[1] ?? '',
			taskId: segments[2] ?? ''
		};
	}
	if (segments.length === 2) {
		return {
			jobId: segments[0] ?? '',
			jobInstanceId: segments[1] ?? '',
			taskId: ''
		};
	}
	if (segments.length === 1) {
		return {
			jobId: segments[0] ?? '',
			jobInstanceId: '',
			taskId: ''
		};
	}
	return {
		jobId: '',
		jobInstanceId: '',
		taskId: ''
	};
}

/**
 * Place kind from stripped route.pathname only (base already removed).
 * Segment count — not route.params.taskId — because sv-router may retain a
 * stale taskId on 2-segment overview, which would mis-classify as task and
 * make Back navigate to the same overview (no-op).
 * Trailing slash → still home at `/`.
 * Flat Task: job / jobInstance / task (3 segments).
 * Static `/archive` is Archive (not a Job id).
 */
export function navPlaceKindFromRoute(input: {
	pathname: string;
}): NavPlaceRouteKind {
	const segments = orgPathSegments(input.pathname);
	if (segments.length === 0) {
		return 'home';
	}
	if (segments.length === 1 && segments[0] === ARCHIVE_SEGMENT) {
		return 'archive';
	}
	if (segments.length > 3) {
		return 'home';
	}
	if (segments.length === 3) {
		return 'task';
	}
	if (segments.length === 2) {
		return 'overview';
	}
	return 'overview';
}

export type NavPlaceInput = {
	kind: NavPlaceRouteKind;
	jobId: string;
	jobInstanceId: string;
	taskId: string;
	/** Job-level Start — provisional empty / `new` jobInstanceId. */
	provisionalJob: boolean;
	worklist: Worklist | null | undefined;
};

export type NavPlaceChrome = {
	showBack: boolean;
	/** Home from Archive/Task/overview (arrival tracking skipped — always home). */
	backTarget: Extract<PostCommitTarget, { kind: 'home' | 'archive' }>;
	primaryLabel: string;
	/**
	 * Muted top line: authored Job.title on overview; on Task pages
	 * `title · instance` (title alone when provisional Start has no instance).
	 * Null omits the line (never raw job id alone as a stand-in title).
	 */
	jobTitleLine: string | null;
};

function authoredJobTitle(
	worklist: Worklist | null | undefined,
	jobId: string
): string | null {
	const title = worklistJobFor(worklist, jobId)?.title?.trim();
	return title || null;
}

/** Task muted line: Job.title · instance label when both exist. */
function taskJobContextLine(
	jobTitle: string | null,
	instanceLabel: string
): string | null {
	const inst = instanceLabel.trim();
	if (jobTitle && inst) return `${jobTitle} · ${inst}`;
	if (jobTitle) return jobTitle;
	if (inst) return inst;
	return null;
}

function taskTitleFromWorklist(
	worklist: Worklist | null | undefined,
	jobId: string,
	taskId: string
): string {
	const job = worklistJobFor(worklist, jobId);
	const chrome = job?.tasks.find((t) => t.id === taskId);
	const title = chrome?.title;
	if (typeof title === 'string' && title.trim()) {
		return title.trim();
	}
	return taskId.trim() || taskId;
}

/**
 * Place chrome DTO for org nav: label + back law only.
 * Reuses overview instance label and Worklist task/job title sources.
 * Overview never composes `·` (instance label only); taskId is ignored when kind is overview.
 */
export function resolveNavPlace(input: NavPlaceInput): NavPlaceChrome {
	const { kind, jobId, jobInstanceId, taskId, provisionalJob, worklist } = input;

	if (kind === 'home') {
		return {
			showBack: false,
			backTarget: { kind: 'home' },
			primaryLabel: m.fallback_home(),
			jobTitleLine: null
		};
	}

	if (kind === 'archive') {
		return {
			showBack: true,
			backTarget: { kind: 'home' },
			primaryLabel: m.archive(),
			jobTitleLine: null
		};
	}

	const jobTitleLine = authoredJobTitle(worklist, jobId);

	if (kind === 'overview') {
		const overview = jobInstanceOverview(worklist, jobId, jobInstanceId);
		return {
			showBack: true,
			backTarget: { kind: 'home' },
			primaryLabel: overview?.label ?? m.fallback_job_instance(),
			jobTitleLine
		};
	}

	const taskTitle = taskTitleFromWorklist(worklist, jobId, taskId);
	const backTarget = resolveTaskExit(jobId, jobInstanceId, provisionalJob);

	if (provisionalJob) {
		return {
			showBack: true,
			backTarget,
			primaryLabel: taskTitle,
			jobTitleLine
		};
	}

	const overview = jobInstanceOverview(worklist, jobId, jobInstanceId);
	const instanceLabel = overview?.label ?? jobInstanceId;
	return {
		showBack: true,
		backTarget,
		primaryLabel: taskTitle,
		jobTitleLine: taskJobContextLine(jobTitleLine, instanceLabel)
	};
}
