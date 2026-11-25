import * as m from '#lib/paraglide/messages.js';

import type { HttpLinkItem } from './commit-port.js';
import { parseHttpLinkItem } from './parse-http-link-item.js';
import { appendStreamScope } from './stream-scope.js';
import {
	TaskRecapError,
	type TaskRecapCol,
	type TaskRecapPort,
	type TaskRecapRequest,
	type TaskRecapRow,
	type TaskRecapTable
} from './task-recap-port.js';

export type HttpTaskRecapPortOptions = {
	fetch?: typeof fetch;
	/** Origin prefix; empty = same-origin (Go hosts `/app` + `/api/org/…`). */
	baseUrl?: string;
};

function taskRecapUrl(base: string, req: TaskRecapRequest): string {
	const params = appendStreamScope(
		new URLSearchParams({
			jobId: req.jobId,
			taskId: req.taskId,
			jobInstanceId: req.jobInstanceId
		}),
		{ stream: req.stream, instance: req.instance }
	);
	return `${base}/api/org/task-recap?${params}`;
}

function parseColumns(raw: unknown): TaskRecapCol[] {
	if (!Array.isArray(raw)) return [];
	const cols: TaskRecapCol[] = [];
	for (const item of raw) {
		if (!item || typeof item !== 'object' || Array.isArray(item)) continue;
		const body = item as Record<string, unknown>;
		const key = typeof body.key === 'string' ? body.key : '';
		if (!key) continue;
		const title = typeof body.title === 'string' && body.title.trim() !== '' ? body.title : key;
		cols.push({ key, title });
	}
	return cols;
}

function parseLinks(raw: unknown): HttpLinkItem[] | undefined {
	if (!Array.isArray(raw) || raw.length === 0) return undefined;
	const links: HttpLinkItem[] = [];
	for (const item of raw) {
		const parsed = parseHttpLinkItem(item);
		if (parsed) links.push(parsed);
	}
	return links.length > 0 ? links : undefined;
}

function parseRow(raw: unknown): TaskRecapRow {
	if (!raw || typeof raw !== 'object' || Array.isArray(raw)) {
		throw new TaskRecapError(m.invalid_task_recap_body(), 500);
	}
	const body = raw as Record<string, unknown>;
	const cells =
		body.cells && typeof body.cells === 'object' && !Array.isArray(body.cells)
			? (body.cells as Record<string, unknown>)
			: {};
	return {
		key: typeof body.key === 'string' ? body.key : '',
		cells,
		submissionId: typeof body.submissionId === 'string' ? body.submissionId : '',
		timestamp: typeof body.timestamp === 'string' ? body.timestamp : '',
		userId: typeof body.userId === 'string' ? body.userId : '',
		userName: typeof body.userName === 'string' ? body.userName : '',
		links: parseLinks(body.links)
	};
}

function parseTaskRecapTable(raw: unknown): TaskRecapTable {
	if (!raw || typeof raw !== 'object' || Array.isArray(raw)) {
		throw new TaskRecapError(m.invalid_task_recap_body(), 500);
	}
	const body = raw as Record<string, unknown>;
	const mode = body.mode === 'entity' || body.mode === 'submission' ? body.mode : null;
	if (!mode) {
		throw new TaskRecapError(m.invalid_task_recap_body(), 500);
	}
	const rowsRaw = Array.isArray(body.rows) ? body.rows : null;
	if (!rowsRaw) {
		throw new TaskRecapError(m.invalid_task_recap_body(), 500);
	}
	return {
		mode,
		columns: parseColumns(body.columns),
		rows: rowsRaw.map(parseRow)
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
					detail || m.org_task_recap_failed({ status: res.status }),
					res.status
				);
			}
			return parseTaskRecapTable(await res.json());
		}
	};
}
