import { taskAddressKey, type TaskAddress } from '../task-address.js';
import { stayAftermath } from './commit-stay-recap.js';

/** Own Commit (or other local write) told to BoardSession.domainChanged. */
export type OwnDomainChange = {
	origin: 'own';
	/** Revision the Commit wrote; null when unknown. */
	revision: number | null;
	/** Committed Task. */
	addr: TaskAddress;
	/** stay: parks on the form; advance: board keeps its scope; leaving: scope changes or off-board. */
	follow: 'stay' | 'advance' | 'leaving';
	submitMany: boolean;
};

/** What happened to the Domain, as told to BoardSession.domainChanged. */
export type DomainChange = OwnDomainChange | { origin: 'foreign'; revision: number };

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
	/** Recaps that must refresh after this Commit (soft if ready; hard if missing/error). */
	recapRefresh: TaskAddress[];
	/** Refetch ready form bundles in place; open forms get a stale marker instead of a swap. */
	revalidateForms: boolean;
	/** Committed addresses in this batch: never marked stale (their own refresh covers them). */
	holdExempt: string[];
	/** Refetch ready recaps in place. */
	revalidateRecaps: boolean;
	/** Worklist hydrate keeps the prior result if the Load fails. */
	background: boolean;
};

export function planBoardRefresh(
	changes: DomainChange[],
	facts: BoardRefreshFacts
): BoardRefreshPlan {
	let timeline = false;
	let worklistOnly = false;
	let remountForms = false;
	let revalidateForms = false;
	let revalidateRecaps = false;
	let background = false;
	const holdExempt: string[] = [];
	const recaps = new Map<string, TaskAddress>();

	for (const change of changes) {
		if (change.origin === 'foreign') {
			timeline = true;
			if (facts.scopeKind === 'instance') {
				revalidateForms = true;
				revalidateRecaps = true;
				background = true;
			}
			continue;
		}

		holdExempt.push(taskAddressKey(change.addr));

		if (change.follow === 'stay') {
			timeline = true;
			remountForms = stayAftermath(change.submitMany).refreshForms || remountForms;
		} else if (change.follow === 'advance' && facts.hasView) {
			timeline = true;
		} else if (change.follow === 'advance' && facts.scopeKind === 'instance') {
			worklistOnly = true;
		}
		recaps.set(taskAddressKey(change.addr), change.addr);
		// Entity recaps on other Tasks may show Domain fields this Commit patched.
		revalidateRecaps = true;
	}

	if (remountForms) {
		revalidateForms = false;
	}

	return {
		timeline,
		worklistOnly: timeline ? false : worklistOnly,
		remountForms,
		recapRefresh: [...recaps.values()],
		revalidateForms,
		holdExempt: revalidateForms ? holdExempt : [],
		revalidateRecaps,
		background
	};
}
