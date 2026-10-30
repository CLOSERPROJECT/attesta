import { describe, expect, test } from 'vitest';

import { displayFromChromeDisplayBag } from './live-task-chrome.svelte.js';
import {
	bundleMatches,
	chromeDisplayFromBundleBindings,
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

	test('dataReport chrome paints bindings.chromeDisplay without Domain walk', () => {
		const bundle = sampleBundle({
			jobId: 'line',
			taskId: 'loadedExtracted',
			jobInstanceId: '1st',
			chrome: [{ id: 'resultsReport', kind: 'dataReport', title: 'Risultati finali' }],
			chromeFieldIds: ['resultsReport'],
			bindings: {
				domain: {
					load: {},
					pallet: {},
					container: {},
					submissions: []
				},
				chromeDisplay: {
					resultsReport: {
						value: null,
						error: null,
						title: 'Risultati finali',
						sections: [
							{
								loadId: 'L1',
								secondaryLoadIds: ['L2'],
								rows: [
									{
										label: 'Camion lordo',
										value: '1000',
										error: null,
										tone: null
									},
									{
										label: 'Pannelli vs estratto',
										value: '70 kg (60%)',
										error: null,
										tone: 'danger'
									}
								]
							}
						]
					}
				},
				instance: { id: '1st' }
			}
		});

		expect(domainFromBundleBindings(bundle.bindings)).toEqual({
			load: {},
			pallet: {},
			container: {},
			submissions: []
		});

		const bag = chromeDisplayFromBundleBindings(bundle.bindings);
		const display = displayFromChromeDisplayBag(bag, {
			id: 'resultsReport',
			kind: 'dataReport',
			title: 'Risultati finali'
		});
		expect(display.error).toBeNull();
		expect(display.title).toBe('Risultati finali');
		expect(display.sections).toEqual([
			{
				loadId: 'L1',
				secondaryLoadIds: ['L2'],
				rows: [
					{ label: 'Camion lordo', value: '1000', error: null, tone: null },
					{
						label: 'Pannelli vs estratto',
						value: '70 kg (60%)',
						error: null,
						tone: 'danger'
					}
				]
			}
		]);
	});

	test('missing chromeDisplay entry yields chrome-display-missing', () => {
		const display = displayFromChromeDisplayBag(undefined, {
			id: 'resultsReport',
			kind: 'dataReport',
			title: 'Risultati finali'
		});
		expect(display.error).toBe('chrome-display-missing');
		expect(display.title).toBe('Risultati finali');
		expect(display.sections).toEqual([]);
	});

	test('dataReport flat rows chrome paints without sections / loadId', () => {
		const bundle = sampleBundle({
			jobId: 'line',
			taskId: 'weightCheck',
			jobInstanceId: 'NIR-1',
			chrome: [{ id: 'registeredData', kind: 'dataReport', title: 'Dati registrati' }],
			chromeFieldIds: ['registeredData'],
			bindings: {
				domain: {
					load: {},
					pallet: {},
					container: {},
					submissions: []
				},
				chromeDisplay: {
					registeredData: {
						value: null,
						error: null,
						title: 'Dati registrati',
						rows: [
							{
								label: 'Peso camion live',
								value: '1200',
								error: null,
								tone: null
							},
							{
								label: 'Somma pesi bancali live',
								value: '1180',
								error: null,
								tone: null
							},
							{
								label: 'Match %',
								value: '98.33333333333333',
								error: null,
								tone: 'green'
							}
						]
					}
				},
				instance: { id: 'NIR-1' }
			}
		});

		const bag = chromeDisplayFromBundleBindings(bundle.bindings);
		const display = displayFromChromeDisplayBag(bag, {
			id: 'registeredData',
			kind: 'dataReport',
			title: 'Dati registrati'
		});
		expect(display.error).toBeNull();
		expect(display.title).toBe('Dati registrati');
		expect(display.sections).toEqual([]);
		expect(display.rows).toEqual([
			{ label: 'Peso camion live', value: '1200', error: null, tone: null },
			{ label: 'Somma pesi bancali live', value: '1180', error: null, tone: null },
			{ label: 'Match %', value: '98.33333333333333', error: null, tone: 'green' }
		]);
	});
});
