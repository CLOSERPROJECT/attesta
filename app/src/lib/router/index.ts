import { createRouter } from 'sv-router';

import { orgSession } from '../org/org-app.svelte.js';
import { runTaskFormBeforeLoad } from '../org/task-form-before-load.js';
import { taskFormMount } from '../org/task-form-mount.svelte.js';
import type { TaskDeepLinkTarget } from '../org/worklist-nav.js';
import { navigateOrgTargetAsync } from './navigate-target.js';
import layout from './layout.svelte';

// Longer Task paths first; sv-router also requires equal path length per pattern.
// Task uses tree shape so `hooks.beforeLoad` can attach (flat leaf cannot).
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
				await runTaskFormBeforeLoad(pathname, {
					mount: taskFormMount,
					session: orgSession,
					navigateRedirect: navigateDeepLinkDeny
				});
			}
		}
	}
});

/** Gate deny → sv-router `throw navigate(...)` (live-bound after createRouter). */
function navigateDeepLinkDeny(
	target: Exclude<TaskDeepLinkTarget, { kind: 'task' }>
): Promise<unknown> {
	return navigateOrgTargetAsync(target);
}
