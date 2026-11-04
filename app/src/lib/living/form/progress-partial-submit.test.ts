/**
 * Regression: cell-map Commit with one enum cell + omit siblings.
 *
 * Stock clearable wipes a bad draft pick → omit/undefined → drop the key.
 * Enum options are real values only. Server filterFilledCells omits untouched.
 *
 * Schema fixture mirrors Go concrete-property projection (no SPA normalize).
 */
import { describe, expect, test } from 'vitest';
import type { FormValue, Schema } from '@sjsf/form';
import { createAjvFormValidator } from './ajv-form-validator.js';

const cellSchema = {
	enum: ['partial', 'done']
};

function lineLoadProjectedSchema(): Schema {
	return {
		type: 'object',
		properties: {
			progress: {
				type: 'object',
				properties: {
					'NIR-1.1': cellSchema,
					'NIR-1.2': cellSchema
				},
				additionalProperties: false
			}
		}
	} as Schema;
}

describe('lineLoad partial cell-map submit (Ajv draft)', () => {
	const validator = createAjvFormValidator<Record<string, unknown>>();
	const schema = lineLoadProjectedSchema();

	test('projection keeps real-value enums on each concrete cell schema', () => {
		const progress = (
			schema as {
				properties?: {
					progress?: {
						properties?: Record<string, { enum?: unknown[]; anyOf?: unknown[] }>;
						additionalProperties?: unknown;
					};
				};
			}
		).properties?.progress;
		expect(progress?.additionalProperties).toBe(false);
		expect(progress?.properties?.['NIR-1.1']?.enum).toEqual(['partial', 'done']);
		expect(progress?.properties?.['NIR-1.2']?.enum).toEqual(['partial', 'done']);
		expect(progress?.properties?.['NIR-1.1']?.enum).not.toContain(null);
		expect(progress?.properties?.['NIR-1.1']?.anyOf).toBeUndefined();
		expect(progress?.properties?.['NIR-1.2']?.anyOf).toBeUndefined();
	});

	test('single enum cell (omit siblings) validates', () => {
		expect(
			validator.isValid(schema, schema, { progress: { 'NIR-1.1': 'partial' } })
		).toBe(true);
	});

	test('empty-string cell fails without SPA normalize', () => {
		expect(
			validator.isValid(schema, schema, {
				progress: { 'NIR-1.1': 'partial', 'NIR-1.2': '' }
			})
		).toBe(false);
	});

	test('enum + omit sibling validates', () => {
		const progress = { 'NIR-1.1': 'partial' };
		const form = validator.validateFormValue(schema, { progress } as FormValue);
		expect(form.errors).toBeUndefined();
	});

	test('enum + own-property undefined sibling validates', () => {
		const progress = { 'NIR-1.1': 'partial', 'NIR-1.2': undefined };
		expect(validator.isValid(schema, schema, { progress })).toBe(true);
	});

	test('null cell fails (not a Domain clear option)', () => {
		expect(
			validator.isValid(schema, schema, {
				progress: { 'NIR-1.1': 'partial', 'NIR-1.2': null }
			})
		).toBe(false);
	});
});
