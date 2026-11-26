import type { InstancePassportsView } from './types.js';

export class PassportsLoadError extends Error {
	readonly status: number;

	constructor(message: string, status: number) {
		super(message);
		this.name = 'PassportsLoadError';
		this.status = status;
	}
}

export type PassportsLoadQuery = {
	stream: string;
	instance: string;
};

export type PassportsPort = {
	load(query: PassportsLoadQuery): Promise<InstancePassportsView>;
};
