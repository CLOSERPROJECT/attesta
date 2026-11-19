import { createRouter } from 'sv-router';

import { orgSession } from '../org/org-app.svelte.js';
import { runTaskFormBeforeLoad } from '../org/task-form-before-load.js';
import { taskFormMount } from '../org/task-form-mount.svelte.js';
import type { TaskDeepLinkTarget } from '../org/worklist-nav.js';
import layout from './layout.svelte';

// Longer Task paths first; sv-router also requires equal path length per pattern.
// Static `/archive` before `/:jobId/...` so it is not treated as a Job id.
// Task uses tree shape so `hooks.beforeLoad` can attach (flat leaf cannot).
export const { p, navigate, isActive, route } = createRouter({
	layout,
	'/': () => import('./home.svelte'),
	'/archive': () => import('./archive.svelte'),
	'/:jobId/:jobInstanceId/:taskId': {
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
	},
	'/:jobId/:jobInstanceId': () => import('./overview.svelte')
});

/** Gate deny → sv-router `throw navigate(...)` (live-bound after createRouter). */
function navigateDeepLinkDeny(
	target: Exclude<TaskDeepLinkTarget, { kind: 'task' }>
): Promise<unknown> {
	if (target.kind === 'overview') {
		return navigate('/:jobId/:jobInstanceId', {
			params: {
				jobId: target.jobId,
				jobInstanceId: target.jobInstanceId
			}
		});
	}
	return navigate('/');
}
