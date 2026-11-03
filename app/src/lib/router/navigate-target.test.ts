import { describe, expect, test, vi } from 'vitest';

vi.mock('./index.js', () => ({
	p: (path: string) => (path === '/' ? '/app' : `/app${path}`),
	navigate: vi.fn()
}));

import { p } from './index.js';
import { hrefLivingTarget } from './navigate-target.js';

describe('hrefLivingTarget', () => {
	test('home target matches p("/")', () => {
		const href = hrefLivingTarget({ kind: 'home' });
		expect(href).toBe(p('/'));
		expect(href).toContain('/app');
	});
});
