import { createRouter } from 'sv-router';

import layout from './layout.svelte';

// Longer Task paths first; sv-router also requires equal path length per pattern.
export const { p, navigate, isActive, route } = createRouter({
	layout,
	'/': () => import('./home.svelte'),
	'/:jobId/:jobInstanceId/:groupId/:groupInstanceId/:taskId': () =>
		import('./task.svelte'),
	'/:jobId/:jobInstanceId/:taskId': () => import('./task.svelte'),
	'/:jobId/:jobInstanceId': () => import('./overview.svelte')
});
