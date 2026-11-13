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

/** Live line Task ids — flattened peer many-submit + lot session (ADR-0008). */
const lineTaskIds = [
	'openTurno',
	'startLot',
	'weigh',
	'markProgress',
	'containerReg',
	'closeLot',
	'closeTurno'
];

type TaskStatus = 'locked' | 'unavailable' | 'available' | 'actionable' | 'done';

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
 * Peer many-submit Tasks gate on open lot (readyWhen); markProgress also
 * requires ≥1 weighted unfinished pallet ($exists(weight)); no Task groups.
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
					taskIds: ['truckArrive', 'register', 'weightCheck'],
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
	const tasks: Record<string, TaskStatus> = {
		openTurno: 'locked',
		startLot: 'locked',
		weigh: 'locked',
		markProgress: 'locked',
		containerReg: 'locked',
		closeLot: 'locked',
		closeTurno: 'locked'
	};
	const lineSubs = domain.submissions.filter(
		(s) => s.jobId === 'line' && s.jobInstanceId === lineInst.id
	);
	const has = (taskId: string) => lineSubs.some((s) => s.taskId === taskId);
	const openLot = Object.values(domain.lot ?? {}).some((lot) => lot?.status === 'open');
	const openLotIds = new Set(
		Object.entries(domain.lot ?? {})
			.filter(([, lot]) => lot?.status === 'open')
			.map(([id]) => id)
	);
	const hasWeightedUnfinishedPallet = Object.entries(domain.pallet ?? {}).some(
		([, facts]) =>
			openLotIds.has(String(facts?.lotId ?? '')) &&
			facts?.weight != null &&
			facts?.progress !== 'done'
	);

	if (has('openTurno')) tasks.openTurno = 'done';
	else tasks.openTurno = 'actionable';

	if (has('openTurno')) {
		tasks.startLot = openLot ? 'unavailable' : 'actionable';
		tasks.weigh = openLot ? 'actionable' : 'unavailable';
		tasks.markProgress =
			openLot && hasWeightedUnfinishedPallet ? 'actionable' : 'unavailable';
		tasks.containerReg = openLot ? 'actionable' : 'unavailable';
		tasks.closeLot = openLot ? 'unavailable' : 'unavailable';
		tasks.closeTurno = has('closeTurno') ? 'done' : 'actionable';
	}

	return {
		jobs: [
			{
				jobId: 'intake',
				canStart: true,
				repeatable: true,
				taskIds: ['truckArrive', 'register', 'weightCheck'],
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

describe('line Commit smoke (session + Worklist + leaf many-submit)', () => {
	test('intake gate then line through openTurno, startLot, markProgress, containerReg', async () => {
		const loadId = 'NO-261019-1';
		let domain: Domain = {
			...emptyDomain(),
			lot: {
				[loadId]: { client: 'NO', truckWeight: 900, zipTare: 50, status: 'fullyProcessed' }
			},
			pallet: {
				[`${loadId}.1`]: { lotId: loadId, weight: 400 },
				[`${loadId}.2`]: { lotId: loadId, weight: 450 }
			},
			jobInstances: [{ jobId: 'intake', id: '261019-1', label: '261019-1' }],
			submissions: [sub('intake', 'register', '261019-1')]
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
				} else if (req.taskPath === 'line/startLot') {
					const lotId = String(req.value.lotId ?? '');
					domain = {
						...domain,
						lot: {
							...domain.lot,
							[lotId]: { ...domain.lot[lotId], status: 'open' }
						},
						submissions: [
							...domain.submissions,
							sub('line', 'startLot', req.jobInstanceId, req.value)
						]
					};
				} else if (req.taskPath === 'line/markProgress') {
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
							sub('line', 'markProgress', req.jobInstanceId, progress)
						]
					};
				} else if (req.taskPath === 'line/containerReg') {
					domain = {
						...domain,
						container: {
							'C-1': {
								weight: 120,
								material: 'Polvere di vetro',
								sourceLotId: loadId,
								incomplete: false
							},
							'C-2': {
								weight: 40,
								material: 'Plastica',
								sourceLotId: loadId,
								incomplete: false
							}
						},
						submissions: [
							...domain.submissions,
							sub('line', 'containerReg', req.jobInstanceId, req.value)
						]
					};
				}
				worklist = worklistFor(domain, true);
				const lineInst = worklistJobFor(worklist, 'line')?.instances[0];
				let nextTaskFormBundle = null;
				let navigation: 'task' | 'overview' | 'home' | 'stay' = 'home';
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
		expect(afterOpen?.startLot).toBe('actionable');
		expect(afterOpen?.markProgress).toBe('unavailable');
		expect(afterOpen?.containerReg).toBe('unavailable');

		expect(
			(
				await session.commit({
					taskPath: 'line/startLot',
					jobInstanceId: '2026-10-19-1st',
					value: { lotId: loadId }
				})
			)?.jobInstanceId
		).toBe('2026-10-19-1st');
		expect(domain.lot[loadId]?.status).toBe('open');
		expect(
			worklistJobFor(session.result!.worklist, 'line')?.instances[0]?.tasks.markProgress
		).toBe('actionable');
		expect(
			worklistJobFor(session.result!.worklist, 'line')?.instances[0]?.tasks.containerReg
		).toBe('actionable');

		expect(
			(
				await session.commit({
					taskPath: 'line/markProgress',
					jobInstanceId: '2026-10-19-1st',
					value: {
						progress: {
							[`${loadId}.1`]: 'done',
							[`${loadId}.2`]: 'done'
						}
					}
				})
			)?.jobInstanceId
		).toBe('2026-10-19-1st');
		expect(
			worklistJobFor(session.result!.worklist, 'line')?.instances[0]?.tasks.markProgress
		).toBe('unavailable');
		expect(
			worklistJobFor(session.result!.worklist, 'line')?.instances[0]?.tasks.containerReg
		).toBe('actionable');

		expect(
			(
				await session.commit({
					taskPath: 'line/containerReg',
					jobInstanceId: '2026-10-19-1st',
					value: {
						sourceLotId: loadId,
						polvereDiVetro: [{ weight: 120 }],
						plastica: [{ weight: 40 }]
					}
				})
			)?.jobInstanceId
		).toBe('2026-10-19-1st');

		const lineSubs = domain.submissions.filter(
			(s) => s.jobId === 'line' && s.jobInstanceId === '2026-10-19-1st'
		);
		expect(lineSubs.map((s) => s.taskId)).toEqual([
			'openTurno',
			'startLot',
			'markProgress',
			'containerReg'
		]);
		expect(Object.keys(domain.container)).toHaveLength(2);
		expect(domain.lot[loadId]?.status).toBe('open');
		expect(worklistJobFor(session.result!.worklist, 'line')?.canStart).toBe(true);
		expect(
			worklistJobFor(session.result!.worklist, 'line')?.instances[0]?.tasks.containerReg
		).toBe('actionable');
		expect(
			worklistJobFor(session.result!.worklist, 'line')?.instances[0]?.tasks
		).not.toHaveProperty('loadedExtracted');
		expect(
			worklistJobFor(session.result!.worklist, 'line')?.instances[0]?.tasks
		).not.toHaveProperty('palletWeight');
		expect(
			worklistJobFor(session.result!.worklist, 'line')?.instances[0]?.tasks
		).not.toHaveProperty('lineLoad');
	});
});
