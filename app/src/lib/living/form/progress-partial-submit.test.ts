/**
 * Regression: lineLoad leftoverTable Commit with Parziale + "—" clear.
 *
 * leftover-table-field.setProgressCell writes `null` for clear. Authored
 * progress.additionalProperties must allow null | enum so Ajv Draft does not
 * block Invia before onSubmit / Commit. Server filterFilledProgress drops nulls.
 *
 * Schema fixture mirrors Go TaskFormBundle projection (no client projectFormSchema).
 */
import { describe, expect, test } from 'vitest';
import type { Schema } from '@sjsf/form';
import { createAjvFormValidator } from './ajv-form-validator.js';
import type { PalletProgress } from './pallet-progress.js';

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

describe('lineLoad partial progress submit (Ajv draft)', () => {
	const validator = createAjvFormValidator<Record<string, unknown>>();
	const schema = lineLoadProjectedSchema();

	/** Mirrors leftover-table-field.setProgressCell write semantics. */
	function setProgressCell(
		value: Record<string, unknown>,
		id: string,
		nextProgress: PalletProgress | undefined
	): Record<string, unknown> {
		const next = { ...value };
		if (nextProgress === undefined) next[id] = null;
		else next[id] = nextProgress;
		return next;
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

	test('single Parziale (omit siblings) validates', () => {
		expect(
			validator.isValid(schema, schema, { progress: { 'NIR-1.1': 'partial' } })
		).toBe(true);
	});

	test('Parziale + — clear (null) validates so Invia / Commit can fire', () => {
		let progress: Record<string, unknown> = {};
		progress = setProgressCell(progress, 'NIR-1.1', 'partial');
		progress = setProgressCell(progress, 'NIR-1.2', undefined); // UI "—"
		const form = validator.validateFormValue(schema, { progress });
		expect(
			form.errors,
			'null clear is intentional wire (server filterFilledProgress); Ajv must not block Invia'
		).toBeUndefined();
	});
});
