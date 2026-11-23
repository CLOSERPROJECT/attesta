import { describe, expect, test } from 'vitest';

import {
	interpretQrResolve,
	takeColdEntityIdRaw
} from './qr-resolve-apply.js';

describe('interpretQrResolve', () => {
	test('one candidate → flat weigh target + draft key + seed', () => {
		const got = interpretQrResolve(
			{
				navigate: {
					jobId: 'line',
					jobInstanceId: 'L1',
					taskId: 'weigh',
					streamInstanceId: '261102-1'
				},
				seed: { palletId: 'load.1' }
			},
			'recycling'
		);
		expect(got).toEqual({
			kind: 'navigate',
			target: {
				kind: 'task',
				stream: 'recycling',
				instance: '261102-1',
				jobId: 'line',
				jobInstanceId: 'L1',
				taskId: 'weigh'
			},
			taskPath: 'line/weigh',
			draftInstanceKey: 'L1',
			seed: { palletId: 'load.1' }
		});
	});

	test('server error message surfaces for toast', () => {
		const got = interpretQrResolve(
			{
				error: {
					code: 'zero',
					message: 'Nessuna pesatura disponibile per questo bancale'
				}
			},
			'recycling'
		);
		expect(got).toEqual({
			kind: 'error',
			message: 'Nessuna pesatura disponibile per questo bancale'
		});
	});
});

describe('takeColdEntityIdRaw', () => {
	test('captures href once and strips entityId from URL', () => {
		const href = 'https://plant.example/app/recycling?entityId=P9&x=1';
		const got = takeColdEntityIdRaw(href);
		expect(got.raw).toBe(href);
		expect(got.strippedHref).toBe('/app/recycling?x=1');
	});

	test('absent entityId yields null raw', () => {
		expect(takeColdEntityIdRaw('https://plant.example/app/recycling').raw).toBeNull();
		expect(
			takeColdEntityIdRaw('https://plant.example/app/recycling?entityId=').raw
		).toBeNull();
	});
});
