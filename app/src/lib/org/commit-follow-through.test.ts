import { describe, expect, test, vi } from 'vitest';

import {
	applyCommitFollowThrough,
	applyContinueFollowThrough,
	decideCommitFollowThrough,
	decideContinueFollowThrough
} from './commit-follow-through.js';
import type { TaskFormBundle } from './form/task-form-bundle.js';
import { normalizeTaskAddress } from './task-address.js';

const stream = 'recycling';
const truck = '261102-1';

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

const sameAddressBundle: TaskFormBundle = {
	...nextBundle,
	taskId: 'register',
	jobInstanceId: 'NIR-1'
};

const current = normalizeTaskAddress({
	stream,
	instance: truck,
	jobId: 'intake',
	jobInstanceId: 'NIR-1',
	taskId: 'register'
});

const homeTarget = { kind: 'home' as const, stream, instance: truck };

describe('decideCommitFollowThrough', () => {
	test('stay parks; other destinations navigate with stream prefix', () => {
		expect(
			decideCommitFollowThrough(
				{
					navigation: 'stay',
					jobInstanceId: 'NIR-1',
					nextTaskFormBundle: null
				},
				current
			)
		).toEqual({ kind: 'stay' });

		expect(
			decideCommitFollowThrough(
				{
					navigation: 'home',
					jobInstanceId: 'NIR-1',
					nextTaskFormBundle: null
				},
				current
			)
		).toEqual({ kind: 'navigate', target: homeTarget });

		expect(
			decideCommitFollowThrough(
				{
					navigation: 'task',
					jobInstanceId: 'NIR-advance',
					nextTaskFormBundle: nextBundle
				},
				{ stream, instance: truck, jobId: 'intake' }
			)
		).toEqual({
			kind: 'navigate',
			target: {
				kind: 'task',
				stream,
				instance: truck,
				jobId: 'intake',
				jobInstanceId: 'NIR-advance',
				taskId: 'codiceFir'
			}
		});
	});

	test('Play Commit (empty instance) navigates to minted instance home', () => {
		expect(
			decideCommitFollowThrough(
				{
					navigation: 'home',
					jobInstanceId: '261102-2',
					nextTaskFormBundle: null
				},
				{ stream, instance: '', jobId: 'intake' }
			)
		).toEqual({
			kind: 'navigate',
			target: { kind: 'home', stream, instance: '261102-2' }
		});
	});
});

describe('decideContinueFollowThrough', () => {
	test('same-address Task Continue → reload', () => {
		expect(
			decideContinueFollowThrough(
				{
					continueNavigation: 'task',
					jobInstanceId: 'NIR-1',
					nextTaskFormBundle: sameAddressBundle
				},
				current
			)
		).toEqual({ kind: 'reload' });
	});

	test('different Task address → navigate', () => {
		expect(
			decideContinueFollowThrough(
				{
					continueNavigation: 'task',
					jobInstanceId: 'NIR-advance',
					nextTaskFormBundle: nextBundle
				},
				current
			)
		).toEqual({
			kind: 'navigate',
			target: {
				kind: 'task',
				stream,
				instance: truck,
				jobId: 'intake',
				jobInstanceId: 'NIR-advance',
				taskId: 'codiceFir'
			}
		});
	});

	test('home Continue → navigate instance home', () => {
		expect(
			decideContinueFollowThrough(
				{
					continueNavigation: 'home',
					jobInstanceId: 'NIR-1',
					nextTaskFormBundle: null
				},
				current
			)
		).toEqual({ kind: 'navigate', target: homeTarget });
	});

	test('null continueNavigation defaults to instance home', () => {
		expect(
			decideContinueFollowThrough(
				{
					continueNavigation: null,
					jobInstanceId: 'NIR-1',
					nextTaskFormBundle: null
				},
				current
			)
		).toEqual({ kind: 'navigate', target: homeTarget });
	});
});

describe('apply follow-through', () => {
	test('Commit navigate clears links then navigates', () => {
		const clearLinksStay = vi.fn();
		const navigate = vi.fn();
		applyCommitFollowThrough(
			{ kind: 'navigate', target: homeTarget },
			{ clearLinksStay, navigate }
		);
		expect(clearLinksStay).toHaveBeenCalledOnce();
		expect(navigate).toHaveBeenCalledWith(homeTarget);
	});

	test('Continue reload clears links then reloads', async () => {
		const clearLinksStay = vi.fn();
		const navigate = vi.fn();
		const reload = vi.fn(async () => undefined);
		await applyContinueFollowThrough(
			{ kind: 'reload' },
			{ clearLinksStay, navigate, reload }
		);
		expect(clearLinksStay).toHaveBeenCalledOnce();
		expect(reload).toHaveBeenCalledOnce();
		expect(navigate).not.toHaveBeenCalled();
	});

	test('Continue navigate clears links then navigates', async () => {
		const clearLinksStay = vi.fn();
		const navigate = vi.fn();
		const reload = vi.fn(async () => undefined);
		await applyContinueFollowThrough(
			{ kind: 'navigate', target: homeTarget },
			{ clearLinksStay, navigate, reload }
		);
		expect(clearLinksStay).toHaveBeenCalledOnce();
		expect(navigate).toHaveBeenCalledWith(homeTarget);
		expect(reload).not.toHaveBeenCalled();
	});
});
