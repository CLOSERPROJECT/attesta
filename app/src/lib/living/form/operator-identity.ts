/**
 * Local operator identity for role-gated Substep availability.
 * No kernel / Appwrite / Cerbos — harness supplies via getOperator().
 */
export type OperatorIdentity = {
	userId: string;
	/** Human-facing name for chrome + submit stamp (C2). */
	displayName: string;
	org: string;
	roles: string[];
};

/**
 * Omit or empty `roles` ⇒ pass. Else operator must hold ≥1 listed role.
 */
export function operatorPassesRoleGate(
	operator: OperatorIdentity,
	roles: readonly string[] | undefined
): boolean {
	if (roles === undefined || roles.length === 0) return true;
	const held = new Set(operator.roles);
	return roles.some((role) => held.has(role));
}
