import { createRouter } from 'sv-router';

import {
	BOARD_TASK_SEARCH_KEY,
	encodeBoardTaskParam
} from '../org/board/board-url.js';
import { orgRouteCoordsFromPath } from '../org/nav-place.js';
import { orgSession } from '../org/org-app.svelte.js';
import { applyUrlSeed, takeSeedQueryParams } from '../org/seed-query.js';
import { normalizeTaskAddress } from '../org/task-address.js';
import { resolveTaskDeepLink } from '../org/worklist-nav.js';
import layout from './layout.svelte';

// Longer Task paths first; sv-router also requires equal path length per pattern.
// Legacy Task paths soft-land on the board with `?task=` (no separate Task remount).
export const { p, navigate, isActive, route } = createRouter({
	layout,
	'/': () => import('./root.svelte'),
	'/:stream': () => import('./stream.svelte'),
	'/:stream/new': () => import('./play.svelte'),
	'/:stream/i/:instance': () => import('./home.svelte'),
	'/:stream/i/:instance/:jobId/:jobInstanceId/:taskId': {
		'/': () => import('./task.svelte'),
		hooks: {
			async beforeLoad({ pathname }) {
				const coords = orgRouteCoordsFromPath({ pathname });
				const addr = normalizeTaskAddress(coords);

				const href =
					typeof window !== 'undefined' ? window.location.href : '';
				if (href) {
					const { seed, strippedHref } = takeSeedQueryParams(href);
					if (Object.keys(seed).length > 0) {
						applyUrlSeed(seed, addr, orgSession);
						window.history.replaceState(window.history.state, '', strippedHref);
					}
				}

				if (addr.stream && addr.instance) {
					await orgSession.hydrate({
						stream: addr.stream,
						instance: addr.instance
					});
				}

				const worklist = orgSession.result?.worklist;
				const gate = resolveTaskDeepLink(
					addr.stream,
					addr.instance,
					addr.jobId,
					addr.jobInstanceId,
					addr.taskId,
					addr.provisionalJob,
					worklist
				);
				if (gate.kind !== 'task') {
					if (gate.kind === 'home') {
						throw await navigate('/:stream/i/:instance', {
							params: {
								stream: gate.stream,
								instance: gate.instance
							},
							replace: true
						});
					}
					if (gate.kind === 'myStream' || gate.kind === 'my') {
						const href =
							gate.kind === 'my'
								? '/my'
								: `/my/streams/org-${gate.stream}/`;
						if (typeof window !== 'undefined') {
							window.location.assign(href);
						}
						return;
					}
					return;
				}

				const search: Record<string, string> = {
					[BOARD_TASK_SEARCH_KEY]: encodeBoardTaskParam({
						jobId: addr.jobId,
						jobInstanceId: addr.jobInstanceId,
						taskId: addr.taskId
					})
				};
				if (typeof window !== 'undefined') {
					for (const [key, value] of new URL(window.location.href).searchParams) {
						if (key === BOARD_TASK_SEARCH_KEY || key.startsWith('seed.')) continue;
						search[key] = value;
					}
				}

				throw await navigate('/:stream/i/:instance', {
					params: {
						stream: addr.stream,
						instance: addr.instance
					},
					search,
					replace: true
				});
			}
		}
	}
});
