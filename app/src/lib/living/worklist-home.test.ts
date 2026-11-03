import { describe, expect, test } from 'vitest';

import type { Worklist } from './types.js';
import {
	availableRowsFromWorklist,
	canStartRowsFromWorklist,
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

/** Viewer-scoped Worklists (statuses shared; viewerAvailable differs). */
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
					tasks: { register: 'done', palletWeight: 'available' },
					viewerAvailable: []
				},
				{
					id: 'NIR-2',
					label: 'Camion B',
					tasks: { register: 'available', palletWeight: 'locked' },
					viewerAvailable: ['register']
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
					tasks: { shift: 'done', lineLoad: 'available' },
					viewerAvailable: []
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
					tasks: { register: 'done', palletWeight: 'available' },
					viewerAvailable: ['palletWeight']
				},
				{
					id: 'NIR-2',
					label: 'Camion B',
					tasks: { register: 'available', palletWeight: 'locked' },
					viewerAvailable: []
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
					tasks: { shift: 'done', lineLoad: 'available' },
					viewerAvailable: []
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
					tasks: { register: 'done', palletWeight: 'available' },
					viewerAvailable: []
				},
				{
					id: 'NIR-2',
					label: 'Camion B',
					tasks: { register: 'available', palletWeight: 'locked' },
					viewerAvailable: []
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
					tasks: { shift: 'done', lineLoad: 'available' },
					viewerAvailable: ['lineLoad']
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
					tasks: { register: 'done', palletWeight: 'available' },
					viewerAvailable: ['palletWeight']
				},
				{
					id: 'NIR-2',
					label: 'Camion B',
					tasks: { register: 'available', palletWeight: 'locked' },
					viewerAvailable: ['register']
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
					tasks: { shift: 'done', lineLoad: 'available' },
					viewerAvailable: ['lineLoad']
				}
			]
		}
	]
};

describe('worklist-home bands', () => {
	test('availableRows sort Job → instance → Task from viewerAvailable', () => {
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
		expect(availableRowsFromWorklist(null)).toEqual([]);
	});
});
