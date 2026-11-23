import { describe, expect, test } from 'vitest';

import type { TimelineSubstep, TimelineView } from '../timeline/types.js';
import {
	firstActionableAddress,
	recapAddressesFromTimeline,
	soleActionableAddress,
	timelineSubstepShown
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

describe('recapAddressesFromTimeline', () => {
	test('includes submitMany and substeps with submissions', () => {
		const view: TimelineView = {
			...base,
			steps: [
				{
					stepId: '1',
					title: 'Step',
					substepCount: 3,
					expanded: true,
					substeps: [
						{
							substepId: 'intake.register',
							title: 'Register',
							status: 'done',
							href: '/app/recycling/i/261102-1/intake/261102-1/register',
							submissions: [{ timestamp: '2024-01-01T00:00:00Z' }]
						},
						{
							substepId: 'intake.photos',
							title: 'Photos',
							status: 'available',
							orgActionable: true,
							submitMany: true,
							href: '/app/recycling/i/261102-1/intake/261102-1/photos'
						},
						{
							substepId: 'intake.skip',
							title: 'Skip',
							status: 'available',
							href: '/app/recycling/i/261102-1/intake/261102-1/skip'
						}
					]
				}
			]
		};
		const addrs = recapAddressesFromTimeline(view);
		expect(addrs.map((a) => a.taskId).sort()).toEqual(['photos', 'register']);
	});
});

describe('timelineSubstepShown', () => {
	const stream = 'recycling';
	const instance = '261102-1';
	const lockedOtherLot: TimelineSubstep = {
		substepId: 'lot_processing.weigh',
		title: 'Pesa',
		status: 'locked',
		orgActionable: false,
		href: `/app/${stream}/i/${instance}/lot_processing/CO-1/weigh`
	};
	const lockedFocusedLot: TimelineSubstep = {
		...lockedOtherLot,
		href: `/app/${stream}/i/${instance}/lot_processing/DA-1/weigh`
	};
	const actionable: TimelineSubstep = {
		...lockedFocusedLot,
		status: 'available',
		orgActionable: true
	};
	const focusDA = {
		jobId: 'lot_processing',
		jobInstanceId: 'DA-1',
		taskId: 'weigh'
	} as const;

	test('showAll and orgActionable; focus is instance-scoped', () => {
		expect(timelineSubstepShown(lockedOtherLot, { showAll: true })).toBe(true);
		expect(timelineSubstepShown(lockedOtherLot, { showAll: false })).toBe(false);
		expect(timelineSubstepShown(actionable, { showAll: false })).toBe(true);
		expect(
			timelineSubstepShown(lockedFocusedLot, {
				showAll: false,
				focus: focusDA,
				stream,
				instance
			})
		).toBe(true);
		expect(
			timelineSubstepShown(lockedOtherLot, {
				showAll: false,
				focus: focusDA,
				stream,
				instance
			})
		).toBe(false);
		expect(
			timelineSubstepShown(
				{ ...lockedOtherLot, href: undefined },
				{ showAll: false, focus: focusDA, stream, instance }
			)
		).toBe(false);
	});
});
