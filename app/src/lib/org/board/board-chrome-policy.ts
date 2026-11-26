/** Board projection + presentation chrome from stream status and the Show-all toggle. */
export type BoardChromePolicy = {
	projectAll: boolean;
	showAllFilterVisible: boolean;
	nestStatusChips: boolean;
	roleBadges: boolean;
	lockedFade: boolean;
	instanceStatusChip: boolean;
};

const activeActionable: BoardChromePolicy = {
	projectAll: false,
	showAllFilterVisible: true,
	nestStatusChips: true,
	roleBadges: false,
	lockedFade: true,
	instanceStatusChip: false
};

const activeShowAll: BoardChromePolicy = {
	projectAll: true,
	showAllFilterVisible: true,
	nestStatusChips: true,
	roleBadges: true,
	lockedFade: true,
	instanceStatusChip: false
};

const doneArchive: BoardChromePolicy = {
	projectAll: true,
	showAllFilterVisible: false,
	nestStatusChips: false,
	roleBadges: false,
	lockedFade: false,
	instanceStatusChip: true
};

/** Done = archive chrome; otherwise toggle picks actionable vs show-all. */
export function boardChromePolicy(
	status: string | undefined | null,
	showAllToggle: boolean
): BoardChromePolicy {
	if (status === 'done') return doneArchive;
	return showAllToggle ? activeShowAll : activeActionable;
}
