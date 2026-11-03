import type { HttpLinkItem } from './commit-port.js';

/** Normalize one Commit / task-recap link wire object; null when incomplete. */
export function parseHttpLinkItem(raw: unknown): HttpLinkItem | null {
	if (!raw || typeof raw !== 'object' || Array.isArray(raw)) return null;
	const rec = raw as Record<string, unknown>;
	const label = typeof rec.label === 'string' ? rec.label : '';
	const url = typeof rec.url === 'string' ? rec.url : '';
	const action = rec.action === 'link' || rec.action === 'download' ? rec.action : null;
	if (!label || !url || !action) return null;
	const item: HttpLinkItem = { label, url, action };
	if (typeof rec.filename === 'string' && rec.filename.length > 0) {
		item.filename = rec.filename;
	}
	return item;
}
