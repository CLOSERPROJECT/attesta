import { describe, expect, test } from 'vitest';

import {
	derivedPanelCount,
	palletWeightLeftoverIds
} from './leftovers.js';
import { emptyDomain } from './types.js';

describe('palletWeightLeftoverIds', () => {
	test('lists pallets for instance lacking weight', () => {
		const domain = emptyDomain();
		domain.pallet = {
			'A.2': { loadId: 'A' },
			'A.1': { loadId: 'A' },
			'B.1': { loadId: 'B' },
			'A.done': { loadId: 'A', weight: 10 }
		};
		expect(palletWeightLeftoverIds(domain, 'A')).toEqual(['A.1', 'A.2']);
	});

	test('empty when all weighed', () => {
		const domain = emptyDomain();
		domain.pallet = { 'A.1': { loadId: 'A', weight: 1 } };
		expect(palletWeightLeftoverIds(domain, 'A')).toEqual([]);
	});
});

describe('derivedPanelCount', () => {
	test('euro tare floor division', () => {
		expect(
			derivedPanelCount({
				weight: 125,
				tareKind: 'euro',
				customTareWeight: '',
				averagePanelWeight: 10
			})
		).toBe(10);
	});

	test('custom tare', () => {
		expect(
			derivedPanelCount({
				weight: 50,
				tareKind: 'custom',
				customTareWeight: 10,
				averagePanelWeight: 10
			})
		).toBe(4);
	});

	test('incomplete inputs yield empty', () => {
		expect(
			derivedPanelCount({
				weight: '',
				tareKind: 'euro',
				customTareWeight: '',
				averagePanelWeight: 10
			})
		).toBe('');
	});
});
