/** Header trail chrome: draft mark is never gated by showAll. */
export function timelineSubstepTrailVisible(parts: {
	showStatusChip: boolean;
	roleBadgeCount: number;
	showDraft: boolean;
}): boolean {
	return parts.showStatusChip || parts.roleBadgeCount > 0 || parts.showDraft;
}
