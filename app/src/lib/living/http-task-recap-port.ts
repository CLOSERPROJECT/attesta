import * as m from '#lib/paraglide/messages.js';

import type { HttpLinkItem } from './commit-port.js';
import { parseHttpLinkItem } from './parse-http-link-item.js';
import {
	TaskRecapError,
	type TaskRecap,
	type TaskRecapPort,
	type TaskRecapRequest
} from './task-recap-port.js';

export type HttpTaskRecapPortOptions = {
	fetch?: typeof fetch;
	/** Origin prefix; empty = same-origin (Go hosts `/app` + `/api/living/…`). */
	baseUrl?: string;
};

function taskRecapUrl(base: string, req: TaskRecapRequest): string {
	const params = new URLSearchParams({
		jobId: req.jobId,
		taskId: req.taskId,
		jobInstanceId: req.jobInstanceId
	});
	if (req.groupId) {
		params.set('groupId', req.groupId);
	}
	if (req.groupInstanceId !== undefined && req.groupId) {
		params.set('groupInstanceId', req.groupInstanceId);
	}
	return `${base}/api/living/task-recap?${params}`;
}

function parseTaskRecap(raw: unknown): TaskRecap {
	if (!raw || typeof raw !== 'object' || Array.isArray(raw)) {
		throw new TaskRecapError(m.invalid_task_recap_body(), 500);
	}
	const body = raw as Record<string, unknown>;
	const linksRaw = Array.isArray(body.links) ? body.links : [];
	const links: HttpLinkItem[] = [];
	for (const item of linksRaw) {
		const parsed = parseHttpLinkItem(item);
		if (parsed) links.push(parsed);
	}
	const facts =
		body.facts && typeof body.facts === 'object' && !Array.isArray(body.facts)
			? (body.facts as Record<string, unknown>)
			: {};
	return {
		jobId: typeof body.jobId === 'string' ? body.jobId : '',
		jobInstanceId: typeof body.jobInstanceId === 'string' ? body.jobInstanceId : '',
		taskId: typeof body.taskId === 'string' ? body.taskId : '',
		groupId: typeof body.groupId === 'string' ? body.groupId : undefined,
		groupInstanceId: typeof body.groupInstanceId === 'string' ? body.groupInstanceId : undefined,
		timestamp: typeof body.timestamp === 'string' ? body.timestamp : '',
		userId: typeof body.userId === 'string' ? body.userId : '',
		facts,
		links
	};
}

export function createHttpTaskRecapPort(
	options: HttpTaskRecapPortOptions = {}
): TaskRecapPort {
	const fetchFn = options.fetch ?? fetch;
	const base = options.baseUrl ?? '';

	return {
		async fetch(req) {
			const res = await fetchFn(taskRecapUrl(base, req), {
				method: 'GET',
				credentials: 'include',
				headers: { Accept: 'application/json' },
				cache: 'no-store'
			});
			if (!res.ok) {
				const detail = (await res.text().catch(() => '')).trim();
				throw new TaskRecapError(
					detail || m.living_task_recap_failed({ status: res.status }),
					res.status
				);
			}
			return parseTaskRecap(await res.json());
		}
	};
}
