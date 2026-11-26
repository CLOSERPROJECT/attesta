import { describe, expect, test } from 'vitest';

import { provisionalSeedDraftKey } from '../drafts.js';
import { taskAddressKey, type TaskAddress } from '../task-address.js';
import { BoardEmbedOpen, embedBodyKey } from './board-embed-open.svelte.js';

const addr = (taskId: string, provisional = false): TaskAddress => ({
	stream: 'pv-recovery',
	instance: 'NIR-1',
	jobId: 'lot_processing',
	jobInstanceId: provisional ? '' : 'DA-1',
	taskId,
	provisionalJob: provisional
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

	test('provisional seed keys isolate accordion open state', () => {
		const store = new BoardEmbedOpen();
		const start = addr('startLot', true);
		const seedL1 = { lotId: 'L1' };
		const seedL2 = { lotId: 'L2' };

		expect(embedBodyKey(start, seedL1)).toBe(
			`${taskAddressKey(start)}|${provisionalSeedDraftKey(seedL1)}`
		);

		store.setOpen(start, true, seedL1);
		expect(store.isOpen(start, seedL1)).toBe(true);
		expect(store.isOpen(start, seedL2)).toBe(false);
		// Address-only query sees any seeded open (hold / collapse callers).
		expect(store.isOpen(start)).toBe(true);

		store.setOpen(start, true, seedL2);
		expect(store.isOpen(start, seedL1)).toBe(true);
		expect(store.isOpen(start, seedL2)).toBe(true);

		store.setOpen(start, false, seedL1);
		expect(store.isOpen(start, seedL1)).toBe(false);
		expect(store.isOpen(start, seedL2)).toBe(true);

		store.closeAll([start]);
		expect(store.isOpen(start, seedL2)).toBe(false);
		expect(Object.keys(store.opened)).toHaveLength(0);
	});
});
