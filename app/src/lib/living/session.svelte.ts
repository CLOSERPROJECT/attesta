import { createHttpLoadDomainPort } from './http-load-port.js';
import type { LoadDomainPort } from './load-port.js';
import {
	orderedJobIds,
	type JobConfig,
	type LoadResult
} from './types.js';

export type LivingSessionOptions = {
	load?: LoadDomainPort;
	/** When false, skip background hydrate (tests inject result). Default true. */
	hydrate?: boolean;
};

/**
 * Living org session: Domain + Active Config via LoadDomainPort.
 * Commit / Task UI arrive in later tickets — this holds Load + Jobs chrome state.
 */
export class LivingSession {
	readonly load: LoadDomainPort;
	result = $state.raw<LoadResult | null>(null);
	loading = $state(false);
	error = $state.raw<string | null>(null);

	constructor(options: LivingSessionOptions = {}) {
		this.load = options.load ?? createHttpLoadDomainPort();
		if (options.hydrate !== false) {
			void this.hydrate();
		}
	}

	jobIds = $derived(orderedJobIds(this.result?.config));

	job = (jobId: string): JobConfig | null => {
		const jobs = this.result?.config.jobs;
		if (!jobs) return null;
		return jobs[jobId] ?? null;
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
}
