import { describe, expect, test } from 'vitest';

import { evaluateFinalResultsEntry } from './live-task-chrome.svelte.js';
import {
	bundleMatches,
	domainFromBundleBindings,
	findTaskFormBundle,
	planFromTaskFormBundle,
	type TaskFormBundle
} from './task-form-bundle.js';

function sampleBundle(overrides: Partial<TaskFormBundle> = {}): TaskFormBundle {
	return {
		jobId: 'intake',
		taskId: 'codiceFir',
		jobInstanceId: 'NIR-1',
		title: 'Codice FIR',
		surface: 'live',
		schema: {
			type: 'object',
			properties: { codiceFir: { type: 'string' } }
		},
		uiSchema: { 'ui:options': { order: ['codiceFir'], hideTitle: true } },
		initialData: {},
		chrome: [],
		chromeFieldIds: [],
		sealFieldIds: ['codiceFir'],
		bindings: {
			domain: { load: {}, pallet: {}, container: {}, submissions: [] },
			instance: { id: 'NIR-1', label: 'NIR-1' }
		},
		...overrides
	};
}

describe('task-form-bundle', () => {
	test('bundleMatches requires job/task/instance', () => {
		const bundle = sampleBundle();
		expect(bundleMatches(bundle, 'intake', 'codiceFir', 'NIR-1')).toBe(true);
		expect(bundleMatches(bundle, 'intake', 'codiceFir', 'other')).toBe(false);
		expect(bundleMatches(null, 'intake', 'codiceFir', 'NIR-1')).toBe(false);
	});

	test('findTaskFormBundle selects by address including provisional empty id', () => {
		const bundles = [
			sampleBundle({ taskId: 'register', jobInstanceId: '' }),
			sampleBundle()
		];
		expect(findTaskFormBundle(bundles, 'intake', 'register', '')?.taskId).toBe('register');
		expect(findTaskFormBundle(bundles, 'intake', 'codiceFir', 'NIR-1')?.taskId).toBe(
			'codiceFir'
		);
		expect(findTaskFormBundle(bundles, 'intake', 'codiceFir', '')).toBeNull();
		expect(findTaskFormBundle([], 'intake', 'register', '')).toBeNull();
	});

	test('planFromTaskFormBundle skips authored compile and strips chrome on seal', () => {
		const bundle = sampleBundle({
			chrome: [
				{
					id: 'hint',
					kind: 'readonly',
					expr: '1'
				}
			],
			chromeFieldIds: ['hint']
		});
		const plan = planFromTaskFormBundle(bundle);
		expect(plan.surface).toBe('live');
		expect(plan.schemas.schema).toBe(bundle.schema);
		expect(plan.chromeFieldIds).toEqual(['hint']);
		expect(plan.sealSubmission({ codiceFir: 'FIR-1', hint: 'x' })).toEqual({
			codiceFir: 'FIR-1'
		});
	});

	test('bindings.domain alone satisfies DomainSnapshot for finalResults chrome', async () => {
		const bundle = sampleBundle({
			jobId: 'line',
			taskId: 'loadedExtracted',
			jobInstanceId: '1st',
			bindings: {
				domain: {
					load: {},
					pallet: {},
					container: {},
					submissions: [
						{
							jobId: 'line',
							taskId: 'lineLoad',
							jobInstanceId: '1st',
							facts: {},
							idsTouched: []
						}
					]
				},
				instance: { id: '1st' }
			}
		});
		const snapshot = domainFromBundleBindings(bundle.bindings);
		const display = await evaluateFinalResultsEntry(
			{ id: 'results', kind: 'finalResults' },
			{},
			snapshot,
			undefined
		);
		expect(display.error).toBeNull();
		expect(display.sections).toEqual([]);
	});
});
