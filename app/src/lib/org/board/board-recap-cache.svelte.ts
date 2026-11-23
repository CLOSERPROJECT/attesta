import { createHttpTaskRecapPort } from '../http-task-recap-port.js';
import {
	taskAddressKey,
	wireJobInstanceId,
	type TaskAddress
} from '../task-address.js';
import {
	TaskRecapError,
	type TaskRecap,
	type TaskRecapPort
} from '../task-recap-port.js';

export type BoardRecapEntryStatus = 'idle' | 'loading' | 'ready' | 'error';

export type BoardRecapEntry = {
	key: string;
	address: TaskAddress;
	status: BoardRecapEntryStatus;
	items: TaskRecap[];
	error: string | null;
};

export type BoardRecapCacheOptions = {
	taskRecap?: TaskRecapPort;
	/** Parallel GET /api/org/task-recap cap after shell paint. */
	concurrency?: number;
};

const DEFAULT_CONCURRENCY = 5;

/**
 * Address-keyed task-recap cache with a bounded parallel preload queue.
 * Abort in-flight work on scope change / dispose. HTTP 404 / empty list → ready [].
 */
export class BoardRecapCache {
	readonly taskRecapPort: TaskRecapPort;
	readonly concurrency: number;

	/** Reactive snapshot of cache entries (reassigned, not deeply mutated). */
	entries = $state.raw<Record<string, BoardRecapEntry>>({});

	#scopeKey = '';
	#queue: TaskAddress[] = [];
	#queued = new Set<string>();
	#inflight = 0;
	#gen = 0;
	#controllers = new Map<string, AbortController>();
	/** Survives beginScope once (Play→instance soft-nav). */
	#carryReady: BoardRecapEntry[] = [];

	constructor(options: BoardRecapCacheOptions = {}) {
		this.taskRecapPort = options.taskRecap ?? createHttpTaskRecapPort();
		this.concurrency = Math.max(1, options.concurrency ?? DEFAULT_CONCURRENCY);
	}

	get(key: string): BoardRecapEntry | undefined {
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
		const next: Record<string, BoardRecapEntry> = {};
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
				items: [],
				error: null
			});
		}
		this.#pump();
	}

	/** Seed after Commit — skips network for this address. */
	putReady(addr: TaskAddress, items: TaskRecap[]): void {
		const key = taskAddressKey(addr);
		this.#controllers.get(key)?.abort();
		this.#controllers.delete(key);
		this.#queued.delete(key);
		this.#queue = this.#queue.filter((a) => taskAddressKey(a) !== key);
		const entry: BoardRecapEntry = {
			key,
			address: addr,
			status: 'ready',
			items,
			error: null
		};
		this.#setEntry(entry);
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

	#setEntry(entry: BoardRecapEntry): void {
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
			items: [],
			error: null
		});

		try {
			const list = await this.taskRecapPort.fetch({
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
				items: list.items,
				error: null
			});
		} catch (err) {
			if (gen !== this.#gen || ac.signal.aborted) return;
			if (err instanceof TaskRecapError && err.status === 404) {
				this.#setEntry({
					key,
					address: addr,
					status: 'ready',
					items: [],
					error: null
				});
				return;
			}
			const message = err instanceof Error ? err.message : String(err);
			this.#setEntry({
				key,
				address: addr,
				status: 'error',
				items: [],
				error: message
			});
		} finally {
			this.#controllers.delete(key);
			this.#inflight = Math.max(0, this.#inflight - 1);
			if (gen === this.#gen) this.#pump();
		}
	}
}

export function createBoardRecapCache(
	options: BoardRecapCacheOptions = {}
): BoardRecapCache {
	return new BoardRecapCache(options);
}

/** Process-wide board recap cache so soft-nav keeps ready entries. */
export const boardRecapCache = createBoardRecapCache();
