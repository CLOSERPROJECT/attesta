import type { TaskFormBundle } from '../form/task-form-bundle.js';
import { createHttpTaskFormPort } from '../http-task-form-port.js';
import {
	taskAddressKey,
	wireJobInstanceId,
	type TaskAddress
} from '../task-address.js';
import type { TaskFormPort } from '../task-form-port.js';

export type BoardFormEntryStatus = 'idle' | 'loading' | 'ready' | 'error';

export type BoardFormEntry = {
	key: string;
	address: TaskAddress;
	status: BoardFormEntryStatus;
	bundle: TaskFormBundle | null;
	error: string | null;
	/**
	 * Form draft identity for this address. Advances when a Domain Commit refresh
	 * lands a new ready bundle (or on beginFreshDraft remount-only) so the embed
	 * remounts against a cleared draft bag. Pre-submit Form↔Recap toggles do not
	 * change this.
	 */
	draftGeneration: number;
};

export type BoardFormCacheOptions = {
	taskForm?: TaskFormPort;
	/** Parallel GET /api/org/task-form cap after shell paint. */
	concurrency?: number;
};

const DEFAULT_CONCURRENCY = 5;

/**
 * Address-keyed form bundle cache with a bounded parallel preload queue.
 * Abort in-flight work on scope change / dispose. Put ready bundles from
 * Commit `nextTaskFormBundle` to avoid a double-fetch.
 */
export class BoardFormCache {
	readonly taskFormPort: TaskFormPort;
	readonly concurrency: number;

	/** Reactive snapshot of cache entries (reassigned, not deeply mutated). */
	entries = $state.raw<Record<string, BoardFormEntry>>({});

	#scopeKey = '';
	#queue: TaskAddress[] = [];
	#queued = new Set<string>();
	#inflight = 0;
	#gen = 0;
	#controllers = new Map<string, AbortController>();
	/** Survives beginScope once (Play→instance soft-nav). */
	#carryReady: BoardFormEntry[] = [];
	/** Keys that should advance draftGeneration when their next fetch becomes ready. */
	#bumpDraftOnReady = new Set<string>();

	constructor(options: BoardFormCacheOptions = {}) {
		this.taskFormPort = options.taskForm ?? createHttpTaskFormPort();
		this.concurrency = Math.max(1, options.concurrency ?? DEFAULT_CONCURRENCY);
	}

	get(key: string): BoardFormEntry | undefined {
		return this.entries[key];
	}

	/** Abort prior scope, clear queue; re-apply staged putReady carry. */
	beginScope(scopeKey: string): void {
		if (this.#scopeKey === scopeKey) return;
		this.#abortAll();
		this.#scopeKey = scopeKey;
		this.#queue = [];
		this.#queued.clear();
		this.#bumpDraftOnReady.clear();
		const carried = this.#carryReady;
		this.#carryReady = [];
		const next: Record<string, BoardFormEntry> = {};
		for (const entry of carried) {
			next[entry.key] = entry;
		}
		this.entries = next;
	}

	/** Enqueue addresses not yet ready/loading; pump up to concurrency. */
	enqueue(addrs: TaskAddress[]): void {
		for (const addr of addrs) {
			const key = taskAddressKey(addr);
			const existing = this.entries[key];
			if (
				existing?.status === 'ready' ||
				existing?.status === 'loading' ||
				existing?.status === 'error'
			) {
				continue;
			}
			if (this.#queued.has(key)) continue;
			this.#queued.add(key);
			this.#queue.push(addr);
			this.#setEntry({
				key,
				address: addr,
				status: 'idle',
				bundle: null,
				error: null,
				draftGeneration: 0
			});
		}
		this.#pump();
	}

	/** Seed from Commit follow-through — skips network for this address. */
	putReady(addr: TaskAddress, bundle: TaskFormBundle): void {
		const key = taskAddressKey(addr);
		this.#controllers.get(key)?.abort();
		this.#controllers.delete(key);
		this.#queued.delete(key);
		this.#queue = this.#queue.filter((a) => taskAddressKey(a) !== key);
		const priorGen = this.entries[key]?.draftGeneration ?? 0;
		const entry: BoardFormEntry = {
			key,
			address: addr,
			status: 'ready',
			bundle,
			error: null,
			draftGeneration: priorGen
		};
		this.#setEntry(entry);
		// Carry across the next beginScope (soft-nav remount).
		this.#carryReady = this.#carryReady.filter((e) => e.key !== key);
		this.#carryReady.push(entry);
	}

	/**
	 * Remount-only: advance draftGeneration while keeping the ready bundle.
	 * Prefer BoardSession.afterDomainCommit `{ refreshForms: true }` (stay
	 * submitMany) — that path drops stale bindings and refetches.
	 * No-op when the address is not cached (nothing to re-bind).
	 */
	beginFreshDraft(addr: TaskAddress): void {
		const key = taskAddressKey(addr);
		const existing = this.entries[key];
		if (!existing) return;
		const entry: BoardFormEntry = {
			...existing,
			draftGeneration: existing.draftGeneration + 1
		};
		this.#setEntry(entry);
		if (entry.status === 'ready' && entry.bundle) {
			this.#carryReady = this.#carryReady.filter((e) => e.key !== key);
			this.#carryReady.push(entry);
		}
	}

	/**
	 * Stay submitMany after Domain Commit: every cached form bundle in the current
	 * scope may hold stale bindings.domain. Drop ready/error/loading entries to
	 * loading (skeleton, not stale paint), refetch via TaskFormPort, and advance
	 * draftGeneration when each becomes ready so Form remounts on a cleared draft
	 * + fresh chrome bindings.
	 */
	refreshAllAfterDomainCommit(): void {
		const targets = Object.values(this.entries).filter(
			(e) =>
				e.status === 'ready' || e.status === 'error' || e.status === 'loading'
		);
		for (const entry of targets) {
			this.#bumpDraftOnReady.add(entry.key);
			this.#restartFetch(entry.address);
		}
	}

	invalidate(key: string): void {
		this.#controllers.get(key)?.abort();
		this.#controllers.delete(key);
		this.#queued.delete(key);
		this.#bumpDraftOnReady.delete(key);
		this.#queue = this.#queue.filter((a) => taskAddressKey(a) !== key);
		const next = { ...this.entries };
		delete next[key];
		this.entries = next;
	}

	dispose(): void {
		this.#abortAll();
		this.#scopeKey = '';
		this.#queue = [];
		this.#queued.clear();
		this.#bumpDraftOnReady.clear();
		this.entries = {};
	}

	#abortAll(): void {
		this.#gen += 1;
		for (const ac of this.#controllers.values()) {
			ac.abort();
		}
		this.#controllers.clear();
		this.#inflight = 0;
	}

	#setEntry(entry: BoardFormEntry): void {
		this.entries = { ...this.entries, [entry.key]: entry };
	}

	/** Abort in-flight work for addr and start a fresh fetch (skeleton immediately). */
	#restartFetch(addr: TaskAddress): void {
		const key = taskAddressKey(addr);
		this.#controllers.get(key)?.abort();
		this.#controllers.delete(key);
		this.#queued.delete(key);
		this.#queue = this.#queue.filter((a) => taskAddressKey(a) !== key);
		this.#carryReady = this.#carryReady.filter((e) => e.key !== key);
		const draftGeneration = this.entries[key]?.draftGeneration ?? 0;
		this.#setEntry({
			key,
			address: addr,
			status: 'loading',
			bundle: null,
			error: null,
			draftGeneration
		});
		if (this.#inflight < this.concurrency) {
			void this.#fetchOne(addr, key);
			return;
		}
		this.#queued.add(key);
		this.#queue.push(addr);
	}

	#pump(): void {
		while (this.#inflight < this.concurrency && this.#queue.length > 0) {
			const addr = this.#queue.shift();
			if (!addr) break;
			const key = taskAddressKey(addr);
			this.#queued.delete(key);
			const existing = this.entries[key];
			if (existing?.status === 'ready') continue;
			void this.#fetchOne(addr, key);
		}
	}

	async #fetchOne(addr: TaskAddress, key: string): Promise<void> {
		const gen = this.#gen;
		const ac = new AbortController();
		this.#controllers.set(key, ac);
		this.#inflight += 1;
		const draftGeneration = this.entries[key]?.draftGeneration ?? 0;
		this.#setEntry({
			key,
			address: addr,
			status: 'loading',
			bundle: null,
			error: null,
			draftGeneration
		});

		try {
			const bundle = await this.taskFormPort.fetch({
				stream: addr.stream,
				instance: addr.instance || undefined,
				jobId: addr.jobId,
				taskId: addr.taskId,
				jobInstanceId: wireJobInstanceId(addr)
			});
			if (gen !== this.#gen || ac.signal.aborted) return;
			const priorGen = this.entries[key]?.draftGeneration ?? draftGeneration;
			const bump = this.#bumpDraftOnReady.has(key);
			if (bump) this.#bumpDraftOnReady.delete(key);
			const entry: BoardFormEntry = {
				key,
				address: addr,
				status: 'ready',
				bundle,
				error: null,
				draftGeneration: bump ? priorGen + 1 : priorGen
			};
			this.#setEntry(entry);
			if (bump) {
				this.#carryReady = this.#carryReady.filter((e) => e.key !== key);
				this.#carryReady.push(entry);
			}
		} catch (err) {
			if (gen !== this.#gen || ac.signal.aborted) return;
			this.#bumpDraftOnReady.delete(key);
			const message = err instanceof Error ? err.message : String(err);
			this.#setEntry({
				key,
				address: addr,
				status: 'error',
				bundle: null,
				error: message,
				draftGeneration: this.entries[key]?.draftGeneration ?? draftGeneration
			});
		} finally {
			if (this.#controllers.get(key) === ac) this.#controllers.delete(key);
			this.#inflight = Math.max(0, this.#inflight - 1);
			if (gen === this.#gen) this.#pump();
		}
	}
}

export function createBoardFormCache(
	options: BoardFormCacheOptions = {}
): BoardFormCache {
	return new BoardFormCache(options);
}

/** Process-wide board cache so Play→instance soft-nav keeps ready bundles. */
export const boardFormCache = createBoardFormCache();
