import { describe, expect, test } from 'vitest';

import { listEntities, type DomainSnapshot } from './domain-snapshot.js';

describe('DomainSnapshot', () => {
	test('accepts a non-well-known kind key', () => {
		const snap: DomainSnapshot = {
			submissions: [],
			widget: { w1: { ok: true } }
		};
		expect(listEntities(snap, 'widget')).toEqual({ w1: { ok: true } });
	});
});
