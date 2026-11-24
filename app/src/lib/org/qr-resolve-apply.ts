import { draftBagInstanceKey } from './drafts.js';
import {
	qrResolveFailedMessage,
	type QrNavigate,
	type QrResolveResult
} from './qr-resolve-port.js';
import { taskPath } from './types.js';
import type { PostCommitTarget } from './worklist-nav.js';

export type QrApplyNavigate = {
	kind: 'navigate';
	target: PostCommitTarget;
	taskPath: string;
	draftInstanceKey: string;
	seed: Record<string, unknown>;
};

export type QrApplyOutcome =
	| QrApplyNavigate
	| { kind: 'error'; message: string };

function seedAsStrings(seed: Record<string, unknown> | undefined): Record<string, string> {
	if (!seed) return {};
	const out: Record<string, string> = {};
	for (const [k, v] of Object.entries(seed)) {
		if (!k || v == null) continue;
		out[k] = typeof v === 'string' ? v : String(v);
	}
	return out;
}

/** Draft bag coords matching TaskForm (provisional + seed → isolated key). */
export function qrDraftCoords(
	nav: QrNavigate,
	seed?: Record<string, unknown>
): {
	taskPath: string;
	draftInstanceKey: string;
} {
	const stringSeed = seedAsStrings(seed);
	return {
		taskPath: taskPath(nav.jobId, nav.taskId),
		draftInstanceKey: draftBagInstanceKey({
			jobInstanceId: nav.jobInstanceId,
			seed: !nav.jobInstanceId && Object.keys(stringSeed).length > 0 ? stringSeed : undefined
		})
	};
}

export function qrNavigateTarget(nav: QrNavigate, stream: string): PostCommitTarget {
	const instance = (nav.streamInstanceId ?? '').trim();
	return {
		kind: 'task',
		stream,
		instance,
		jobId: nav.jobId,
		jobInstanceId: nav.jobInstanceId,
		taskId: nav.taskId
	};
}

export function interpretQrResolve(
	result: QrResolveResult,
	stream: string
): QrApplyOutcome {
	if (result.error?.message) {
		return { kind: 'error', message: result.error.message };
	}
	const nav = result.navigate;
	if (!nav) {
		return { kind: 'error', message: qrResolveFailedMessage() };
	}
	const coords = qrDraftCoords(nav, result.seed);
	return {
		kind: 'navigate',
		target: qrNavigateTarget(nav, stream),
		taskPath: coords.taskPath,
		draftInstanceKey: coords.draftInstanceKey,
		seed: result.seed ?? {}
	};
}

/** Cold `/app/{stream}?entityId=` — capture raw then strip query so resolve runs once. */
export function takeColdEntityIdRaw(href: string): {
	raw: string | null;
	strippedHref: string;
} {
	let url: URL;
	try {
		url = new URL(href);
	} catch {
		return { raw: null, strippedHref: href };
	}
	const entityId = url.searchParams.get('entityId');
	if (entityId == null || entityId.trim() === '') {
		return { raw: null, strippedHref: href };
	}
	const raw = href;
	url.searchParams.delete('entityId');
	const qs = url.searchParams.toString();
	const strippedHref = `${url.pathname}${qs ? `?${qs}` : ''}${url.hash}`;
	return { raw, strippedHref };
}
