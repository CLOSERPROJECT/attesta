import type { CommitOutcome } from './commit-port.js';
import {
	normalizeTaskAddress,
	taskAddressKey,
	type TaskAddress
} from './task-address.js';
import type { Worklist } from './types.js';
import {
	resolveCommitFollowUp,
	resolveContinueNavigation,
	type PostCommitTarget
} from './worklist-nav.js';

/** Post-Commit follow-through: stay on form, or leave to a server target. */
export type CommitFollowThrough =
	| { kind: 'stay' }
	| { kind: 'navigate'; target: PostCommitTarget };

/** Continue after stay: same-address reload, or navigate out. */
export type ContinueFollowThrough =
	| { kind: 'reload' }
	| { kind: 'navigate'; target: PostCommitTarget };

export type CommitFollowThroughDeps = {
	clearLinksStay: () => void;
	navigate: (target: PostCommitTarget) => void;
	reload: (worklist?: Worklist | null) => Promise<unknown>;
};

/**
 * Commit outcome + current Task address → stay | navigate (ADR-0006).
 * Destination from server navigation only; stay never inferred from links.
 */
export function decideCommitFollowThrough(
	outcome: Pick<
		CommitOutcome,
		'navigation' | 'jobInstanceId' | 'nextTaskFormBundle'
	>,
	current: Pick<TaskAddress, 'jobId'>
): CommitFollowThrough {
	const followUp = resolveCommitFollowUp(
		outcome.navigation,
		current.jobId,
		outcome.jobInstanceId,
		outcome.nextTaskFormBundle
	);
	if (followUp.kind === 'stay') return { kind: 'stay' };
	return { kind: 'navigate', target: followUp };
}

/**
 * Continue after stay + current Task address → reload | navigate.
 * Same-address Task Continue reloads the form instead of navigating.
 */
export function decideContinueFollowThrough(
	outcome: Pick<
		CommitOutcome,
		'continueNavigation' | 'jobInstanceId' | 'nextTaskFormBundle'
	>,
	current: TaskAddress
): ContinueFollowThrough {
	const target = resolveContinueNavigation(
		outcome.continueNavigation,
		current.jobId,
		outcome.jobInstanceId,
		outcome.nextTaskFormBundle
	);
	const sameTaskAddress =
		target.kind === 'task' &&
		taskAddressKey(normalizeTaskAddress(target)) === taskAddressKey(current);
	if (sameTaskAddress) return { kind: 'reload' };
	return { kind: 'navigate', target };
}

/** Leave after Commit: drop links then navigate (stay is page UI only). */
export function applyCommitFollowThrough(
	decision: Extract<CommitFollowThrough, { kind: 'navigate' }>,
	deps: Pick<CommitFollowThroughDeps, 'clearLinksStay' | 'navigate'>
): void {
	deps.clearLinksStay();
	deps.navigate(decision.target);
}

/** Continue after stay UI cleared: drop links then reload or navigate. */
export async function applyContinueFollowThrough(
	decision: ContinueFollowThrough,
	deps: CommitFollowThroughDeps,
	worklist?: Worklist | null
): Promise<void> {
	deps.clearLinksStay();
	if (decision.kind === 'reload') {
		await deps.reload(worklist);
		return;
	}
	deps.navigate(decision.target);
}
