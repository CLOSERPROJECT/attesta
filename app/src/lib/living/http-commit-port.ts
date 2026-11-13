import * as m from '#lib/paraglide/messages.js';

import {
	CommitDomainError,
	type CommitDomainPort,
	type CommitNavigation,
	type CommitResult,
	type ContinueNavigation,
	type HttpLinkItem
} from './commit-port.js';
import { parseHttpLinkItem } from './parse-http-link-item.js';
import { emptyWorklist, type TaskFormBundle, type Worklist } from './types.js';

export type HttpCommitDomainPortOptions = {
	fetch?: typeof fetch;
	/** Origin prefix; empty = same-origin (Go hosts `/app` + `/api/living/…`). */
	baseUrl?: string;
};

function parseWorklist(raw: unknown): Worklist {
	if (!raw || typeof raw !== 'object' || Array.isArray(raw)) return emptyWorklist();
	const jobs = (raw as { jobs?: unknown }).jobs;
	if (!Array.isArray(jobs)) return emptyWorklist();
	return { jobs: jobs as Worklist['jobs'] };
}

function parseOptionalTaskFormBundle(raw: unknown): TaskFormBundle | null {
	if (raw == null || typeof raw !== 'object' || Array.isArray(raw)) return null;
	return raw as TaskFormBundle;
}

function parseCommitNavigation(raw: unknown): CommitNavigation {
	if (raw === 'stay' || raw === 'task' || raw === 'overview' || raw === 'home') {
		return raw;
	}
	return 'home';
}

function parseContinueNavigation(raw: unknown): ContinueNavigation | null {
	if (raw === 'task' || raw === 'overview' || raw === 'home') return raw;
	return null;
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
	const nextBundle = parseOptionalTaskFormBundle(
		(raw as { nextTaskFormBundle?: unknown })?.nextTaskFormBundle
	);
	const navigation = parseCommitNavigation(
		(raw as { navigation?: unknown })?.navigation
	);
	const continueNavigation =
		navigation === 'stay'
			? parseContinueNavigation(
					(raw as { continueNavigation?: unknown })?.continueNavigation
				)
			: null;
	return {
		...body,
		worklist: parseWorklist((raw as { worklist?: unknown })?.worklist),
		navigation,
		continueNavigation,
		nextTaskFormBundle: nextBundle,
		links,
		httpErrors
	};
}

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
					value: req.value,
					mint: req.mint ?? { kind: 'none' }
				})
			});
			if (!res.ok) {
				const detail = (await res.text().catch(() => '')).trim();
				throw new CommitDomainError(
					detail || m.living_commit_failed({ status: res.status }),
					res.status
				);
			}
			return parseCommitResult(await res.json());
		}
	};
}
