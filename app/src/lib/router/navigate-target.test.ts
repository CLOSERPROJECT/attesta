import { describe, expect, test, vi } from 'vitest';

vi.mock('./index.js', () => ({
	p: (path: string, opts?: { params?: Record<string, string> }) => {
		if (!opts?.params) return path === '/' ? '/app' : `/app${path}`;
		let out = path;
		for (const [key, value] of Object.entries(opts.params)) {
			out = out.replace(`:${key}`, value);
		}
		return `/app${out}`;
	},
	navigate: vi.fn()
}));

import {
	hrefOrgTarget,
	orgMyInstanceTimelinePath,
	orgMyStreamDashboardPath
} from './navigate-target.js';

describe('hrefOrgTarget', () => {
	test('my target is /my', () => {
		expect(hrefOrgTarget({ kind: 'my' })).toBe('/my');
	});

	test('myStream uses org catalog key', () => {
		expect(hrefOrgTarget({ kind: 'myStream', stream: 'recycling' })).toBe(
			'/my/streams/org-recycling/'
		);
		expect(orgMyStreamDashboardPath('recycling')).toBe('/my/streams/org-recycling/');
	});

	test('home target is /my instance timeline', () => {
		expect(
			hrefOrgTarget({
				kind: 'home',
				stream: 'recycling',
				instance: '261102-1'
			})
		).toBe('/my/streams/org-recycling/instance/261102-1');
		expect(orgMyInstanceTimelinePath('recycling', '261102-1')).toBe(
			'/my/streams/org-recycling/instance/261102-1'
		);
	});

	test('task target keeps stream instance prefix', () => {
		expect(
			hrefOrgTarget({
				kind: 'task',
				stream: 'recycling',
				instance: '261102-1',
				jobId: 'intake',
				jobInstanceId: '261102-1',
				taskId: 'register'
			})
		).toBe('/app/recycling/i/261102-1/intake/261102-1/register');
	});
});
