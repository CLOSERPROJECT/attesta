/**
 * Cascade #7 regression: after lineLoad leftover projection seed, Completo on
 * one leftover must Ajv-green. seedMapTableDraft must not write weigh-row
 * objects into sibling progress cells.
 */
import { describe, expect, test } from 'vitest';
import type { Schema } from '@sjsf/form';
import { emptySnapshot } from './domain-snapshot.js';
import truckLoopStepsYaml from './fixtures/truck-loop-steps.yaml?raw';
import { loadTruckLoop, taskConfig } from './load-truck-loop.js';
import { projectFormSchema } from './project-form-schema.js';
import { createAjvFormValidator } from './ajv-form-validator.js';
import {
	domainProgressForIds,
	normalizeFormProgress,
	seedProgressDraftFromDomain
} from './pallet-progress.js';
import { seedMapTableDraft } from './map-table-mount.js';
import { compileTaskFormPlan } from './task-form-plan.js';
import { emptyObjectRowWeighCell } from './object-row-weigh.js';

describe('cascade #7 gate mapTable seed off leftoverTable progress', () => {
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
		return { snapshot, slice: projected.value };
	}

	/** Mirrors TaskForm.seedProjectedDraft for leftoverTable progress primary. */
	function seedLikeTaskForm(
		draft: Record<string, unknown>,
		snapshot: ReturnType<typeof emptySnapshot>,
		leftoverIds: readonly string[]
	): Record<string, unknown> {
		const domainProg = domainProgressForIds(snapshot, leftoverIds);
		const seeded = seedProgressDraftFromDomain(
			draft.progress,
			domainProg,
			leftoverIds
		);
		let current: Record<string, unknown> = { ...draft, progress: seeded };
		const mapSeeded = seedMapTableDraft(current, authored, leftoverIds);
		if (mapSeeded) current = mapSeeded;
		return current;
	}

	test('seedMapTableDraft returns null for lineLoad leftoverTable', async () => {
		const { slice } = await projected();
		expect(seedMapTableDraft({}, authored, slice.leftoverIds)).toBeNull();
	});

	test('Completo-on-one after projection seed is Ajv-green', async () => {
		const { snapshot, slice } = await projected();
		const schema = slice.schema as Schema;
		const [touched, sibling] = slice.leftoverIds;
		expect(touched && sibling).toBeTruthy();

		const afterSeed = seedLikeTaskForm({}, snapshot, slice.leftoverIds);
		const progress = afterSeed.progress as Record<string, unknown>;
		expect(progress[sibling!]).not.toEqual(expect.objectContaining({ weight: '' }));
		expect(
			Object.values(progress).every(
				(cell) =>
					cell === null ||
					cell === 'partial' ||
					cell === 'done' ||
					cell === undefined
			)
		).toBe(true);

		const formValue = {
			progress: { ...progress, [touched!]: 'done' }
		};
		expect(validator.isValid(schema, schema, formValue)).toBe(true);
		const result = validator.validateFormValue(schema, formValue);
		expect(result.errors).toBeUndefined();
		expect(result.value).toEqual({ progress: { [touched!]: 'done' } });
	});

	test('normalize / seal drop leftover weigh-row cells if somehow present', () => {
		const junk = emptyObjectRowWeighCell({ tareKind: 'euro', idoneita: 'NIR' });
		expect(
			normalizeFormProgress({
				progress: { 'NIR-1.1': 'done', 'NIR-1.2': junk }
			})
		).toEqual({ progress: { 'NIR-1.1': 'done' } });

		const plan = compileTaskFormPlan(authored as Record<string, unknown>);
		expect(
			plan.sealSubmission({
				progress: { 'NIR-1.1': 'done', 'NIR-1.2': junk }
			})
		).toEqual({ progress: { 'NIR-1.1': 'done' } });
	});
});
