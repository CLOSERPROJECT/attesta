/**
 * Local operator identity for chrome eval `$operator` + submit stamp.
 * No kernel / Appwrite / Cerbos — harness supplies via getOperator().
 */
export type OperatorIdentity = {
	userId: string;
	/** Human-facing name for chrome + submit stamp (C2). */
	displayName: string;
	org: string;
	roles: string[];
};
