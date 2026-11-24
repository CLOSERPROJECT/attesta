import { describe, expect, test } from 'vitest';

import { taskAddressKey, type TaskAddress } from '../task-address.js';
import { BoardJustCompleted } from './board-just-completed.svelte.js';
import type { BoardTaskFocus } from './board-url.js';

const addr = (taskId: string, jobInstanceId = 'DA-1'): TaskAddress => ({
	stream: 'recycling',
	instance: '261102-1',
	jobId: 'lot_processing',
	jobInstanceId,
	taskId,
	provisionalJob: false
});

const focus = (taskId: string, jobInstanceId = 'DA-1'): BoardTaskFocus => ({
	jobId: 'lot_processing',
	jobInstanceId,
	taskId
});

describe('BoardJustCompleted', () => {
	test('pin replaces; has/get keyed by taskAddressKey', () => {
		const store = new BoardJustCompleted();
		const a = addr('weigh');
		const b = addr('nir');

		expect(store.has(a)).toBe(false);
		expect(store.get()).toBeNull();

		store.pin(a, 'sub-1');
		expect(store.has(a)).toBe(true);
		expect(store.has(b)).toBe(false);
		expect(store.get()).toEqual({ address: a, submissionId: 'sub-1' });
		expect(taskAddressKey(store.get()!.address)).toBe(taskAddressKey(a));

		store.pin(b);
		expect(store.has(a)).toBe(false);
		expect(store.has(b)).toBe(true);
		expect(store.get()).toEqual({ address: b });
	});

	test('clear drops the active pin', () => {
		const store = new BoardJustCompleted();
		const a = addr('weigh');
		store.pin(a, 'sub-1');
		store.clear();
		expect(store.has(a)).toBe(false);
		expect(store.get()).toBeNull();
	});

	test('onFocusChange keeps pin when focus stays on pinned task', () => {
		const store = new BoardJustCompleted();
		const a = addr('weigh');
		store.pin(a);
		store.onFocusChange(focus('weigh'), a.stream, a.instance);
		expect(store.has(a)).toBe(true);
	});

	test('onFocusChange clears when focus leaves or is cleared', () => {
		const store = new BoardJustCompleted();
		const a = addr('weigh');
		store.pin(a);

		store.onFocusChange(focus('nir'), a.stream, a.instance);
		expect(store.get()).toBeNull();

		store.pin(a);
		store.onFocusChange(null, a.stream, a.instance);
		expect(store.get()).toBeNull();

		store.pin(a);
		store.onFocusChange(focus('weigh', 'CO-1'), a.stream, a.instance);
		expect(store.get()).toBeNull();
	});
});
