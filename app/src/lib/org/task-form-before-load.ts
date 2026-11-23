import { orgRouteCoordsFromPath } from './nav-place.js';
import { applyUrlSeed, takeSeedQueryParams } from './seed-query.js';
import { normalizeTaskAddress } from './task-address.js';
import type { TaskFormMount } from './task-form-mount.svelte.js';
import type { Worklist } from './types.js';
import type { TaskDeepLinkTarget } from './worklist-nav.js';

export type TaskFormBeforeLoadDeps = {
	mount: TaskFormMount;
	session: {
		result: { worklist: Worklist } | null;
		getDraft: (
			taskPath: string,
			draftInstanceKey: string
		) => Record<string, unknown> | null;
		seedDraft: (
			taskPath: string,
			draftInstanceKey: string,
			value: Record<string, unknown>
		) => void;
	};
	/**
	 * On gate deny — return `navigate(...)` so caller can `throw` it
	 * (sv-router hooks pattern).
	 */
	navigateRedirect: (
		target: Exclude<TaskDeepLinkTarget, { kind: 'task' }>
	) => Promise<unknown>;
	/** Optional href for `?seed.*` (defaults to window.location.href in browser). */
	href?: string;
	replaceHref?: (href: string) => void;
};

/**
 * Shared Task route beforeLoad: URL seed → gate → mount.load.
 * If Worklist is already present, gate first (deny → throw navigateRedirect;
 * blocked → stay without form fetch). If not, start fetch and return; page
 * applies gate later via navigate.
 */
export async function runTaskFormBeforeLoad(
	pathname: string,
	deps: TaskFormBeforeLoadDeps
): Promise<void> {
	const coords = orgRouteCoordsFromPath({ pathname });
	const addr = normalizeTaskAddress(coords);

	const href =
		deps.href ??
		(typeof window !== 'undefined' ? window.location.href : '');
	if (href) {
		const { seed, strippedHref } = takeSeedQueryParams(href);
		if (Object.keys(seed).length > 0) {
			applyUrlSeed(seed, addr, deps.session);
			const replace =
				deps.replaceHref ??
				((next: string) => {
					if (typeof window !== 'undefined') {
						window.history.replaceState(window.history.state, '', next);
					}
				});
			replace(strippedHref);
		}
	}

	const worklist = deps.session.result?.worklist;

	if (deps.session.result) {
		const gate = deps.mount.applyGate(worklist, addr);
		if (gate.kind === 'denied') {
			throw deps.navigateRedirect(gate.redirect);
		}
		if (gate.kind === 'blocked') {
			return;
		}
		await deps.mount.load(addr);
		return;
	}

	void deps.mount.load(addr);
}
