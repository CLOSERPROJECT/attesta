/**
 * Cascade #6 regression: Completo on one leftover + untouched siblings.
 * Untouched wire (`""` / own-property undefined) must be omitted before Ajv
 * so Invia can Commit; null clear stays.
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
} from './pallet-progress.js';
import { planFromTaskFormBundle, type TaskFormBundle } from './task-form-bundle.js';

/** Fixture: lineLoad leftoverTable after Go leftover projection. */
function lineLoadProjectedSchema(leftoverIds: string[]): Schema {
	return {
		type: 'object',
		properties: {
			progress: {
				type: 'object',
				propertyNames: { enum: leftoverIds },
				additionalProperties: {
					anyOf: [{ type: 'null' }, { type: 'string', enum: ['partial', 'done'] }]
				}
			}
		}
	} as Schema;
}

function sealPlan() {
	const leftoverIds = ['NIR-1.1', 'NIR-1.2', 'NIR-1.3'];
	const bundle: TaskFormBundle = {
		jobId: 'line',
		taskId: 'lineLoad',
		jobInstanceId: '1st',
		title: 'lineLoad',
		surface: 'leftover-table',
		schema: lineLoadProjectedSchema(leftoverIds),
		uiSchema: {} as UiSchemaRoot,
		initialData: {},
		leftoverIds,
		chrome: [],
		chromeFieldIds: [],
		sealFieldIds: [],
		bindings: {}
	};
	return planFromTaskFormBundle(bundle);
}

describe('cascade #6 omit untouched progress cells', () => {
	const leftoverIds = ['NIR-1.1', 'NIR-1.2'];
	const schema = lineLoadProjectedSchema(leftoverIds);
	const validator = createAjvFormValidator<Record<string, unknown>>();

	test('omitUntouchedProgressCells drops "" and undefined; keeps null|partial|done', () => {
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

	test('createAjvFormValidator: Completo + sibling "" validates (Invia path)', () => {
		const [touched, sibling] = leftoverIds;
		const progress = { [touched]: 'done', [sibling]: '' };
		expect(validator.isValid(schema, schema, { progress })).toBe(true);
		const result = validator.validateFormValue(schema, { progress });
		expect(result.errors).toBeUndefined();
		expect(result.value).toEqual({ progress: { [touched]: 'done' } });
	});

	test('createAjvFormValidator: Completo + own-property undefined validates', () => {
		const [touched, sibling] = leftoverIds;
		const progress = { [touched]: 'done', [sibling]: undefined };
		expect(validator.isValid(schema, schema, { progress })).toBe(true);
		const result = validator.validateFormValue(schema, { progress });
		expect(result.errors).toBeUndefined();
		expect(result.value).toEqual({ progress: { [touched]: 'done' } });
	});

	test('null clear sibling still validates; normalizeFormProgress keeps null', () => {
		const [touched, sibling] = leftoverIds;
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
