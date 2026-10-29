import { describe, expect, test } from 'vitest';

import { computeIntakeInstanceId } from './intake-id.js';
import { emptyDomain } from './types.js';

describe('computeIntakeInstanceId', () => {
	test('builds grammar and progressive n', () => {
		const domain = emptyDomain();
		domain.load = {
			'NIR-261019-NO-1': { client: 'NO', date: '2026-10-19' }
		};
		expect(
			computeIntakeInstanceId(domain, {
				client: 'NO',
				date: '2026-10-19',
				idoneitaRiutilizzo: 'NIR'
			})
		).toBe('NIR-261019-NO-2');
	});

	test('first of day is n=1', () => {
		expect(
			computeIntakeInstanceId(emptyDomain(), {
				client: 'SU',
				date: '2026-10-19',
				idoneitaRiutilizzo: 'IR'
			})
		).toBe('IR-261019-SU-1');
	});

	test('incomplete draft returns null', () => {
		expect(
			computeIntakeInstanceId(emptyDomain(), {
				client: '',
				date: '2026-10-19',
				idoneitaRiutilizzo: 'NIR'
			})
		).toBeNull();
	});
});
