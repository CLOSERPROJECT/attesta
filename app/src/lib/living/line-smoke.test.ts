import { describe, expect, test, vi } from 'vitest';

import type { TaskFormBundle } from './form/task-form-bundle.js';
import { lineLoadLeftoverIds } from './leftovers.js';
import { LivingSession } from './session.svelte.js';
import {
	emptyDomain,
	worklistJobFor,
	type Domain,
	type LoadResult,
	type Submission,
	type Worklist
} from './types.js';

const lineTaskIds = ['shift', 'lineLoad', 'containerReg', 'loadedExtracted'];

function sub(
	jobId: string,
	taskId: string,
	jobInstanceId: string,
	facts: Record<string, unknown> = {}
): Submission {
	return {
		jobId,
		taskId,
		jobInstanceId,
		timestamp: 't',
		userId: 'ada',
		facts,
		idsTouched: []
	};
}

/** Fixture Worklist — mirrors Go ProjectWorklist for this smoke path (no SPA gates). */
function worklistFor(domain: Domain, canStartLine: boolean): Worklist {
	const lineInst = domain.jobInstances.find((i) => i.jobId === 'line');
	if (!lineInst) {
		return {
			jobs: [
				{
					jobId: 'intake',
					canStart: true,
					repeatable: true,
					taskIds: ['register', 'codiceFir', 'palletWeight', 'weightCheck'],
					instances: []
				},
				{
					jobId: 'line',
					canStart: canStartLine,
					repeatable: true,
					startLabel: 'Nuovo turno',
					taskIds: lineTaskIds,
					instances: []
				}
			]
		};
	}
	const tasks: Record<string, 'locked' | 'available' | 'done'> = {
		shift: 'locked',
		lineLoad: 'locked',
		containerReg: 'locked',
		loadedExtracted: 'locked'
	};
	const lineSubs = domain.submissions.filter(
		(s) => s.jobId === 'line' && s.jobInstanceId === lineInst.id
	);
	const has = (taskId: string) => lineSubs.some((s) => s.taskId === taskId);
	const leftoversOpen = lineLoadLeftoverIds(domain).length > 0;

	if (has('shift')) tasks.shift = 'done';
	else tasks.shift = 'available';

	if (has('shift')) {
		tasks.lineLoad = leftoversOpen || !has('lineLoad') ? 'available' : 'done';
		tasks.containerReg = has('containerReg') ? 'done' : 'available';
	}
	if (has('containerReg')) {
		tasks.loadedExtracted = has('loadedExtracted') ? 'done' : 'available';
	}

	return {
		jobs: [
			{
				jobId: 'intake',
				canStart: true,
				repeatable: true,
				taskIds: ['register', 'codiceFir', 'palletWeight', 'weightCheck'],
				instances: []
			},
			{
				jobId: 'line',
				canStart: canStartLine,
				repeatable: true,
				startLabel: 'Nuovo turno',
				taskIds: lineTaskIds,
				instances: [
					{
						id: lineInst.id,
						label: lineInst.label,
						tasks
					}
				]
			}
		]
	};
}

describe('line Commit smoke (session + Worklist + leftovers some)', () => {
	test('intake gate then one line instance through loadedExtracted', async () => {
		const intakeId = 'NIR-261019-NO-1';
		let domain: Domain = {
			...emptyDomain(),
			load: {
				[intakeId]: { client: 'NO', truckWeight: 900, truckTare: 50 }
			},
			pallet: {
				[`${intakeId}.1`]: { loadId: intakeId, weight: 400 },
				[`${intakeId}.2`]: { loadId: intakeId, weight: 450 }
			},
			jobInstances: [{ jobId: 'intake', id: intakeId, label: intakeId }],
			submissions: [sub('intake', 'palletWeight', intakeId)]
		};
		let worklist = worklistFor(domain, true);

		const commit = vi.fn(
			async (req: {
				taskPath: string;
				jobInstanceId: string;
				value: Record<string, unknown>;
			}) => {
				if (req.taskPath === 'line/shift') {
					domain = {
						...domain,
						jobInstances: [
							...domain.jobInstances,
							{
								jobId: 'line',
								id: req.jobInstanceId,
								label: `Turno: ${req.jobInstanceId}`
							}
						],
						submissions: [
							...domain.submissions,
							sub('line', 'shift', req.jobInstanceId, req.value)
						]
					};
				} else if (req.taskPath === 'line/lineLoad') {
					const progress = (req.value.progress ?? {}) as Record<string, string | null>;
					const pallet = { ...domain.pallet };
					for (const [id, cell] of Object.entries(progress)) {
						const facts = { ...(pallet[id] ?? {}) };
						if (cell === null) delete facts.progress;
						else facts.progress = cell;
						pallet[id] = facts;
					}
					domain = {
						...domain,
						pallet,
						submissions: [
							...domain.submissions,
							sub('line', 'lineLoad', req.jobInstanceId, progress)
						]
					};
				} else if (req.taskPath === 'line/containerReg') {
					domain = {
						...domain,
						container: {
							'C-1': {
								weight: 120,
								material: 'Polvere di vetro',
								sourceLoadId: intakeId
							},
							'C-2': {
								weight: 40,
								material: 'Plastica',
								sourceLoadId: intakeId
							}
						},
						submissions: [
							...domain.submissions,
							sub('line', 'containerReg', req.jobInstanceId, req.value)
						]
					};
				} else if (req.taskPath === 'line/loadedExtracted') {
					domain = {
						...domain,
						submissions: [
							...domain.submissions,
							sub('line', 'loadedExtracted', req.jobInstanceId, req.value)
						]
					};
				}
				worklist = worklistFor(domain, true);
				const lineJob = worklistJobFor(worklist, 'line');
				const lineInst = lineJob?.instances[0];
				const taskFormBundles: TaskFormBundle[] = [];
				if (lineJob?.canStart) {
					taskFormBundles.push({
						jobId: 'line',
						taskId: 'shift',
						jobInstanceId: '',
						title: 'Shift',
						surface: 'live',
						schema: { type: 'object' },
						uiSchema: {},
						initialData: {},
						chrome: [],
						chromeFieldIds: [],
						sealFieldIds: ['shift'],
						bindings: { domain: {} }
					});
				}
				if (lineInst) {
					for (const taskId of lineTaskIds) {
						const st = lineInst.tasks[taskId];
						if (st !== 'available') continue;
						taskFormBundles.push({
							jobId: 'line',
							taskId,
							jobInstanceId: lineInst.id,
							title: taskId,
							surface: taskId === 'lineLoad' ? 'leftover-table' : 'live',
							schema: { type: 'object' },
							uiSchema: {},
							initialData: {},
							chrome: [],
							chromeFieldIds: [],
							sealFieldIds: [],
							bindings: { domain: {} }
						});
					}
				}
				return {
					org: 'feragame',
					jobInstanceId: req.jobInstanceId,
					worklist,
					taskFormBundles
				};
			}
		);

		const initial: LoadResult = {
			org: 'feragame',
			activeConfigVersion: '1',
			configId: 'truck-loop',
			worklist,
			taskFormBundles: []
		};

		const session = new LivingSession({
			load: {
				load: async () => ({
					...initial,
					worklist,
					taskFormBundles: []
				})
			},
			commit: { commit },
			hydrate: false
		});
		session.result = { ...initial, worklist, taskFormBundles: [] };

		expect(worklistJobFor(session.result.worklist, 'line')?.canStart).toBe(true);
		expect(lineLoadLeftoverIds(domain)).toEqual([`${intakeId}.1`, `${intakeId}.2`]);

		expect(
			await session.commit({
				taskPath: 'line/shift',
				jobInstanceId: '1st',
				value: { shift: '1st' },
				mint: { kind: 'startingNew' }
			})
		).toBe('1st');

		expect(session.result?.worklist).toEqual(worklist);
		expect(
			worklistJobFor(session.result!.worklist, 'line')?.instances[0]?.tasks.containerReg
		).toBe('available');
		expect(
			session.result?.taskFormBundles.some(
				(b) => b.jobId === 'line' && b.taskId === 'lineLoad' && b.jobInstanceId === '1st'
			)
		).toBe(true);
		expect(
			session.result?.taskFormBundles.some(
				(b) => b.jobId === 'line' && b.taskId === 'shift' && b.jobInstanceId === ''
			)
		).toBe(true);

		expect(
			await session.commit({
				taskPath: 'line/lineLoad',
				jobInstanceId: '1st',
				value: {
					progress: {
						[`${intakeId}.1`]: 'done',
						[`${intakeId}.2`]: 'partial'
					}
				},
				mint: { kind: 'none' }
			})
		).toBe('1st');
		expect(lineLoadLeftoverIds(domain)).toEqual([`${intakeId}.2`]);
		expect(
			worklistJobFor(session.result!.worklist, 'line')?.instances[0]?.tasks.lineLoad
		).toBe('available');
		expect(
			session.result?.taskFormBundles.some(
				(b) => b.jobId === 'line' && b.taskId === 'containerReg' && b.jobInstanceId === '1st'
			)
		).toBe(true);

		expect(
			await session.commit({
				taskPath: 'line/lineLoad',
				jobInstanceId: '1st',
				value: { progress: { [`${intakeId}.2`]: 'done' } },
				mint: { kind: 'none' }
			})
		).toBe('1st');
		expect(lineLoadLeftoverIds(domain)).toEqual([]);
		expect(
			worklistJobFor(session.result!.worklist, 'line')?.instances[0]?.tasks.lineLoad
		).toBe('done');

		expect(
			await session.commit({
				taskPath: 'line/containerReg',
				jobInstanceId: '1st',
				value: {
					sourceLoadId: intakeId,
					polvereDiVetro: [{ weight: 120 }],
					plastica: [{ weight: 40 }]
				},
				mint: { kind: 'none' }
			})
		).toBe('1st');

		expect(
			await session.commit({
				taskPath: 'line/loadedExtracted',
				jobInstanceId: '1st',
				value: { checkedLoadedWeight: 850, checkedContainerSum: 160 },
				mint: { kind: 'none' }
			})
		).toBe('1st');

		const lineSubs = domain.submissions.filter(
			(s) => s.jobId === 'line' && s.jobInstanceId === '1st'
		);
		expect(lineSubs.map((s) => s.taskId)).toEqual([
			'shift',
			'lineLoad',
			'lineLoad',
			'containerReg',
			'loadedExtracted'
		]);
		expect(Object.keys(domain.container)).toHaveLength(2);
		expect(worklistJobFor(session.result!.worklist, 'line')?.canStart).toBe(true);
		expect(
			worklistJobFor(session.result!.worklist, 'line')?.instances[0]?.tasks.loadedExtracted
		).toBe('done');
	});
});
