import {
	CommitDomainError,
	type CommitDomainPort,
	type CommitResult,
	type HttpLinkItem
} from './commit-port.js';

export type HttpCommitDomainPortOptions = {
	fetch?: typeof fetch;
	/** Origin prefix; empty = same-origin (Go hosts `/app` + `/api/living/…`). */
	baseUrl?: string;
};

function parseHttpLinkItem(raw: unknown): HttpLinkItem | null {
	if (!raw || typeof raw !== 'object' || Array.isArray(raw)) return null;
	const rec = raw as Record<string, unknown>;
	const label = typeof rec.label === 'string' ? rec.label : '';
	const url = typeof rec.url === 'string' ? rec.url : '';
	const action = rec.action === 'link' || rec.action === 'download' ? rec.action : null;
	if (!label || !url || !action) return null;
	const item: HttpLinkItem = { label, url, action };
	if (typeof rec.filename === 'string' && rec.filename.length > 0) {
		item.filename = rec.filename;
	}
	return item;
}

/** Normalize Commit JSON so SPA always sees arrays for links / httpErrors. */
export function parseCommitResult(raw: unknown): CommitResult {
	const body = raw as CommitResult;
	const linksRaw = Array.isArray((raw as { links?: unknown })?.links)
		? ((raw as { links: unknown[] }).links)
		: [];
	const errorsRaw = Array.isArray((raw as { httpErrors?: unknown })?.httpErrors)
		? ((raw as { httpErrors: unknown[] }).httpErrors)
		: [];
	const links: HttpLinkItem[] = [];
	for (const item of linksRaw) {
		const parsed = parseHttpLinkItem(item);
		if (parsed) links.push(parsed);
	}
	const httpErrors = errorsRaw.filter((e): e is string => typeof e === 'string');
	return {
		...body,
		links,
		httpErrors
	};
}

/** Browser Adapter: POST /api/living/commit → Domain. Drafts stay client-side. */
export function createHttpCommitDomainPort(
	options: HttpCommitDomainPortOptions = {}
): CommitDomainPort {
	const fetchFn = options.fetch ?? fetch;
	const base = options.baseUrl ?? '';

	return {
		async commit(req) {
			const res = await fetchFn(`${base}/api/living/commit`, {
				method: 'POST',
				credentials: 'include',
				headers: {
					Accept: 'application/json',
					'Content-Type': 'application/json'
				},
				cache: 'no-store',
				body: JSON.stringify({
					taskPath: req.taskPath,
					jobInstanceId: req.jobInstanceId,
					value: req.value
				})
			});
			if (!res.ok) {
				const detail = (await res.text().catch(() => '')).trim();
				throw new CommitDomainError(
					detail || `living commit failed (${res.status})`,
					res.status
				);
			}
			return parseCommitResult(await res.json());
		}
	};
}
