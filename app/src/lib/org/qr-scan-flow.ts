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
	rememberOpenSeed?: (taskPath: string, seed: Record<string, string>) => void;
};

export type QrScanFlowDeps = {
	resolve: QrResolvePort['resolve'];
	stream: string;
	instance?: string;
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
		result = await deps.resolve(trimmed, {
			stream: deps.stream,
			instance: deps.instance
		});
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
	const outcome = interpretQrResolve(result, deps.stream);
	if (outcome.kind === 'error') {
		deps.toast(outcome.message);
		return 'toasted';
	}
	const prior = deps.drafts.getDraft(outcome.taskPath, outcome.draftInstanceKey) ?? {};
	const stringSeed: Record<string, string> = {};
	for (const [k, v] of Object.entries(outcome.seed)) {
		if (!k || v == null) continue;
		stringSeed[k] = typeof v === 'string' ? v : String(v);
	}
	if (Object.keys(stringSeed).length > 0) {
		deps.drafts.rememberOpenSeed?.(outcome.taskPath, stringSeed);
	}
	deps.drafts.setDraft(outcome.taskPath, outcome.draftInstanceKey, {
		...prior,
		...outcome.seed
	});
	const fromInstance = (deps.instance ?? '').trim();
	const toInstance =
		outcome.target.kind === 'task' ? outcome.target.instance.trim() : '';
	const switched =
		Boolean(fromInstance) && Boolean(toInstance) && fromInstance !== toInstance;
	if (switched) {
		deps.toast(m.scan_switched_instance());
	}
	deps.navigate(outcome.target);
	return 'navigated';
}
