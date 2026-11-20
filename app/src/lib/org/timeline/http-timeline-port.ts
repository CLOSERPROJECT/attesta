import * as m from '#lib/paraglide/messages.js';

import { streamScopeQuery, type StreamScope } from '../stream-scope.js';
import { TimelineLoadError, type TimelinePort } from './timeline-port.js';
import type { TimelineView } from './types.js';

export type HttpTimelinePortOptions = {
	fetch?: typeof fetch;
	baseUrl?: string;
};

export function createHttpTimelinePort(
	options: HttpTimelinePortOptions = {}
): TimelinePort {
	const fetchFn = options.fetch ?? fetch;
	const base = options.baseUrl ?? '';

	return {
		async load(scope: StreamScope & { instance: string }) {
			const qs = streamScopeQuery({
				stream: scope.stream,
				instance: scope.instance
			});
			const res = await fetchFn(`${base}/api/org/timeline?${qs}`, {
				method: 'GET',
				credentials: 'include',
				headers: { Accept: 'application/json' },
				cache: 'no-store'
			});
			if (!res.ok) {
				const detail = (await res.text().catch(() => '')).trim();
				throw new TimelineLoadError(
					detail || m.org_timeline_load_failed({ status: res.status }),
					res.status
				);
			}
			return (await res.json()) as TimelineView;
		}
	};
}
