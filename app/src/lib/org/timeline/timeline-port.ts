import type { StreamScope } from '../stream-scope.js';
import type { TimelineView } from './types.js';

export class TimelineLoadError extends Error {
	readonly status: number;

	constructor(message: string, status: number) {
		super(message);
		this.name = 'TimelineLoadError';
		this.status = status;
	}
}

export type TimelinePort = {
	load(scope: StreamScope & { instance: string }): Promise<TimelineView>;
};
