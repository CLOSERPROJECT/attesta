import { describe, expect, test } from 'vitest';

import type { TaskFormBundle } from './form/task-form-bundle.js';
import type { Worklist } from './types.js';
import {
	commitHasLinks,
	resolveCommitFollowUp,
	resolveGroupStartAnotherTarget,
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
	const pallet = viewer === 'logistic' ? 'actionable' : 'available';
	const codice = viewer === 'admin' ? 'actionable' : 'available';
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
							palletWeight: pallet
						}
					},
					{
						id: 'NIR-advance',
						label: 'Advance',
						tasks: {
							register: 'done',
							codiceFir: codice,
							palletWeight: 'locked'
						}
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

	test('Task exit: provisional Job Home; existing overview', () => {
		expect(resolveTaskExit('intake', '', true)).toEqual({ kind: 'home' });
		expect(resolveTaskExit('intake', 'NIR-1', false)).toEqual({
			kind: 'overview',
			jobId: 'intake',
			jobInstanceId: 'NIR-1'
		});
	});

	test('nested provisional Spawn deep-link mounts; minted uses group actionable', () => {
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
							label: 'T1',
							tasks: { lots: 'available', weightCheck: 'locked' },
							groups: {
								lots: {
									groupId: 'lots',
									startLabel: 'Nuovo lotto',
									repeatable: true,
									canSpawn: true,
									canClose: false,
									childTaskIds: ['register', 'labels'],
									instances: [
										{
											id: 'lots-1',
											tasks: { register: 'actionable', labels: 'locked' }
										}
									]
								}
							}
						}
					]
				}
			]
		};

		expect(
			resolveTaskDeepLink('intake', 'T1', 'register', false, wl, 'lots', 'new')
		).toEqual({ kind: 'task' });
		expect(
			resolveTaskDeepLink('intake', 'T1', 'register', false, wl, 'lots', 'lots-1')
		).toEqual({ kind: 'task' });
		expect(
			resolveTaskDeepLink('intake', 'T1', 'labels', false, wl, 'lots', 'lots-1')
		).toEqual({ kind: 'overview', jobId: 'intake', jobInstanceId: 'T1' });
	});

	test('task next with nested bundle includes group coordinates', () => {
		const nested: TaskFormBundle = {
			...nextBundle,
			taskId: 'register',
			jobInstanceId: 'T1',
			groupId: 'lots',
			groupInstanceId: 'lots-1'
		};
		expect(resolvePostCommitNavigation('task', 'intake', 'T1', nested)).toEqual({
			kind: 'task',
			jobId: 'intake',
			jobInstanceId: 'T1',
			taskId: 'register',
			groupId: 'lots',
			groupInstanceId: 'lots-1'
		});
	});
});

describe('post-Commit stay-for-links', () => {
	const downloadLinks = [
		{
			label: 'Labels',
			url: 'https://example.test/labels.pdf',
			action: 'download' as const,
			filename: 'labels.pdf'
		}
	];

	test('any non-empty links are detected', () => {
		expect(commitHasLinks(downloadLinks)).toBe(true);
		expect(
			commitHasLinks([
				{ label: 'Open', url: 'https://example.test/x', action: 'link' }
			])
		).toBe(true);
		expect(commitHasLinks([])).toBe(false);
		expect(commitHasLinks(null)).toBe(false);
	});

	test('with any links → stay; empty → existing next law', () => {
		expect(
			resolveCommitFollowUp('overview', 'intake', 'T1', null, downloadLinks)
		).toEqual({ kind: 'stay' });
		expect(resolveCommitFollowUp('overview', 'intake', 'T1', null, [])).toEqual({
			kind: 'overview',
			jobId: 'intake',
			jobInstanceId: 'T1'
		});
		expect(
			resolveCommitFollowUp('task', 'intake', 'NIR-advance', nextBundle, [
				{ label: 'doc', url: 'https://example.test/d', action: 'link' }
			])
		).toEqual({ kind: 'stay' });
		expect(
			resolveCommitFollowUp('overview', 'passport', 'L1', null, [
				{
					label: 'Digital Product Passport',
					url: 'https://example.test/dpp',
					action: 'link'
				}
			])
		).toEqual({ kind: 'stay' });
	});

	test('Start another targets provisional first-leaf form', () => {
		expect(resolveGroupStartAnotherTarget('intake', 'T1', 'lots', 'register')).toEqual({
			kind: 'task',
			jobId: 'intake',
			jobInstanceId: 'T1',
			groupId: 'lots',
			groupInstanceId: 'new',
			taskId: 'register'
		});
	});
});
