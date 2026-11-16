import { createRouter } from 'sv-router';

import layout from './layout.svelte';

// Longer Task paths first; sv-router also requires equal path length per pattern.
// Static `/archive` before `/:jobId/...` so it is not treated as a Job id.
export const { p, navigate, isActive, route } = createRouter({
	layout,
	'/': () => import('./home.svelte'),
	'/archive': () => import('./archive.svelte'),
	'/:jobId/:jobInstanceId/:taskId': () => import('./task.svelte'),
	'/:jobId/:jobInstanceId': () => import('./overview.svelte')
});
