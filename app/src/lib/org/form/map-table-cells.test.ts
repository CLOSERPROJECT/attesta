import { describe, expect, test } from 'vitest';
import { MapTableCells } from './map-table-cells.svelte.js';

describe('MapTableCells', () => {
	test('omit stays omit (undefined→undefined no-op)', () => {
		let value: Record<string, unknown> | undefined;
		const rowCalls: unknown[] = [];
		const setValueCalls: unknown[] = [];
		const cells = new MapTableCells({
			getValue: () => value,
			setValue: (next) => {
				setValueCalls.push(next);
				value = next as Record<string, unknown> | undefined;
			},
			setRow: (rowKey, cell) => {
				rowCalls.push([rowKey, cell]);
			}
		});
		cells.setScalarCell('a', undefined);
		expect(value).toBeUndefined();
		expect(setValueCalls).toEqual([]);
		expect(rowCalls).toEqual([]);
	});

	test('null on omit stays omit (never invents null key)', () => {
		let value: Record<string, unknown> | undefined;
		const rowCalls: unknown[] = [];
		const setValueCalls: unknown[] = [];
		const cells = new MapTableCells({
			getValue: () => value,
			setValue: (next) => {
				setValueCalls.push(next);
				value = next as Record<string, unknown> | undefined;
			},
			setRow: (rowKey, cell) => {
				rowCalls.push([rowKey, cell]);
			}
		});
		cells.setScalarCell('a', null);
		expect(value).toBeUndefined();
		expect(setValueCalls).toEqual([]);
		expect(rowCalls).toEqual([]);
	});

	test('undefined on filled cell drops draft key', () => {
		let value: Record<string, unknown> | undefined = { a: 'done', b: 'partial' };
		const rowCalls: unknown[] = [];
		const cells = new MapTableCells({
			getValue: () => value,
			setValue: (next) => {
				value = next as Record<string, unknown> | undefined;
			},
			setRow: (rowKey, cell) => {
				rowCalls.push([rowKey, cell]);
			}
		});
		cells.setScalarCell('a', undefined);
		expect(value).toEqual({ b: 'partial' });
		expect(Object.prototype.hasOwnProperty.call(value, 'a')).toBe(false);
		expect(rowCalls).toEqual([]);
	});

	test('undefined on sole filled cell drops map to undefined', () => {
		let value: Record<string, unknown> | undefined = { a: 'done' };
		const cells = new MapTableCells({
			getValue: () => value,
			setValue: (next) => {
				value = next as Record<string, unknown> | undefined;
			},
			setRow: () => {
				throw new Error('setRow must not run on stock clear');
			}
		});
		cells.setScalarCell('a', undefined);
		expect(value).toBeUndefined();
	});

	test('null on filled cell drops draft key (never write Domain null)', () => {
		let value: Record<string, unknown> | undefined = { a: 'done' };
		const rowCalls: unknown[] = [];
		const cells = new MapTableCells({
			getValue: () => value,
			setValue: (next) => {
				value = next as Record<string, unknown> | undefined;
			},
			setRow: (rowKey, cell) => {
				rowCalls.push([rowKey, cell]);
			}
		});
		cells.setScalarCell('a', null);
		expect(value).toBeUndefined();
		expect(rowCalls).toEqual([]);
	});

	test('enum write goes through setRow', () => {
		let value: Record<string, unknown> | undefined;
		const rowCalls: unknown[] = [];
		const cells = new MapTableCells({
			getValue: () => value,
			setValue: (next) => {
				value = next as Record<string, unknown> | undefined;
			},
			setRow: (rowKey, cell) => {
				rowCalls.push([rowKey, cell]);
				value = { ...(value ?? {}), [rowKey]: cell };
			}
		});
		cells.setScalarCell('a', 'done');
		expect(rowCalls).toEqual([['a', 'done']]);
		expect(value).toEqual({ a: 'done' });
	});
});
