/**
 * Why: sjsf array Add (SimpleKeyedArray.push) and objectCtx.set mutate deep
 * `$state` form local in place and never call createForm's value setter, so
 * DraftSession / localStorage never see the row. Wire once from createConfigForm
 * so every deep change republishes to Draft.
 *
 * `$effect.root` — createConfigForm is called from StepStackSession.#liveForm
 * (not component init). JSON.stringify walks the proxy so Add / new keys
 * re-subscribe; `$state.snapshot` alone does not. `untrack` + lastJson gate —
 * setDraft must not re-enter the effect (effect_update_depth_exceeded).
 * Tests: wrap `$state` + sync in `$effect.root`, `flushSync`, then cleanup
 * (Svelte Vitest docs).
 */
import { flushSync, untrack } from 'svelte';
import { republishBoundValue } from './republish-form-value.js';

export function syncDeepLocalToDraft<T extends Record<string, unknown>>(
	getLocal: () => T,
	setDraft: (next: T) => void
): () => void {
	let lastJson = '';
	return $effect.root(() => {
		$effect(() => {
			// JSON.stringify walks the proxy graph so array push / new keys re-run;
			// $state.snapshot alone does not always subscribe to deep adds.
			const json = JSON.stringify(getLocal());
			if (json === lastJson) return;
			lastJson = json;
			untrack(() => {
				republishBoundValue(
					() => JSON.parse(json) as Record<string, unknown>,
					(next) => setDraft(next as T)
				);
			});
		});
	});
}

/** Test probe: deep local + array Add, asserting syncDeepLocalToDraft → setDraft. */
export function probeSyncDeepLocalArrayAdd(
	setDraft: (next: Record<string, unknown>) => void
): { calls: Record<string, unknown>[] } {
	const calls: Record<string, unknown>[] = [];
	let mutate = (): void => {};
	const cleanup = $effect.root(() => {
		let local = $state({} as Record<string, unknown>);
		syncDeepLocalToDraft(() => local, (next) => {
			calls.push(next);
			setDraft(next);
		});
		mutate = () => {
			local.polvereDiVetro = [];
			(local.polvereDiVetro as Record<string, unknown>[]).push({ weight: 12 });
		};
	});
	flushSync();
	mutate();
	flushSync();
	cleanup();
	return { calls };
}

/** Test probe: whole-value local assign (createConfigForm setter path) → setDraft via effect. */
export function probeSyncDeepLocalWholeAssign(
	setDraft: (next: Record<string, unknown>) => void
): { calls: Record<string, unknown>[] } {
	const calls: Record<string, unknown>[] = [];
	let assign = (): void => {};
	const cleanup = $effect.root(() => {
		let local = $state({} as Record<string, unknown>);
		syncDeepLocalToDraft(() => local, (next) => {
			calls.push(next);
			setDraft(next);
		});
		// Mimic createConfigForm setter: local = next only (effect writes Draft)
		assign = () => {
			local = { sourceLoadId: 'x', polvereDiVetro: [{ weight: 1 }] };
		};
	});
	flushSync();
	assign();
	flushSync();
	cleanup();
	return { calls };
}
