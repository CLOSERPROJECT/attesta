import { createRouter } from 'sv-router';

import layout from './layout.svelte';

export const { p, navigate, isActive, route } = createRouter({
	layout,
	'/': () => import('./home.svelte'),
	'/about': () => import('./about.svelte')
});
