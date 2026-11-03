/** Client-only Draft bag — never part of Domain until Commit succeeds. */
export type DraftKey = string;

export function draftKey(taskPath: string, jobInstanceId: string): DraftKey {
	return `${taskPath}::${jobInstanceId || '_new'}`;
}

/**
 * Instance id passed to getDraft/setDraft/clearDraft.
 * Nested: `jobInstanceId::groupId::groupInstanceId` (provisional group → '').
 * Flat: jobInstanceId as-is (including '' for Start).
 */
export function draftBagInstanceKey(parts: {
	jobInstanceId: string;
	groupId?: string;
	groupInstanceId?: string;
}): string {
	const groupId = parts.groupId ?? '';
	if (!groupId) return parts.jobInstanceId;
	return `${parts.jobInstanceId}::${groupId}::${parts.groupInstanceId ?? ''}`;
}
