import { CommitDomainError, type CommitDomainPort, type CommitResult } from './commit-port.js';

export type HttpCommitDomainPortOptions = {
	fetch?: typeof fetch;
	/** Origin prefix; empty = same-origin (Go hosts `/app` + `/api/living/…`). */
	baseUrl?: string;
};

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
			return (await res.json()) as CommitResult;
		}
	};
}
