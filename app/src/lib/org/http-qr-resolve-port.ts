import * as m from '#lib/paraglide/messages.js';

import {
	parseQrResolveResult,
	QrResolveHttpError,
	type QrResolvePort
} from './qr-resolve-port.js';
import { streamScopeQuery } from './stream-scope.js';

export type HttpQrResolvePortOptions = {
	fetch?: typeof fetch;
	/** Origin prefix; empty = same-origin (Go hosts `/app` + `/api/org/…`). */
	baseUrl?: string;
};

export function createHttpQrResolvePort(
	options: HttpQrResolvePortOptions = {}
): QrResolvePort {
	const fetchFn = options.fetch ?? fetch;
	const base = options.baseUrl ?? '';

	return {
		async resolve(raw, scope) {
			const qs = streamScopeQuery({
				stream: scope.stream,
				instance: scope.instance
			});
			const res = await fetchFn(`${base}/api/org/qr-resolve?${qs}`, {
				method: 'POST',
				credentials: 'include',
				headers: {
					Accept: 'application/json',
					'Content-Type': 'application/json'
				},
				cache: 'no-store',
				body: JSON.stringify({ raw })
			});
			// 200 = navigate+seed; 422 = toastable Config error — both JSON bodies.
			if (res.status === 200 || res.status === 422) {
				return parseQrResolveResult(await res.json());
			}
			const detail = (await res.text().catch(() => '')).trim();
			throw new QrResolveHttpError(
				detail || m.org_qr_resolve_failed({ status: res.status }),
				res.status
			);
		}
	};
}
