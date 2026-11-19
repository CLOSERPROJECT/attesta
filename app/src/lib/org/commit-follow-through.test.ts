import { describe, expect, test, vi } from 'vitest';

import {
	applyCommitFollowThrough,
	applyContinueFollowThrough,
	decideCommitFollowThrough,
	decideContinueFollowThrough
} from './commit-follow-through.js';
import type { TaskFormBundle } from './form/task-form-bundle.js';
import { normalizeTaskAddress } from './task-address.js';

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
	jobId: 'intake',
	jobInstanceId: 'NIR-1',
	taskId: 'register'
});

describe('decideCommitFollowThrough', () => {
	test('stay parks; other destinations navigate', () => {
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
		).toEqual({ kind: 'navigate', target: { kind: 'home' } });

		expect(
			decideCommitFollowThrough(
				{
					navigation: 'task',
					jobInstanceId: 'NIR-advance',
					nextTaskFormBundle: nextBundle
				},
				{ jobId: 'intake' }
			)
		).toEqual({
			kind: 'navigate',
			target: {
				kind: 'task',
				jobId: 'intake',
				jobInstanceId: 'NIR-advance',
				taskId: 'codiceFir'
			}
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
				jobId: 'intake',
				jobInstanceId: 'NIR-advance',
				taskId: 'codiceFir'
			}
		});
	});

	test('home Continue → navigate home', () => {
		expect(
			decideContinueFollowThrough(
				{
					continueNavigation: 'home',
					jobInstanceId: 'NIR-1',
					nextTaskFormBundle: null
				},
				current
			)
		).toEqual({ kind: 'navigate', target: { kind: 'home' } });
	});

	test('null continueNavigation defaults to home', () => {
		expect(
			decideContinueFollowThrough(
				{
					continueNavigation: null,
					jobInstanceId: 'NIR-1',
					nextTaskFormBundle: null
				},
				current
			)
		).toEqual({ kind: 'navigate', target: { kind: 'home' } });
	});
});

describe('apply follow-through', () => {
	test('Commit navigate clears links then navigates', () => {
		const clearLinksStay = vi.fn();
		const navigate = vi.fn();
		applyCommitFollowThrough(
			{ kind: 'navigate', target: { kind: 'home' } },
			{ clearLinksStay, navigate }
		);
		expect(clearLinksStay).toHaveBeenCalledOnce();
		expect(navigate).toHaveBeenCalledWith({ kind: 'home' });
	});

	test('Continue reload clears links then reloads worklist', async () => {
		const clearLinksStay = vi.fn();
		const navigate = vi.fn();
		const reload = vi.fn(async () => undefined);
		const worklist = { jobs: [] };
		await applyContinueFollowThrough(
			{ kind: 'reload' },
			{ clearLinksStay, navigate, reload },
			worklist
		);
		expect(clearLinksStay).toHaveBeenCalledOnce();
		expect(reload).toHaveBeenCalledWith(worklist);
		expect(navigate).not.toHaveBeenCalled();
	});

	test('Continue navigate clears links then navigates', async () => {
		const clearLinksStay = vi.fn();
		const navigate = vi.fn();
		const reload = vi.fn(async () => undefined);
		await applyContinueFollowThrough(
			{ kind: 'navigate', target: { kind: 'home' } },
			{ clearLinksStay, navigate, reload }
		);
		expect(clearLinksStay).toHaveBeenCalledOnce();
		expect(navigate).toHaveBeenCalledWith({ kind: 'home' });
		expect(reload).not.toHaveBeenCalled();
	});
});
