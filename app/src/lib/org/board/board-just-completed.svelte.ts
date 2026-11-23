import { taskAddressKey, type TaskAddress } from '../task-address.js';
import {
	focusMatchesAddress,
	type BoardTaskFocus
} from './board-url.js';

/** Session-sticky just-completed pin: one active address under available-only. */
export type JustCompletedPin = {
	address: TaskAddress;
	submissionId?: string;
};

/**
 * Address-keyed just-completed pin for the Org board.
 * Replace-on-pin; single active. Process-wide default survives Play→instance remount.
 */
export class BoardJustCompleted {
	#pin = $state.raw<JustCompletedPin | null>(null);

	pin(address: TaskAddress, submissionId?: string): void {
		this.#pin =
			submissionId === undefined
				? { address }
				: { address, submissionId };
	}

	clear(): void {
		if (this.#pin == null) return;
		this.#pin = null;
	}

	has(addr: TaskAddress): boolean {
		const pin = this.#pin;
		if (!pin) return false;
		return taskAddressKey(pin.address) === taskAddressKey(addr);
	}

	get(): JustCompletedPin | null {
		return this.#pin;
	}

	/** Focus moved: drop pin when leaving the pinned task. */
	onFocusChange(
		next: BoardTaskFocus | null,
		_stream: string,
		_instance: string
	): void {
		const pin = this.#pin;
		if (!pin) return;
		if (next != null && focusMatchesAddress(next, pin.address)) return;
		this.clear();
	}
}

/** Process-wide so Play→instance remount keeps the just-completed pin. */
export const boardJustCompleted = new BoardJustCompleted();
