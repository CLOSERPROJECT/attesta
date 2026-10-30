import { describe, expect, test } from 'vitest';
import { emptySnapshot } from './domain-snapshot.js';
import truckLoopStepsYaml from './fixtures/truck-loop-steps.yaml?raw';
import { loadTruckLoop, taskConfig } from './load-truck-loop.js';
import { projectFormSchema, readColumns } from './project-form-schema.js';

describe('readColumns', () => {
	test('reads ui:options.columns; defaults Id / Valore', () => {
		expect(readColumns({})).toEqual({ id: 'Id', value: 'Valore' });
		expect(
			readColumns({ 'ui:options': { columns: { id: 'Bancale', value: 'Peso' } } })
		).toEqual({ id: 'Bancale', value: 'Peso' });
	});
});

describe('projectFormSchema', () => {
	const config = loadTruckLoop(truckLoopStepsYaml);

	test('injects propertyNames.enum for palletWeight leftovers', async () => {
		const authored = taskConfig(config.jobs.intake, 'palletWeight')!;
		const snapshot = emptySnapshot();
		snapshot.load['NIR-1'] = { client: 'NO' };
		snapshot.pallet['NIR-1.1'] = { loadId: 'NIR-1' };
		snapshot.pallet['NIR-1.2'] = { loadId: 'NIR-1' };

		const projected = await projectFormSchema(snapshot, authored, {
			instance: { id: 'NIR-1', label: 'NIR-1' }
		});
		expect(projected.ok).toBe(true);
		if (!projected.ok) return;
		expect(projected.value.leftoverIds).toEqual(['NIR-1.1', 'NIR-1.2']);
		const weights = (
			projected.value.schema as { properties?: Record<string, { propertyNames?: { enum?: string[] } }> }
		).properties?.weights;
		expect(weights?.propertyNames?.enum).toEqual(['NIR-1.1', 'NIR-1.2']);
	});

	test('lineLoad leftoverTable projects progress propertyNames', async () => {
		const authored = taskConfig(config.jobs.line, 'lineLoad')!;
		const snapshot = emptySnapshot();
		snapshot.pallet['P1'] = { loadId: 'L1', weight: 10 };
		snapshot.pallet['P2'] = { loadId: 'L1', weight: 20, progress: 'done' };

		const projected = await projectFormSchema(snapshot, authored, {
			instance: { id: '1st', label: '1st' }
		});
		expect(projected.ok).toBe(true);
		if (!projected.ok) return;
		expect(projected.value.leftoverIds).toEqual(['P1']);
	});
});
