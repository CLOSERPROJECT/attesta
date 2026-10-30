import { describe, expect, test, vi } from 'vitest';

import { lineLoadLeftoverIds } from './leftovers.js';
import { LivingSession } from './session.svelte.js';
import { canStartJob } from './starts-when.js';
import {
	emptyDomain,
	type Domain,
	type LoadResult,
	type Submission,
	type TaskConfig
} from './types.js';
import { isTaskUnlocked, orderedTaskIds } from './unlock.js';

const lineTasks: TaskConfig[] = [
	{ id: 'shift' },
	{
		id: 'lineLoad',
		dynamicProperties: {
			progress: { read: 'leftovers', completeness: 'some' }
		}
	},
	{ id: 'containerReg', unlockFrom: 'instance' },
	{ id: 'loadedExtracted' }
];

const sampleConfig: LoadResult['config'] = {
	id: 'truck-loop',
	org: 'feragame',
	version: '1',
	jobs: {
		intake: {
			repeatable: true,
			label: 'intake',
			tasks: [
				{
					id: 'palletWeight',
					dynamicProperties: {
						weights: { read: 'leftovers', completeness: 'all' }
					}
				}
			]
		},
		line: {
			repeatable: true,
			label: 'line',
			startLabel: 'Nuovo turno',
			startsWhen: {
				task: { jobId: 'intake', taskId: 'palletWeight' },
				quantifier: 'some'
			},
			tasks: lineTasks
		}
	}
};

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

describe('line Commit smoke (session + unlock + leftovers some)', () => {
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

		const commit = vi.fn(async (req: { taskPath: string; jobInstanceId: string; value: Record<string, unknown> }) => {
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
			return { org: 'feragame', jobInstanceId: req.jobInstanceId, domain };
		});

		const session = new LivingSession({
			load: {
				load: async () => ({
					org: 'feragame',
					activeConfigVersion: '1',
					domain,
					config: sampleConfig
				})
			},
			commit: { commit },
			hydrate: false
		});
		session.result = {
			org: 'feragame',
			activeConfigVersion: '1',
			domain,
			config: sampleConfig
		};

		const line = sampleConfig.jobs.line!;
		expect(canStartJob({ job: line, jobId: 'line', domain, config: sampleConfig })).toBe(
			true
		);
		expect(lineLoadLeftoverIds(domain)).toEqual([`${intakeId}.1`, `${intakeId}.2`]);

		expect(
			await session.commit({
				taskPath: 'line/shift',
				jobInstanceId: '1st',
				value: { shift: '1st' },
				mint: { kind: 'startingNew' }
			})
		).toBe('1st');

		domain = session.result!.domain;
		const ordered = orderedTaskIds(lineTasks);
		expect(
			isTaskUnlocked({
				orderedIds: ordered,
				jobId: 'line',
				taskId: 'containerReg',
				jobInstanceId: '1st',
				submissions: domain.submissions,
				tasks: lineTasks
			})
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
		domain = session.result!.domain;
		expect(lineLoadLeftoverIds(domain)).toEqual([`${intakeId}.2`]);
		// some Submission must not be treated as an unlock gate for a synthetic successor.
		expect(
			isTaskUnlocked({
				orderedIds: ['lineLoad', 'next'],
				jobId: 'line',
				taskId: 'next',
				jobInstanceId: '1st',
				submissions: domain.submissions,
				tasks: [
					lineTasks.find((t) => t.id === 'lineLoad')!,
					{ id: 'next' }
				]
			})
		).toBe(false);

		expect(
			await session.commit({
				taskPath: 'line/lineLoad',
				jobInstanceId: '1st',
				value: { progress: { [`${intakeId}.2`]: 'done' } },
				mint: { kind: 'none' }
			})
		).toBe('1st');
		domain = session.result!.domain;
		expect(lineLoadLeftoverIds(domain)).toEqual([]);

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

		const final = session.result!.domain;
		const lineSubs = final.submissions.filter(
			(s) => s.jobId === 'line' && s.jobInstanceId === '1st'
		);
		expect(lineSubs.map((s) => s.taskId)).toEqual([
			'shift',
			'lineLoad',
			'lineLoad',
			'containerReg',
			'loadedExtracted'
		]);
		expect(Object.keys(final.container)).toHaveLength(2);
		expect(canStartJob({ job: line, jobId: 'line', domain: final, config: sampleConfig })).toBe(
			true
		);
	});
});
