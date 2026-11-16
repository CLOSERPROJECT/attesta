import type { HttpFollowUp, HttpLinkItem } from './http-port.js';

/** Chrome model for the per-Substep HTTP link strip. */
export type HttpLinkStripModel = {
	items: readonly HttpLinkItem[];
	followUp: HttpFollowUp;
};

/** Strip-owned dispatch; only click path into HttpFollowUp. */
export function activateHttpLinkItem(
	item: HttpLinkItem,
	followUp: HttpFollowUp
): void {
	if (item.action === 'link') {
		followUp.openTab(item.url);
		return;
	}
	followUp.download(item.url, item.filename ?? 'download');
}
