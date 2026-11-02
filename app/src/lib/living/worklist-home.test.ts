import { describe, expect, test } from 'vitest';

import type { Worklist } from './types.js';
import {
	availableRowsFromWorklist,
	canStartRowsFromWorklist,
	rolesIntersect,
	taskRouteInstanceSegment
} from './worklist-home.js';

const worklist: Worklist = {
	jobs: [
		{
			jobId: 'intake',
			canStart: true,
			repeatable: true,
			startLabel: 'Nuovo camion',
			taskIds: ['register', 'palletWeight'],
			tasks: [
				{ id: 'register', title: 'Registrazione', roles: ['administration'] },
				{ id: 'palletWeight', title: 'Peso pallet', roles: ['logistic'] }
			],
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
			startLabel: 'Nuovo turno',
			taskIds: ['shift', 'lineLoad'],
			tasks: [
				{ id: 'shift', title: 'Turno', roles: ['line'] },
				{ id: 'lineLoad', title: 'Carico', roles: ['line'] }
			],
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

describe('worklist-home bands', () => {
	test('rolesIntersect is slug∩roles', () => {
		expect(rolesIntersect(['administration'], ['administration', 'logistic'])).toBe(true);
		expect(rolesIntersect(['line'], ['administration'])).toBe(false);
		expect(rolesIntersect([], ['line'])).toBe(false);
	});

	test('availableRows sort Job → instance → Task and filter by viewer roles', () => {
		const admin = availableRowsFromWorklist(worklist, ['administration']);
		expect(admin.map((r) => `${r.jobId}/${r.jobInstanceId}/${r.taskId}`)).toEqual([
			'intake/NIR-2/register'
		]);
		expect(admin[0]?.title).toBe('Registrazione');
		expect(admin[0]?.instanceLabel).toBe('Camion B');

		const logistic = availableRowsFromWorklist(worklist, ['logistic']);
		expect(logistic.map((r) => `${r.jobId}/${r.jobInstanceId}/${r.taskId}`)).toEqual([
			'intake/NIR-1/palletWeight'
		]);
		expect(logistic[0]?.title).toBe('Peso pallet');

		const line = availableRowsFromWorklist(worklist, ['line']);
		expect(line.map((r) => `${r.jobId}/${r.jobInstanceId}/${r.taskId}`)).toEqual([
			'line/1st/lineLoad'
		]);

		const all = availableRowsFromWorklist(worklist, [
			'administration',
			'logistic',
			'line'
		]);
		expect(all.map((r) => `${r.jobId}/${r.jobInstanceId}/${r.taskId}`)).toEqual([
			'intake/NIR-1/palletWeight',
			'intake/NIR-2/register',
			'line/1st/lineLoad'
		]);
	});

	test('canStartRows use startLabel and first Task; provisional segment is new', () => {
		expect(canStartRowsFromWorklist(worklist)).toEqual([
			{ jobId: 'intake', startLabel: 'Nuovo camion', firstTaskId: 'register' }
		]);
		expect(taskRouteInstanceSegment('')).toBe('new');
		expect(taskRouteInstanceSegment('NIR-1')).toBe('NIR-1');
	});

	test('empty Worklist yields empty bands', () => {
		expect(availableRowsFromWorklist({ jobs: [] }, ['administration'])).toEqual([]);
		expect(canStartRowsFromWorklist({ jobs: [] })).toEqual([]);
		expect(availableRowsFromWorklist(null, ['administration'])).toEqual([]);
	});
});
