import { describe, expect, test, vi } from 'vitest';

import { LivingSession } from './session.svelte.js';
import {
	emptyDomain,
	worklistJobFor,
	type Domain,
	type LoadResult,
	type Submission,
	type Worklist
} from './types.js';

/** Live line Task ids — plant fixture (no parked loadedExtracted). */
const lineTaskIds = ['openTurno', 'palletWeight', 'lineLoad', 'containerReg', 'closeTurno'];

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

/**
 * Fixture Worklist — mirrors Go ProjectWorklist for this smoke path.
 * containerReg unlocks with openTurno (unlockFrom); lineLoad also needs openTurno.
 * Group stays available until Close (one register Submission does not mark done).
 */
function worklistFor(domain: Domain, canStartLine: boolean): Worklist {
	const lineInst = domain.jobInstances.find((i) => i.jobId === 'line');
	if (!lineInst) {
		return {
			jobs: [
				{
					jobId: 'intake',
					canStart: true,
					repeatable: true,
					taskIds: ['truckArrive', 'lots', 'weightCheck'],
					tasks: [],
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
	const tasks: Record<string, 'locked' | 'available' | 'actionable' | 'done'> = {
		openTurno: 'locked',
		palletWeight: 'locked',
		lineLoad: 'locked',
		containerReg: 'locked',
		closeTurno: 'locked'
	};
	const lineSubs = domain.submissions.filter(
		(s) => s.jobId === 'line' && s.jobInstanceId === lineInst.id
	);
	const has = (taskId: string) => lineSubs.some((s) => s.taskId === taskId);

	if (has('openTurno')) tasks.openTurno = 'done';
	else tasks.openTurno = 'actionable';

	if (has('openTurno')) {
		// Smoke skips weigh; treat palletWeight as already closed (seed-shaped).
		tasks.palletWeight = 'done';
		tasks.lineLoad = has('lineLoad') ? 'done' : 'actionable';
		// unlockFrom openTurno — group stays available until Close
		tasks.containerReg = 'available';
		// unlockFrom openTurno; smoke leaves shift open (no closeTurno Commit).
		tasks.closeTurno = has('closeTurno') ? 'done' : 'actionable';
	}

	return {
		jobs: [
			{
				jobId: 'intake',
				canStart: true,
				repeatable: true,
				taskIds: ['truckArrive', 'lots', 'weightCheck'],
				tasks: [],
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

describe('line Commit smoke (session + Worklist + group register)', () => {
	test('intake gate then line through openTurno, lineLoad, and containerReg register', async () => {
		const loadId = 'NO-261019-1';
		let domain: Domain = {
			...emptyDomain(),
			load: {
				[loadId]: { client: 'NO', truckWeight: 900, truckTare: 50 }
			},
			pallet: {
				[`${loadId}.1`]: { loadId, weight: 400 },
				[`${loadId}.2`]: { loadId, weight: 450 }
			},
			jobInstances: [{ jobId: 'intake', id: '261019-1', label: '261019-1' }],
			submissions: [sub('intake', 'lots', '261019-1')]
		};
		let worklist = worklistFor(domain, true);

		const commit = vi.fn(
			async (req: {
				taskPath: string;
				jobInstanceId: string;
				value: Record<string, unknown>;
				mint?: { kind: string };
			}) => {
				if (req.taskPath === 'line/openTurno') {
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
							sub('line', 'openTurno', req.jobInstanceId, req.value)
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
				} else if (req.taskPath === 'line/containerReg/new/register') {
					domain = {
						...domain,
						container: {
							'C-1': {
								weight: 120,
								material: 'Polvere di vetro',
								sourceLoadId: loadId,
								incomplete: false
							},
							'C-2': {
								weight: 40,
								material: 'Plastica',
								sourceLoadId: loadId,
								incomplete: false
							}
						},
						load: {
							...domain.load,
							[loadId]: { ...domain.load[loadId], fullyProcessed: true }
						},
						submissions: [
							...domain.submissions,
							sub('line', 'register', req.jobInstanceId, req.value)
						]
					};
				}
				worklist = worklistFor(domain, true);
				const lineInst = worklistJobFor(worklist, 'line')?.instances[0];
				let nextTaskFormBundle = null;
				let navigation: 'task' | 'overview' | 'home' = 'home';
				if (lineInst) {
					for (const taskId of lineTaskIds) {
						if (lineInst.tasks[taskId] !== 'actionable') continue;
						nextTaskFormBundle = {
							jobId: 'line',
							taskId,
							jobInstanceId: lineInst.id,
							title: taskId,
							surface: 'live' as const,
							schema: { type: 'object' },
							uiSchema: {},
							initialData: {},
							chrome: [],
							chromeFieldIds: [],
							sealFieldIds: [],
							bindings: { domain: {} }
						};
						navigation = 'task';
						break;
					}
					if (navigation !== 'task') {
						const hasRemaining = lineTaskIds.some(
							(tid) => lineInst.tasks[tid] !== 'done'
						);
						navigation = hasRemaining ? 'overview' : 'home';
					}
				}
				return {
					org: 'feragame',
					jobInstanceId: req.jobInstanceId,
					worklist,
					navigation,
					nextTaskFormBundle
				};
			}
		);

		const initial: LoadResult = {
			org: 'feragame',
			activeConfigVersion: '1',
			configId: 'feragame-config',
			userId: 'ada',
			displayName: 'Ada',
			roles: ['line', 'administration', 'logistic'],
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

		expect(
			(
				await session.commit({
					taskPath: 'line/openTurno',
					jobInstanceId: '2026-10-19-1st',
					value: { date: '2026-10-19', turno: '1st' },
					mint: { kind: 'startingNew' }
				})
			)?.jobInstanceId
		).toBe('2026-10-19-1st');

		expect(session.result?.worklist).toEqual(worklist);
		const afterOpen = worklistJobFor(session.result!.worklist, 'line')?.instances[0]?.tasks;
		expect(afterOpen?.lineLoad).toBe('actionable');
		expect(afterOpen?.containerReg).toBe('available');

		expect(
			(
				await session.commit({
					taskPath: 'line/lineLoad',
					jobInstanceId: '2026-10-19-1st',
					value: {
						progress: {
							[`${loadId}.1`]: 'done',
							[`${loadId}.2`]: 'done'
						}
					},
					mint: { kind: 'none' }
				})
			)?.jobInstanceId
		).toBe('2026-10-19-1st');
		expect(
			worklistJobFor(session.result!.worklist, 'line')?.instances[0]?.tasks.lineLoad
		).toBe('done');
		expect(
			worklistJobFor(session.result!.worklist, 'line')?.instances[0]?.tasks.containerReg
		).toBe('available');

		expect(
			(
				await session.commit({
					taskPath: 'line/containerReg/new/register',
					jobInstanceId: '2026-10-19-1st',
					value: {
						sourceLoadId: loadId,
						polvereDiVetro: [{ weight: 120 }],
						plastica: [{ weight: 40 }]
					},
					mint: { kind: 'spawningNew' }
				})
			)?.jobInstanceId
		).toBe('2026-10-19-1st');

		const lineSubs = domain.submissions.filter(
			(s) => s.jobId === 'line' && s.jobInstanceId === '2026-10-19-1st'
		);
		expect(lineSubs.map((s) => s.taskId)).toEqual(['openTurno', 'lineLoad', 'register']);
		expect(Object.keys(domain.container)).toHaveLength(2);
		expect(domain.load[loadId]?.fullyProcessed).toBe(true);
		expect(worklistJobFor(session.result!.worklist, 'line')?.canStart).toBe(true);
		expect(
			worklistJobFor(session.result!.worklist, 'line')?.instances[0]?.tasks.containerReg
		).toBe('available');
		expect(
			worklistJobFor(session.result!.worklist, 'line')?.instances[0]?.tasks
		).not.toHaveProperty('loadedExtracted');
	});
});
