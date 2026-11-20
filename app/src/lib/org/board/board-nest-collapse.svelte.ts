import { createContext } from 'svelte';

/**
 * Board-scoped collapse-all signal for nest Collapsibles (steps/rows/details).
 * Bump epoch; children set open=false when epoch changes from 0.
 * Embeds stay on BoardEmbedOpen.closeAll — do not remount the tree (forms).
 */
export class BoardNestCollapse {
	epoch = $state(0);

	collapseAll(): void {
		this.epoch += 1;
	}
}

export const [getBoardNestCollapse, setBoardNestCollapse] =
	createContext<BoardNestCollapse>();
