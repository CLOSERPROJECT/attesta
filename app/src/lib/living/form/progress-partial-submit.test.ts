/**
 * Regression: lineLoad leftoverTable Commit with Parziale + "—" clear.
 *
 * leftover-table-field.setProgressCell writes `null` for clear. Authored
 * progress.additionalProperties must allow null | enum so Ajv Draft does not
 * block Invia before onSubmit / Commit. Server filterFilledProgress drops nulls.
 */
import { describe, expect, test } from 'vitest';
import type { Schema } from '@sjsf/form';
import { emptySnapshot } from './domain-snapshot.js';
import truckLoopStepsYaml from './fixtures/truck-loop-steps.yaml?raw';
import { loadTruckLoop, taskConfig } from './load-truck-loop.js';
import { projectFormSchema } from './project-form-schema.js';
import { createAjvFormValidator } from './ajv-form-validator.js';
import type { PalletProgress } from './pallet-progress.js';

describe('lineLoad partial progress submit (Ajv draft)', () => {
	const config = loadTruckLoop(truckLoopStepsYaml);
	const authored = taskConfig(config.jobs.line, 'lineLoad')!;
	const validator = createAjvFormValidator<Record<string, unknown>>();

	async function projectedSchema() {
		const snapshot = emptySnapshot();
		snapshot.pallet['NIR-1.1'] = { loadId: 'NIR-1', weight: 400 };
		snapshot.pallet['NIR-1.2'] = { loadId: 'NIR-1', weight: 450 };
		const projected = await projectFormSchema(snapshot, authored, {
			instance: { id: '1st', label: '1st' }
		});
		expect(projected.ok).toBe(true);
		if (!projected.ok) throw new Error('project failed');
		return projected.value.schema as Schema;
	}

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

	test('projection keeps nullability on progress.additionalProperties', async () => {
		const schema = await projectedSchema();
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

	test('single Parziale (omit siblings) validates', async () => {
		const schema = await projectedSchema();
		expect(
			validator.isValid(schema, schema, { progress: { 'NIR-1.1': 'partial' } })
		).toBe(true);
	});

	test('Parziale + — clear (null) validates so Invia / Commit can fire', async () => {
		const schema = await projectedSchema();
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
