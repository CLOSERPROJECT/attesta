import { orgRouteCoordsFromPath } from './nav-place.js';
import { normalizeTaskAddress } from './task-address.js';
import type { TaskFormMount } from './task-form-mount.svelte.js';
import type { Worklist } from './types.js';
import type { TaskDeepLinkTarget } from './worklist-nav.js';

export type TaskFormBeforeLoadDeps = {
	mount: TaskFormMount;
	session: { result: { worklist: Worklist } | null };
	/**
	 * On gate deny — return `navigate(...)` so caller can `throw` it
	 * (sv-router hooks pattern).
	 */
	navigateRedirect: (
		target: Exclude<TaskDeepLinkTarget, { kind: 'task' }>
	) => Promise<unknown>;
};

/**
 * Shared Task route beforeLoad: parse address → mount.load.
 * If Worklist is already present, gate first (deny → throw navigateRedirect).
 * If not, start fetch and return; page applies gate later via navigate.
 */
export async function runTaskFormBeforeLoad(
	pathname: string,
	deps: TaskFormBeforeLoadDeps
): Promise<void> {
	const coords = orgRouteCoordsFromPath({ pathname });
	const addr = normalizeTaskAddress(coords);
	const worklist = deps.session.result?.worklist;

	if (deps.session.result) {
		const gate = deps.mount.applyGate(worklist, addr);
		if (gate.kind === 'denied') {
			throw deps.navigateRedirect(gate.redirect);
		}
		await deps.mount.load(addr);
		return;
	}

	void deps.mount.load(addr);
}
