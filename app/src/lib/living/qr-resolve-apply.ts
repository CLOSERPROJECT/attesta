import { draftBagInstanceKey } from './drafts.js';
import {
	QR_RESOLVE_FAILED,
	type QrNavigate,
	type QrResolveResult
} from './qr-resolve-port.js';
import {
	groupTaskPath,
	normalizeGroupInstanceId,
	PROVISIONAL_GROUP_INSTANCE_ID,
	taskPath
} from './types.js';
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

/** Draft bag coords matching TaskForm (provisional group → ''). */
export function qrDraftCoords(nav: QrNavigate): {
	taskPath: string;
	draftInstanceKey: string;
} {
	const groupId = nav.groupId ?? '';
	const provisionalSpawn =
		Boolean(groupId) && normalizeGroupInstanceId(nav.groupInstanceId ?? '') === '';
	const path = groupId
		? groupTaskPath(
				nav.jobId,
				groupId,
				provisionalSpawn
					? PROVISIONAL_GROUP_INSTANCE_ID
					: (nav.groupInstanceId as string),
				nav.taskId
			)
		: taskPath(nav.jobId, nav.taskId);
	return {
		taskPath: path,
		draftInstanceKey: draftBagInstanceKey({
			jobInstanceId: nav.jobInstanceId,
			groupId: groupId || undefined,
			groupInstanceId: provisionalSpawn ? '' : nav.groupInstanceId
		})
	};
}

export function qrNavigateTarget(nav: QrNavigate): PostCommitTarget {
	if (nav.groupId) {
		return {
			kind: 'task',
			jobId: nav.jobId,
			jobInstanceId: nav.jobInstanceId,
			groupId: nav.groupId,
			groupInstanceId: nav.groupInstanceId || PROVISIONAL_GROUP_INSTANCE_ID,
			taskId: nav.taskId
		};
	}
	return {
		kind: 'task',
		jobId: nav.jobId,
		jobInstanceId: nav.jobInstanceId,
		taskId: nav.taskId
	};
}

export function interpretQrResolve(result: QrResolveResult): QrApplyOutcome {
	if (result.error?.message) {
		return { kind: 'error', message: result.error.message };
	}
	const nav = result.navigate;
	if (!nav) {
		return { kind: 'error', message: QR_RESOLVE_FAILED };
	}
	const coords = qrDraftCoords(nav);
	return {
		kind: 'navigate',
		target: qrNavigateTarget(nav),
		taskPath: coords.taskPath,
		draftInstanceKey: coords.draftInstanceKey,
		seed: result.seed ?? {}
	};
}

/** Cold `/app?entityId=` — capture raw then strip query so resolve runs once. */
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
