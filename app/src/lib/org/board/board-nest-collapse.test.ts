import { describe, expect, test } from 'vitest';

import { BoardNestCollapse } from './board-nest-collapse.svelte.js';

describe('BoardNestCollapse', () => {
	// Epoch signals substep details only; steps/rows do not listen.
	test('collapseAll bumps epoch from 0', () => {
		const nest = new BoardNestCollapse();
		expect(nest.epoch).toBe(0);
		nest.collapseAll();
		expect(nest.epoch).toBe(1);
		nest.collapseAll();
		expect(nest.epoch).toBe(2);
	});
});
