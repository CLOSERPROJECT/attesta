import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, test, vi } from 'vitest';

import { OrgSession } from '../session.svelte.js';
import {
	probeSyncDeepLocalArrayAdd,
	probeSyncDeepLocalWholeAssign,
	publishDeepLocalJson,
	type DeepLocalDraftSyncState
} from './deep-local-draft-sync.svelte.js';

const libDir = dirname(fileURLToPath(import.meta.url));

describe('publishDeepLocalJson / probes', () => {
	test('array Add: first observe seeds; mutate dirties via setDraft', () => {
		const setDraft = vi.fn();
		const seedDraft = vi.fn();
		const { seedCalls, setCalls, calls } = probeSyncDeepLocalArrayAdd(setDraft, seedDraft);

		expect(seedCalls).toHaveLength(1);
		expect(seedCalls[0]).toEqual({});
		expect(setCalls).toHaveLength(1);
		expect(setCalls[0]).toEqual({ rows: [{ weight: 12 }] });
		expect(calls).toHaveLength(2);
		expect(seedDraft).toHaveBeenCalledTimes(1);
		expect(setDraft).toHaveBeenCalledTimes(1);
		expect(setDraft).toHaveBeenCalledWith({ rows: [{ weight: 12 }] });
	});

	test('whole assign: first observe seeds; assign dirties via setDraft', () => {
		const setDraft = vi.fn();
		const seedDraft = vi.fn();
		const { seedCalls, setCalls } = probeSyncDeepLocalWholeAssign(setDraft, seedDraft);

		expect(seedCalls).toEqual([{}]);
		expect(setCalls).toEqual([{ material: 'polvereDiVetro', weight: 1 }]);
		expect(seedDraft).toHaveBeenCalledTimes(1);
		expect(setDraft).toHaveBeenCalledTimes(1);
	});

	test('identical json is a no-op after prime', () => {
		const setDraft = vi.fn();
		const seedDraft = vi.fn();
		const state: DeepLocalDraftSyncState = { lastJson: '', primed: false };
		const publish = { seedDraft, setDraft };
		publishDeepLocalJson('{"a":1}', state, publish);
		publishDeepLocalJson('{"a":1}', state, publish);
		expect(seedDraft).toHaveBeenCalledTimes(1);
		expect(setDraft).not.toHaveBeenCalled();
	});

	test('OrgSession: first tick leaves hasDirtyDraft false; edit dirties', () => {
		const session = new OrgSession({
			load: { load: async () => null! },
			hydrate: false
		});
		const path = 'intake/register';
		const key = '_new';
		const state: DeepLocalDraftSyncState = { lastJson: '', primed: false };
		const publish = {
			seedDraft: (next: Record<string, unknown>) => session.seedDraft(path, key, next),
			setDraft: (next: Record<string, unknown>) => session.setDraft(path, key, next)
		};

		publishDeepLocalJson('{}', state, publish);
		expect(session.getDraft(path, key)).toEqual({});
		expect(session.hasDirtyDraft(path, key)).toBe(false);

		publishDeepLocalJson(JSON.stringify({ rows: [{ weight: 12 }] }), state, publish);
		expect(session.getDraft(path, key)).toEqual({ rows: [{ weight: 12 }] });
		expect(session.hasDirtyDraft(path, key)).toBe(true);
	});
});

describe('syncDeepLocalToDraft seed vs dirty wiring (source)', () => {
	test('effect delegates to publishDeepLocalJson; createConfigForm / task-form wire seedValue', () => {
		const syncSrc = readFileSync(join(libDir, 'deep-local-draft-sync.svelte.ts'), 'utf8');
		const createSrc = readFileSync(join(libDir, 'create-config-form.svelte.ts'), 'utf8');
		const liveSrc = readFileSync(join(libDir, 'live-task-form.svelte.ts'), 'utf8');
		const portsSrc = readFileSync(join(libDir, 'live-task-chrome.svelte.ts'), 'utf8');
		const taskSrc = readFileSync(join(libDir, '../task-form.svelte'), 'utf8');

		expect(syncSrc).toContain('publishDeepLocalJson');
		expect(syncSrc).toMatch(/primed\s*\?\s*publish\.setDraft\s*:\s*publish\.seedDraft/);
		expect(createSrc).toContain('seedValue');
		expect(createSrc).toContain('seedDraft: (next) => options.seedValue');
		expect(createSrc).toContain('setDraft: (next) => setValue');
		expect(portsSrc).toContain('seedValue:');
		expect(liveSrc).toContain('seedValue: ports.seedValue');
		expect(taskSrc).toContain('seedValue:');
		expect(taskSrc).toContain('seedDraftValue');
		expect(taskSrc).toContain('orgSession.seedDraft');
	});
});
