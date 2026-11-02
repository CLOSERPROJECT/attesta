import { describe, expect, test } from 'vitest';
import { readColumns } from './read-columns.js';

describe('readColumns', () => {
	test('reads ui:options.columns; defaults Id / Value', () => {
		expect(readColumns({})).toEqual({ id: 'Id', value: 'Value' });
		expect(
			readColumns({ 'ui:options': { columns: { id: 'Bancale', value: 'Peso' } } })
		).toEqual({ id: 'Bancale', value: 'Peso' });
	});
});
