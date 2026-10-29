import type { LoadResult } from './types.js';

/** Org Domain Load — no run id. */
export type LoadDomainPort = {
	load(): Promise<LoadResult>;
};

export class LoadDomainError extends Error {
	readonly status: number;

	constructor(message: string, status: number) {
		super(message);
		this.name = 'LoadDomainError';
		this.status = status;
	}
}
