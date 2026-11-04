import { describe, expect, test } from 'vitest';

import * as m from '#lib/paraglide/messages.js';

import { readColumns } from './read-columns.js';

describe('readColumns', () => {
	test('reads ui:options.columns; defaults Id / Value from locale', () => {
		expect(readColumns({})).toEqual({ id: m.column_id(), value: m.column_value() });
		expect(
			readColumns({ 'ui:options': { columns: { id: 'Bancale', value: 'Peso' } } })
		).toEqual({ id: 'Bancale', value: 'Peso' });
	});
});
