import { describe, expect, test } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import truckLoopStepsYaml from './fixtures/truck-loop-steps.yaml?raw';
import { loadTruckLoop, substepConfig, taskConfig } from './load-truck-loop.js';
import { emptyObjectRowWeighCell } from './object-row-weigh.js';
import {
	prepareMapTableForm,
	seedMapTableDraft
} from './map-table-mount.js';

const libDir = dirname(fileURLToPath(import.meta.url));

describe('mapTable Substep mount', () => {
	test('seedMapTableDraft fills missing leftover rows from form tareKind default', () => {
		const config = loadTruckLoop(truckLoopStepsYaml);
		const palletWeight = substepConfig(config.jobs.intake, 'palletWeight')!;
		const seeded = seedMapTableDraft(
			{
				tareKind: 'industrial',
				weights: {
					keep: {
						weight: 9,
						tareKind: 'euro',
						idoneita: 'IR',
						panelCount: 3
					}
				}
			},
			palletWeight,
			['keep', 'new']
		);
		expect(seeded).toEqual({
			tareKind: 'industrial',
			weights: {
				keep: {
					weight: 9,
					tareKind: 'euro',
					idoneita: 'IR',
					panelCount: 3
				},
				new: emptyObjectRowWeighCell({ tareKind: 'industrial', idoneita: 'NIR' })
			}
		});
		expect(seedMapTableDraft({}, palletWeight, [])).toBeNull();
	});

	test('seedMapTableDraft skips leftoverTable progress (lineLoad)', () => {
		const config = loadTruckLoop(truckLoopStepsYaml);
		const lineLoad = taskConfig(config.jobs.line, 'lineLoad')!;
		expect(
			seedMapTableDraft({ progress: { a: 'done' } }, lineLoad, ['a', 'b'])
		).toBeNull();
	});

	test('prepareMapTableForm expands map-star Derived and injects tareKinds domain', () => {
		const config = loadTruckLoop(truckLoopStepsYaml);
		const cfg = substepConfig(config.jobs.intake, 'palletWeight') as {
			derived?: Record<string, { expr: string; overridable?: boolean }>;
			tareKinds?: Record<string, number>;
		};
		const prepared = prepareMapTableForm(
			{
				schema: { type: 'object' },
				uiSchema: {},
				leftoverIds: ['p1', 'p2']
			},
			cfg
		);
		expect(prepared.options.derived?.['weights.p1.panelCount']?.expr).toContain('p1');
		expect(prepared.options.derived?.['weights.p2.panelCount']?.expr).toContain('p2');
		expect(prepared.options.getDomain?.()).toEqual({ tareKinds: cfg.tareKinds });
	});

	test('living TaskForm mounts from Task form bundle; mapTable seed helpers stay pure', () => {
		const taskForm = readFileSync(join(libDir, '../TaskForm.svelte'), 'utf8');
		const planSource = readFileSync(join(libDir, './task-form-plan.ts'), 'utf8');
		const mountSvelte = readFileSync(join(libDir, './map-table-mount.svelte.ts'), 'utf8');
		const mountPure = readFileSync(join(libDir, './map-table-mount.ts'), 'utf8');
		expect(taskForm).toContain('LiveTaskForm');
		expect(taskForm).toContain('planFromTaskFormBundle');
		expect(taskForm).not.toContain('compileTaskFormPlan');
		expect(taskForm).not.toContain('buildLiveTaskPlanContext');
		expect(taskForm).not.toContain('seedMapTableDraft');
		expect(taskForm).toContain('startingNew');
		expect(taskForm).not.toMatch(/jobInstanceId:\s*['"]_new['"]/);
		expect(planSource).toContain('projectedSchemas');
		expect(mountSvelte).toContain('createConfigForm');
		expect(mountSvelte).toContain('prepareMapTableForm');
		expect(mountPure).toContain('expandMapDerived');
		expect(mountPure).toContain('seedEmptyObjectRowWeighRows');
		expect(mountPure).toContain('getDomain:');
	});
});
