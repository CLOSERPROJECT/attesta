import { taskAddressKey, type TaskAddress } from '../task-address.js';
import { stayAftermath } from './commit-stay-recap.js';

/** What happened to the Domain, as told to BoardSession.domainChanged. */
export type DomainChange = {
	origin: 'own';
	/** Revision the Commit wrote; null when unknown. */
	revision: number | null;
	/** Committed Task. */
	addr: TaskAddress;
	/** stay: parks on the form; advance: board keeps its scope; leaving: scope changes or off-board. */
	follow: 'stay' | 'advance' | 'leaving';
	submitMany: boolean;
};

/** Board facts the policy needs. */
export type BoardRefreshFacts = {
	scopeKind: 'boot' | 'instance';
	hasView: boolean;
};

export type BoardRefreshPlan = {
	/** Reload the timeline (also hydrates the Worklist on instance scope). */
	timeline: boolean;
	/** Hydrate the Worklist without a timeline reload. */
	worklistOnly: boolean;
	/** Skeleton + refetch every cached form bundle (own submitMany stay). */
	remountForms: boolean;
	/** Recaps to invalidate and re-enqueue. */
	recapRefresh: TaskAddress[];
};

export function planBoardRefresh(
	changes: DomainChange[],
	facts: BoardRefreshFacts
): BoardRefreshPlan {
	let timeline = false;
	let worklistOnly = false;
	let remountForms = false;
	const recaps = new Map<string, TaskAddress>();

	for (const change of changes) {
		if (change.follow === 'stay') {
			timeline = true;
			remountForms = stayAftermath(change.submitMany).refreshForms || remountForms;
		} else if (change.follow === 'advance' && facts.hasView) {
			timeline = true;
		} else if (change.follow === 'advance' && facts.scopeKind === 'instance') {
			worklistOnly = true;
		}
		recaps.set(taskAddressKey(change.addr), change.addr);
	}

	return {
		timeline,
		worklistOnly: timeline ? false : worklistOnly,
		remountForms,
		recapRefresh: [...recaps.values()]
	};
}
