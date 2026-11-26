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
	orgMyStreamDashboardPath,
	orgSpaInstanceHomePath
} from './navigate-target.js';

describe('hrefOrgTarget', () => {
	test('my target is /my', () => {
		expect(hrefOrgTarget({ kind: 'my' })).toBe('/my');
	});

	test('myStream uses org catalog key', () => {
		expect(hrefOrgTarget({ kind: 'myStream', stream: 'pv-recovery' })).toBe(
			'/my/streams/org-pv-recovery/'
		);
		expect(orgMyStreamDashboardPath('pv-recovery')).toBe('/my/streams/org-pv-recovery/');
	});

	test('home target is SPA instance timeline', () => {
		expect(
			hrefOrgTarget({
				kind: 'home',
				stream: 'pv-recovery',
				instance: '261102-1'
			})
		).toBe('/app/pv-recovery/i/261102-1');
		expect(orgSpaInstanceHomePath('pv-recovery', '261102-1')).toBe(
			'/app/pv-recovery/i/261102-1'
		);
		expect(orgMyInstanceTimelinePath('pv-recovery', '261102-1')).toBe(
			'/app/pv-recovery/i/261102-1'
		);
	});

	test('task target is board soft-focus ?task=', () => {
		expect(
			hrefOrgTarget({
				kind: 'task',
				stream: 'pv-recovery',
				instance: '261102-1',
				jobId: 'intake',
				jobInstanceId: '261102-1',
				taskId: 'register'
			})
		).toBe(
			'/app/pv-recovery/i/261102-1?task=intake%2F261102-1%2Fregister'
		);
	});
});
