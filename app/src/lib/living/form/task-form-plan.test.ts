import { describe, expect, test } from 'vitest';
import truckLoopStepsYaml from './fixtures/truck-loop-steps.yaml?raw';
import { loadTruckLoop, substepConfig } from './load-truck-loop.js';
import { evaluateChromeEntry } from './live-task-chrome.svelte.js';
import { leftoversIdentity } from './project-form-schema.js';
import { compileTaskFormPlan } from './task-form-plan.js';

function isPlan(
	result: ReturnType<typeof compileTaskFormPlan>
): result is Exclude<typeof result, { error: string }> {
	return !('error' in result);
}

describe('compileTaskFormPlan', () => {
	const config = loadTruckLoop(truckLoopStepsYaml);

	test('register: placeholders injected; schema properties match authored (chrome ids separate)', () => {
		const authored = substepConfig(config.jobs.intake, 'register')!;
		const plan = compileTaskFormPlan(authored);
		expect(isPlan(plan)).toBe(true);
		if (!isPlan(plan)) return;

		expect(plan.surface).toBe('live');
		expect(plan.projectionKey).toBe('');
		const authoredProps = (authored.schema as { properties?: Record<string, unknown> }).properties!;
		const planProps = (plan.schemas.schema as { properties?: Record<string, unknown> }).properties!;
		for (const key of Object.keys(authoredProps)) {
			expect(planProps[key]).toEqual(authoredProps[key]);
		}

		const clientUi = (plan.schemas.uiSchema as Record<string, unknown>).client as {
			'ui:options'?: {
				enumNames?: string[];
				shadcn4Text?: { placeholder?: string };
			};
		};
		expect(clientUi['ui:options']?.enumNames).toEqual(['Nord', 'Sud']);
		expect(clientUi['ui:options']?.shadcn4Text?.placeholder).toBe('Inserisci Cliente…');
		const wasteUi = (plan.schemas.uiSchema as Record<string, unknown>).wasteCode as {
			'ui:options'?: {
				enumNames?: string[];
				shadcn4Text?: { placeholder?: string };
			};
		};
		expect(wasteUi['ui:options']?.enumNames).toEqual([
			'15 01 01 — Imballaggi in carta e cartone',
			'15 01 02 — Imballaggi in plastica',
			'15 01 03 — Imballaggi in legno',
			'15 01 04 — Imballaggi metallici',
			'19 12 01 — Carta e cartone',
			'19 12 04 — Plastica e gomma',
			'19 12 12 — Altri rifiuti da trattamento meccanico'
		]);
		const truckUi = (plan.schemas.uiSchema as Record<string, unknown>).truckWeight as {
			'ui:options'?: { shadcn4Number?: { placeholder?: string } };
		};
		expect(truckUi['ui:options']?.shadcn4Number?.placeholder).toBe(
			'Inserisci Peso totale camion…'
		);
		// Does not mutate authored YAML in place (enumNames stay; no placeholder bleed).
		const authoredClient = (authored.uiSchema as Record<string, unknown>).client as {
			'ui:options'?: { enumNames?: string[]; shadcn4Text?: unknown };
		};
		expect(authoredClient['ui:options']?.enumNames).toEqual(['Nord', 'Sud']);
		expect(authoredClient['ui:options']?.shadcn4Text).toBeUndefined();
		const authoredWaste = (authored.uiSchema as Record<string, unknown>).wasteCode as {
			'ui:options'?: { enumNames?: string[]; shadcn4Text?: unknown };
		};
		expect(authoredWaste['ui:options']?.enumNames?.[6]).toBe(
			'19 12 12 — Altri rifiuti da trattamento meccanico'
		);
		expect(authoredWaste['ui:options']?.shadcn4Text).toBeUndefined();

		const draft = { client: 'NO', truckWeight: 1 };
		expect(plan.sealSubmission(draft)).toEqual(draft);
	});

	test('truck-loop register effectiveWeight readonly chrome; seal strips id', async () => {
		const register = substepConfig(config.jobs.intake, 'register')!;
		const plan = compileTaskFormPlan(register);
		expect(isPlan(plan)).toBe(true);
		if (!isPlan(plan)) return;

		expect(plan.chromeFieldIds).toEqual(['progressiveNumber', 'effectiveWeight']);
		const progressiveNumber = plan.chrome.find((c) => c.id === 'progressiveNumber');
		expect(progressiveNumber).toEqual({
			id: 'progressiveNumber',
			kind: 'readonly',
			expr: "$count($filter($domain.load.*, function($v){$v.client = $.client and $v.date = $.date})) + 1",
			after: 'client'
		});
		const effectiveWeight = plan.chrome.find((c) => c.id === 'effectiveWeight');
		expect(effectiveWeight).toEqual({
			id: 'effectiveWeight',
			kind: 'readonly',
			expr: "$type($.truckWeight) = 'number' and $type($.truckTare) = 'number' ? $.truckWeight - $.truckTare : null",
			after: 'truckTare'
		});

		const order = (
			plan.schemas.uiSchema as { 'ui:options'?: { order?: string[] } }
		)['ui:options']?.order;
		expect(order).toEqual([
			'client',
			'progressiveNumber',
			'date',
			'wasteCode',
			'idoneitaRiutilizzo',
			'truckWeight',
			'truckTare',
			'effectiveWeight',
			'palletCount',
			'notes'
		]);

		const props = (plan.schemas.schema as { properties?: Record<string, unknown> }).properties!;
		expect(props.progressiveNumber).toEqual({
			type: 'string',
			title: 'Progressivo cliente',
			readOnly: true
		});
		expect(props.effectiveWeight).toEqual({
			type: 'string',
			title: 'Peso netto',
			readOnly: true
		});

		const progressiveUi = (plan.schemas.uiSchema as Record<string, unknown>)
			.progressiveNumber as Record<string, unknown>;
		expect(progressiveUi['ui:components']).toEqual({ stringField: 'readonlyField' });
		expect(progressiveUi['ui:options']).toEqual({ chromeId: 'progressiveNumber' });

		const readonlyUi = (plan.schemas.uiSchema as Record<string, unknown>).effectiveWeight as Record<
			string,
			unknown
		>;
		expect(readonlyUi['ui:components']).toEqual({ stringField: 'readonlyField' });
		expect(readonlyUi['ui:options']).toEqual({ chromeId: 'effectiveWeight' });

		const sealed = plan.sealSubmission({
			client: 'NO',
			date: '2026-10-26',
			wasteCode: '150101',
			truckWeight: 1200,
			truckTare: 400,
			progressiveNumber: '1',
			effectiveWeight: '800',
			palletCount: 2
		});
		expect(sealed).toEqual({
			client: 'NO',
			date: '2026-10-26',
			wasteCode: '150101',
			truckWeight: 1200,
			truckTare: 400,
			palletCount: 2
		});
		expect(sealed).not.toHaveProperty('effectiveWeight');
		expect(sealed).not.toHaveProperty('progressiveNumber');

		expect(effectiveWeight?.kind).toBe('readonly');
		if (effectiveWeight?.kind !== 'readonly' && effectiveWeight?.kind !== 'valuePrint') return;
		expect(await evaluateChromeEntry(effectiveWeight.expr, sealed, {}, undefined)).toEqual({
			value: '800',
			error: null
		});
		expect(
			await evaluateChromeEntry(effectiveWeight.expr, { truckWeight: 1200 }, {}, undefined)
		).toEqual({ value: null, error: null });
	});

	test('default placeholders skip chrome; leave authored placeholder alone', () => {
		const authored = {
			schema: {
				type: 'object',
				properties: {
					client: { type: 'string', title: 'Cliente' },
					notes: { type: 'string', title: 'Note' }
				}
			},
			uiSchema: {
				'ui:options': { order: ['client', 'notes'] },
				notes: {
					'ui:options': { placeholder: 'Optional remarks' }
				}
			},
			chrome: {
				registeredData: {
					kind: 'recap',
					title: 'Dati registrati',
					before: 'client',
					rows: [{ label: 'X', expr: '1' }]
				}
			}
		};
		const plan = compileTaskFormPlan(authored);
		expect(isPlan(plan)).toBe(true);
		if (!isPlan(plan)) return;

		const clientUi = (plan.schemas.uiSchema as Record<string, unknown>).client as {
			'ui:options'?: { shadcn4Text?: { placeholder?: string } };
		};
		expect(clientUi['ui:options']?.shadcn4Text?.placeholder).toBe('Inserisci Cliente…');

		const notesUi = (plan.schemas.uiSchema as Record<string, unknown>).notes as {
			'ui:options'?: { placeholder?: string; shadcn4Text?: { placeholder?: string } };
		};
		expect(notesUi['ui:options']?.placeholder).toBe('Optional remarks');
		expect(notesUi['ui:options']?.shadcn4Text).toBeUndefined();

		const chromeUi = (plan.schemas.uiSchema as Record<string, unknown>).registeredData as Record<
			string,
			unknown
		>;
		expect(chromeUi['ui:components']).toEqual({ stringField: 'recapField' });
		expect(
			(chromeUi['ui:options'] as { shadcn4Text?: unknown } | undefined)?.shadcn4Text
		).toBeUndefined();
	});

	test('truck-loop weightCheck recap; seal strips recap ids', async () => {
		const weightCheck = substepConfig(config.jobs.intake, 'weightCheck')!;
		const plan = compileTaskFormPlan(weightCheck);
		expect(isPlan(plan)).toBe(true);
		if (!isPlan(plan)) return;

		expect(plan.chromeFieldIds).toEqual(['registeredData', 'currentOperator']);
		const recap = plan.chrome.find((c) => c.kind === 'recap');
		expect(recap?.kind).toBe('recap');
		if (recap?.kind !== 'recap') return;
		expect(recap.title).toBe('Dati registrati');
		expect(recap.before).toBe('checkedTruckWeight');
		expect(recap.rows).toHaveLength(3);
		expect(recap.rows[2]?.bands).toEqual([
			{ min: 98, tone: 'green' },
			{ min: 90, tone: 'warning' },
			{ min: 0, tone: 'danger' }
		]);

		const currentOperator = plan.chrome.find((c) => c.kind === 'readonly');
		expect(currentOperator).toEqual({
			id: 'currentOperator',
			kind: 'readonly',
			expr: '$operator.displayName',
			after: 'checkedPalletSum'
		});

		const order = (
			plan.schemas.uiSchema as { 'ui:options'?: { order?: string[] } }
		)['ui:options']?.order;
		expect(order).toEqual([
			'registeredData',
			'checkedTruckWeight',
			'checkedPalletSum',
			'currentOperator'
		]);

		const chromeUi = (plan.schemas.uiSchema as Record<string, unknown>).registeredData as Record<
			string,
			unknown
		>;
		expect(chromeUi['ui:components']).toEqual({ stringField: 'recapField' });
		expect(chromeUi['ui:options']).toEqual({ chromeId: 'registeredData' });

		const readonlyUi = (plan.schemas.uiSchema as Record<string, unknown>).currentOperator as Record<
			string,
			unknown
		>;
		expect(readonlyUi['ui:components']).toEqual({ stringField: 'readonlyField' });
		expect(readonlyUi['ui:options']).toEqual({ chromeId: 'currentOperator' });

		const sealed = plan.sealSubmission({
			checkedTruckWeight: 880,
			checkedPalletSum: 850,
			registeredData: 'ignored',
			currentOperator: 'Ada'
		});
		expect(sealed).toEqual({
			checkedTruckWeight: 880,
			checkedPalletSum: 850
		});

		const domain = {
			load: { 'load-1': { truckWeight: 900 } },
			pallet: {
				'load-1.1': { loadId: 'load-1', weight: 400 },
				'load-1.2': { loadId: 'load-1', weight: 450 }
			}
		};
		const instance = { id: 'load-1' };
		expect(await evaluateChromeEntry(recap.rows[0]!.expr, sealed, domain, instance)).toEqual({
			value: '900',
			error: null
		});
		expect(await evaluateChromeEntry(recap.rows[1]!.expr, sealed, domain, instance)).toEqual({
			value: '850',
			error: null
		});
		expect(await evaluateChromeEntry(recap.rows[2]!.expr, sealed, domain, instance)).toEqual({
			value: String((100 * 850) / 900),
			error: null
		});

		expect(JSON.stringify(weightCheck)).not.toContain('liveTruckWeight');
	});

	test('truck-loop loadedExtracted finalResults chrome; seal strips chrome id', () => {
		const loaded = substepConfig(config.jobs.line, 'loadedExtracted')!;
		const loadedPlan = compileTaskFormPlan(loaded);
		expect(isPlan(loadedPlan)).toBe(true);
		if (!isPlan(loadedPlan)) return;

		expect(loadedPlan.chromeFieldIds).toEqual(['feragameFinalResults']);
		const finalResults = loadedPlan.chrome.find((c) => c.kind === 'finalResults');
		expect(finalResults).toEqual({
			id: 'feragameFinalResults',
			kind: 'finalResults',
			title: 'Risultati finali Feragame',
			before: 'checkedLoadedWeight'
		});
		expect(loadedPlan.chrome.some((c) => c.kind === 'recap')).toBe(false);

		const loadedOrder = (
			loadedPlan.schemas.uiSchema as { 'ui:options'?: { order?: string[] } }
		)['ui:options']?.order;
		expect(loadedOrder).toEqual([
			'feragameFinalResults',
			'checkedLoadedWeight',
			'checkedContainerSum'
		]);

		const chromeUi = (loadedPlan.schemas.uiSchema as Record<string, unknown>)
			.feragameFinalResults as Record<string, unknown>;
		expect(chromeUi['ui:components']).toEqual({ stringField: 'feragameFinalResultsField' });
		expect(chromeUi['ui:options']).toEqual({ chromeId: 'feragameFinalResults' });

		expect(
			loadedPlan.sealSubmission({
				checkedLoadedWeight: 850,
				checkedContainerSum: 120,
				feragameFinalResults: 'ignored'
			})
		).toEqual({
			checkedLoadedWeight: 850,
			checkedContainerSum: 120
		});

		expect(JSON.stringify(loaded)).not.toContain('liveLoadedWeight');
		expect(JSON.stringify(loaded)).not.toContain('registeredData');
		expect(JSON.stringify(loaded)).not.toContain('Match %');
	});

	test('injects recap chrome before named field with recapField marker', () => {
		const authored = {
			schema: {
				type: 'object',
				required: ['checkedTruckWeight'],
				properties: {
					checkedTruckWeight: { type: 'number', title: 'Peso camion verificato' }
				}
			},
			uiSchema: {
				'ui:options': { order: ['checkedTruckWeight'] }
			},
			chrome: {
				registeredData: {
					kind: 'recap',
					title: 'Dati registrati',
					before: 'checkedTruckWeight',
					rows: [
						{ label: 'Truck', expr: '$domain.truckWeight' },
						{
							label: 'Match %',
							expr: '100',
							bands: [
								{ min: 98, tone: 'green' },
								{ min: 0, tone: 'danger' }
							]
						}
					]
				}
			}
		};
		const plan = compileTaskFormPlan(authored);
		expect(isPlan(plan)).toBe(true);
		if (!isPlan(plan)) return;

		expect(plan.chrome).toEqual([
			{
				id: 'registeredData',
				kind: 'recap',
				title: 'Dati registrati',
				before: 'checkedTruckWeight',
				rows: [
					{ label: 'Truck', expr: '$domain.truckWeight' },
					{
						label: 'Match %',
						expr: '100',
						bands: [
							{ min: 98, tone: 'green' },
							{ min: 0, tone: 'danger' }
						]
					}
				]
			}
		]);

		const props = (plan.schemas.schema as { properties?: Record<string, unknown> }).properties!;
		expect(props.registeredData).toEqual({
			type: 'string',
			title: 'Dati registrati',
			readOnly: true
		});

		const order = (
			plan.schemas.uiSchema as { 'ui:options'?: { order?: string[] } }
		)['ui:options']?.order;
		expect(order).toEqual(['registeredData', 'checkedTruckWeight']);

		const chromeUi = (plan.schemas.uiSchema as Record<string, unknown>).registeredData as Record<
			string,
			unknown
		>;
		expect(chromeUi['ui:components']).toEqual({ stringField: 'recapField' });
		expect(chromeUi['ui:options']).toEqual({ chromeId: 'registeredData' });
	});

	test('injects finalResults chrome with feragameFinalResultsField marker; seal strips id', () => {
		const authored = {
			schema: {
				type: 'object',
				required: ['checkedLoadedWeight', 'checkedContainerSum'],
				properties: {
					checkedLoadedWeight: { type: 'number', title: 'Peso caricato verificato' },
					checkedContainerSum: { type: 'number', title: 'Somma pesi contenitori verificata' }
				}
			},
			uiSchema: {
				'ui:options': { order: ['checkedLoadedWeight', 'checkedContainerSum'] }
			},
			chrome: {
				feragameFinalResults: {
					kind: 'finalResults',
					title: 'Risultati finali Feragame',
					before: 'checkedLoadedWeight'
				}
			}
		};
		const plan = compileTaskFormPlan(authored);
		expect(isPlan(plan)).toBe(true);
		if (!isPlan(plan)) return;

		expect(plan.chrome).toEqual([
			{
				id: 'feragameFinalResults',
				kind: 'finalResults',
				title: 'Risultati finali Feragame',
				before: 'checkedLoadedWeight'
			}
		]);
		expect(plan.chromeFieldIds).toEqual(['feragameFinalResults']);

		const props = (plan.schemas.schema as { properties?: Record<string, unknown> }).properties!;
		expect(props.feragameFinalResults).toEqual({
			type: 'string',
			title: 'Risultati finali Feragame',
			readOnly: true
		});

		const order = (
			plan.schemas.uiSchema as { 'ui:options'?: { order?: string[] } }
		)['ui:options']?.order;
		expect(order).toEqual([
			'feragameFinalResults',
			'checkedLoadedWeight',
			'checkedContainerSum'
		]);

		const chromeUi = (plan.schemas.uiSchema as Record<string, unknown>)
			.feragameFinalResults as Record<string, unknown>;
		expect(chromeUi['ui:components']).toEqual({ stringField: 'feragameFinalResultsField' });
		expect(chromeUi['ui:options']).toEqual({ chromeId: 'feragameFinalResults' });

		expect(
			plan.sealSubmission({
				checkedLoadedWeight: 100,
				checkedContainerSum: 90,
				feragameFinalResults: 'ignored'
			})
		).toEqual({
			checkedLoadedWeight: 100,
			checkedContainerSum: 90
		});
	});

	test('ignores unknown chrome kinds (finalResults sibling still parses)', () => {
		const plan = compileTaskFormPlan({
			schema: {
				type: 'object',
				properties: { fieldA: { type: 'string' } }
			},
			uiSchema: { 'ui:options': { order: ['fieldA'] } },
			chrome: {
				bad: { kind: 'notARealKind', expr: '1' },
				ok: { kind: 'finalResults', before: 'fieldA' }
			}
		});
		expect(isPlan(plan)).toBe(true);
		if (!isPlan(plan)) return;
		expect(plan.chrome).toEqual([{ id: 'ok', kind: 'finalResults', before: 'fieldA' }]);
		expect(plan.chromeFieldIds).toEqual(['ok']);
	});

	test('surface leftover-table for flat leftoverTable substeps', () => {
		const authored = substepConfig(config.jobs.line, 'lineLoad')!;
		const plan = compileTaskFormPlan(authored);
		expect(isPlan(plan)).toBe(true);
		if (!isPlan(plan)) return;
		expect(plan.surface).toBe('leftover-table');
	});

	test('injects readonly chrome fields, order after, and ui markers', () => {
		const authored = {
			schema: {
				type: 'object',
				required: ['checkedTruckWeight'],
				properties: {
					checkedTruckWeight: { type: 'number', title: 'Peso camion verificato' }
				}
			},
			uiSchema: {
				'ui:options': { order: ['checkedTruckWeight'] }
			},
			chrome: {
				currentOperator: {
					kind: 'readonly',
					expr: '$operator.displayName',
					after: 'checkedTruckWeight',
					title: 'Operatore corrente'
				}
			}
		};
		const plan = compileTaskFormPlan(authored);
		expect(isPlan(plan)).toBe(true);
		if (!isPlan(plan)) return;

		expect(plan.chrome).toEqual([
			{
				id: 'currentOperator',
				kind: 'readonly',
				expr: '$operator.displayName',
				after: 'checkedTruckWeight'
			}
		]);
		expect(plan.chromeFieldIds).toEqual(['currentOperator']);

		const props = (plan.schemas.schema as { properties?: Record<string, unknown> }).properties!;
		expect(props.currentOperator).toEqual({
			type: 'string',
			title: 'Operatore corrente',
			readOnly: true
		});

		const order = (
			plan.schemas.uiSchema as { 'ui:options'?: { order?: string[] } }
		)['ui:options']?.order;
		expect(order).toEqual(['checkedTruckWeight', 'currentOperator']);

		const chromeUi = (plan.schemas.uiSchema as Record<string, unknown>).currentOperator as Record<
			string,
			unknown
		>;
		expect(chromeUi['ui:components']).toEqual({ stringField: 'readonlyField' });
		expect(chromeUi['ui:options']).toEqual({ chromeId: 'currentOperator' });

		expect(
			plan.sealSubmission({
				checkedTruckWeight: 1,
				currentOperator: 'Ada'
			})
		).toEqual({ checkedTruckWeight: 1 });
	});

	test('injects valuePrint chrome fields, order after, and ui markers', () => {
		const authored = {
			schema: {
				type: 'object',
				required: ['checkedTruckWeight'],
				properties: {
					checkedTruckWeight: { type: 'number', title: 'Peso camion verificato' }
				}
			},
			uiSchema: {
				'ui:options': { order: ['checkedTruckWeight'] }
			},
			chrome: {
				liveTruckWeight: {
					kind: 'valuePrint',
					expr: '$domain.load.truckWeight',
					after: 'checkedTruckWeight',
					title: 'Peso camion live'
				}
			}
		};
		const plan = compileTaskFormPlan(authored);
		expect(isPlan(plan)).toBe(true);
		if (!isPlan(plan)) return;

		expect(plan.chrome).toEqual([
			{
				id: 'liveTruckWeight',
				kind: 'valuePrint',
				expr: '$domain.load.truckWeight',
				after: 'checkedTruckWeight'
			}
		]);
		expect(plan.chromeFieldIds).toEqual(['liveTruckWeight']);

		const props = (plan.schemas.schema as { properties?: Record<string, unknown> }).properties!;
		expect(props.liveTruckWeight).toEqual({
			type: 'string',
			title: 'Peso camion live',
			readOnly: true
		});
		expect((plan.schemas.schema as { required?: string[] }).required).toEqual([
			'checkedTruckWeight'
		]);

		const order = (
			plan.schemas.uiSchema as { 'ui:options'?: { order?: string[] } }
		)['ui:options']?.order;
		expect(order).toEqual(['checkedTruckWeight', 'liveTruckWeight']);

		const chromeUi = (plan.schemas.uiSchema as Record<string, unknown>).liveTruckWeight as Record<
			string,
			unknown
		>;
		expect(chromeUi['ui:components']).toEqual({ stringField: 'valuePrintField' });
		expect(chromeUi['ui:options']).toEqual({ chromeId: 'liveTruckWeight' });
	});

	test('sealSubmission strips chrome field ids', () => {
		const authored = {
			schema: {
				type: 'object',
				properties: { fieldA: { type: 'string' } }
			},
			uiSchema: { 'ui:options': { order: ['fieldA'] } },
			chrome: {
				displayA: { kind: 'valuePrint', expr: '1' }
			}
		};
		const plan = compileTaskFormPlan(authored);
		expect(isPlan(plan)).toBe(true);
		if (!isPlan(plan)) return;

		expect(
			plan.sealSubmission({
				fieldA: 'x',
				displayA: 'ignored'
			})
		).toEqual({ fieldA: 'x' });
	});

	test('chrome id colliding with authored property returns error', () => {
		const plan = compileTaskFormPlan({
			schema: {
				type: 'object',
				properties: { liveTruckWeight: { type: 'number' } }
			},
			uiSchema: {},
			chrome: {
				liveTruckWeight: { kind: 'valuePrint', expr: '1' }
			}
		});
		expect(plan).toEqual({ error: 'chrome id "liveTruckWeight" collides with authored property' });
	});

	test('passes through concrete derived keys', () => {
		const authored = {
			schema: { type: 'object', properties: { x: { type: 'number' } } },
			uiSchema: {},
			derived: {
				'x.total': { expr: '$sum($data.x)', overridable: true }
			}
		};
		const plan = compileTaskFormPlan(authored);
		expect(isPlan(plan)).toBe(true);
		if (!isPlan(plan)) return;
		expect(plan.derived).toEqual(authored.derived);
	});

	test('expands map-star derived when ctx supplies leftoverIds', () => {
		const palletWeight = substepConfig(config.jobs.intake, 'palletWeight') as {
			derived?: Record<string, { expr: string; overridable?: boolean }>;
		};
		const plan = compileTaskFormPlan(palletWeight, { leftoverIds: ['p1', 'p2'] });
		expect(isPlan(plan)).toBe(true);
		if (!isPlan(plan)) return;

		expect(plan.derived?.['weights.p1.panelCount']?.expr).toContain('p1');
		expect(plan.derived?.['weights.p2.panelCount']?.expr).toContain('p2');
		expect(plan.derived?.['weights.*.panelCount']).toBeUndefined();
	});

	test('projectionKey uses leftoversIdentity for mapTable with leftoverIds', () => {
		const palletWeight = substepConfig(config.jobs.intake, 'palletWeight')!;
		const ids = ['load-1/2', 'load-1/1'];
		const plan = compileTaskFormPlan(palletWeight, { leftoverIds: ids });
		expect(isPlan(plan)).toBe(true);
		if (!isPlan(plan)) return;

		expect(plan.surface).toBe('live');
		expect(plan.projectionKey).toBe(leftoversIdentity(ids));
	});

	test('rejects non-object authored', () => {
		expect(compileTaskFormPlan(null)).toEqual({ error: 'authored substep must be an object' });
	});

	test('projectedSchemas replace display base before chrome merge (mapTable paint)', () => {
		const authored = {
			schema: {
				type: 'object',
				properties: { placeholder: { type: 'string' } }
			},
			uiSchema: { 'ui:options': { order: ['placeholder'] } },
			chrome: {
				hint: { kind: 'valuePrint', expr: '1', after: 'placeholder' }
			}
		};
		const projected = {
			schema: {
				type: 'object' as const,
				properties: { weights: { type: 'object' as const } }
			},
			uiSchema: { weights: { 'ui:field': 'mapTableField' } }
		};
		const plan = compileTaskFormPlan(authored, {
			projectedSchemas: projected as {
				schema: import('@sjsf/form').Schema;
				uiSchema: import('@sjsf/form').UiSchemaRoot;
			}
		});
		expect(isPlan(plan)).toBe(true);
		if (!isPlan(plan)) return;

		const props = (plan.schemas.schema as { properties?: Record<string, unknown> }).properties!;
		expect(props.weights).toEqual({ type: 'object' });
		expect(props.placeholder).toBeUndefined();
		expect(props.hint).toBeDefined();
	});

	test('hides root form title for every truck-loop Substep; authored titles stay for accordion', () => {
		const withRootTitle: Array<{ stepId: string; substepId: string; title: string }> = [
			{ stepId: 'intake', substepId: 'palletWeight', title: 'Pesi bancali' },
			{ stepId: 'intake', substepId: 'weightCheck', title: 'Controllo camion vs bancali' },
			{ stepId: 'line', substepId: 'shift', title: 'Seleziona turno' },
			{ stepId: 'line', substepId: 'lineLoad', title: 'Segna avanzamento' },
			{ stepId: 'line', substepId: 'containerReg', title: 'Registrazione contenitori' },
			{ stepId: 'line', substepId: 'loadedExtracted', title: 'Controllo caricato vs estratto' }
		];

		for (const { stepId, substepId, title } of withRootTitle) {
			const authored = substepConfig(config.jobs[stepId]!, substepId)!;
			expect((authored.schema as { title?: string }).title).toBe(title);

			const plan = compileTaskFormPlan(authored);
			expect(isPlan(plan)).toBe(true);
			if (!isPlan(plan)) return;

			const rootOpts = (plan.schemas.uiSchema as { 'ui:options'?: { hideTitle?: boolean } })[
				'ui:options'
			];
			expect(rootOpts?.hideTitle).toBe(true);

			const authoredUi = authored.uiSchema as { 'ui:options'?: { hideTitle?: boolean } };
			expect(authoredUi['ui:options']?.hideTitle).toBeUndefined();
			expect((authored.schema as { title?: string }).title).toBe(title);
		}

		// No root title in YAML — still paint hideTitle so form chrome stays consistent.
		const register = substepConfig(config.jobs.intake, 'register')!;
		expect((register.schema as { title?: string }).title).toBeUndefined();
		const registerPlan = compileTaskFormPlan(register);
		expect(isPlan(registerPlan)).toBe(true);
		if (!isPlan(registerPlan)) return;
		expect(
			(registerPlan.schemas.uiSchema as { 'ui:options'?: { hideTitle?: boolean } })['ui:options']
				?.hideTitle
		).toBe(true);
		// Field titles remain (placeholders / labels).
		expect(
			(
				(registerPlan.schemas.schema as { properties?: Record<string, { title?: string }> })
					.properties!.client as { title?: string }
			).title
		).toBe('Cliente');
	});
});
