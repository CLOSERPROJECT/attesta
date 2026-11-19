import * as m from '#lib/paraglide/messages.js';

import type { Worklist } from './types.js';
import { worklistJobFor } from './types.js';
import { resolveTaskExit, type PostCommitTarget } from './worklist-nav.js';

export type NavPlaceRouteKind = 'app-root' | 'stream' | 'play' | 'home' | 'task';

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

/** `/my` dashboard for an Org stream (`org-{stream}` catalog key). */
export function orgMyDashboardPath(stream: string): string {
	return `/my/streams/org-${stream}/`;
}

/** `/my` instance timeline for an Org stream instance. */
export function orgMyInstancePath(stream: string, instance: string): string {
	return `/my/streams/org-${stream}/instance/${instance}`;
}

/** Anchor / stream-instance label from the filtered Worklist (truck id). */
export function streamInstanceLabel(
	worklist: Worklist | null | undefined,
	instanceId: string
): string {
	const id = instanceId.trim();
	if (!id || !worklist?.jobs?.length) return id;
	for (const job of worklist.jobs) {
		const inst = job.instances.find((row) => row.id === id);
		if (inst) {
			const label = inst.label?.trim();
			return label || id;
		}
	}
	return id;
}

export function orgPlayPath(stream: string): string {
	return `/${stream}/new`;
}

export function orgHomePath(stream: string, instance: string): string {
	return `/${stream}/i/${instance}`;
}

export function orgTaskPath(
	stream: string,
	instance: string,
	jobId: string,
	jobInstanceId: string,
	taskId: string
): string {
	return `${orgHomePath(stream, instance)}/${jobId}/${jobInstanceId}/${taskId}`;
}

/**
 * Place coords from pathname only (base already removed).
 * §6: / | /{stream} | /{stream}/new | /{stream}/i/{instance} | …/task
 */
export type OrgRouteCoords = {
	stream: string;
	instance: string;
	jobId: string;
	jobInstanceId: string;
	taskId: string;
};

const emptyCoords = (): OrgRouteCoords => ({
	stream: '',
	instance: '',
	jobId: '',
	jobInstanceId: '',
	taskId: ''
});

export function orgRouteCoordsFromPath(input: {
	pathname: string;
}): OrgRouteCoords {
	const segments = orgPathSegments(input.pathname);
	if (segments.length === 0) {
		return emptyCoords();
	}
	const stream = segments[0] ?? '';
	if (!stream || stream === 'i' || stream === 'new') {
		return emptyCoords();
	}
	if (segments.length === 1) {
		return { ...emptyCoords(), stream };
	}
	if (segments.length === 2 && segments[1] === 'new') {
		return { ...emptyCoords(), stream };
	}
	if (segments[1] === 'i') {
		const instance = segments[2] ?? '';
		if (!instance) return emptyCoords();
		if (segments.length === 3) {
			return { ...emptyCoords(), stream, instance };
		}
		if (segments.length === 6) {
			return {
				stream,
				instance,
				jobId: segments[3] ?? '',
				jobInstanceId: segments[4] ?? '',
				taskId: segments[5] ?? ''
			};
		}
	}
	return emptyCoords();
}

/**
 * Place kind from stripped route.pathname only (base already removed).
 * Segment shapes match the §6 Routes table — not route.params.
 */
export function navPlaceKindFromRoute(input: {
	pathname: string;
}): NavPlaceRouteKind {
	const segments = orgPathSegments(input.pathname);
	if (segments.length === 0) {
		return 'app-root';
	}
	const stream = segments[0] ?? '';
	if (!stream || stream === 'i' || stream === 'new') {
		return 'app-root';
	}
	if (segments.length === 1) {
		return 'stream';
	}
	if (segments.length === 2 && segments[1] === 'new') {
		return 'play';
	}
	if (segments[1] === 'i') {
		if (segments.length === 3 && segments[2]) {
			return 'home';
		}
		if (segments.length === 6) {
			return 'task';
		}
	}
	return 'app-root';
}

export type NavPlaceInput = {
	kind: NavPlaceRouteKind;
	stream: string;
	instance: string;
	jobId: string;
	jobInstanceId: string;
	taskId: string;
	/** Job-level Start — provisional empty / `new` jobInstanceId. */
	provisionalJob: boolean;
	worklist: Worklist | null | undefined;
};

export type NavPlaceChrome = {
	showBack: boolean;
	backTarget: PostCommitTarget;
	primaryLabel: string;
	/**
	 * Muted top line: on Task pages `title · instance` (title alone when
	 * provisional Start has no instance). Null omits the line.
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

function instanceLabelFromWorklist(
	worklist: Worklist | null | undefined,
	jobId: string,
	jobInstanceId: string
): string {
	const job = worklistJobFor(worklist, jobId);
	const inst = job?.instances.find((row) => row.id === jobInstanceId);
	const label = inst?.label?.trim();
	return label || jobInstanceId;
}

/**
 * Place chrome DTO for org nav: label + back law only.
 * Back from Task → instance home; Play → `/my` stream dashboard.
 */
export function resolveNavPlace(input: NavPlaceInput): NavPlaceChrome {
	const {
		kind,
		stream,
		instance,
		jobId,
		jobInstanceId,
		taskId,
		provisionalJob,
		worklist
	} = input;

	if (kind === 'app-root') {
		return {
			showBack: false,
			backTarget: { kind: 'my' },
			primaryLabel: m.fallback_home(),
			jobTitleLine: null
		};
	}

	if (kind === 'stream') {
		return {
			showBack: false,
			backTarget: { kind: 'myStream', stream },
			primaryLabel: m.fallback_home(),
			jobTitleLine: null
		};
	}

	if (kind === 'home') {
		return {
			showBack: false,
			backTarget: { kind: 'home', stream, instance },
			primaryLabel: m.fallback_home(),
			jobTitleLine: null
		};
	}

	const jobTitleLine = authoredJobTitle(worklist, jobId);
	const taskTitle = taskTitleFromWorklist(worklist, jobId, taskId);
	const backTarget = resolveTaskExit(stream, instance, provisionalJob || kind === 'play');

	if (kind === 'play' || provisionalJob) {
		return {
			showBack: true,
			backTarget,
			primaryLabel: taskTitle,
			jobTitleLine
		};
	}

	const instanceLabel = instanceLabelFromWorklist(worklist, jobId, jobInstanceId);
	return {
		showBack: true,
		backTarget,
		primaryLabel: taskTitle,
		jobTitleLine: taskJobContextLine(jobTitleLine, instanceLabel)
	};
}
