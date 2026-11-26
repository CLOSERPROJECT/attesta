import { provisionalSeedDraftKey } from '../drafts.js';
import { taskAddressKey, type TaskAddress } from '../task-address.js';

export type EmbedSeed = Record<string, string>;

/** Address key, plus optional `seed:…` suffix for provisional accordion isolation. */
export function embedBodyKey(addr: TaskAddress, seed?: EmbedSeed): string {
	const base = taskAddressKey(addr);
	if (!seed) return base;
	const keys = Object.keys(seed).filter(Boolean);
	if (keys.length === 0) return base;
	return `${base}|${provisionalSeedDraftKey(seed)}`;
}

function keyMatchesAddress(key: string, base: string): boolean {
	return key === base || key.startsWith(`${base}|`);
}

/**
 * Address-keyed embed open state for the Org board.
 * Default closed on load (nests expand; substep bodies stay collapsed).
 * Only open keys are stored. setFocus / Continue / Play force-open — no effects.
 * Provisional seeded Starts (`?seed.lotId=`) use distinct keys so one Apri lotto
 * accordion does not open every eligible lot row.
 */
export class BoardEmbedOpen {
	/** Reactive: keys present here are expanded. */
	opened = $state.raw<Record<string, true>>({});

	/**
	 * With seed: exact key. Without: address-only key or any seeded variant
	 * (form-cache hold / collapse-all callers that only know the Task address).
	 */
	isOpen(addr: TaskAddress, seed?: EmbedSeed): boolean {
		const opened = this.opened;
		if (seed && Object.keys(seed).filter(Boolean).length > 0) {
			return opened[embedBodyKey(addr, seed)] === true;
		}
		const base = taskAddressKey(addr);
		if (opened[base] === true) return true;
		for (const key of Object.keys(opened)) {
			if (keyMatchesAddress(key, base)) return true;
		}
		return false;
	}

	setOpen(addr: TaskAddress, open: boolean, seed?: EmbedSeed): void {
		if (!open) {
			if (seed && Object.keys(seed).filter(Boolean).length > 0) {
				const key = embedBodyKey(addr, seed);
				if (!(key in this.opened)) return;
				const next = { ...this.opened };
				delete next[key];
				this.opened = next;
				return;
			}
			// No seed → collapse every seeded/unseeded variant for this address.
			this.closeAll([addr]);
			return;
		}
		const key = embedBodyKey(addr, seed);
		if (key in this.opened) return;
		this.opened = { ...this.opened, [key]: true };
	}

	toggle(addr: TaskAddress, seed?: EmbedSeed): boolean {
		const next = !this.isOpen(addr, seed);
		this.setOpen(addr, next, seed);
		return next;
	}

	clear(): void {
		if (Object.keys(this.opened).length === 0) return;
		this.opened = {};
	}

	/** Collapse every listed address (including seeded provisional variants). */
	closeAll(addrs: Iterable<TaskAddress>): void {
		const bases = new Set<string>();
		for (const addr of addrs) {
			bases.add(taskAddressKey(addr));
		}
		if (bases.size === 0) return;
		const next = { ...this.opened };
		let changed = false;
		for (const key of Object.keys(next)) {
			let match = false;
			for (const base of bases) {
				if (keyMatchesAddress(key, base)) {
					match = true;
					break;
				}
			}
			if (!match) continue;
			delete next[key];
			changed = true;
		}
		if (changed) this.opened = next;
	}
}

/** Process-wide so Play→instance remount keeps user-expanded embeds. */
export const boardEmbedOpen = new BoardEmbedOpen();
