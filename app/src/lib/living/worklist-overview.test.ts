import { describe, expect, test } from 'vitest';

import type { Worklist } from './types.js';
import { formatHandoffCopy, jobInstanceOverview } from './worklist-overview.js';

const taskChrome = [
	{ id: 'register', title: 'Registrazione', roles: ['administration'] },
	{ id: 'codiceFir', title: 'Codice FIR', roles: ['administration'] },
	{ id: 'palletWeight', title: 'Peso pallet', roles: ['logistic'] }
];

function worklistFor(viewer: 'admin' | 'logistic'): Worklist {
	const nir1Avail = viewer === 'logistic' ? ['palletWeight'] : [];
	const bothAvail =
		viewer === 'admin'
			? ['register']
			: viewer === 'logistic'
				? ['palletWeight']
				: [];
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
							palletWeight: 'available'
						},
						viewerAvailable: nir1Avail
					},
					{
						id: 'NIR-done',
						label: 'Done truck',
						tasks: {
							register: 'done',
							codiceFir: 'done',
							palletWeight: 'done'
						},
						viewerAvailable: []
					},
					{
						id: 'NIR-both-available',
						label: 'Split available',
						tasks: {
							register: 'available',
							codiceFir: 'locked',
							palletWeight: 'available'
						},
						viewerAvailable: bothAvail
					}
				]
			}
		]
	};
}

const adminWorklist = worklistFor('admin');
const logisticWorklist = worklistFor('logistic');

describe('job instance overview model', () => {
	test('lists statuses; linkable only for viewerAvailable', () => {
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

	test('handoff uses first available only; no banner when next Task is theirs', () => {
		const admin = jobInstanceOverview(adminWorklist, 'intake', 'NIR-both-available');
		expect(admin?.tasks.map((t) => [t.taskId, t.status, t.linkable])).toEqual([
			['register', 'available', true],
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

	test('finished model marks all done', () => {
		const done = jobInstanceOverview(adminWorklist, 'intake', 'NIR-done');
		expect(done?.finished).toBe(true);
		expect(done?.handoffRoles).toBeNull();
	});

	test('unknown instance yields null', () => {
		expect(jobInstanceOverview(adminWorklist, 'intake', 'missing')).toBeNull();
		expect(jobInstanceOverview(adminWorklist, 'intake', 'new')).toBeNull();
	});
});
