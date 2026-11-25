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

/** Prefer human-facing userName; fall back to userId. */
export function formatSubmissionUser(
	item: { userName?: string; userId?: string }
): string {
	const name = item.userName?.trim();
	if (name) return name;
	const id = item.userId?.trim();
	return id || '—';
}

/** True when this recap row is the Commit submissionId. */
export function recapRowMatchesSubmissionId(
	row: { submissionId: string },
	submissionId: string | undefined | null
): boolean {
	if (submissionId == null || submissionId === '') return false;
	return row.submissionId === submissionId;
}
