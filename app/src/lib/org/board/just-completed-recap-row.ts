import { taskAddressKey, type TaskAddress } from '../task-address.js';
import type { JustCompletedPin } from './board-just-completed.svelte.js';

/**
 * Commit `submissionId` for this many-recap address, when the pin targets it.
 * Matches TaskRecapRow.submissionId (latest-touch Submission timestamp).
 */
export function pinSubmissionIdForAddress(
	pin: JustCompletedPin | null,
	addr: TaskAddress
): string | undefined {
	if (!pin?.submissionId) return undefined;
	if (taskAddressKey(pin.address) !== taskAddressKey(addr)) return undefined;
	return pin.submissionId;
}
