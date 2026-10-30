/**
 * Cascade #6 regression: Completo on one leftover + untouched siblings.
 * Untouched wire (`""` / own-property undefined) must be omitted before Ajv
 * so Invia can Commit; null clear stays.
 */
import { describe, expect, test } from 'vitest';
import type { Schema } from '@sjsf/form';
import { emptySnapshot } from './domain-snapshot.js';
import truckLoopStepsYaml from './fixtures/truck-loop-steps.yaml?raw';
import { loadTruckLoop, taskConfig } from './load-truck-loop.js';
import { projectFormSchema } from './project-form-schema.js';
import { createAjvFormValidator } from './ajv-form-validator.js';
import {
	normalizeFormProgress,
	omitUntouchedProgressCells
} from './pallet-progress.js';
import { compileTaskFormPlan } from './task-form-plan.js';

describe('cascade #6 omit untouched progress cells', () => {
	const config = loadTruckLoop(truckLoopStepsYaml);
	const authored = taskConfig(config.jobs.line, 'lineLoad')!;
	const validator = createAjvFormValidator<Record<string, unknown>>();

	async function projected() {
		const snapshot = emptySnapshot();
		snapshot.pallet['NIR-1.1'] = { loadId: 'NIR-1', weight: 400 };
		snapshot.pallet['NIR-1.2'] = { loadId: 'NIR-1', weight: 450 };
		const projected = await projectFormSchema(snapshot, authored, {
			instance: { id: '1st', label: '1st' }
		});
		expect(projected.ok).toBe(true);
		if (!projected.ok) throw new Error('project failed');
		return projected.value;
	}

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

	test('createAjvFormValidator: Completo + sibling "" validates (Invia path)', async () => {
		const slice = await projected();
		const schema = slice.schema as Schema;
		const [touched, sibling] = slice.leftoverIds;
		expect(touched && sibling).toBeTruthy();
		const progress = { [touched!]: 'done', [sibling!]: '' };
		expect(validator.isValid(schema, schema, { progress })).toBe(true);
		const result = validator.validateFormValue(schema, { progress });
		expect(result.errors).toBeUndefined();
		expect(result.value).toEqual({ progress: { [touched!]: 'done' } });
	});

	test('createAjvFormValidator: Completo + own-property undefined validates', async () => {
		const slice = await projected();
		const schema = slice.schema as Schema;
		const [touched, sibling] = slice.leftoverIds;
		expect(touched && sibling).toBeTruthy();
		const progress = { [touched!]: 'done', [sibling!]: undefined };
		expect(validator.isValid(schema, schema, { progress })).toBe(true);
		const result = validator.validateFormValue(schema, { progress });
		expect(result.errors).toBeUndefined();
		expect(result.value).toEqual({ progress: { [touched!]: 'done' } });
	});

	test('null clear sibling still validates; normalizeFormProgress keeps null', async () => {
		const slice = await projected();
		const schema = slice.schema as Schema;
		const [touched, sibling] = slice.leftoverIds;
		expect(touched && sibling).toBeTruthy();
		const progress = { [touched!]: 'partial', [sibling!]: null };
		expect(normalizeFormProgress({ progress })).toEqual({ progress });
		expect(validator.isValid(schema, schema, { progress })).toBe(true);
	});

	test('sealSubmission omits untouched progress cells', () => {
		const plan = compileTaskFormPlan(authored as Record<string, unknown>);
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
