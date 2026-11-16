import * as m from '#lib/paraglide/messages.js';

import { interpretQrResolve } from './qr-resolve-apply.js';
import type { QrResolvePort, QrResolveResult } from './qr-resolve-port.js';
import { qrResolveFailedMessage, QrResolveHttpError } from './qr-resolve-port.js';
import type { PostCommitTarget } from './worklist-nav.js';

export type QrScanDraftBag = {
	getDraft: (
		taskPath: string,
		draftInstanceKey: string
	) => Record<string, unknown> | null;
	setDraft: (
		taskPath: string,
		draftInstanceKey: string,
		value: Record<string, unknown>
	) => void;
};

export type QrScanFlowDeps = {
	resolve: QrResolvePort['resolve'];
	drafts: QrScanDraftBag;
	navigate: (target: PostCommitTarget) => void;
	toast: (message: string) => void;
};

export async function runQrScan(
	deps: QrScanFlowDeps,
	raw: string
): Promise<'navigated' | 'toasted'> {
	const trimmed = raw.trim();
	if (!trimmed) {
		deps.toast(m.empty_scan());
		return 'toasted';
	}
	let result: QrResolveResult;
	try {
		result = await deps.resolve(trimmed);
	} catch (err) {
		const message =
			err instanceof QrResolveHttpError
				? err.message
				: err instanceof Error
					? err.message
					: String(err);
		deps.toast(message || qrResolveFailedMessage());
		return 'toasted';
	}
	const outcome = interpretQrResolve(result);
	if (outcome.kind === 'error') {
		deps.toast(outcome.message);
		return 'toasted';
	}
	const prior = deps.drafts.getDraft(outcome.taskPath, outcome.draftInstanceKey) ?? {};
	deps.drafts.setDraft(outcome.taskPath, outcome.draftInstanceKey, {
		...prior,
		...outcome.seed
	});
	deps.navigate(outcome.target);
	return 'navigated';
}
