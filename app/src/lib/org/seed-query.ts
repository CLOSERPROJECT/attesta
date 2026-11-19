import { draftBagInstanceKey } from './drafts.js';
import { taskPath } from './types.js';

/**
 * §7 / #311: `?seed.<field>=…` on start Task (and Play) routes — capture then
 * strip so seedDraft runs once, mirroring cold `entityId`.
 */
export function takeSeedQueryParams(href: string): {
	seed: Record<string, string>;
	strippedHref: string;
} {
	let url: URL;
	try {
		url = new URL(href);
	} catch {
		return { seed: {}, strippedHref: href };
	}
	const seed: Record<string, string> = {};
	const toDelete: string[] = [];
	for (const [key, value] of url.searchParams.entries()) {
		if (!key.startsWith('seed.')) continue;
		const field = key.slice('seed.'.length);
		if (!field) continue;
		seed[field] = value;
		toDelete.push(key);
	}
	for (const key of toDelete) {
		url.searchParams.delete(key);
	}
	const qs = url.searchParams.toString();
	const strippedHref = `${url.pathname}${qs ? `?${qs}` : ''}${url.hash}`;
	return { seed, strippedHref };
}

export type SeedQueryDraftBag = {
	getDraft: (
		taskPath: string,
		draftInstanceKey: string
	) => Record<string, unknown> | null;
	seedDraft: (
		taskPath: string,
		draftInstanceKey: string,
		value: Record<string, unknown>
	) => void;
};

/** Merge seed fields into the Task draft bag (non-dirty). */
export function applyUrlSeed(
	seed: Record<string, string>,
	addr: { jobId: string; taskId: string; jobInstanceId: string },
	drafts: SeedQueryDraftBag
): void {
	if (Object.keys(seed).length === 0) return;
	const path = taskPath(addr.jobId, addr.taskId);
	const key = draftBagInstanceKey({ jobInstanceId: addr.jobInstanceId });
	const prior = drafts.getDraft(path, key) ?? {};
	drafts.seedDraft(path, key, { ...prior, ...seed });
}

/** Apply `?seed.*` once via seedDraft; returns stripped href when seed was present. */
export function applySeedQueryParams(
	href: string,
	drafts: SeedQueryDraftBag,
	addr: { jobId: string; taskId: string; jobInstanceId: string }
): string | null {
	const { seed, strippedHref } = takeSeedQueryParams(href);
	if (Object.keys(seed).length === 0) return null;
	applyUrlSeed(seed, addr, drafts);
	return strippedHref;
}
