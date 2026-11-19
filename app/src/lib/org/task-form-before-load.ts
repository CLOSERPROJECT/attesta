import { orgRouteCoordsFromPath } from './nav-place.js';
import { normalizeTaskAddress } from './task-address.js';
import type { TaskFormMount } from './task-form-mount.svelte.js';
import type { Worklist } from './types.js';
import type { TaskDeepLinkTarget } from './worklist-nav.js';

export type WorklistSessionSnapshot = {
	result: { worklist: Worklist } | null;
	loading: boolean;
};

export type AwaitWorklistOptions = {
	/** Brief wait for layout hydrate to flip `loading` on cold deep-link. */
	settleMs?: number;
	pollMs?: number;
	maxMs?: number;
};

function sleep(ms: number): Promise<void> {
	return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * Wait for Worklist without calling hydrate (layout owns that).
 * If idle with no result, settle briefly for hydrate to start; then await `!loading`.
 */
export async function awaitOrgSessionWorklist(
	session: WorklistSessionSnapshot,
	options: AwaitWorklistOptions = {}
): Promise<Worklist | null | undefined> {
	const pollMs = options.pollMs ?? 16;
	const maxMs = options.maxMs ?? 10_000;
	const settleMs = options.settleMs ?? 50;
	const start = Date.now();

	if (!session.result && !session.loading) {
		const settleUntil = Date.now() + settleMs;
		while (!session.result && !session.loading && Date.now() < settleUntil) {
			await sleep(pollMs);
		}
	}

	while (session.loading && Date.now() - start < maxMs) {
		await sleep(pollMs);
	}

	return session.result?.worklist;
}

export type TaskFormBeforeLoadDeps = {
	mount: TaskFormMount;
	session: WorklistSessionSnapshot;
	/**
	 * On gate deny — return `navigate(...)` so caller can `throw` it
	 * (sv-router hooks pattern).
	 */
	navigateRedirect: (
		target: Exclude<TaskDeepLinkTarget, { kind: 'task' }>
	) => Promise<unknown>;
	awaitWorklist?: typeof awaitOrgSessionWorklist;
};

/** Shared Task route beforeLoad: parse address, await Worklist, mount.load, redirect on deny. */
export async function runTaskFormBeforeLoad(
	pathname: string,
	deps: TaskFormBeforeLoadDeps
): Promise<void> {
	const coords = orgRouteCoordsFromPath({ pathname });
	const addr = normalizeTaskAddress(coords);

	const awaitWorklist = deps.awaitWorklist ?? awaitOrgSessionWorklist;
	const worklist = await awaitWorklist(deps.session);

	const result = await deps.mount.load(addr, worklist);

	if (result.kind === 'denied') {
		throw deps.navigateRedirect(result.redirect);
	}
}
