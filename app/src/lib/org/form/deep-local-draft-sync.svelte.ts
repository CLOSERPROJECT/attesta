/**
 * Why: sjsf array Add (SimpleKeyedArray.push) and objectCtx.set mutate deep
 * `$state` form local in place and never call createForm's value setter, so
 * DraftSession / localStorage never see the row. Wire once from createConfigForm
 * so every deep change republishes to Draft.
 *
 * First observation must not dirty: schema-default merge / remount bag populate
 * goes through `seedDraft` (non-dirty). Later local changes use dirtying `setDraft`.
 *
 * `$effect.root` — createConfigForm is called from LiveTaskForm.#mountForm
 * (not component init). JSON.stringify walks the proxy so Add / new keys
 * re-subscribe; `$state.snapshot` alone does not. `untrack` + lastJson gate —
 * publish must not re-enter the effect (effect_update_depth_exceeded).
 * Tests: drive `publishDeepLocalJson` (shared with the effect) via probes;
 * `$effect` itself is not runnable under Vitest node.
 */
import { untrack } from 'svelte';
import { republishBoundValue } from './republish-form-value.js';

/** Seed (first tick, non-dirty) vs set (later edits, dirty) publish ports. */
export type DeepLocalDraftPublish<T extends Record<string, unknown>> = {
	seedDraft: (next: T) => void;
	setDraft: (next: T) => void;
};

export type DeepLocalDraftSyncState = {
	lastJson: string;
	primed: boolean;
};

/** Shared tick: first json → seedDraft; later changes → setDraft. */
export function publishDeepLocalJson<T extends Record<string, unknown>>(
	json: string,
	state: DeepLocalDraftSyncState,
	publish: DeepLocalDraftPublish<T>
): void {
	if (json === state.lastJson) return;
	state.lastJson = json;
	const publishDraft = state.primed ? publish.setDraft : publish.seedDraft;
	state.primed = true;
	untrack(() => {
		republishBoundValue(
			() => JSON.parse(json) as Record<string, unknown>,
			(next) => publishDraft(next as T)
		);
	});
}

export function syncDeepLocalToDraft<T extends Record<string, unknown>>(
	getLocal: () => T,
	publish: DeepLocalDraftPublish<T>
): () => void {
	const state: DeepLocalDraftSyncState = { lastJson: '', primed: false };
	return $effect.root(() => {
		$effect(() => {
			// JSON.stringify walks the proxy graph so array push / new keys re-run;
			// $state.snapshot alone does not always subscribe to deep adds.
			publishDeepLocalJson(JSON.stringify(getLocal()), state, publish);
		});
	});
}

export type ProbeSyncDeepLocalResult = {
	calls: Record<string, unknown>[];
	seedCalls: Record<string, unknown>[];
	setCalls: Record<string, unknown>[];
};

function probePublish(
	setDraft: (next: Record<string, unknown>) => void,
	seedDraft: (next: Record<string, unknown>) => void
): {
	publish: DeepLocalDraftPublish<Record<string, unknown>>;
	calls: Record<string, unknown>[];
	seedCalls: Record<string, unknown>[];
	setCalls: Record<string, unknown>[];
} {
	const calls: Record<string, unknown>[] = [];
	const seedCalls: Record<string, unknown>[] = [];
	const setCalls: Record<string, unknown>[] = [];
	return {
		calls,
		seedCalls,
		setCalls,
		publish: {
			seedDraft: (next) => {
				calls.push(next);
				seedCalls.push(next);
				seedDraft(next);
			},
			setDraft: (next) => {
				calls.push(next);
				setCalls.push(next);
				setDraft(next);
			}
		}
	};
}

/** Test probe: deep local + array Add via publishDeepLocalJson (effect body). */
export function probeSyncDeepLocalArrayAdd(
	setDraft: (next: Record<string, unknown>) => void,
	seedDraft: (next: Record<string, unknown>) => void = () => {}
): ProbeSyncDeepLocalResult {
	const { publish, calls, seedCalls, setCalls } = probePublish(setDraft, seedDraft);
	const state: DeepLocalDraftSyncState = { lastJson: '', primed: false };
	let local: Record<string, unknown> = {};
	publishDeepLocalJson(JSON.stringify(local), state, publish);
	local = { ...local, rows: [] };
	(local.rows as Record<string, unknown>[]).push({ weight: 12 });
	publishDeepLocalJson(JSON.stringify(local), state, publish);
	return { calls, seedCalls, setCalls };
}

/** Test probe: whole-value local assign via publishDeepLocalJson (effect body). */
export function probeSyncDeepLocalWholeAssign(
	setDraft: (next: Record<string, unknown>) => void,
	seedDraft: (next: Record<string, unknown>) => void = () => {}
): ProbeSyncDeepLocalResult {
	const { publish, calls, seedCalls, setCalls } = probePublish(setDraft, seedDraft);
	const state: DeepLocalDraftSyncState = { lastJson: '', primed: false };
	let local: Record<string, unknown> = {};
	publishDeepLocalJson(JSON.stringify(local), state, publish);
	local = { material: 'polvereDiVetro', weight: 1 };
	publishDeepLocalJson(JSON.stringify(local), state, publish);
	return { calls, seedCalls, setCalls };
}
