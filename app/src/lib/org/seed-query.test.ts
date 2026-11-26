import { describe, expect, test } from 'vitest';

import { provisionalSeedDraftKey } from './drafts.js';
import {
	applySeedQueryParams,
	applyUrlSeed,
	takeSeedQueryParams
} from './seed-query.js';

describe('takeSeedQueryParams', () => {
	test('collects seed.* params and strips them from the URL', () => {
		const href =
			'https://plant.example/app/pv-recovery/i/T1/lot_processing/new/startLot?seed.lotId=L9&x=1';
		const got = takeSeedQueryParams(href);
		expect(got.seed).toEqual({ lotId: 'L9' });
		expect(got.strippedHref).toBe(
			'/app/pv-recovery/i/T1/lot_processing/new/startLot?x=1'
		);
	});

	test('supports multiple seed fields', () => {
		const href =
			'https://plant.example/app/pv-recovery/i/T1/lot_processing/new/startLot?seed.lotId=L9&seed.material=PET';
		const got = takeSeedQueryParams(href);
		expect(got.seed).toEqual({ lotId: 'L9', material: 'PET' });
		expect(got.strippedHref).toBe(
			'/app/pv-recovery/i/T1/lot_processing/new/startLot'
		);
	});

	test('absent seed.* yields empty seed and unchanged path', () => {
		const href = 'https://plant.example/app/pv-recovery/i/T1/intake/T1/register';
		const got = takeSeedQueryParams(href);
		expect(got.seed).toEqual({});
		expect(got.strippedHref).toBe('/app/pv-recovery/i/T1/intake/T1/register');
	});

	test('empty seed value is kept as empty string', () => {
		const href = 'https://plant.example/app/pv-recovery/new?seed.note=';
		expect(takeSeedQueryParams(href).seed).toEqual({ note: '' });
	});
});

describe('applySeedQueryParams', () => {
	test('seeds isolated provisional bag, records provenance, returns stripped href', () => {
		const bag = new Map<string, Record<string, unknown>>();
		const seededMeta = new Map<string, readonly string[]>();
		let openSeed: Record<string, string> | null = null;
		const seedKey = provisionalSeedDraftKey({ lotId: 'L9' });
		const stripped = applySeedQueryParams(
			'https://plant.example/app/pv-recovery/i/T1/lot_processing/new/startLot?seed.lotId=L9',
			{
				getDraft: (path, key) => bag.get(`${path}::${key}`) ?? null,
				seedDraft: (path, key, value, seededFields) => {
					bag.set(`${path}::${key}`, value);
					if (seededFields) seededMeta.set(`${path}::${key}`, seededFields);
				},
				rememberOpenSeed: (_path, seed) => {
					openSeed = seed;
				}
			},
			{ jobId: 'lot_processing', taskId: 'startLot', jobInstanceId: '' }
		);
		expect(stripped).toBe('/app/pv-recovery/i/T1/lot_processing/new/startLot');
		expect(bag.get(`lot_processing/startLot::${seedKey}`)).toEqual({ lotId: 'L9' });
		expect(seededMeta.get(`lot_processing/startLot::${seedKey}`)).toEqual(['lotId']);
		expect(openSeed).toEqual({ lotId: 'L9' });
		expect(bag.has('lot_processing/startLot::')).toBe(false);
	});

	test('seed overwrites prior values for seeded keys', () => {
		const bag = new Map<string, Record<string, unknown>>();
		const seedKey = provisionalSeedDraftKey({ lotId: 'L2' });
		bag.set(`lot_processing/startLot::${seedKey}`, { lotId: 'L1', note: 'keep' });
		applyUrlSeed(
			{ lotId: 'L2' },
			{ jobId: 'lot_processing', taskId: 'startLot', jobInstanceId: '' },
			{
				getDraft: (path, key) => bag.get(`${path}::${key}`) ?? null,
				seedDraft: (path, key, value) => {
					bag.set(`${path}::${key}`, value);
				}
			}
		);
		expect(bag.get(`lot_processing/startLot::${seedKey}`)).toEqual({
			lotId: 'L2',
			note: 'keep'
		});
	});

	test('no seed.* → null and no draft write', () => {
		const seedDraft = (path: string, key: string, value: Record<string, unknown>) => {
			void path;
			void key;
			void value;
			throw new Error('should not seed');
		};
		expect(
			applySeedQueryParams(
				'https://plant.example/app/pv-recovery/i/T1/lot_processing/new/startLot',
				{ getDraft: () => null, seedDraft },
				{ jobId: 'lot_processing', taskId: 'startLot', jobInstanceId: '' }
			)
		).toBeNull();
	});
});

describe('provisional draft isolation', () => {
	test('seeded L-wait then unseeded start uses a separate empty bag', () => {
		const bag = new Map<string, Record<string, unknown>>();
		let openSeed: Record<string, string> | null = { stale: 'x' };
		const drafts = {
			getDraft: (path: string, key: string) => bag.get(`${path}::${key}`) ?? null,
			seedDraft: (path: string, key: string, value: Record<string, unknown>) => {
				bag.set(`${path}::${key}`, value);
			},
			rememberOpenSeed: (_path: string, seed: Record<string, string>) => {
				openSeed = seed;
			},
			clearOpenSeed: () => {
				openSeed = null;
			}
		};
		applyUrlSeed(
			{ lotId: 'L-wait' },
			{ jobId: 'lot_processing', taskId: 'startLot', jobInstanceId: '' },
			drafts
		);
		const seededKey = provisionalSeedDraftKey({ lotId: 'L-wait' });
		expect(bag.get(`lot_processing/startLot::${seededKey}`)).toEqual({ lotId: 'L-wait' });

		const unseeded = applyUrlSeed(
			{},
			{ jobId: 'lot_processing', taskId: 'startLot', jobInstanceId: '' },
			drafts
		);
		expect(unseeded.draftBagInstanceKey).toBe('');
		expect(unseeded.seededFields).toEqual([]);
		expect(openSeed).toBeNull();
		expect(bag.get('lot_processing/startLot::')).toBeUndefined();
		expect(bag.get(`lot_processing/startLot::${seededKey}`)).toEqual({ lotId: 'L-wait' });
	});
});
