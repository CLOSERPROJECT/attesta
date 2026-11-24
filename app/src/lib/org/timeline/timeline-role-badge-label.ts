import type { TimelineRoleBadge } from './types.js';

/** Visible pill text: label when present, else id. */
export function timelineRoleBadgeLabel(badge: TimelineRoleBadge): string {
	const label = badge.label?.trim();
	return label || badge.id;
}

/** Role pills appear only in Show all (API order preserved by caller). */
export function timelineRoleBadgesVisible(
	showAll: boolean,
	badges: TimelineRoleBadge[] | undefined
): TimelineRoleBadge[] {
	if (!showAll || !badges?.length) return [];
	return badges;
}
