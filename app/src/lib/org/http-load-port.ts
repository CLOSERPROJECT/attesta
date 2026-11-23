import * as m from '#lib/paraglide/messages.js';

import { LoadDomainError, type LoadDomainPort } from './load-port.js';
import { streamScopeQuery, type StreamScope } from './stream-scope.js';
import type { LoadResult } from './types.js';

export type HttpLoadDomainPortOptions = {
	fetch?: typeof fetch;
	/** Origin prefix; empty = same-origin (Go hosts `/app` + `/api/org/…`). */
	baseUrl?: string;
};

export function createHttpLoadDomainPort(
	options: HttpLoadDomainPortOptions = {}
): LoadDomainPort {
	const fetchFn = options.fetch ?? fetch;
	const base = options.baseUrl ?? '';

	return {
		async load(scope: StreamScope & { instance: string }) {
			const qs = streamScopeQuery({
				stream: scope.stream,
				instance: scope.instance
			});
			const res = await fetchFn(`${base}/api/org/domain?${qs}`, {
				method: 'GET',
				credentials: 'include',
				headers: { Accept: 'application/json' },
				cache: 'no-store'
			});
			if (!res.ok) {
				const detail = (await res.text().catch(() => '')).trim();
				throw new LoadDomainError(
					detail || m.org_domain_load_failed({ status: res.status }),
					res.status
				);
			}
			return (await res.json()) as LoadResult;
		}
	};
}
