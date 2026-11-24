import { describe, expect, test, vi } from 'vitest';

import { createHttpTimelinePort } from './http-timeline-port.js';
import { TimelineLoadError } from './timeline-port.js';
import type { TimelineView } from './types.js';

const bootView: TimelineView = {
	stream: 'recycling',
	instance: '',
	mode: 'boot',
	label: 'Recycling',
	status: 'active',
	steps: []
};

const instanceView: TimelineView = {
	stream: 'recycling',
	instance: '261102-1',
	mode: 'instance',
	label: 'Camion',
	status: 'active',
	steps: []
};

describe('HttpTimelinePort', () => {
	test('GET /api/org/timeline with stream only (boot)', async () => {
		const fetchFn = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
			expect(String(input)).toBe('/api/org/timeline?stream=recycling');
			expect(init?.method).toBe('GET');
			expect(init?.credentials).toBe('include');
			return new Response(JSON.stringify(bootView), {
				status: 200,
				headers: { 'Content-Type': 'application/json' }
			});
		});

		const port = createHttpTimelinePort({ fetch: fetchFn, baseUrl: '' });
		const got = await port.load({ stream: 'recycling' });
		expect(got).toEqual(bootView);
		expect(fetchFn).toHaveBeenCalledOnce();
	});

	test('GET /api/org/timeline with stream and instance', async () => {
		const fetchFn = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
			expect(String(input)).toBe(
				'/api/org/timeline?stream=recycling&instance=261102-1'
			);
			expect(init?.method).toBe('GET');
			return new Response(JSON.stringify(instanceView), {
				status: 200,
				headers: { 'Content-Type': 'application/json' }
			});
		});

		const port = createHttpTimelinePort({ fetch: fetchFn, baseUrl: '' });
		const got = await port.load({ stream: 'recycling', instance: '261102-1' });
		expect(got).toEqual(instanceView);
		expect(fetchFn).toHaveBeenCalledOnce();
	});

	test('non-OK response becomes TimelineLoadError', async () => {
		const fetchFn = vi.fn(
			async () => new Response('organization required', { status: 403 })
		);
		const port = createHttpTimelinePort({ fetch: fetchFn });
		await expect(port.load({ stream: 'recycling' })).rejects.toBeInstanceOf(
			TimelineLoadError
		);
		await expect(port.load({ stream: 'recycling' })).rejects.toMatchObject({
			status: 403
		});
	});

	test('preserves roleBadges from timeline JSON', async () => {
		const withBadges: TimelineView = {
			...instanceView,
			steps: [
				{
					stepId: 'intake',
					title: 'Intake',
					substepCount: 1,
					expanded: true,
					substeps: [
						{
							substepId: 'intake.truckArrive',
							title: 'Truck arrive',
							status: 'available',
							roleBadges: [
								{ id: 'dep1', label: 'Depot', palette: 'blue' },
								{ id: 'ops', label: 'Ops', palette: 'emerald' }
							]
						}
					]
				}
			]
		};
		const fetchFn = vi.fn(
			async () =>
				new Response(JSON.stringify(withBadges), {
					status: 200,
					headers: { 'Content-Type': 'application/json' }
				})
		);
		const port = createHttpTimelinePort({ fetch: fetchFn });
		const got = await port.load({ stream: 'recycling', instance: '261102-1' });
		expect(got.steps[0]?.substeps?.[0]?.roleBadges).toEqual([
			{ id: 'dep1', label: 'Depot', palette: 'blue' },
			{ id: 'ops', label: 'Ops', palette: 'emerald' }
		]);
	});
});
