import { describe, expect, test, vi } from 'vitest';

vi.mock('./index.js', () => ({
	p: (path: string) => (path === '/' ? '/app' : `/app${path}`),
	navigate: vi.fn()
}));

import { p } from './index.js';
import { hrefOrgTarget } from './navigate-target.js';

describe('hrefOrgTarget', () => {
	test('home target matches p("/")', () => {
		const href = hrefOrgTarget({ kind: 'home' });
		expect(href).toBe(p('/'));
		expect(href).toContain('/app');
	});

	test('archive target matches p("/archive")', () => {
		const href = hrefOrgTarget({ kind: 'archive' });
		expect(href).toBe(p('/archive'));
		expect(href).toContain('/app/archive');
	});

});
