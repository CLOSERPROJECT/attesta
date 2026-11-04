import * as m from '#lib/paraglide/messages.js';

import { CommitDomainError } from './commit-port.js';
import type { CloseGroupPort } from './group-close-port.js';
import { parseCommitResult } from './http-commit-port.js';

export type HttpGroupClosePortOptions = {
	fetch?: typeof fetch;
	baseUrl?: string;
};

export function createHttpCloseGroupPort(
	options: HttpGroupClosePortOptions = {}
): CloseGroupPort {
	return {
		close: async (req) => {
			const fetchFn = options.fetch ?? fetch;
			const base = options.baseUrl ?? '';
			const res = await fetchFn(`${base}/api/living/close`, {
				method: 'POST',
				credentials: 'include',
				headers: {
					Accept: 'application/json',
					'Content-Type': 'application/json'
				},
				cache: 'no-store',
				body: JSON.stringify(req)
			});
			if (!res.ok) {
				const detail = (await res.text().catch(() => '')).trim();
				throw new CommitDomainError(
					detail || m.living_close_failed({ status: res.status }),
					res.status
				);
			}
			return parseCommitResult(await res.json());
		}
	};
}
