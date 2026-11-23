import { taskAddressKey, type TaskAddress } from '../task-address.js';

/**
 * Address-keyed embed open state for the Org board.
 * Default closed on load (nests expand; substep bodies stay collapsed).
 * Only open keys are stored. setFocus / Continue / Play force-open — no effects.
 */
export class BoardEmbedOpen {
	/** Reactive: keys present here are expanded. */
	opened = $state.raw<Record<string, true>>({});

	isOpen(addr: TaskAddress): boolean {
		return this.opened[taskAddressKey(addr)] === true;
	}

	setOpen(addr: TaskAddress, open: boolean): void {
		const key = taskAddressKey(addr);
		if (open) {
			if (key in this.opened) return;
			this.opened = { ...this.opened, [key]: true };
			return;
		}
		if (!(key in this.opened)) return;
		const next = { ...this.opened };
		delete next[key];
		this.opened = next;
	}

	toggle(addr: TaskAddress): boolean {
		const next = !this.isOpen(addr);
		this.setOpen(addr, next);
		return next;
	}

	clear(): void {
		if (Object.keys(this.opened).length === 0) return;
		this.opened = {};
	}

	/** Collapse every listed address (ignore unknown keys). */
	closeAll(addrs: Iterable<TaskAddress>): void {
		const next = { ...this.opened };
		let changed = false;
		for (const addr of addrs) {
			const key = taskAddressKey(addr);
			if (!(key in next)) continue;
			delete next[key];
			changed = true;
		}
		if (changed) this.opened = next;
	}
}

/** Process-wide so Play→instance remount keeps user-expanded embeds. */
export const boardEmbedOpen = new BoardEmbedOpen();
