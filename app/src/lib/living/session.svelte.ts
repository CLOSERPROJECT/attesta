import type { CommitDomainPort, CommitRequest, HttpLinkItem } from './commit-port.js';
import { createHttpCommitDomainPort } from './http-commit-port.js';
import { draftKey } from './drafts.js';
import { createHttpLoadDomainPort } from './http-load-port.js';
import type { LoadDomainPort } from './load-port.js';
import { orderedJobIdsFromWorklist, type LoadResult, type WorklistJob } from './types.js';

export type LivingSessionOptions = {
	load?: LoadDomainPort;
	commit?: CommitDomainPort;
	/** When false, skip background hydrate (tests inject result). Default true. */
	hydrate?: boolean;
};

/** Living org session: Load Worklist + bundles; Commit sealed Tasks; client Drafts. */
export class LivingSession {
	readonly load: LoadDomainPort;
	readonly commitPort: CommitDomainPort;
	result = $state.raw<LoadResult | null>(null);
	loading = $state(false);
	committing = $state(false);
	error = $state.raw<string | null>(null);
	commitError = $state.raw<string | null>(null);
	/** Last Commit HttpPort links (FormPlan strip can wait for ticket 14). */
	lastHttpLinks = $state.raw<HttpLinkItem[]>([]);
	lastHttpErrors = $state.raw<string[]>([]);
	/** taskPath::jobInstanceId → sealed-shape draft object */
	drafts = $state.raw<Record<string, Record<string, unknown>>>({});

	constructor(options: LivingSessionOptions = {}) {
		this.load = options.load ?? createHttpLoadDomainPort();
		this.commitPort = options.commit ?? createHttpCommitDomainPort();
		if (options.hydrate !== false) {
			void this.hydrate();
		}
	}

	jobIds = $derived(orderedJobIdsFromWorklist(this.result?.worklist));

	worklistJob = (jobId: string): WorklistJob | null => {
		const jobs = this.result?.worklist.jobs;
		if (!jobs?.length) return null;
		return jobs.find((row) => row.jobId === jobId) ?? null;
	};

	getDraft = (taskPath: string, jobInstanceId: string): Record<string, unknown> | null => {
		return this.drafts[draftKey(taskPath, jobInstanceId)] ?? null;
	};

	setDraft = (
		taskPath: string,
		jobInstanceId: string,
		value: Record<string, unknown>
	): void => {
		const key = draftKey(taskPath, jobInstanceId);
		this.drafts = { ...this.drafts, [key]: value };
	};

	clearDraft = (taskPath: string, jobInstanceId: string): void => {
		const key = draftKey(taskPath, jobInstanceId);
		if (!(key in this.drafts)) return;
		const next = { ...this.drafts };
		delete next[key];
		this.drafts = next;
	};

	hydrate = async (): Promise<void> => {
		this.loading = true;
		this.error = null;
		try {
			this.result = await this.load.load();
		} catch (err) {
			this.result = null;
			this.error = err instanceof Error ? err.message : String(err);
		} finally {
			this.loading = false;
		}
	};

	/**
	 * Re-Load Worklist + bundles without wiping client Drafts.
	 * Used after Commit denial so chrome matches server truth.
	 */
	refreshKeepDrafts = async (): Promise<void> => {
		this.loading = true;
		try {
			this.result = await this.load.load();
			this.error = null;
		} catch (err) {
			// Keep prior result; surface Load failure separately from Commit denial.
			this.error = err instanceof Error ? err.message : String(err);
		} finally {
			this.loading = false;
		}
	};

	/**
	 * Commit sealed value. On success: replace Worklist + Task form bundles
	 * from response, clear Draft keyed by the request id (provisional '' → `_new`),
	 * return response jobInstanceId. On failure: keep Draft, re-Load, surface error.
	 */
	commit = async (req: CommitRequest): Promise<string | null> => {
		this.committing = true;
		this.commitError = null;
		try {
			const committed = await this.commitPort.commit(req);
			const prior = this.result;
			if (prior) {
				this.result = {
					...prior,
					org: committed.org || prior.org,
					worklist: committed.worklist,
					taskFormBundles: committed.taskFormBundles
				};
			} else {
				await this.refreshKeepDrafts();
				const loaded = this.result;
				if (loaded) {
					this.result = {
						...loaded,
						worklist: committed.worklist,
						taskFormBundles: committed.taskFormBundles
					};
				}
			}
			this.lastHttpLinks = committed.links ?? [];
			this.lastHttpErrors = committed.httpErrors ?? [];
			this.clearDraft(req.taskPath, req.jobInstanceId);
			return committed.jobInstanceId;
		} catch (err) {
			this.commitError = err instanceof Error ? err.message : String(err);
			this.lastHttpLinks = [];
			this.lastHttpErrors = [];
			await this.refreshKeepDrafts();
			return null;
		} finally {
			this.committing = false;
		}
	};
}
