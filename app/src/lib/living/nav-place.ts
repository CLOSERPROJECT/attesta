import * as m from '#lib/paraglide/messages.js';

import type { Worklist } from './types.js';
import { worklistJobFor } from './types.js';
import { resolveTaskExit, type PostCommitTarget } from './worklist-nav.js';
import { jobInstanceOverview } from './worklist-overview.js';

export type NavPlaceRouteKind = 'home' | 'overview' | 'task';

/** Living path segments after base strip (trailing slash normalized). */
export function livingPathSegments(pathname: string): string[] {
	let path = pathname;
	if (path.length > 1 && path.endsWith('/')) {
		path = path.slice(0, -1);
	}
	if (path === '/' || path === '') {
		return [];
	}
	return path.replace(/^\//, '').split('/').filter(Boolean);
}

/**
 * Task/overview coords from pathname only.
 * Ignore route.params.groupId / groupInstanceId unless the path has 5 living
 * segments — sv-router matchRoute leaks those keys from a failed 5-seg
 * candidate into the winning 3-seg (flat Task) or 2-seg (overview) match.
 */
export type LivingRouteCoords = {
	jobId: string;
	jobInstanceId: string;
	taskId: string;
	groupId: string;
	groupInstanceId: string;
};

export function livingRouteCoordsFromPath(input: {
	pathname: string;
}): LivingRouteCoords {
	const segments = livingPathSegments(input.pathname);
	if (segments.length >= 5) {
		return {
			jobId: segments[0] ?? '',
			jobInstanceId: segments[1] ?? '',
			groupId: segments[2] ?? '',
			groupInstanceId: segments[3] ?? '',
			taskId: segments[4] ?? ''
		};
	}
	if (segments.length >= 3) {
		return {
			jobId: segments[0] ?? '',
			jobInstanceId: segments[1] ?? '',
			groupId: '',
			groupInstanceId: '',
			taskId: segments[2] ?? ''
		};
	}
	if (segments.length >= 2) {
		return {
			jobId: segments[0] ?? '',
			jobInstanceId: segments[1] ?? '',
			groupId: '',
			groupInstanceId: '',
			taskId: ''
		};
	}
	if (segments.length >= 1) {
		return {
			jobId: segments[0] ?? '',
			jobInstanceId: '',
			groupId: '',
			groupInstanceId: '',
			taskId: ''
		};
	}
	return {
		jobId: '',
		jobInstanceId: '',
		groupId: '',
		groupInstanceId: '',
		taskId: ''
	};
}

/**
 * Place kind from stripped route.pathname only (base already removed).
 * Segment count — not route.params.taskId — because sv-router may retain a
 * stale taskId on 2-segment overview, which would mis-classify as task and
 * make Back navigate to the same overview (no-op).
 * Trailing slash → still home at `/`.
 * Nested Task: job / jobInstance / group / groupInstance / task (5 segments).
 */
export function navPlaceKindFromRoute(input: {
	pathname: string;
}): NavPlaceRouteKind {
	const segments = livingPathSegments(input.pathname);
	if (segments.length === 0) {
		return 'home';
	}
	if (segments.length >= 3) {
		return 'task';
	}
	if (segments.length >= 2) {
		return 'overview';
	}
	return 'overview';
}

export type NavPlaceInput = {
	kind: NavPlaceRouteKind;
	jobId: string;
	jobInstanceId: string;
	taskId: string;
	/** Job-level Start only — nested Spawn keeps a real jobInstanceId. */
	provisionalJob: boolean;
	/** Nested provisional Spawn — same chrome as minted Task; Back → home. */
	provisionalSpawn?: boolean;
	worklist: Worklist | null | undefined;
};

export type NavPlaceChrome = {
	showBack: boolean;
	backTarget: Extract<PostCommitTarget, { kind: 'home' }>;
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
 * Place chrome DTO for Living nav: label + back law only.
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
