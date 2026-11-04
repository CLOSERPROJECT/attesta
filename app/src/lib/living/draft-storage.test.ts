import { describe, expect, test } from 'vitest';

import {
	DRAFT_STORAGE_PREFIX,
	livingDraftStorageKey,
	livingDraftStoragePrefix,
	loadDirtyDraftsForIdentity,
	memoryStorage,
	readStoredDraft,
	removeStoredDraft,
	writeStoredDraft
} from './draft-storage.js';

describe('draft-storage keys', () => {
	test('prefix and full key format', () => {
		expect(DRAFT_STORAGE_PREFIX).toBe('attesta:living-draft:');
		expect(livingDraftStoragePrefix('feragame', 'u1')).toBe('attesta:living-draft:feragame:u1:');
		expect(livingDraftStorageKey('feragame', 'u1', 'register::NIR-1')).toBe(
			'attesta:living-draft:feragame:u1:register::NIR-1'
		);
	});
});

describe('draft-storage read/write/remove', () => {
	test('memoryStorage roundtrip persists raw JSON object', () => {
		const storage = memoryStorage();
		const key = livingDraftStorageKey('feragame', 'u1', 'task::_new');
		const value = { plate: 'AB123', weight: 42 };

		writeStoredDraft(storage, key, value);
		expect(storage.getItem(key)).toBe(JSON.stringify(value));
		expect(readStoredDraft(storage, key)).toEqual(value);

		removeStoredDraft(storage, key);
		expect(readStoredDraft(storage, key)).toBeNull();
	});

	test('readStoredDraft returns null for missing, invalid JSON, and non-objects', () => {
		const storage = memoryStorage({
			badJson: '{',
			array: '[]',
			nullish: 'null',
			string: '"hi"'
		});
		expect(readStoredDraft(storage, 'missing')).toBeNull();
		expect(readStoredDraft(storage, 'badJson')).toBeNull();
		expect(readStoredDraft(storage, 'array')).toBeNull();
		expect(readStoredDraft(storage, 'nullish')).toBeNull();
		expect(readStoredDraft(storage, 'string')).toBeNull();
	});

	test('null storage is a no-op for write/remove and read', () => {
		const key = livingDraftStorageKey('org', 'user', 'k');
		expect(readStoredDraft(null, key)).toBeNull();
		writeStoredDraft(undefined, key, { a: 1 });
		removeStoredDraft(null, key);
	});
});

describe('loadDirtyDraftsForIdentity', () => {
	test('loads drafts for org/user prefix and ignores other identities', () => {
		const storage = memoryStorage();
		const ownA = livingDraftStorageKey('feragame', 'u1', 'register::inst-1');
		const ownB = livingDraftStorageKey('feragame', 'u1', 'palletWeight::inst-1');
		const otherOrg = livingDraftStorageKey('other', 'u1', 'register::inst-1');
		const otherUser = livingDraftStorageKey('feragame', 'u2', 'register::inst-1');
		const unrelated = 'some-other-app-key';

		writeStoredDraft(storage, ownA, { field: 'a' });
		writeStoredDraft(storage, ownB, { field: 'b' });
		writeStoredDraft(storage, otherOrg, { field: 'x' });
		writeStoredDraft(storage, otherUser, { field: 'y' });
		storage.setItem(unrelated, '{"ignored":true}');

		expect(loadDirtyDraftsForIdentity(storage, 'feragame', 'u1')).toEqual({
			'register::inst-1': { field: 'a' },
			'palletWeight::inst-1': { field: 'b' }
		});
	});

	test('skips keys under prefix with invalid JSON', () => {
		const storage = memoryStorage();
		const goodKey = livingDraftStorageKey('feragame', 'u1', 'ok');
		const badKey = livingDraftStorageKey('feragame', 'u1', 'broken');
		writeStoredDraft(storage, goodKey, { ok: true });
		storage.setItem(badKey, 'not-json');

		expect(loadDirtyDraftsForIdentity(storage, 'feragame', 'u1')).toEqual({
			ok: { ok: true }
		});
	});

	test('returns empty map when storage is null', () => {
		expect(loadDirtyDraftsForIdentity(null, 'feragame', 'u1')).toEqual({});
	});
});
