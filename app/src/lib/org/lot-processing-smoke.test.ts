import { describe, expect, test, vi } from 'vitest';

import { OrgSession } from './session.svelte.js';
import {
	emptyDomain,
	worklistJobFor,
	type Domain,
	type LoadResult,
	type Submission,
	type Worklist
} from './types.js';

/** Live lot_processing Task ids — flattened peer many-submit + lot session. */
const lotProcessingTaskIds = [
	'startLot',
	'weigh',
	'markProgress',
	'containerReg',
	'closeLot'
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
 * Job instance id = lot id (minted on startLot).
 */
function worklistFor(domain: Domain, canStartLotProcessing: boolean): Worklist {
	const lotInst = domain.jobInstances.find((i) => i.jobId === 'lot_processing');
	if (!lotInst) {
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
					jobId: 'lot_processing',
					canStart: canStartLotProcessing,
					repeatable: true,
					startLabel: 'Apri lotto',
					taskIds: lotProcessingTaskIds,
					instances: []
				}
			]
		};
	}
	const tasks: Record<string, TaskStatus> = {
		startLot: 'done',
		weigh: 'locked',
		markProgress: 'locked',
		containerReg: 'locked',
		closeLot: 'locked'
	};
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

	tasks.weigh = openLot ? 'actionable' : 'unavailable';
	tasks.markProgress =
		openLot && hasWeightedUnfinishedPallet ? 'actionable' : 'unavailable';
	tasks.containerReg = openLot ? 'actionable' : 'unavailable';
	tasks.closeLot = 'unavailable';

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
				jobId: 'lot_processing',
				canStart: canStartLotProcessing,
				repeatable: true,
				startLabel: 'Apri lotto',
				taskIds: lotProcessingTaskIds,
				instances: [
					{
						id: lotInst.id,
						label: lotInst.label,
						tasks
					}
				]
			}
		]
	};
}

describe('lot_processing Commit smoke (session + Worklist + leaf many-submit)', () => {
	test('intake gate then lot_processing through startLot, markProgress, containerReg', async () => {
		const loadId = 'NO-261019-1';
		let domain: Domain = {
			...emptyDomain(),
			lot: {
				[loadId]: { client: 'NO', truckWeight: 900, zipTare: 50 }
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
				if (req.taskPath === 'lot_processing/startLot') {
					const lotId = String(req.value.lotId ?? '');
					domain = {
						...domain,
						lot: {
							...domain.lot,
							[lotId]: { ...domain.lot[lotId], status: 'open' }
						},
						jobInstances: [
							...domain.jobInstances,
							{
								jobId: 'lot_processing',
								id: req.jobInstanceId,
								label: `Lotto: ${req.jobInstanceId}`
							}
						],
						submissions: [
							...domain.submissions,
							sub('lot_processing', 'startLot', req.jobInstanceId, req.value)
						]
					};
				} else if (req.taskPath === 'lot_processing/markProgress') {
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
							sub('lot_processing', 'markProgress', req.jobInstanceId, progress)
						]
					};
				} else if (req.taskPath === 'lot_processing/containerReg') {
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
							sub('lot_processing', 'containerReg', req.jobInstanceId, req.value)
						]
					};
				}
				worklist = worklistFor(domain, true);
				const lotInst = worklistJobFor(worklist, 'lot_processing')?.instances[0];
				let nextTaskFormBundle = null;
				let navigation: 'task' | 'home' | 'stay' = 'home';
				if (lotInst) {
					for (const taskId of lotProcessingTaskIds) {
						if (lotInst.tasks[taskId] !== 'actionable') continue;
						nextTaskFormBundle = {
							jobId: 'lot_processing',
							taskId,
							jobInstanceId: lotInst.id,
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

		const session = new OrgSession({
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

		expect(worklistJobFor(session.result.worklist, 'lot_processing')?.canStart).toBe(true);

		expect(
			(
				await session.commit({
					taskPath: 'lot_processing/startLot',
					jobInstanceId: loadId,
					value: { lotId: loadId },
					mint: { kind: 'startingNew' }
				})
			)?.jobInstanceId
		).toBe(loadId);

		expect(session.result?.worklist).toEqual(worklist);
		expect(domain.lot[loadId]?.status).toBe('open');
		const afterStart = worklistJobFor(session.result!.worklist, 'lot_processing')?.instances[0]
			?.tasks;
		expect(afterStart?.startLot).toBe('done');
		expect(afterStart?.markProgress).toBe('actionable');
		expect(afterStart?.containerReg).toBe('actionable');

		expect(
			(
				await session.commit({
					taskPath: 'lot_processing/markProgress',
					jobInstanceId: loadId,
					value: {
						progress: {
							[`${loadId}.1`]: 'done',
							[`${loadId}.2`]: 'done'
						}
					}
				})
			)?.jobInstanceId
		).toBe(loadId);
		expect(
			worklistJobFor(session.result!.worklist, 'lot_processing')?.instances[0]?.tasks
				.markProgress
		).toBe('unavailable');
		expect(
			worklistJobFor(session.result!.worklist, 'lot_processing')?.instances[0]?.tasks
				.containerReg
		).toBe('actionable');

		expect(
			(
				await session.commit({
					taskPath: 'lot_processing/containerReg',
					jobInstanceId: loadId,
					value: {
						sourceLotId: loadId,
						polvereDiVetro: [{ weight: 120 }],
						plastica: [{ weight: 40 }]
					}
				})
			)?.jobInstanceId
		).toBe(loadId);

		const lotSubs = domain.submissions.filter(
			(s) => s.jobId === 'lot_processing' && s.jobInstanceId === loadId
		);
		expect(lotSubs.map((s) => s.taskId)).toEqual([
			'startLot',
			'markProgress',
			'containerReg'
		]);
		expect(Object.keys(domain.container)).toHaveLength(2);
		expect(domain.lot[loadId]?.status).toBe('open');
		expect(worklistJobFor(session.result!.worklist, 'lot_processing')?.canStart).toBe(true);
		expect(
			worklistJobFor(session.result!.worklist, 'lot_processing')?.instances[0]?.tasks
				.containerReg
		).toBe('actionable');
		expect(
			worklistJobFor(session.result!.worklist, 'lot_processing')?.instances[0]?.tasks
		).not.toHaveProperty('openTurno');
		expect(
			worklistJobFor(session.result!.worklist, 'lot_processing')?.instances[0]?.tasks
		).not.toHaveProperty('closeTurno');
		expect(
			worklistJobFor(session.result!.worklist, 'lot_processing')?.instances[0]?.tasks
		).not.toHaveProperty('loadedExtracted');
		expect(
			worklistJobFor(session.result!.worklist, 'lot_processing')?.instances[0]?.tasks
		).not.toHaveProperty('palletWeight');
		expect(
			worklistJobFor(session.result!.worklist, 'lot_processing')?.instances[0]?.tasks
		).not.toHaveProperty('lineLoad');
	});
});
