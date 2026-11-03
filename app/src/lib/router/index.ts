import { createRouter } from 'sv-router';

import layout from './layout.svelte';

// Task (3 segs) before overview (2 segs); sv-router also requires equal path length.
export const { p, navigate, isActive, route } = createRouter({
	layout,
	'/': () => import('./home.svelte'),
	'/:jobId/:jobInstanceId/:taskId': () => import('./task.svelte'),
	'/:jobId/:jobInstanceId': () => import('./overview.svelte')
});
