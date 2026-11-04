import type {
	CommitDomainPort,
	CommitOutcome,
	CommitRequest,
	HttpLinkItem
} from './commit-port.js';
import { createHttpCommitDomainPort } from './http-commit-port.js';
import { draftKey, type DraftEntry } from './drafts.js';
import {
	browserStorage,
	livingDraftStorageKey,
	loadDirtyDraftsForIdentity,
	removeStoredDraft,
	writeStoredDraft,
	type StorageLike
} from './draft-storage.js';
import type { CloseGroupPort, CloseGroupRequest } from './group-close-port.js';
import { createHttpCloseGroupPort } from './http-group-close-port.js';
import { createHttpLoadDomainPort } from './http-load-port.js';
import { createHttpTaskFormPort } from './http-task-form-port.js';
import type { LoadDomainPort } from './load-port.js';
import type { TaskFormPort, TaskFormRequest } from './task-form-port.js';
import type { TaskFormBundle } from './form/task-form-bundle.js';
import {
	operatorFromLoad,
	type OperatorIdentity
} from './form/operator-identity.js';
import { commitHasDownloadLinks } from './worklist-nav.js';
import { orderedJobIdsFromWorklist, type LoadResult, type WorklistJob } from './types.js';

/** Debounce for dirty Draft localStorage writes. */
export const DEFAULT_DRAFT_DEBOUNCE_MS = 300;

export type DraftPersistClock = {
	setTimeout: (fn: () => void, ms: number) => unknown;
	clearTimeout: (id: unknown) => void;
};

const defaultClock: DraftPersistClock = {
	setTimeout: (fn, ms) => globalThis.setTimeout(fn, ms),
	clearTimeout: (id) => globalThis.clearTimeout(id as ReturnType<typeof setTimeout>)
};

export type LivingSessionOptions = {
	load?: LoadDomainPort;
	commit?: CommitDomainPort;
	close?: CloseGroupPort;
	taskForm?: TaskFormPort;
	/** When false, skip background hydrate (tests inject result). Default true. */
	hydrate?: boolean;
	/** Injected storage (browser localStorage + memory fake). Default browser. */
	storage?: StorageLike | null;
	debounceMs?: number;
	clock?: DraftPersistClock;
	/** When false, skip beforeunload / pagehide flush listeners. Default true in browser. */
	flushOnHide?: boolean;
};

type DraftIdentity = { org: string; userId: string };

export class LivingSession {
	readonly load: LoadDomainPort;
	readonly commitPort: CommitDomainPort;
	readonly closePort: CloseGroupPort;
	readonly taskFormPort: TaskFormPort;
	readonly #storage: StorageLike | null;
	readonly #debounceMs: number;
	readonly #clock: DraftPersistClock;
	readonly #pendingTimers = new Map<string, unknown>();
	#identity: DraftIdentity | null = null;
	#hideCleanup: (() => void) | null = null;

	result = $state.raw<LoadResult | null>(null);
	loading = $state(false);
	committing = $state(false);
	error = $state.raw<string | null>(null);
	commitError = $state.raw<string | null>(null);
	/** Last Commit HttpPort links (FormPlan strip can wait for ticket 14). */
	lastHttpLinks = $state.raw<HttpLinkItem[]>([]);
	lastHttpErrors = $state.raw<string[]>([]);
	/**
	 * Set synchronously with Commit downloads so Task deep-link $effect
	 * does not redirect before the route can enter stay-for-links.
	 */
	linksStayActive = $state(false);
	drafts = $state.raw<Record<string, DraftEntry>>({});

	constructor(options: LivingSessionOptions = {}) {
		this.load = options.load ?? createHttpLoadDomainPort();
		this.commitPort = options.commit ?? createHttpCommitDomainPort();
		this.closePort = options.close ?? createHttpCloseGroupPort();
		this.taskFormPort = options.taskForm ?? createHttpTaskFormPort();
		this.#storage =
			options.storage !== undefined ? options.storage : browserStorage();
		this.#debounceMs = options.debounceMs ?? DEFAULT_DRAFT_DEBOUNCE_MS;
		this.#clock = options.clock ?? defaultClock;
		if (options.hydrate !== false) {
			void this.hydrate();
		}
		const attachHide =
			options.flushOnHide !== false &&
			typeof window !== 'undefined' &&
			this.#storage != null;
		if (attachHide) {
			const onHide = () => this.flushPendingDrafts();
			const onVis = () => {
				if (document.visibilityState === 'hidden') onHide();
			};
			window.addEventListener('beforeunload', onHide);
			window.addEventListener('pagehide', onHide);
			document.addEventListener('visibilitychange', onVis);
			this.#hideCleanup = () => {
				window.removeEventListener('beforeunload', onHide);
				window.removeEventListener('pagehide', onHide);
				document.removeEventListener('visibilitychange', onVis);
			};
		}
	}

	/** Test / teardown: drop hide listeners. */
	dispose = (): void => {
		this.#hideCleanup?.();
		this.#hideCleanup = null;
		this.#cancelAllPending();
	};

	jobIds = $derived(orderedJobIdsFromWorklist(this.result?.worklist));

	/** Chrome `$operator` from Load projection (session Gate → LoadResult). */
	operator = (): OperatorIdentity | undefined => operatorFromLoad(this.result);

	worklistJob = (jobId: string): WorklistJob | null => {
		const jobs = this.result?.worklist.jobs;
		if (!jobs?.length) return null;
		return jobs.find((row) => row.jobId === jobId) ?? null;
	};

	getDraft = (taskPath: string, jobInstanceId: string): Record<string, unknown> | null => {
		return this.drafts[draftKey(taskPath, jobInstanceId)]?.value ?? null;
	};

	/** Dirty operator/QR write — counts for marks/persist until clearDraft. */
	setDraft = (
		taskPath: string,
		jobInstanceId: string,
		value: Record<string, unknown>
	): void => {
		const key = draftKey(taskPath, jobInstanceId);
		this.drafts = { ...this.drafts, [key]: { value, dirty: true } };
		this.#schedulePersist(key, value);
	};

	/**
	 * Non-dirty seed write. Does not count for marks/persist.
	 * If the key is already dirty, dirty sticks (no undirty-by-seed).
	 */
	seedDraft = (
		taskPath: string,
		jobInstanceId: string,
		value: Record<string, unknown>
	): void => {
		const key = draftKey(taskPath, jobInstanceId);
		const prior = this.drafts[key];
		this.drafts = {
			...this.drafts,
			[key]: { value, dirty: prior?.dirty ?? false }
		};
	};

	hasDirtyDraft = (taskPath: string, draftBagInstanceKey: string): boolean => {
		return this.drafts[draftKey(taskPath, draftBagInstanceKey)]?.dirty === true;
	};

	clearDraft = (taskPath: string, jobInstanceId: string): void => {
		const key = draftKey(taskPath, jobInstanceId);
		this.#cancelPersist(key);
		if (key in this.drafts) {
			const next = { ...this.drafts };
			delete next[key];
			this.drafts = next;
		}
		const id = this.#identity;
		if (id) {
			removeStoredDraft(
				this.#storage,
				livingDraftStorageKey(id.org, id.userId, key)
			);
		}
	};

	/** Force pending debounced writes to storage (page hide / tests). */
	flushPendingDrafts = (): void => {
		const pending = [...this.#pendingTimers.keys()];
		for (const key of pending) {
			this.#cancelPersist(key);
			const entry = this.drafts[key];
			if (entry?.dirty) {
				this.#persistNow(key, entry.value);
			}
		}
	};

	hydrate = async (): Promise<void> => {
		this.loading = true;
		this.error = null;
		try {
			this.result = await this.load.load();
			this.#syncIdentityFromResult(this.result);
		} catch (err) {
			this.result = null;
			this.error = err instanceof Error ? err.message : String(err);
		} finally {
			this.loading = false;
		}
	};

	/** Addressed Task form hydrate; does not mutate home LoadResult bundles. */
	fetchTaskForm = (req: TaskFormRequest): Promise<TaskFormBundle> => {
		return this.taskFormPort.fetch(req);
	};

	/**
	 * Re-Load Worklist without wiping client Drafts.
	 * Used after Commit denial so chrome matches server truth.
	 * Does not rehydrate from storage (in-memory dirty wins).
	 */
	refreshKeepDrafts = async (): Promise<void> => {
		this.loading = true;
		try {
			this.result = await this.load.load();
			this.error = null;
			// Identity change still resets bag; same-identity refresh does not rehydrate.
			this.#syncIdentityFromResult(this.result, { rehydrateOnlyOnChange: true });
		} catch (err) {
			// Keep prior result; surface Load failure separately from Commit denial.
			this.error = err instanceof Error ? err.message : String(err);
		} finally {
			this.loading = false;
		}
	};

	/**
	 * Commit sealed value. On success: replace Worklist from response, clear Draft
	 * keyed by the request id (provisional '' → `_new`), return outcome for
	 * post-Commit routing. On failure: keep Draft, re-Load, surface error.
	 */
	commit = async (req: CommitRequest): Promise<CommitOutcome | null> => {
		if (this.committing) return null;
		this.committing = true;
		this.commitError = null;
		try {
			const { draftBagInstanceKey: bagKey, ...wire } = req;
			const committed = await this.commitPort.commit(wire);
			const prior = this.result;
			if (prior) {
				this.result = {
					...prior,
					org: committed.org || prior.org,
					worklist: committed.worklist
				};
			} else {
				await this.refreshKeepDrafts();
				const loaded = this.result;
				if (loaded) {
					this.result = {
						...loaded,
						worklist: committed.worklist
					};
				}
			}
			this.lastHttpLinks = committed.links ?? [];
			this.lastHttpErrors = committed.httpErrors ?? [];
			this.linksStayActive = commitHasDownloadLinks(this.lastHttpLinks);
			this.clearDraft(req.taskPath, bagKey ?? req.jobInstanceId);
			return {
				jobInstanceId: committed.jobInstanceId,
				worklist: committed.worklist,
				next: committed.next,
				nextTaskFormBundle: committed.nextTaskFormBundle ?? null
			};
		} catch (err) {
			this.commitError = err instanceof Error ? err.message : String(err);
			this.lastHttpLinks = [];
			this.lastHttpErrors = [];
			this.linksStayActive = false;
			await this.refreshKeepDrafts();
			return null;
		} finally {
			this.committing = false;
		}
	};

	/** Clear stay-for-links before navigating away (Continue / Start another / Close). */
	clearLinksStay = (): void => {
		this.linksStayActive = false;
		this.lastHttpLinks = [];
		this.lastHttpErrors = [];
	};

	/**
	 * Close a Task group. On success: replace Worklist from response, return
	 * Commit-shaped outcome for post-Close routing. On failure: re-Load, surface error.
	 */
	closeGroup = async (req: CloseGroupRequest): Promise<CommitOutcome | null> => {
		this.committing = true;
		this.commitError = null;
		try {
			const closed = await this.closePort.close(req);
			const prior = this.result;
			if (prior) {
				this.result = {
					...prior,
					org: closed.org || prior.org,
					worklist: closed.worklist
				};
			} else {
				await this.refreshKeepDrafts();
				const loaded = this.result;
				if (loaded) {
					this.result = {
						...loaded,
						worklist: closed.worklist
					};
				}
			}
			this.lastHttpLinks = [];
			this.lastHttpErrors = [];
			this.linksStayActive = false;
			return {
				jobInstanceId: closed.jobInstanceId,
				worklist: closed.worklist,
				next: closed.next,
				nextTaskFormBundle: closed.nextTaskFormBundle ?? null
			};
		} catch (err) {
			this.commitError = err instanceof Error ? err.message : String(err);
			await this.refreshKeepDrafts();
			return null;
		} finally {
			this.committing = false;
		}
	};

	#syncIdentityFromResult = (
		result: LoadResult | null,
		opts: { rehydrateOnlyOnChange?: boolean } = {}
	): void => {
		if (!result?.org || !result.userId) return;
		const next: DraftIdentity = { org: result.org, userId: result.userId };
		const prev = this.#identity;
		const changed =
			prev == null || prev.org !== next.org || prev.userId !== next.userId;
		if (!changed) {
			// Same identity: never rebuild bag from storage on refreshKeepDrafts.
			if (opts.rehydrateOnlyOnChange) return;
			return;
		}
		if (prev) {
			this.flushPendingDrafts();
			this.drafts = {};
			this.#cancelAllPending();
		}
		this.#identity = next;
		this.#rehydrateFromStorage(next, { keepMemoryDirty: prev == null });
		if (prev == null) {
			// Dirty writes that landed before identity can now persist.
			for (const [key, entry] of Object.entries(this.drafts)) {
				if (entry.dirty) this.#schedulePersist(key, entry.value);
			}
		}
	};

	#rehydrateFromStorage = (
		id: DraftIdentity,
		opts: { keepMemoryDirty: boolean }
	): void => {
		const loaded = loadDirtyDraftsForIdentity(this.#storage, id.org, id.userId);
		const next: Record<string, DraftEntry> = opts.keepMemoryDirty
			? { ...this.drafts }
			: {};
		for (const [key, value] of Object.entries(loaded)) {
			const prior = next[key];
			if (prior?.dirty) continue; // in-memory dirty wins
			next[key] = { value, dirty: true };
		}
		this.drafts = next;
	};

	#schedulePersist = (key: string, value: Record<string, unknown>): void => {
		if (!this.#identity || !this.#storage) return;
		this.#cancelPersist(key);
		if (this.#debounceMs <= 0) {
			this.#persistNow(key, value);
			return;
		}
		this.#pendingTimers.set(
			key,
			this.#clock.setTimeout(() => {
				this.#pendingTimers.delete(key);
				this.#persistNow(key, value);
			}, this.#debounceMs)
		);
	};

	#persistNow = (key: string, value: Record<string, unknown>): void => {
		const id = this.#identity;
		if (!id || !this.#storage) return;
		writeStoredDraft(
			this.#storage,
			livingDraftStorageKey(id.org, id.userId, key),
			value
		);
	};

	#cancelPersist = (key: string): void => {
		const timer = this.#pendingTimers.get(key);
		if (timer == null) return;
		this.#clock.clearTimeout(timer);
		this.#pendingTimers.delete(key);
	};

	#cancelAllPending = (): void => {
		for (const key of [...this.#pendingTimers.keys()]) {
			this.#cancelPersist(key);
		}
	};
}
