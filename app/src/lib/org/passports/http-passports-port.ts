import * as m from '#lib/paraglide/messages.js';

import { streamScopeQuery } from '../stream-scope.js';
import {
	PassportsLoadError,
	type PassportsLoadQuery,
	type PassportsPort
} from './passports-port.js';
import type { InstancePassportsView } from './types.js';

export type HttpPassportsPortOptions = {
	fetch?: typeof fetch;
	baseUrl?: string;
};

export function createHttpPassportsPort(
	options: HttpPassportsPortOptions = {}
): PassportsPort {
	const fetchFn = options.fetch ?? fetch;
	const base = options.baseUrl ?? '';

	return {
		async load(query: PassportsLoadQuery) {
			const qs = streamScopeQuery({
				stream: query.stream,
				instance: query.instance
			});
			const res = await fetchFn(`${base}/api/org/passports?${qs}`, {
				method: 'GET',
				credentials: 'include',
				headers: { Accept: 'application/json' },
				cache: 'no-store'
			});
			if (!res.ok) {
				const detail = (await res.text().catch(() => '')).trim();
				throw new PassportsLoadError(
					detail || m.org_passports_load_failed({ status: res.status }),
					res.status
				);
			}
			const body = (await res.json()) as InstancePassportsView;
			return {
				passports: Array.isArray(body.passports) ? body.passports : []
			};
		}
	};
}
