import { describe, expect, test } from 'vitest';
import { readColumns } from './read-columns.js';

describe('readColumns', () => {
	test('reads ui:options.columns; defaults Id / Valore', () => {
		expect(readColumns({})).toEqual({ id: 'Id', value: 'Valore' });
		expect(
			readColumns({ 'ui:options': { columns: { id: 'Bancale', value: 'Peso' } } })
		).toEqual({ id: 'Bancale', value: 'Peso' });
	});
});
