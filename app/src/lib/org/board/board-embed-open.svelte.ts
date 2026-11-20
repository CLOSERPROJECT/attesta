import { taskAddressKey, type TaskAddress } from '../task-address.js';

/**
 * Address-keyed embed open state for the Org board.
 * Default open (actionable embeds); only closed keys are stored.
 * setFocus / Continue force-open by clearing the closed flag — no effects.
 */
export class BoardEmbedOpen {
	/** Reactive: keys present here are collapsed. */
	closed = $state.raw<Record<string, true>>({});

	isOpen(addr: TaskAddress): boolean {
		return this.closed[taskAddressKey(addr)] !== true;
	}

	setOpen(addr: TaskAddress, open: boolean): void {
		const key = taskAddressKey(addr);
		if (open) {
			if (!(key in this.closed)) return;
			const next = { ...this.closed };
			delete next[key];
			this.closed = next;
			return;
		}
		if (key in this.closed) return;
		this.closed = { ...this.closed, [key]: true };
	}

	toggle(addr: TaskAddress): boolean {
		const next = !this.isOpen(addr);
		this.setOpen(addr, next);
		return next;
	}

	clear(): void {
		if (Object.keys(this.closed).length === 0) return;
		this.closed = {};
	}
}

/** Process-wide so Play→instance remount keeps user-collapsed embeds. */
export const boardEmbedOpen = new BoardEmbedOpen();
