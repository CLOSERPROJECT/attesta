/**
 * Cascade #6 regression: one row enum + untouched siblings.
 * Untouched wire (`""` / own-property undefined) must be omitted before Ajv
 * so submit can Commit; null clear stays.
 *
 * Projected schema shape mirrors Go TaskFormBundle (propertyNames + nullability);
 * seal uses planFromTaskFormBundle — no client compile / projectFormSchema.
 */
import { describe, expect, test } from 'vitest';
import type { Schema, UiSchemaRoot } from '@sjsf/form';
import { createAjvFormValidator } from './ajv-form-validator.js';
import {
	normalizeFormProgress,
	omitUntouchedProgressCells
} from './form-progress.js';
import { planFromTaskFormBundle, type TaskFormBundle } from './task-form-bundle.js';

/** Fixture: lineLoad rowTable after Go row projection. */
function lineLoadProjectedSchema(rowIds: string[]): Schema {
	return {
		type: 'object',
		properties: {
			progress: {
				type: 'object',
				propertyNames: { enum: rowIds },
				additionalProperties: {
					anyOf: [{ type: 'null' }, { type: 'string', enum: ['partial', 'done'] }]
				}
			}
		}
	} as Schema;
}

function sealPlan() {
	const rowIds = ['NIR-1.1', 'NIR-1.2', 'NIR-1.3'];
	const bundle: TaskFormBundle = {
		jobId: 'line',
		taskId: 'lineLoad',
		jobInstanceId: '1st',
		title: 'lineLoad',
		surface: 'row-table',
		schema: lineLoadProjectedSchema(rowIds),
		uiSchema: {} as UiSchemaRoot,
		initialData: {},
		rowIds,
		chrome: [],
		chromeFieldIds: [],
		sealFieldIds: [],
		bindings: {}
	};
	return planFromTaskFormBundle(bundle);
}

describe('cascade #6 omit untouched progress cells', () => {
	const rowIds = ['NIR-1.1', 'NIR-1.2'];
	const schema = lineLoadProjectedSchema(rowIds);
	const validator = createAjvFormValidator<Record<string, unknown>>();

	test('omitUntouchedProgressCells drops "" and undefined; keeps null|enum strings', () => {
		expect(
			omitUntouchedProgressCells({
				a: 'done',
				b: '',
				c: '  ',
				d: undefined,
				e: null,
				f: 'partial'
			})
		).toEqual({ a: 'done', e: null, f: 'partial' });
	});

	test('omitUntouchedProgressCells drops weigh-row objects (mis-seed)', () => {
		expect(
			omitUntouchedProgressCells({
				a: 'done',
				b: { weight: '', tareKind: 'euro', idoneita: 'NIR', panelCount: 0 }
			})
		).toEqual({ a: 'done' });
	});

	test('createAjvFormValidator: enum + sibling "" validates (submit path)', () => {
		const [touched, sibling] = rowIds;
		const progress = { [touched]: 'done', [sibling]: '' };
		expect(validator.isValid(schema, schema, { progress })).toBe(true);
		const result = validator.validateFormValue(schema, { progress });
		expect(result.errors).toBeUndefined();
		expect(result.value).toEqual({ progress: { [touched]: 'done' } });
	});

	test('createAjvFormValidator: enum + own-property undefined validates', () => {
		const [touched, sibling] = rowIds;
		const progress = { [touched]: 'done', [sibling]: undefined };
		expect(validator.isValid(schema, schema, { progress })).toBe(true);
		const result = validator.validateFormValue(schema, { progress });
		expect(result.errors).toBeUndefined();
		expect(result.value).toEqual({ progress: { [touched]: 'done' } });
	});

	test('null clear sibling still validates; normalizeFormProgress keeps null', () => {
		const [touched, sibling] = rowIds;
		const progress = { [touched]: 'partial', [sibling]: null };
		expect(normalizeFormProgress({ progress })).toEqual({ progress });
		expect(validator.isValid(schema, schema, { progress })).toBe(true);
	});

	test('sealSubmission omits untouched progress cells', () => {
		const plan = sealPlan();
		expect(
			plan.sealSubmission({
				progress: { 'NIR-1.1': 'done', 'NIR-1.2': '', 'NIR-1.3': undefined }
			})
		).toEqual({ progress: { 'NIR-1.1': 'done' } });
		expect(
			plan.sealSubmission({
				progress: { 'NIR-1.1': 'done', 'NIR-1.2': null }
			})
		).toEqual({ progress: { 'NIR-1.1': 'done', 'NIR-1.2': null } });
	});
});
