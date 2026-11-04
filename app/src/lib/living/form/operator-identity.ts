/**
 * Local operator identity for chrome eval `$operator` + submit stamp.
 * SPA: Load projects userId/displayName/org/roles; harness may supply via getOperator().
 */
export type OperatorIdentity = {
	userId: string;
	/** Human-facing name for chrome + submit stamp (C2). */
	displayName: string;
	org: string;
	roles: string[];
};

/** Map living LoadResult viewer fields → chrome `$operator` bag. */
export function operatorFromLoad(result: {
	org: string;
	userId?: string;
	displayName?: string;
	roles?: string[];
} | null | undefined): OperatorIdentity | undefined {
	if (result == null) return undefined;
	const userId = typeof result.userId === 'string' ? result.userId.trim() : '';
	const displayNameRaw =
		typeof result.displayName === 'string' ? result.displayName.trim() : '';
	const displayName = displayNameRaw || userId;
	if (!userId && !displayName) return undefined;
	return {
		userId: userId || displayName,
		displayName,
		org: result.org,
		roles: Array.isArray(result.roles) ? [...result.roles] : []
	};
}
