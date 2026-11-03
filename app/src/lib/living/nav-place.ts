import type { Worklist } from './types.js';
import { worklistJobFor } from './types.js';
import { resolveTaskExit, type PostCommitTarget } from './worklist-nav.js';
import { jobInstanceOverview } from './worklist-overview.js';

export type NavPlaceRouteKind = 'home' | 'overview' | 'task';

/**
 * Place kind from stripped route.pathname only (base already removed).
 * Segment count — not route.params.taskId — because sv-router may retain a
 * stale taskId on 2-segment overview, which would mis-classify as task and
 * make Back navigate to the same overview (no-op).
 * Trailing slash → still home at `/`.
 */
export function navPlaceKindFromRoute(input: {
	pathname: string;
}): NavPlaceRouteKind {
	let { pathname } = input;
	if (pathname.length > 1 && pathname.endsWith('/')) {
		pathname = pathname.slice(0, -1);
	}
	if (pathname === '/' || pathname === '') {
		return 'home';
	}

	const segments = pathname.replace(/^\//, '').split('/').filter(Boolean);
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
	provisional: boolean;
	worklist: Worklist | null | undefined;
};

export type NavPlaceChrome = {
	showBack: boolean;
	backTarget: Extract<PostCommitTarget, { kind: 'home' | 'overview' }>;
	primaryLabel: string;
	/** Authored Job.title only; null omits line 2 (never raw job id). */
	jobTitleLine: string | null;
};

function authoredJobTitle(
	worklist: Worklist | null | undefined,
	jobId: string
): string | null {
	const title = worklistJobFor(worklist, jobId)?.title?.trim();
	return title || null;
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
	const { kind, jobId, jobInstanceId, taskId, provisional, worklist } = input;

	if (kind === 'home') {
		return {
			showBack: false,
			backTarget: { kind: 'home' },
			primaryLabel: 'Home',
			jobTitleLine: null
		};
	}

	const jobTitleLine = authoredJobTitle(worklist, jobId);

	if (kind === 'overview') {
		const overview = jobInstanceOverview(worklist, jobId, jobInstanceId);
		return {
			showBack: true,
			backTarget: { kind: 'home' },
			primaryLabel: overview?.label ?? 'Job instance',
			jobTitleLine
		};
	}

	const taskTitle = taskTitleFromWorklist(worklist, jobId, taskId);
	const backTarget = resolveTaskExit(jobId, jobInstanceId, provisional) as Extract<
		PostCommitTarget,
		{ kind: 'home' | 'overview' }
	>;

	if (provisional) {
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
		primaryLabel: `${instanceLabel} · ${taskTitle}`,
		jobTitleLine
	};
}
