import { describe, expect, test } from 'vitest';

import type { TimelineView } from '../timeline/types.js';
import {
	firstActionableAddress,
	soleActionableAddress
} from './timeline-actionable.js';

const base = {
	stream: 'recycling',
	instance: '261102-1',
	label: 'Camion',
	status: 'active'
} as const;

function viewWithActionables(
	hrefs: { substepId: string; href: string }[]
): TimelineView {
	return {
		...base,
		steps: [
			{
				stepId: '1',
				title: 'Step',
				substepCount: hrefs.length,
				expanded: true,
				substeps: hrefs.map((h) => ({
					substepId: h.substepId,
					title: h.substepId,
					status: 'available',
					orgActionable: true,
					href: h.href
				}))
			}
		]
	};
}

describe('soleActionableAddress', () => {
	test('null when none or many; address when exactly one', () => {
		expect(soleActionableAddress({ ...base, steps: [] })).toBeNull();

		const one = viewWithActionables([
			{
				substepId: 'lot_processing.weigh',
				href: '/app/recycling/i/261102-1/lot_processing/DA-1/weigh'
			}
		]);
		expect(soleActionableAddress(one)?.taskId).toBe('weigh');
		expect(firstActionableAddress(one)?.taskId).toBe('weigh');

		const many = viewWithActionables([
			{
				substepId: 'intake.register',
				href: '/app/recycling/i/261102-1/intake/261102-1/register'
			},
			{
				substepId: 'lot_processing.weigh',
				href: '/app/recycling/i/261102-1/lot_processing/DA-1/weigh'
			}
		]);
		expect(soleActionableAddress(many)).toBeNull();
		expect(firstActionableAddress(many)?.taskId).toBe('register');
	});
});
