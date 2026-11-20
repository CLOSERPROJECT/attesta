import { describe, expect, test } from 'vitest';

import { taskAddressKey, type TaskAddress } from '../task-address.js';
import { BoardEmbedOpen } from './board-embed-open.svelte.js';

const addr = (taskId: string): TaskAddress => ({
	stream: 'recycling',
	instance: 'NIR-1',
	jobId: 'lot_processing',
	jobInstanceId: 'DA-1',
	taskId
});

describe('BoardEmbedOpen', () => {
	test('defaults open; collapse and reopen by key', () => {
		const store = new BoardEmbedOpen();
		const a = addr('weigh');
		const b = addr('nir');

		expect(store.isOpen(a)).toBe(true);
		expect(store.toggle(a)).toBe(false);
		expect(store.isOpen(a)).toBe(false);
		expect(store.isOpen(b)).toBe(true);

		store.setOpen(a, true);
		expect(store.isOpen(a)).toBe(true);
		expect(Object.keys(store.closed)).toHaveLength(0);
	});

	test('setOpen is idempotent', () => {
		const store = new BoardEmbedOpen();
		const a = addr('weigh');
		store.setOpen(a, false);
		store.setOpen(a, false);
		expect(Object.keys(store.closed)).toEqual([taskAddressKey(a)]);
		store.setOpen(a, true);
		store.setOpen(a, true);
		expect(Object.keys(store.closed)).toHaveLength(0);
	});
});
