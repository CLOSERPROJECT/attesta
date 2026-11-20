import { createContext } from 'svelte';

import type { CommitOutcome } from '../commit-port.js';
import type { TaskAddress } from '../task-address.js';
import type { BoardFormEntry } from './board-form-cache.svelte.js';
import type { BoardTaskFocus } from './board-url.js';

export type BoardEmbedApi = {
	getEntry: (addr: TaskAddress) => BoardFormEntry | undefined;
	focus: BoardTaskFocus | null;
	isFocused: (addr: TaskAddress) => boolean;
	setFocus: (addr: TaskAddress) => void;
	/** Embed panel open; default true until user collapses. */
	isEmbedOpen: (addr: TaskAddress) => boolean;
	setEmbedOpen: (addr: TaskAddress, open: boolean) => void;
	toggleEmbedOpen: (addr: TaskAddress) => boolean;
	stayedFor: (addr: TaskAddress) => CommitOutcome | null;
	onCommitted: (addr: TaskAddress, outcome: CommitOutcome) => void;
	onContinue: (addr: TaskAddress) => void;
	play?: boolean;
};

export const [getBoardEmbed, setBoardEmbed] = createContext<BoardEmbedApi>();
