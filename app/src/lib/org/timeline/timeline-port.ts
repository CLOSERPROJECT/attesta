import type { TimelineView } from './types.js';

export class TimelineLoadError extends Error {
	readonly status: number;

	constructor(message: string, status: number) {
		super(message);
		this.name = 'TimelineLoadError';
		this.status = status;
	}
}

/** Load query: stream alone (boot) or stream+instance. Never boot=1. */
export type TimelineLoadQuery =
	| { stream: string }
	| { stream: string; instance: string };

export type TimelinePort = {
	load(query: TimelineLoadQuery): Promise<TimelineView>;
};
