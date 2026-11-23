import type { TaskRecap } from './task-recap-port.js';

/**
 * Format a submission ISO timestamp with the runtime locale
 * (browser i18n when `locale` is omitted).
 */
export function formatSubmissionTimestamp(
	raw: string,
	locale?: string | string[]
): string {
	if (!raw) return '—';
	const date = new Date(raw);
	if (Number.isNaN(date.getTime())) return raw;
	return new Intl.DateTimeFormat(locale, {
		dateStyle: 'medium',
		timeStyle: 'short'
	}).format(date);
}

/** Stable fact-key union: walk items in order, append unseen keys. */
export function unionFactKeys(items: readonly TaskRecap[]): string[] {
	const keys: string[] = [];
	const seen = new Set<string>();
	for (const item of items) {
		for (const key of Object.keys(item.facts)) {
			if (key.length === 0 || seen.has(key)) continue;
			seen.add(key);
			keys.push(key);
		}
	}
	return keys;
}

/** First non-empty authored title for `key`, else the key itself. */
export function factTitle(items: readonly TaskRecap[], key: string): string {
	for (const item of items) {
		const title = item.titles?.[key];
		if (title) return title;
	}
	return key;
}
