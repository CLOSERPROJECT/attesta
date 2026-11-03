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

	test('nested task target includes group coordinates', () => {
		const href = hrefLivingTarget({
			kind: 'task',
			jobId: 'intake',
			jobInstanceId: 'T1',
			groupId: 'lots',
			groupInstanceId: 'new',
			taskId: 'register'
		});
		expect(href).toBe(
			p('/:jobId/:jobInstanceId/:groupId/:groupInstanceId/:taskId', {
				params: {
					jobId: 'intake',
					jobInstanceId: 'T1',
					groupId: 'lots',
					groupInstanceId: 'new',
					taskId: 'register'
				}
			})
		);
	});
});
