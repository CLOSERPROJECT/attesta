/**
 * Regression: cell-map Commit with one enum cell + "—" clear.
 *
 * Clear writes `null`. Authored progress.additionalProperties must allow
 * null | enum so Ajv Draft does not block submit before onSubmit / Commit.
 * Server filterFilledCells keeps null (clear) and omits untouched "".
 *
 * Schema fixture mirrors Go TaskFormBundle projection (no client projectFormSchema).
 */
import { describe, expect, test } from 'vitest';
import type { Schema } from '@sjsf/form';
import { createAjvFormValidator } from './ajv-form-validator.js';

function lineLoadProjectedSchema(): Schema {
	return {
		type: 'object',
		properties: {
			progress: {
				type: 'object',
				propertyNames: { enum: ['NIR-1.1', 'NIR-1.2'] },
				additionalProperties: {
					anyOf: [{ type: 'null' }, { type: 'string', enum: ['partial', 'done'] }]
				}
			}
		}
	} as Schema;
}

describe('lineLoad partial cell-map submit (Ajv draft)', () => {
	const validator = createAjvFormValidator<Record<string, unknown>>();
	const schema = lineLoadProjectedSchema();

	/** Mirrors scalar mapTable clear write semantics. */
	function setCell(
		value: Record<string, unknown>,
		id: string,
		next: string | undefined
	): Record<string, unknown> {
		const out = { ...value };
		if (next === undefined) out[id] = null;
		else out[id] = next;
		return out;
	}

	test('projection keeps nullability on progress.additionalProperties', () => {
		const progress = (
			schema as {
				properties?: {
					progress?: { additionalProperties?: { anyOf?: unknown[] } };
				};
			}
		).properties?.progress;
		const anyOf = progress?.additionalProperties?.anyOf;
		expect(anyOf).toEqual(
			expect.arrayContaining([{ type: 'null' }, { type: 'string', enum: ['partial', 'done'] }])
		);
	});

	test('single enum cell (omit siblings) validates', () => {
		expect(
			validator.isValid(schema, schema, { progress: { 'NIR-1.1': 'partial' } })
		).toBe(true);
	});

	test('enum + — clear (null) validates so submit / Commit can fire', () => {
		let progress: Record<string, unknown> = {};
		progress = setCell(progress, 'NIR-1.1', 'partial');
		progress = setCell(progress, 'NIR-1.2', undefined); // UI "—"
		const form = validator.validateFormValue(schema, { progress });
		expect(
			form.errors,
			'null clear is intentional wire (server filterFilledCells); Ajv must not block submit'
		).toBeUndefined();
	});
});
