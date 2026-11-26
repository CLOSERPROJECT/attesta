import type { TimelineRoleBadge } from './types.js';

/** Visible pill text: label when present, else id. */
export function timelineRoleBadgeLabel(badge: TimelineRoleBadge): string {
	const label = badge.label?.trim();
	return label || badge.id;
}

/** Role pills when chrome.roleBadges (API order preserved by caller). */
export function timelineRoleBadgesVisible(
	roleBadges: boolean,
	badges: TimelineRoleBadge[] | undefined
): TimelineRoleBadge[] {
	if (!roleBadges || !badges?.length) return [];
	return badges;
}
