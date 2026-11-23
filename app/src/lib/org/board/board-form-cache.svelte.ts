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
			if (existing?.status === 'ready' || existing?.status === 'loading') {
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
				error: null
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
		const entry: BoardFormEntry = {
			key,
			address: addr,
			status: 'ready',
			bundle,
			error: null
		};
		this.#setEntry(entry);
		// Carry across the next beginScope (soft-nav remount).
		this.#carryReady = this.#carryReady.filter((e) => e.key !== key);
		this.#carryReady.push(entry);
	}

	invalidate(key: string): void {
		this.#controllers.get(key)?.abort();
		this.#controllers.delete(key);
		this.#queued.delete(key);
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
		this.#setEntry({
			key,
			address: addr,
			status: 'loading',
			bundle: null,
			error: null
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
			this.#setEntry({
				key,
				address: addr,
				status: 'ready',
				bundle,
				error: null
			});
		} catch (err) {
			if (gen !== this.#gen || ac.signal.aborted) return;
			const message = err instanceof Error ? err.message : String(err);
			this.#setEntry({
				key,
				address: addr,
				status: 'error',
				bundle: null,
				error: message
			});
		} finally {
			this.#controllers.delete(key);
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
