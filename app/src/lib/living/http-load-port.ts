import { LoadDomainError, type LoadDomainPort } from './load-port.js';
import type { LoadResult } from './types.js';

export type HttpLoadDomainPortOptions = {
	fetch?: typeof fetch;
	/** Origin prefix; empty = same-origin (Go hosts `/app` + `/api/living/…`). */
	baseUrl?: string;
};

/** Browser Adapter: GET /api/living/domain → Worklist + bundles + config metadata. */
export function createHttpLoadDomainPort(
	options: HttpLoadDomainPortOptions = {}
): LoadDomainPort {
	const fetchFn = options.fetch ?? fetch;
	const base = options.baseUrl ?? '';

	return {
		async load() {
			const res = await fetchFn(`${base}/api/living/domain`, {
				method: 'GET',
				credentials: 'include',
				headers: { Accept: 'application/json' },
				cache: 'no-store'
			});
			if (!res.ok) {
				const detail = (await res.text().catch(() => '')).trim();
				throw new LoadDomainError(
					detail || `living domain load failed (${res.status})`,
					res.status
				);
			}
			return (await res.json()) as LoadResult;
		}
	};
}
