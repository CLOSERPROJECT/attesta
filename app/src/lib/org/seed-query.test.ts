import { describe, expect, test } from 'vitest';

import { applySeedQueryParams, takeSeedQueryParams } from './seed-query.js';

describe('takeSeedQueryParams', () => {
	test('collects seed.* params and strips them from the URL', () => {
		const href =
			'https://plant.example/app/recycling/i/T1/lot_processing/new/startLot?seed.lotId=L9&x=1';
		const got = takeSeedQueryParams(href);
		expect(got.seed).toEqual({ lotId: 'L9' });
		expect(got.strippedHref).toBe(
			'/app/recycling/i/T1/lot_processing/new/startLot?x=1'
		);
	});

	test('supports multiple seed fields', () => {
		const href =
			'https://plant.example/app/recycling/i/T1/lot_processing/new/startLot?seed.lotId=L9&seed.material=PET';
		const got = takeSeedQueryParams(href);
		expect(got.seed).toEqual({ lotId: 'L9', material: 'PET' });
		expect(got.strippedHref).toBe(
			'/app/recycling/i/T1/lot_processing/new/startLot'
		);
	});

	test('absent seed.* yields empty seed and unchanged path', () => {
		const href = 'https://plant.example/app/recycling/i/T1/intake/T1/register';
		const got = takeSeedQueryParams(href);
		expect(got.seed).toEqual({});
		expect(got.strippedHref).toBe('/app/recycling/i/T1/intake/T1/register');
	});

	test('empty seed value is kept as empty string', () => {
		const href = 'https://plant.example/app/recycling/new?seed.note=';
		expect(takeSeedQueryParams(href).seed).toEqual({ note: '' });
	});
});

describe('applySeedQueryParams', () => {
	test('seeds draft bag and returns stripped href', () => {
		const bag = new Map<string, Record<string, unknown>>();
		const stripped = applySeedQueryParams(
			'https://plant.example/app/recycling/i/T1/lot_processing/new/startLot?seed.lotId=L9',
			{
				getDraft: (path, key) => bag.get(`${path}::${key}`) ?? null,
				seedDraft: (path, key, value) => {
					bag.set(`${path}::${key}`, value);
				}
			},
			{ jobId: 'lot_processing', taskId: 'startLot', jobInstanceId: '' }
		);
		expect(stripped).toBe('/app/recycling/i/T1/lot_processing/new/startLot');
		expect(bag.get('lot_processing/startLot::')).toEqual({ lotId: 'L9' });
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
				'https://plant.example/app/recycling/i/T1/lot_processing/new/startLot',
				{ getDraft: () => null, seedDraft },
				{ jobId: 'lot_processing', taskId: 'startLot', jobInstanceId: '' }
			)
		).toBeNull();
	});
});
