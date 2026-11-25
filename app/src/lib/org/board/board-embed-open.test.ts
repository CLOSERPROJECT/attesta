import { describe, expect, test } from 'vitest';

import { taskAddressKey, type TaskAddress } from '../task-address.js';
import { BoardEmbedOpen } from './board-embed-open.svelte.js';

const addr = (taskId: string): TaskAddress => ({
	stream: 'pv-recovery',
	instance: 'NIR-1',
	jobId: 'lot_processing',
	jobInstanceId: 'DA-1',
	taskId
});

describe('BoardEmbedOpen', () => {
	test('defaults closed; open and collapse by key', () => {
		const store = new BoardEmbedOpen();
		const a = addr('weigh');
		const b = addr('nir');

		expect(store.isOpen(a)).toBe(false);
		expect(store.toggle(a)).toBe(true);
		expect(store.isOpen(a)).toBe(true);
		expect(store.isOpen(b)).toBe(false);

		store.setOpen(a, false);
		expect(store.isOpen(a)).toBe(false);
		expect(Object.keys(store.opened)).toHaveLength(0);
	});

	test('setOpen is idempotent', () => {
		const store = new BoardEmbedOpen();
		const a = addr('weigh');
		store.setOpen(a, true);
		store.setOpen(a, true);
		expect(Object.keys(store.opened)).toEqual([taskAddressKey(a)]);
		store.setOpen(a, false);
		store.setOpen(a, false);
		expect(Object.keys(store.opened)).toHaveLength(0);
	});

	test('closeAll clears open keys for listed addresses', () => {
		const store = new BoardEmbedOpen();
		const a = addr('weigh');
		const b = addr('nir');
		store.setOpen(a, true);
		store.setOpen(b, true);
		store.closeAll([a, b]);
		expect(store.isOpen(a)).toBe(false);
		expect(store.isOpen(b)).toBe(false);
		store.setOpen(a, true);
		store.setOpen(b, true);
		store.closeAll([a]);
		expect(Object.keys(store.opened)).toEqual([taskAddressKey(b)]);
	});
});
