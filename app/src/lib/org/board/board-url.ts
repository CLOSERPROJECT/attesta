import {
	normalizeTaskAddress,
	taskRouteInstanceSegment,
	type TaskAddress
} from '../task-address.js';
import { orgPathSegments } from '../nav-place.js';

/**
 * Soft SPA focus on the timeline board: `?task=jobId/jobInstanceId/taskId`
 * on `/{stream}/new` or `/{stream}/i/{instance}` (provisional segment is `new`).
 * Legacy task paths rewrite to this scheme so the board stays mounted.
 */

export type BoardTaskFocus = {
	jobId: string;
	jobInstanceId: string;
	taskId: string;
};

const TASK_PARAM = 'task';

export function encodeBoardTaskParam(focus: BoardTaskFocus): string {
	const jobInstanceId = taskRouteInstanceSegment(focus.jobInstanceId);
	return `${focus.jobId}/${jobInstanceId}/${focus.taskId}`;
}

export function parseBoardTaskParam(raw: string | null | undefined): BoardTaskFocus | null {
	if (!raw) return null;
	const parts = raw.split('/').filter(Boolean);
	if (parts.length !== 3) return null;
	const [jobId, jobInstanceId, taskId] = parts;
	if (!jobId || !jobInstanceId || !taskId) return null;
	return { jobId, jobInstanceId, taskId };
}

export function boardTaskFromAddress(addr: TaskAddress): BoardTaskFocus {
	return {
		jobId: addr.jobId,
		jobInstanceId: addr.jobInstanceId,
		taskId: addr.taskId
	};
}

export function addressFromBoardTask(
	stream: string,
	instance: string,
	focus: BoardTaskFocus
): TaskAddress {
	return normalizeTaskAddress({
		stream,
		instance,
		jobId: focus.jobId,
		jobInstanceId: focus.jobInstanceId,
		taskId: focus.taskId
	});
}

export function readBoardTaskFromSearch(
	search: string | URLSearchParams | Record<string, string | number | boolean>
): BoardTaskFocus | null {
	if (typeof search === 'string') {
		const q = search.startsWith('?') ? search.slice(1) : search;
		return parseBoardTaskParam(new URLSearchParams(q).get(TASK_PARAM));
	}
	if (search instanceof URLSearchParams) {
		return parseBoardTaskParam(search.get(TASK_PARAM));
	}
	const raw = search[TASK_PARAM];
	return parseBoardTaskParam(raw == null ? null : String(raw));
}

/** Build board href with optional `?task=`; drops the param when focus is null. */
export function boardHrefWithTask(pathname: string, focus: BoardTaskFocus | null): string {
	const url = new URL(pathname, 'http://board.local');
	if (focus) {
		url.searchParams.set(TASK_PARAM, encodeBoardTaskParam(focus));
	} else {
		url.searchParams.delete(TASK_PARAM);
	}
	const qs = url.searchParams.toString();
	return qs ? `${url.pathname}?${qs}` : url.pathname;
}

export function focusMatchesAddress(focus: BoardTaskFocus | null, addr: TaskAddress): boolean {
	if (!focus) return false;
	const normalized = addressFromBoardTask(addr.stream, addr.instance, focus);
	return (
		normalized.jobId === addr.jobId &&
		normalized.taskId === addr.taskId &&
		normalized.jobInstanceId === addr.jobInstanceId &&
		normalized.provisionalJob === addr.provisionalJob
	);
}

/**
 * Parse a timeline/card href (with or without `/app` base) into a Task address
 * plus optional `seed.*` query bag.
 */
export function parseTaskHref(
	href: string,
	fallbackStream = '',
	fallbackInstance = ''
): { address: TaskAddress; seed: Record<string, string> } | null {
	let path = href.trim();
	if (!path) return null;
	let seed: Record<string, string> = {};
	try {
		const asUrl = new URL(path, 'http://board.local');
		path = asUrl.pathname;
		for (const [key, value] of asUrl.searchParams.entries()) {
			if (key.startsWith('seed.')) {
				const field = key.slice('seed.'.length);
				if (field) seed[field] = value;
			}
		}
	} catch {
		const q = path.indexOf('?');
		if (q >= 0) {
			const params = new URLSearchParams(path.slice(q + 1));
			path = path.slice(0, q);
			for (const [key, value] of params.entries()) {
				if (key.startsWith('seed.')) {
					const field = key.slice('seed.'.length);
					if (field) seed[field] = value;
				}
			}
		}
	}
	if (path.startsWith('/app/')) {
		path = path.slice('/app'.length);
	}
	const segments = orgPathSegments(path);
	if (segments.length === 6 && segments[1] === 'i') {
		return {
			address: normalizeTaskAddress({
				stream: segments[0] ?? fallbackStream,
				instance: segments[2] ?? fallbackInstance,
				jobId: segments[3] ?? '',
				jobInstanceId: segments[4] ?? '',
				taskId: segments[5] ?? ''
			}),
			seed
		};
	}
	return null;
}

export const BOARD_TASK_SEARCH_KEY = TASK_PARAM;
