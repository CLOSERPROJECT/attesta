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

import type { Worklist } from './types.js';
import {
	actNowRowKey,
	actNowRowsFromWorklist,
	availableRowKey,
	availableRowsFromWorklist,
	canStartRowsFromWorklist,
	completedRowKey,
	completedRowsFromWorklist,
	instanceTaskProgress,
	progressFraction,
	taskRouteInstanceSegment
} from './worklist-home.js';
const chrome = {
	intake: [
		{ id: 'register', title: 'Registrazione', roles: ['administration'] },
		{ id: 'palletWeight', title: 'Peso pallet', roles: ['logistic'] }
	],
	lot_processing: [
		{ id: 'startLot', title: 'Apri lotto', roles: ['logistic'] },
		{ id: 'markProgress', title: 'Segna avanzamento', roles: ['line'] }
	]
} as const;

/** Viewer-scoped Worklists (not-done leaf is actionable vs available). */
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
					tasks: { register: 'actionable', palletWeight: 'unavailable' }
				}
			]
		},
		{
			jobId: 'lot_processing',
			canStart: false,
			repeatable: true,
			title: 'Lavorazione',
			startLabel: 'Apri lotto',
			taskIds: ['startLot', 'markProgress'],
			tasks: [...chrome.lot_processing],
			instances: [
				{
					id: 'CO-1',
					label: 'Lotto CO-1',
					tasks: { startLot: 'done', markProgress: 'available' }
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
					tasks: { register: 'available', palletWeight: 'unavailable' }
				}
			]
		},
		{
			jobId: 'lot_processing',
			canStart: false,
			repeatable: true,
			title: 'Lavorazione',
			startLabel: 'Apri lotto',
			taskIds: ['startLot', 'markProgress'],
			tasks: [...chrome.lot_processing],
			instances: [
				{
					id: 'CO-1',
					label: 'Lotto CO-1',
					tasks: { startLot: 'done', markProgress: 'available' }
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
					tasks: { register: 'available', palletWeight: 'unavailable' }
				}
			]
		},
		{
			jobId: 'lot_processing',
			canStart: false,
			repeatable: true,
			title: 'Lavorazione',
			startLabel: 'Apri lotto',
			taskIds: ['startLot', 'markProgress'],
			tasks: [...chrome.lot_processing],
			instances: [
				{
					id: 'CO-1',
					label: 'Lotto CO-1',
					tasks: { startLot: 'done', markProgress: 'actionable' }
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
					tasks: { register: 'actionable', palletWeight: 'unavailable' }
				}
			]
		},
		{
			jobId: 'lot_processing',
			canStart: false,
			repeatable: true,
			title: 'Lavorazione',
			startLabel: 'Apri lotto',
			taskIds: ['startLot', 'markProgress'],
			tasks: [...chrome.lot_processing],
			instances: [
				{
					id: 'CO-1',
					label: 'Lotto CO-1',
					tasks: { startLot: 'done', markProgress: 'actionable' }
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
			'lot_processing/CO-1/markProgress'
		]);
		expect(line[0]?.jobTitle).toBe('Lavorazione');
		expect(line[0]?.tasksDone).toBe(1);
		expect(line[0]?.tasksTotal).toBe(2);

		const all = availableRowsFromWorklist(allRolesWorklist);
		expect(all.map((r) => `${r.jobId}/${r.jobInstanceId}/${r.taskId}`)).toEqual([
			'intake/NIR-1/palletWeight',
			'intake/NIR-2/register',
			'lot_processing/CO-1/markProgress'
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

	test('actNowRows: actionable first, Can start after; keys discriminate kinds', () => {
		const rows = actNowRowsFromWorklist(adminWorklist);
		expect(rows.map((r) => (r.kind === 'task' ? r.taskId : `start:${r.jobId}`))).toEqual([
			'register',
			'start:intake'
		]);
		expect(rows[0]).toMatchObject({
			kind: 'task',
			jobId: 'intake',
			jobInstanceId: 'NIR-2',
			taskId: 'register',
			title: 'Registrazione'
		});
		expect(rows[1]).toMatchObject({
			kind: 'start',
			jobId: 'intake',
			startLabel: 'Nuovo camion',
			firstTaskId: 'register'
		});
		expect(actNowRowKey(rows[0]!)).toBe('task:intake/NIR-2/register');
		expect(actNowRowKey(rows[1]!)).toBe('start:intake');

		const tasksOnly = actNowRowsFromWorklist(logisticWorklist);
		expect(tasksOnly.every((r) => r.kind === 'task')).toBe(true);
		expect(tasksOnly.map((r) => (r.kind === 'task' ? r.taskId : ''))).toEqual([
			'palletWeight'
		]);
	});

	test('empty Worklist yields empty bands', () => {
		expect(availableRowsFromWorklist({ jobs: [] })).toEqual([]);
		expect(canStartRowsFromWorklist({ jobs: [] })).toEqual([]);
		expect(actNowRowsFromWorklist({ jobs: [] })).toEqual([]);
		expect(completedRowsFromWorklist({ jobs: [] })).toEqual([]);
		expect(availableRowsFromWorklist(null)).toEqual([]);
		expect(actNowRowsFromWorklist(null)).toEqual([]);
		expect(completedRowsFromWorklist(null)).toEqual([]);
	});

	test('home rows omit Spawn/Close / group chrome', () => {
		const banned = /canSpawn|canClose|"groups"|spawnGroup|closeGroup|closeLabel/;
		expect(JSON.stringify(adminWorklist)).not.toMatch(banned);
		for (const row of actNowRowsFromWorklist(adminWorklist)) {
			expect(['task', 'start']).toContain(row.kind);
			expect(row).not.toHaveProperty('canSpawn');
			expect(row).not.toHaveProperty('canClose');
			expect(row).not.toHaveProperty('groups');
			expect(JSON.stringify(row)).not.toMatch(banned);
		}
		for (const row of [
			...availableRowsFromWorklist(adminWorklist),
			...canStartRowsFromWorklist(adminWorklist),
			...completedRowsFromWorklist(adminWorklist)
		]) {
			expect(row).not.toHaveProperty('canSpawn');
			expect(row).not.toHaveProperty('canClose');
			expect(row).not.toHaveProperty('groups');
		}
	});

	test('completedRows include finished Job instances; unfinished excluded; Job → instance order', () => {
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
							tasks: { register: 'actionable', palletWeight: 'unavailable' }
						}
					]
				},
				{
					jobId: 'lot_processing',
					canStart: false,
					repeatable: true,
					title: 'Lavorazione',
					taskIds: ['startLot', 'markProgress'],
					tasks: [...chrome.lot_processing],
					instances: [
						{
							id: 'CO-1',
							label: 'Lotto CO-1',
							tasks: { startLot: 'done', markProgress: 'available' }
						},
						{
							id: 'CO-done',
							label: 'Lotto CO-done',
							tasks: { startLot: 'done', markProgress: 'done' }
						}
					]
				}
			]
		};

		const rows = completedRowsFromWorklist(mixed);
		expect(rows.map((r) => completedRowKey(r))).toEqual([
			'intake/NIR-done',
			'lot_processing/CO-done'
		]);
		expect(rows[0]).toMatchObject({
			jobId: 'intake',
			jobInstanceId: 'NIR-done',
			instanceLabel: 'Finished truck',
			jobTitle: 'Camion',
			tasksDone: 2,
			tasksTotal: 2
		});
		expect(rows[1]).toMatchObject({
			jobId: 'lot_processing',
			jobInstanceId: 'CO-done',
			instanceLabel: 'Lotto CO-done',
			jobTitle: 'Lavorazione',
			tasksDone: 2,
			tasksTotal: 2
		});

		for (const key of ['intake/NIR-1', 'intake/NIR-2', 'lot_processing/CO-1']) {
			expect(rows.map((r) => completedRowKey(r))).not.toContain(key);
		}
	});

	test('completedRows stay off act-now home (no Archive place)', () => {
		const mixed: Worklist = {
			jobs: [
				{
					jobId: 'intake',
					canStart: true,
					repeatable: true,
					title: 'Camion',
					taskIds: ['register', 'palletWeight'],
					tasks: [...chrome.intake],
					instances: [
						{
							id: 'NIR-open',
							label: 'Open',
							tasks: { register: 'actionable', palletWeight: 'unavailable' }
						},
						{
							id: 'NIR-done',
							label: 'Done',
							tasks: { register: 'done', palletWeight: 'done' }
						}
					]
				}
			]
		};
		const completed = completedRowsFromWorklist(mixed);
		expect(completed.map((r) => completedRowKey(r))).toEqual(['intake/NIR-done']);
		expect(actNowRowsFromWorklist(mixed).map((r) => actNowRowKey(r))).toEqual([
			'task:intake/NIR-open/register',
			'start:intake'
		]);
	});
});
