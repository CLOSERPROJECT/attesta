import { createContext } from 'svelte';

/**
 * Board-scoped collapse-all signal for substep detail Collapsibles only.
 * Bump epoch; substeps set detailsOpen=false when epoch changes from 0.
 * Step/row nests stay expanded (initial board state). Embeds use BoardEmbedOpen.closeAll.
 */
export class BoardNestCollapse {
	epoch = $state(0);

	collapseAll(): void {
		this.epoch += 1;
	}
}

export const [getBoardNestCollapse, setBoardNestCollapse] =
	createContext<BoardNestCollapse>();
