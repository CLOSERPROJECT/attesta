import { describe, expect, test } from 'vitest';

import {
	interpretQrResolve,
	qrDraftCoords,
	takeColdEntityIdRaw
} from './qr-resolve-apply.js';

describe('interpretQrResolve', () => {
	test('one candidate → nested weigh target + provisional draft key + seed', () => {
		const got = interpretQrResolve({
			navigate: {
				jobId: 'line',
				jobInstanceId: 'L1',
				groupId: 'palletWeight',
				groupInstanceId: 'new',
				taskId: 'weigh'
			},
			seed: { palletId: 'load.1' }
		});
		expect(got).toEqual({
			kind: 'navigate',
			target: {
				kind: 'task',
				jobId: 'line',
				jobInstanceId: 'L1',
				groupId: 'palletWeight',
				groupInstanceId: 'new',
				taskId: 'weigh'
			},
			taskPath: 'line/palletWeight/new/weigh',
			draftInstanceKey: 'L1::palletWeight::',
			seed: { palletId: 'load.1' }
		});
	});

	test('open group instance uses real groupInstanceId in draft key', () => {
		const coords = qrDraftCoords({
			jobId: 'line',
			jobInstanceId: 'L1',
			groupId: 'palletWeight',
			groupInstanceId: 'pw-3',
			taskId: 'weigh'
		});
		expect(coords).toEqual({
			taskPath: 'line/palletWeight/pw-3/weigh',
			draftInstanceKey: 'L1::palletWeight::pw-3'
		});
	});

	test('server error message surfaces for toast', () => {
		const got = interpretQrResolve({
			error: {
				code: 'zero',
				message: 'Nessuna pesatura disponibile per questo bancale'
			}
		});
		expect(got).toEqual({
			kind: 'error',
			message: 'Nessuna pesatura disponibile per questo bancale'
		});
	});
});

describe('takeColdEntityIdRaw', () => {
	test('captures href once and strips entityId from URL', () => {
		const href = 'https://plant.example/app?entityId=P9&x=1';
		const got = takeColdEntityIdRaw(href);
		expect(got.raw).toBe(href);
		expect(got.strippedHref).toBe('/app?x=1');
	});

	test('absent entityId yields null raw', () => {
		expect(takeColdEntityIdRaw('https://plant.example/app').raw).toBeNull();
		expect(takeColdEntityIdRaw('https://plant.example/app?entityId=').raw).toBeNull();
	});
});
