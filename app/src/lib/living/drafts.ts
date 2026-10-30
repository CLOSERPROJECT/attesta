/** Client-only Draft bag — never part of Domain until Commit succeeds. */
export type DraftKey = string;

export function draftKey(taskPath: string, jobInstanceId: string): DraftKey {
	return `${taskPath}::${jobInstanceId || '_new'}`;
}
