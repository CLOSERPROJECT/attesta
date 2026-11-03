import { describe, expect, test } from 'vitest';

import type { TaskFormBundle } from './form/task-form-bundle.js';
import type { Worklist } from './types.js';
import {
	resolvePostCommitNavigation,
	resolveTaskDeepLink,
	resolveTaskExit,
	resolveUnmountableTaskTarget
} from './worklist-nav.js';

const taskChrome = [
	{ id: 'register', title: 'Registrazione', roles: ['administration'] },
	{ id: 'codiceFir', title: 'Codice FIR', roles: ['administration'] },
	{ id: 'palletWeight', title: 'Peso pallet', roles: ['logistic'] }
];

function worklistFor(viewer: 'admin' | 'logistic'): Worklist {
	const nir1Avail = viewer === 'logistic' ? ['palletWeight'] : [];
	const advanceAvail = viewer === 'admin' ? ['codiceFir'] : [];
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
						id: 'NIR-advance',
						label: 'Advance',
						tasks: {
							register: 'done',
							codiceFir: 'available',
							palletWeight: 'locked'
						},
						viewerAvailable: advanceAvail
					}
				]
			}
		]
	};
}

const adminWorklist = worklistFor('admin');

const nextBundle: TaskFormBundle = {
	jobId: 'intake',
	taskId: 'codiceFir',
	jobInstanceId: 'NIR-advance',
	title: 'Codice FIR',
	surface: 'live',
	schema: { type: 'object' },
	uiSchema: {},
	initialData: {},
	chrome: [],
	chromeFieldIds: [],
	sealFieldIds: [],
	bindings: { domain: {} }
};

describe('post-Commit navigation adapter', () => {
	test('task next uses nextTaskFormBundle address', () => {
		expect(
			resolvePostCommitNavigation('task', 'intake', 'NIR-advance', nextBundle)
		).toEqual({
			kind: 'task',
			jobId: 'intake',
			jobInstanceId: 'NIR-advance',
			taskId: 'codiceFir'
		});
	});

	test('task next without usable bundle falls back to home', () => {
		expect(resolvePostCommitNavigation('task', 'intake', 'NIR-advance', null)).toEqual({
			kind: 'home'
		});
	});

	test('overview next opens overview for the committed instance', () => {
		expect(resolvePostCommitNavigation('overview', 'intake', 'NIR-1', null)).toEqual({
			kind: 'overview',
			jobId: 'intake',
			jobInstanceId: 'NIR-1'
		});
	});

	test('home next goes Home', () => {
		expect(resolvePostCommitNavigation('home', 'intake', 'NIR-done', null)).toEqual({
			kind: 'home'
		});
	});
});

describe('Task deep link + exit', () => {
	test('unauthorized / locked Task URL → overview; unknown → Home', () => {
		expect(
			resolveTaskDeepLink('intake', 'NIR-1', 'palletWeight', false, adminWorklist)
		).toEqual({ kind: 'overview', jobId: 'intake', jobInstanceId: 'NIR-1' });

		expect(
			resolveTaskDeepLink('intake', 'NIR-advance', 'codiceFir', false, adminWorklist)
		).toEqual({ kind: 'task' });

		expect(
			resolveTaskDeepLink('intake', 'missing', 'register', false, adminWorklist)
		).toEqual({ kind: 'home' });

		expect(
			resolveTaskDeepLink('intake', 'new', 'register', true, adminWorklist)
		).toEqual({ kind: 'task' });
	});

	test('task-form deny shares unmountable redirect with deep-link', () => {
		expect(
			resolveUnmountableTaskTarget('intake', 'NIR-1', false, adminWorklist)
		).toEqual({ kind: 'overview', jobId: 'intake', jobInstanceId: 'NIR-1' });
		expect(resolveUnmountableTaskTarget('intake', '', true, adminWorklist)).toEqual({
			kind: 'home'
		});
		expect(
			resolveUnmountableTaskTarget('intake', 'missing', false, adminWorklist)
		).toEqual({ kind: 'home' });
	});

	test('Task exit: provisional Home; existing overview', () => {
		expect(resolveTaskExit('intake', '', true)).toEqual({ kind: 'home' });
		expect(resolveTaskExit('intake', 'NIR-1', false)).toEqual({
			kind: 'overview',
			jobId: 'intake',
			jobInstanceId: 'NIR-1'
		});
	});
});
