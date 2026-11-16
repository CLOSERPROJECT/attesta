import { describe, expect, test } from 'vitest';

import * as m from '#lib/paraglide/messages.js';

import type { Worklist } from './types.js';
import {
	aggregateGroupInstanceStatus,
	formatFactValue,
	formatHandoffCopy,
	formatSubmissionCount,
	jobInstanceOverview,
	manySubmitHistoryVisible,
	openGroupLoopChrome,
	provisionalSpawnDraftRowVisible,
	taskStatusLabel
} from './worklist-overview.js';

const taskChrome = [
	{ id: 'register', title: 'Registrazione', roles: ['administration'] },
	{ id: 'codiceFir', title: 'Codice FIR', roles: ['administration'] },
	{ id: 'palletWeight', title: 'Peso pallet', roles: ['logistic'] }
];

function worklistFor(viewer: 'admin' | 'logistic'): Worklist {
	const palletMine = viewer === 'logistic' ? 'actionable' : 'available';
	const registerMine = viewer === 'admin' ? 'actionable' : 'available';
	return {
		jobs: [
			{
				jobId: 'intake',
				canStart: viewer === 'admin',
				repeatable: true,
				startLabel: 'Nuovo camion',
				taskIds: ['register', 'codiceFir', 'palletWeight'],
				tasks: taskChrome,
				instances: [
					{
						id: 'NIR-1',
						label: 'NIR-1',
						tasks: {
							register: 'done',
							codiceFir: 'done',
							palletWeight: palletMine
						}
					},
					{
						id: 'NIR-done',
						label: 'Done truck',
						tasks: {
							register: 'done',
							codiceFir: 'done',
							palletWeight: 'done'
						}
					},
					{
						id: 'NIR-both-available',
						label: 'Split available',
						tasks: {
							register: registerMine,
							codiceFir: 'locked',
							palletWeight: palletMine
						}
					}
				]
			}
		]
	};
}

const adminWorklist = worklistFor('admin');
const logisticWorklist = worklistFor('logistic');

describe('job instance overview model', () => {
	test('taskStatusLabel maps five-value status via Paraglide', () => {
		expect(taskStatusLabel('locked')).toBe(m.status_locked());
		expect(taskStatusLabel('unavailable')).toBe(m.status_unavailable());
		expect(taskStatusLabel('available')).toBe(m.status_available());
		expect(taskStatusLabel('actionable')).toBe(m.status_actionable());
		expect(taskStatusLabel('done')).toBe(m.status_done());
	});

	test('formatFactValue stringifies recap values', () => {
		expect(formatFactValue(null)).toBe('—');
		expect(formatFactValue('x')).toBe('x');
		expect(formatFactValue(42)).toBe('42');
		expect(formatFactValue({ a: 1 })).toBe('{"a":1}');
	});

	test('lists statuses; linkable only for actionable', () => {
		const admin = jobInstanceOverview(adminWorklist, 'intake', 'NIR-1');
		expect(admin?.label).toBe('NIR-1');
		expect(admin?.tasks.map((t) => [t.taskId, t.status, t.linkable])).toEqual([
			['register', 'done', false],
			['codiceFir', 'done', false],
			['palletWeight', 'available', false]
		]);
		expect(admin?.handoffRoles).toEqual(['logistic']);
		expect(formatHandoffCopy(admin!.handoffRoles!)).toContain('logistic');
		expect(admin?.finished).toBe(false);

		const logistic = jobInstanceOverview(logisticWorklist, 'intake', 'NIR-1');
		expect(logistic?.tasks.find((t) => t.taskId === 'palletWeight')?.linkable).toBe(
			true
		);
		expect(logistic?.handoffRoles).toBeNull();
	});

	test('handoff uses first open only; no banner when next Task is theirs', () => {
		const admin = jobInstanceOverview(adminWorklist, 'intake', 'NIR-both-available');
		expect(admin?.tasks.map((t) => [t.taskId, t.status, t.linkable])).toEqual([
			['register', 'actionable', true],
			['codiceFir', 'locked', false],
			['palletWeight', 'available', false]
		]);
		expect(admin?.handoffRoles).toBeNull();

		const logistic = jobInstanceOverview(
			logisticWorklist,
			'intake',
			'NIR-both-available'
		);
		expect(logistic?.handoffRoles).toEqual(['administration']);
		expect(logistic?.tasks.find((t) => t.taskId === 'register')?.linkable).toBe(false);
		expect(logistic?.tasks.find((t) => t.taskId === 'palletWeight')?.linkable).toBe(
			true
		);
	});

	test('finished model marks all done and remains projectable for overview', () => {
		const done = jobInstanceOverview(adminWorklist, 'intake', 'NIR-done');
		// Completed band deep-links need a finished overview model (not null).
		expect(done).not.toBeNull();
		expect(done?.finished).toBe(true);
		expect(done?.handoffRoles).toBeNull();
		expect(done?.tasks.every((t) => t.status === 'done')).toBe(true);
	});

	test('finished ignores submit: many; group chrome only when Worklist groups present', () => {
		const flat: Worklist = {
			jobs: [
				{
					jobId: 'intake',
					canStart: false,
					repeatable: true,
					taskIds: ['truckArrive', 'register', 'weightCheck'],
					tasks: [
						{
							id: 'truckArrive',
							title: 'Arrive',
							roles: ['administration'],
							submit: 'once'
						},
						{
							id: 'register',
							title: 'Register',
							roles: ['administration'],
							submit: 'many'
						},
						{
							id: 'weightCheck',
							title: 'Weight',
							roles: ['administration'],
							submit: 'once'
						}
					],
					instances: [
						{
							id: 'T1',
							label: 'Truck 1',
							tasks: {
								truckArrive: 'done',
								register: 'actionable',
								weightCheck: 'done'
							}
						}
					]
				}
			]
		};
		const model = jobInstanceOverview(flat, 'intake', 'T1');
		expect(model?.finished).toBe(true);
		const register = model?.tasks.find((t) => t.taskId === 'register');
		expect(register).toMatchObject({
			status: 'actionable',
			linkable: true,
			submit: 'many',
			submissionCount: 0
		});
		expect(register?.group).toBeUndefined();
		expect(model?.tasks.every((t) => t.group == null)).toBe(true);
		expect(model?.tasks.find((t) => t.taskId === 'truckArrive')?.submit).toBeUndefined();
	});

	test('many-submit leaf carries submissionCounts; history gate needs N > 0', () => {
		const wl: Worklist = {
			jobs: [
				{
					jobId: 'intake',
					canStart: false,
					repeatable: true,
					taskIds: ['register'],
					tasks: [
						{
							id: 'register',
							title: 'Register',
							roles: ['administration'],
							submit: 'many'
						}
					],
					instances: [
						{
							id: 'T1',
							label: 'Truck 1',
							tasks: { register: 'actionable' },
							submissionCounts: { register: 2 }
						}
					]
				}
			]
		};
		const register = jobInstanceOverview(wl, 'intake', 'T1')?.tasks[0];
		expect(register).toMatchObject({
			submit: 'many',
			submissionCount: 2,
			linkable: true
		});
		expect(manySubmitHistoryVisible(register?.submit, register?.submissionCount)).toBe(
			true
		);
		expect(manySubmitHistoryVisible('many', 0)).toBe(false);
		expect(manySubmitHistoryVisible(undefined, 3)).toBe(false);
		expect(formatSubmissionCount(1)).toBe(m.submission_count_one({ count: 1 }));
		expect(formatSubmissionCount(2)).toBe(m.submission_count_many({ count: 2 }));
	});

	test('group child many-submit reads submissionCounts from group instance', () => {
		const wl: Worklist = {
			jobs: [
				{
					jobId: 'intake',
					canStart: false,
					repeatable: true,
					taskIds: ['lots'],
					tasks: [
						{ id: 'lots', title: 'Lotti', roles: ['administration'] },
						{
							id: 'register',
							title: 'Registrazione',
							roles: ['administration'],
							submit: 'many'
						}
					],
					instances: [
						{
							id: 'T1',
							label: 'Truck 1',
							tasks: { lots: 'available' },
							groups: {
								lots: {
									groupId: 'lots',
									repeatable: true,
									canSpawn: false,
									canClose: false,
									childTaskIds: ['register'],
									instances: [
										{
											id: 'lots-1',
											label: 'load-A',
											tasks: { register: 'actionable' },
											submissionCounts: { register: 3 }
										}
									]
								}
							}
						}
					]
				}
			]
		};
		const flatChild = jobInstanceOverview(wl, 'intake', 'T1')?.tasks[0]?.groupInstances?.[0]
			?.flatChild;
		expect(flatChild).toMatchObject({
			taskId: 'register',
			submit: 'many',
			submissionCount: 3,
			linkable: true
		});
	});

	test('unknown instance yields null', () => {
		expect(jobInstanceOverview(adminWorklist, 'intake', 'missing')).toBeNull();
		expect(jobInstanceOverview(adminWorklist, 'intake', 'new')).toBeNull();
	});

	test('provisionalSpawnDraftRowVisible when canSpawn + dirty provisional Draft', () => {
		const group = {
			canSpawn: true,
			startLabel: 'Nuova registrazione',
			firstChildTaskId: 'register',
			canClose: false,
			closeLabel: 'Close'
		};
		expect(provisionalSpawnDraftRowVisible(group, true)).toBe(true);
		expect(provisionalSpawnDraftRowVisible(group, false)).toBe(false);
		expect(
			provisionalSpawnDraftRowVisible({ ...group, canSpawn: false }, true)
		).toBe(false);
		expect(
			provisionalSpawnDraftRowVisible(
				{ ...group, firstChildTaskId: '' },
				true
			)
		).toBe(false);
		expect(provisionalSpawnDraftRowVisible(undefined, true)).toBe(false);
	});

	test('group Spawn chrome when canSpawn; not linkable as flat Task', () => {
		const wl: Worklist = {
			jobs: [
				{
					jobId: 'intake',
					canStart: false,
					repeatable: true,
					taskIds: ['lots', 'weightCheck'],
					tasks: [
						{ id: 'lots', title: 'Lotti', roles: ['administration'] },
						{ id: 'weightCheck', title: 'Peso', roles: ['logistic'] }
					],
					instances: [
						{
							id: 'T1',
							label: 'Truck 1',
							tasks: { lots: 'available', weightCheck: 'locked' },
							groups: {
								lots: {
									groupId: 'lots',
									startLabel: 'Nuovo lotto',
									closeLabel: 'Chiudi scarico',
									repeatable: true,
									canSpawn: true,
									canClose: false,
									childTaskIds: ['register', 'labels'],
									instances: []
								}
							}
						}
					]
				}
			]
		};
		const model = jobInstanceOverview(wl, 'intake', 'T1');
		expect(model?.tasks[0]).toMatchObject({
			taskId: 'lots',
			linkable: false,
			group: {
				canSpawn: true,
				startLabel: 'Nuovo lotto',
				firstChildTaskId: 'register',
				canClose: false,
				closeLabel: 'Chiudi scarico'
			},
			groupInstances: []
		});
		expect(model?.handoffRoles).toBeNull();
	});

	test('group Close chrome when canClose; suppresses handoff', () => {
		const wl: Worklist = {
			jobs: [
				{
					jobId: 'intake',
					canStart: false,
					repeatable: true,
					taskIds: ['lots', 'weightCheck'],
					tasks: [
						{ id: 'lots', title: 'Lotti', roles: ['administration'] },
						{ id: 'weightCheck', title: 'Peso', roles: ['logistic'] },
						{ id: 'register', title: 'Registrazione', roles: ['administration'] }
					],
					instances: [
						{
							id: 'T1',
							label: 'Truck 1',
							tasks: { lots: 'available', weightCheck: 'locked' },
							groups: {
								lots: {
									groupId: 'lots',
									startLabel: 'Nuovo lotto',
									closeLabel: 'Chiudi scarico',
									repeatable: true,
									canSpawn: false,
									canClose: true,
									childTaskIds: ['register'],
									instances: [
										{
											id: 'lots-1',
											tasks: { register: 'done' }
										}
									]
								}
							}
						}
					]
				}
			]
		};
		const model = jobInstanceOverview(wl, 'intake', 'T1');
		expect(model?.tasks[0]?.group).toMatchObject({
			canSpawn: false,
			canClose: true,
			closeLabel: 'Chiudi scarico',
			firstChildTaskId: 'register'
		});
		expect(model?.tasks[0]?.groupInstances).toEqual([
			{
				id: 'lots-1',
				displayId: 'lots-1',
				status: 'done',
				children: [],
				flatChild: {
					taskId: 'register',
					title: 'Registrazione',
					status: 'done',
					linkable: false
				}
			}
		]);
		expect(model?.handoffRoles).toBeNull();
		expect(openGroupLoopChrome(wl, 'intake', 'T1', 'lots')).toMatchObject({
			canSpawn: false,
			canClose: true,
			closeLabel: 'Chiudi scarico',
			firstChildTaskId: 'register'
		});
		expect(openGroupLoopChrome(wl, 'intake', 'T1', 'missing')).toBeNull();
	});

	test('groupInstances displayId prefers projected label (lot id)', () => {
		const wl: Worklist = {
			jobs: [
				{
					jobId: 'intake',
					canStart: false,
					repeatable: true,
					taskIds: ['lots'],
					tasks: [
						{ id: 'lots', title: 'Lotti', roles: ['administration'] },
						{ id: 'register', title: 'Registrazione', roles: ['administration'] }
					],
					instances: [
						{
							id: 'T1',
							label: 'Truck 1',
							tasks: { lots: 'available' },
							groups: {
								lots: {
									groupId: 'lots',
									repeatable: true,
									canSpawn: false,
									canClose: true,
									childTaskIds: ['register'],
									instances: [
										{
											id: 'lots-1',
											label: '261019-1st-1-L1',
											tasks: { register: 'done' }
										}
									]
								}
							}
						}
					]
				}
			]
		};
		const model = jobInstanceOverview(wl, 'intake', 'T1');
		expect(model?.tasks[0]?.groupInstances).toEqual([
			{
				id: 'lots-1',
				displayId: '261019-1st-1-L1',
				status: 'done',
				children: [],
				flatChild: {
					taskId: 'register',
					title: 'Registrazione',
					status: 'done',
					linkable: false
				}
			}
		]);
	});

	test('single-child group occurrence flattens to flatChild; multi-child keeps nested children', () => {
		const wl: Worklist = {
			jobs: [
				{
					jobId: 'intake',
					canStart: false,
					repeatable: true,
					taskIds: ['solo', 'lots'],
					tasks: [
						{ id: 'solo', title: 'Solo group', roles: ['administration'] },
						{ id: 'lots', title: 'Lotti', roles: ['administration'] },
						{ id: 'register', title: 'Registrazione', roles: ['administration'] },
						{ id: 'labels', title: 'Etichette', roles: ['administration'] }
					],
					instances: [
						{
							id: 'T1',
							label: 'Truck 1',
							tasks: { solo: 'actionable', lots: 'available' },
							groups: {
								solo: {
									groupId: 'solo',
									repeatable: true,
									canSpawn: false,
									canClose: false,
									childTaskIds: ['register'],
									instances: [
										{
											id: 'solo-1',
											label: 'load-A',
											tasks: { register: 'actionable' }
										},
										{
											id: 'solo-2',
											tasks: { register: 'available' }
										},
										{
											id: 'solo-3',
											tasks: { register: 'locked' }
										},
										{
											id: 'solo-4',
											tasks: { register: 'done' }
										}
									]
								},
								lots: {
									groupId: 'lots',
									repeatable: true,
									canSpawn: false,
									canClose: false,
									childTaskIds: ['register', 'labels'],
									instances: [
										{
											id: 'lots-1',
											tasks: { register: 'done', labels: 'actionable' }
										}
									]
								}
							}
						}
					]
				}
			]
		};
		const model = jobInstanceOverview(wl, 'intake', 'T1');
		expect(model?.tasks.find((t) => t.taskId === 'solo')?.groupInstances).toEqual([
			{
				id: 'solo-1',
				displayId: 'load-A',
				status: 'actionable',
				children: [],
				flatChild: {
					taskId: 'register',
					title: 'Registrazione',
					status: 'actionable',
					linkable: true
				}
			},
			{
				id: 'solo-2',
				displayId: 'solo-2',
				status: 'available',
				children: [],
				flatChild: {
					taskId: 'register',
					title: 'Registrazione',
					status: 'available',
					linkable: false
				}
			},
			{
				id: 'solo-3',
				displayId: 'solo-3',
				status: 'locked',
				children: [],
				flatChild: {
					taskId: 'register',
					title: 'Registrazione',
					status: 'locked',
					linkable: false
				}
			},
			{
				id: 'solo-4',
				displayId: 'solo-4',
				status: 'done',
				children: [],
				flatChild: {
					taskId: 'register',
					title: 'Registrazione',
					status: 'done',
					linkable: false
				}
			}
		]);
		expect(model?.tasks.find((t) => t.taskId === 'lots')?.groupInstances).toEqual([
			{
				id: 'lots-1',
				displayId: 'lots-1',
				status: 'actionable',
				children: [
					{
						taskId: 'register',
						title: 'Registrazione',
						status: 'done',
						linkable: false
					},
					{
						taskId: 'labels',
						title: 'Etichette',
						status: 'actionable',
						linkable: true
					}
				]
			}
		]);
	});

	test('groupInstances aggregates prefer actionable over available', () => {
		const wl: Worklist = {
			jobs: [
				{
					jobId: 'intake',
					canStart: false,
					repeatable: true,
					taskIds: ['lots', 'weightCheck'],
					tasks: [
						{ id: 'lots', title: 'Lotti', roles: ['administration'] },
						{ id: 'weightCheck', title: 'Peso', roles: ['logistic'] },
						{ id: 'register', title: 'Registrazione', roles: ['administration'] },
						{ id: 'labels', title: 'Etichette', roles: ['administration'] }
					],
					instances: [
						{
							id: 'T1',
							label: 'Truck 1',
							tasks: { lots: 'available', weightCheck: 'locked' },
							groups: {
								lots: {
									groupId: 'lots',
									startLabel: 'Nuovo lotto',
									closeLabel: 'Chiudi scarico',
									repeatable: true,
									canSpawn: true,
									canClose: false,
									childTaskIds: ['register', 'labels'],
									instances: [
										{
											id: 'lots-1',
											tasks: { register: 'done', labels: 'done' }
										},
										{
											id: 'lots-2',
											tasks: { register: 'done', labels: 'available' }
										},
										{
											id: 'lots-3',
											tasks: { register: 'locked', labels: 'locked' }
										},
										{
											id: 'lots-4',
											tasks: { register: 'done', labels: 'actionable' }
										}
									]
								}
							}
						}
					]
				}
			]
		};
		const model = jobInstanceOverview(wl, 'intake', 'T1');
		expect(model?.tasks[0]?.groupInstances).toEqual([
			{
				id: 'lots-1',
				displayId: 'lots-1',
				status: 'done',
				children: [
					{
						taskId: 'register',
						title: 'Registrazione',
						status: 'done',
						linkable: false
					},
					{ taskId: 'labels', title: 'Etichette', status: 'done', linkable: false }
				]
			},
			{
				id: 'lots-2',
				displayId: 'lots-2',
				status: 'available',
				children: [
					{
						taskId: 'register',
						title: 'Registrazione',
						status: 'done',
						linkable: false
					},
					{ taskId: 'labels', title: 'Etichette', status: 'available', linkable: false }
				]
			},
			{
				id: 'lots-3',
				displayId: 'lots-3',
				status: 'locked',
				children: [
					{
						taskId: 'register',
						title: 'Registrazione',
						status: 'locked',
						linkable: false
					},
					{ taskId: 'labels', title: 'Etichette', status: 'locked', linkable: false }
				]
			},
			{
				id: 'lots-4',
				displayId: 'lots-4',
				status: 'actionable',
				children: [
					{
						taskId: 'register',
						title: 'Registrazione',
						status: 'done',
						linkable: false
					},
					{ taskId: 'labels', title: 'Etichette', status: 'actionable', linkable: true }
				]
			}
		]);
		expect(model?.tasks[0]?.group).toMatchObject({
			canSpawn: true,
			startLabel: 'Nuovo lotto',
			firstChildTaskId: 'register'
		});
		expect(model?.tasks[0]?.linkable).toBe(false);
		expect(
			aggregateGroupInstanceStatus(
				{ id: 'x', tasks: { register: 'done', labels: 'locked' } },
				['register', 'labels']
			)
		).toBe('locked');
		expect(
			aggregateGroupInstanceStatus(
				{ id: 'y', tasks: { register: 'available', labels: 'actionable' } },
				['register', 'labels']
			)
		).toBe('actionable');
	});
});
