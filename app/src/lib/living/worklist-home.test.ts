import { describe, expect, test, vi } from 'vitest';

vi.mock('../router/index.js', () => ({
	p: (path: string, opts?: { params?: Record<string, string> }) => {
		if (!opts?.params) return path === '/' ? '/app' : `/app${path}`;
		let out = path;
		for (const [key, value] of Object.entries(opts.params)) {
			out = out.replace(`:${key}`, value);
		}
		return `/app${out}`;
	},
	navigate: vi.fn()
}));

import { hrefLivingTarget } from '../router/navigate-target.js';
import type { Worklist } from './types.js';
import {
	availableRowKey,
	availableRowsFromWorklist,
	availableRowTarget,
	canStartRowsFromWorklist,
	inProgressRowKey,
	inProgressRowsFromWorklist,
	inProgressRowTarget,
	instanceTaskProgress,
	progressFraction,
	taskRouteInstanceSegment
} from './worklist-home.js';

const chrome = {
	intake: [
		{ id: 'register', title: 'Registrazione', roles: ['administration'] },
		{ id: 'palletWeight', title: 'Peso pallet', roles: ['logistic'] }
	],
	line: [
		{ id: 'shift', title: 'Turno', roles: ['line'] },
		{ id: 'lineLoad', title: 'Carico', roles: ['line'] }
	]
} as const;

/** Viewer-scoped Worklists (unlocked leaf is actionable vs available). */
const adminWorklist: Worklist = {
	jobs: [
		{
			jobId: 'intake',
			canStart: true,
			repeatable: true,
			title: 'Camion',
			startLabel: 'Nuovo camion',
			taskIds: ['register', 'palletWeight'],
			tasks: [...chrome.intake],
			instances: [
				{
					id: 'NIR-1',
					label: 'NIR-1',
					tasks: { register: 'done', palletWeight: 'available' }
				},
				{
					id: 'NIR-2',
					label: 'Camion B',
					tasks: { register: 'actionable', palletWeight: 'locked' }
				}
			]
		},
		{
			jobId: 'line',
			canStart: false,
			repeatable: true,
			title: 'Linea',
			startLabel: 'Nuovo turno',
			taskIds: ['shift', 'lineLoad'],
			tasks: [...chrome.line],
			instances: [
				{
					id: '1st',
					label: 'Turno: 1st',
					tasks: { shift: 'done', lineLoad: 'available' }
				}
			]
		}
	]
};

const logisticWorklist: Worklist = {
	jobs: [
		{
			jobId: 'intake',
			canStart: false,
			repeatable: true,
			title: 'Camion',
			startLabel: 'Nuovo camion',
			taskIds: ['register', 'palletWeight'],
			tasks: [...chrome.intake],
			instances: [
				{
					id: 'NIR-1',
					label: 'NIR-1',
					tasks: { register: 'done', palletWeight: 'actionable' }
				},
				{
					id: 'NIR-2',
					label: 'Camion B',
					tasks: { register: 'available', palletWeight: 'locked' }
				}
			]
		},
		{
			jobId: 'line',
			canStart: false,
			repeatable: true,
			title: 'Linea',
			startLabel: 'Nuovo turno',
			taskIds: ['shift', 'lineLoad'],
			tasks: [...chrome.line],
			instances: [
				{
					id: '1st',
					label: 'Turno: 1st',
					tasks: { shift: 'done', lineLoad: 'available' }
				}
			]
		}
	]
};

const lineWorklist: Worklist = {
	jobs: [
		{
			jobId: 'intake',
			canStart: false,
			repeatable: true,
			title: 'Camion',
			startLabel: 'Nuovo camion',
			taskIds: ['register', 'palletWeight'],
			tasks: [...chrome.intake],
			instances: [
				{
					id: 'NIR-1',
					label: 'NIR-1',
					tasks: { register: 'done', palletWeight: 'available' }
				},
				{
					id: 'NIR-2',
					label: 'Camion B',
					tasks: { register: 'available', palletWeight: 'locked' }
				}
			]
		},
		{
			jobId: 'line',
			canStart: false,
			repeatable: true,
			title: 'Linea',
			startLabel: 'Nuovo turno',
			taskIds: ['shift', 'lineLoad'],
			tasks: [...chrome.line],
			instances: [
				{
					id: '1st',
					label: 'Turno: 1st',
					tasks: { shift: 'done', lineLoad: 'actionable' }
				}
			]
		}
	]
};

const allRolesWorklist: Worklist = {
	jobs: [
		{
			jobId: 'intake',
			canStart: true,
			repeatable: true,
			title: 'Camion',
			startLabel: 'Nuovo camion',
			taskIds: ['register', 'palletWeight'],
			tasks: [...chrome.intake],
			instances: [
				{
					id: 'NIR-1',
					label: 'NIR-1',
					tasks: { register: 'done', palletWeight: 'actionable' }
				},
				{
					id: 'NIR-2',
					label: 'Camion B',
					tasks: { register: 'actionable', palletWeight: 'locked' }
				}
			]
		},
		{
			jobId: 'line',
			canStart: false,
			repeatable: true,
			title: 'Linea',
			startLabel: 'Nuovo turno',
			taskIds: ['shift', 'lineLoad'],
			tasks: [...chrome.line],
			instances: [
				{
					id: '1st',
					label: 'Turno: 1st',
					tasks: { shift: 'done', lineLoad: 'actionable' }
				}
			]
		}
	]
};

describe('worklist-home bands', () => {
	test('availableRows sort Job → instance → Task from actionable', () => {
		const admin = availableRowsFromWorklist(adminWorklist);
		expect(admin.map((r) => `${r.jobId}/${r.jobInstanceId}/${r.taskId}`)).toEqual([
			'intake/NIR-2/register'
		]);
		expect(admin[0]?.title).toBe('Registrazione');
		expect(admin[0]?.instanceLabel).toBe('Camion B');
		expect(admin[0]?.jobTitle).toBe('Camion');
		expect(admin[0]?.tasksDone).toBe(0);
		expect(admin[0]?.tasksTotal).toBe(2);

		const logistic = availableRowsFromWorklist(logisticWorklist);
		expect(logistic.map((r) => `${r.jobId}/${r.jobInstanceId}/${r.taskId}`)).toEqual([
			'intake/NIR-1/palletWeight'
		]);
		expect(logistic[0]?.title).toBe('Peso pallet');
		expect(logistic[0]?.jobTitle).toBe('Camion');
		expect(logistic[0]?.tasksDone).toBe(1);
		expect(logistic[0]?.tasksTotal).toBe(2);

		const line = availableRowsFromWorklist(lineWorklist);
		expect(line.map((r) => `${r.jobId}/${r.jobInstanceId}/${r.taskId}`)).toEqual([
			'line/1st/lineLoad'
		]);
		expect(line[0]?.jobTitle).toBe('Linea');
		expect(line[0]?.tasksDone).toBe(1);
		expect(line[0]?.tasksTotal).toBe(2);

		const all = availableRowsFromWorklist(allRolesWorklist);
		expect(all.map((r) => `${r.jobId}/${r.jobInstanceId}/${r.taskId}`)).toEqual([
			'intake/NIR-1/palletWeight',
			'intake/NIR-2/register',
			'line/1st/lineLoad'
		]);
		expect(all.map((r) => `${r.tasksDone}/${r.tasksTotal}`)).toEqual(['1/2', '0/2', '1/2']);
	});

	test('instanceTaskProgress counts done against Config taskIds', () => {
		const job = adminWorklist.jobs[0]!;
		expect(instanceTaskProgress(job, job.instances[0]!)).toEqual({ done: 1, total: 2 });
		expect(instanceTaskProgress(job, job.instances[1]!)).toEqual({ done: 0, total: 2 });
		expect(progressFraction(1, 3)).toBeCloseTo(1 / 3);
		expect(progressFraction(0, 0)).toBe(0);
		expect(progressFraction(5, 3)).toBe(1);
	});

	test('canStartRows use startLabel and first Task; provisional segment is new', () => {
		expect(canStartRowsFromWorklist(adminWorklist)).toEqual([
			{
				jobId: 'intake',
				jobTitle: 'Camion',
				startLabel: 'Nuovo camion',
				firstTaskId: 'register'
			}
		]);
		expect(taskRouteInstanceSegment('')).toBe('new');
		expect(taskRouteInstanceSegment('NIR-1')).toBe('NIR-1');
	});

	test('empty Worklist yields empty bands', () => {
		expect(availableRowsFromWorklist({ jobs: [] })).toEqual([]);
		expect(canStartRowsFromWorklist({ jobs: [] })).toEqual([]);
		expect(inProgressRowsFromWorklist({ jobs: [] })).toEqual([]);
		expect(availableRowsFromWorklist(null)).toEqual([]);
		expect(inProgressRowsFromWorklist(null)).toEqual([]);
	});

	test('inProgressRows include unfinished Job instances; finished excluded; Job → instance order', () => {
		const mixed: Worklist = {
			jobs: [
				{
					jobId: 'intake',
					canStart: false,
					repeatable: true,
					title: 'Camion',
					taskIds: ['register', 'palletWeight'],
					tasks: [...chrome.intake],
					instances: [
						{
							id: 'NIR-1',
							label: 'NIR-1',
							tasks: { register: 'done', palletWeight: 'actionable' }
						},
						{
							id: 'NIR-done',
							label: 'Finished truck',
							tasks: { register: 'done', palletWeight: 'done' }
						},
						{
							id: 'NIR-2',
							label: 'Camion B',
							tasks: { register: 'actionable', palletWeight: 'locked' }
						}
					]
				},
				{
					jobId: 'line',
					canStart: false,
					repeatable: true,
					title: 'Linea',
					taskIds: ['shift', 'lineLoad'],
					tasks: [...chrome.line],
					instances: [
						{
							id: '1st',
							label: 'Turno: 1st',
							tasks: { shift: 'done', lineLoad: 'available' }
						}
					]
				}
			]
		};

		const rows = inProgressRowsFromWorklist(mixed);
		expect(rows.map((r) => inProgressRowKey(r))).toEqual([
			'intake/NIR-1',
			'intake/NIR-2',
			'line/1st'
		]);
		expect(rows[0]).toMatchObject({
			jobId: 'intake',
			jobInstanceId: 'NIR-1',
			instanceLabel: 'NIR-1',
			jobTitle: 'Camion',
			tasksDone: 1,
			tasksTotal: 2
		});
		expect(rows[1]).toMatchObject({
			instanceLabel: 'Camion B',
			tasksDone: 0,
			tasksTotal: 2
		});
		expect(rows[2]).toMatchObject({
			jobId: 'line',
			jobInstanceId: '1st',
			jobTitle: 'Linea',
			tasksDone: 1,
			tasksTotal: 2
		});

		// Overlap-friendly: instances with actionable Tasks still appear in In progress.
		const availableKeys = availableRowsFromWorklist(mixed).map(
			(r) => `${r.jobId}/${r.jobInstanceId}`
		);
		expect(availableKeys).toContain('intake/NIR-1');
		expect(rows.map((r) => inProgressRowKey(r))).toContain('intake/NIR-1');

		expect(inProgressRowTarget(rows[0]!)).toEqual({
			kind: 'overview',
			jobId: 'intake',
			jobInstanceId: 'NIR-1'
		});
		expect(hrefLivingTarget(inProgressRowTarget(rows[0]!))).toBe('/app/intake/NIR-1');
	});

	test('nested ready Tasks and canSpawn provisional first-leaf appear in Available', () => {
		const nested: Worklist = {
			jobs: [
				{
					jobId: 'intake',
					canStart: false,
					repeatable: true,
					title: 'Camion',
					taskIds: ['truckArrive', 'lots', 'weightCheck'],
					tasks: [
						{ id: 'truckArrive', title: 'Arrive', roles: ['administration'] },
						{ id: 'lots', title: 'Lots', roles: [] },
						{ id: 'weightCheck', title: 'Weight', roles: ['administration'] },
						{ id: 'register', title: 'Register lot', roles: ['administration'] }
					],
					instances: [
						{
							id: 'T1',
							label: 'Truck 1',
							tasks: {
								truckArrive: 'done',
								lots: 'available',
								weightCheck: 'locked'
							},
							groups: {
								lots: {
									groupId: 'lots',
									startLabel: 'Nuovo lotto',
									closeLabel: 'Chiudi scarico',
									repeatable: true,
									canSpawn: true,
									canClose: true,
									childTaskIds: ['register'],
									instances: [
										{
											id: 'L1',
											tasks: { register: 'actionable' }
										}
									]
								}
							}
						}
					]
				}
			]
		};

		const rows = availableRowsFromWorklist(nested);
		expect(rows).toHaveLength(2);
		expect(rows[0]).toMatchObject({
			jobId: 'intake',
			jobInstanceId: 'T1',
			groupId: 'lots',
			groupInstanceId: 'L1',
			taskId: 'register',
			title: 'Register lot',
			instanceLabel: 'Truck 1'
		});
		expect(availableRowKey(rows[0]!)).toBe('intake/T1/lots/L1/register');
		expect(hrefLivingTarget(availableRowTarget(rows[0]!))).toBe(
			'/app/intake/T1/lots/L1/register'
		);

		expect(rows[1]).toMatchObject({
			jobId: 'intake',
			jobInstanceId: 'T1',
			groupId: 'lots',
			groupInstanceId: 'new',
			taskId: 'register',
			title: 'Register lot',
			instanceLabel: 'Truck 1'
		});
		expect(availableRowKey(rows[1]!)).toBe('intake/T1/lots/new/register');
		expect(availableRowTarget(rows[1]!)).toEqual({
			kind: 'task',
			jobId: 'intake',
			jobInstanceId: 'T1',
			groupId: 'lots',
			groupInstanceId: 'new',
			taskId: 'register'
		});
		expect(hrefLivingTarget(availableRowTarget(rows[1]!))).toBe(
			'/app/intake/T1/lots/new/register'
		);

		// truckArrive-shaped: canSpawn, empty open group → provisional first-leaf only.
		const spawnOnly: Worklist = {
			jobs: [
				{
					...nested.jobs[0]!,
					instances: [
						{
							...nested.jobs[0]!.instances[0]!,
							groups: {
								lots: {
									...nested.jobs[0]!.instances[0]!.groups!.lots,
									canSpawn: true,
									canClose: false,
									instances: []
								}
							}
						}
					]
				}
			]
		};
		const spawnRows = availableRowsFromWorklist(spawnOnly);
		expect(spawnRows).toHaveLength(1);
		expect(spawnRows[0]).toMatchObject({
			groupId: 'lots',
			groupInstanceId: 'new',
			taskId: 'register',
			title: 'Register lot'
		});
		expect(hrefLivingTarget(availableRowTarget(spawnRows[0]!))).toBe(
			'/app/intake/T1/lots/new/register'
		);
		// Still no separate Spawn / Can start peer for the group.
		expect(canStartRowsFromWorklist(spawnOnly)).toEqual([]);

		const noSpawn: Worklist = {
			jobs: [
				{
					...nested.jobs[0]!,
					instances: [
						{
							...nested.jobs[0]!.instances[0]!,
							groups: {
								lots: {
									...nested.jobs[0]!.instances[0]!.groups!.lots,
									canSpawn: false,
									canClose: false,
									instances: []
								}
							}
						}
					]
				}
			]
		};
		expect(availableRowsFromWorklist(noSpawn)).toEqual([]);
	});
});
