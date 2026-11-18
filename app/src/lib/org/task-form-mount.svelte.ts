import type { TaskFormBundle } from './form/task-form-bundle.js';
import { createHttpTaskFormPort } from './http-task-form-port.js';
import type { TaskFormPort } from './task-form-port.js';
import type { Worklist } from './types.js';
import {
	resolveTaskDeepLink,
	type TaskDeepLinkTarget
} from './worklist-nav.js';

/** Address for Task form hydrate (aligns with task.svelte route coords). */
export type TaskFormAddress = {
	jobId: string;
	taskId: string;
	/** Empty when provisionalJob; URL segment `new` is normalized before load. */
	jobInstanceId: string;
	provisionalJob: boolean;
};

export type TaskFormMountStatus = 'idle' | 'loading' | 'ready' | 'denied' | 'error';

export type TaskFormLoadResult =
	| { kind: 'ready' }
	| { kind: 'denied'; redirect: Exclude<TaskDeepLinkTarget, { kind: 'task' }> }
	| { kind: 'error'; message: string }
	| { kind: 'aborted' };

export type TaskFormMountOptions = {
	taskForm?: TaskFormPort;
};

export function taskFormAddressKey(addr: TaskFormAddress): string {
	const instance = addr.provisionalJob ? 'new' : addr.jobInstanceId;
	return `${addr.jobId}|${instance}|${addr.taskId}`;
}

/**
 * Address-owned Task form slot. Gates via resolveTaskDeepLink, fetches via
 * TaskFormPort. Does not hydrate Worklist or touch OrgSession.
 */
export class TaskFormMount {
	readonly taskFormPort: TaskFormPort;

	bundle = $state.raw<TaskFormBundle | null>(null);
	status = $state<TaskFormMountStatus>('idle');
	addressKey = $state.raw<string | null>(null);
	error = $state.raw<string | null>(null);
	redirect = $state.raw<Exclude<TaskDeepLinkTarget, { kind: 'task' }> | null>(null);

	#gen = 0;
	#abort: AbortController | null = null;
	#lastAddr: TaskFormAddress | null = null;
	#lastWorklist: Worklist | null | undefined = undefined;

	constructor(options: TaskFormMountOptions = {}) {
		this.taskFormPort = options.taskForm ?? createHttpTaskFormPort();
	}

	load = async (
		addr: TaskFormAddress,
		worklist: Worklist | null | undefined
	): Promise<TaskFormLoadResult> => {
		this.#lastAddr = addr;
		this.#lastWorklist = worklist;

		const gen = ++this.#gen;
		this.#abort?.abort();
		const ac = new AbortController();
		this.#abort = ac;

		this.addressKey = taskFormAddressKey(addr);
		this.redirect = null;
		this.error = null;

		const gate = resolveTaskDeepLink(
			addr.jobId,
			addr.jobInstanceId,
			addr.taskId,
			addr.provisionalJob,
			worklist
		);
		if (gate.kind !== 'task') {
			if (gen !== this.#gen) return { kind: 'aborted' };
			this.bundle = null;
			this.status = 'denied';
			this.redirect = gate;
			return { kind: 'denied', redirect: gate };
		}

		this.status = 'loading';
		this.bundle = null;

		try {
			const next = await this.taskFormPort.fetch({
				jobId: addr.jobId,
				taskId: addr.taskId,
				jobInstanceId: addr.provisionalJob ? 'new' : addr.jobInstanceId
			});
			if (gen !== this.#gen || ac.signal.aborted) return { kind: 'aborted' };
			this.bundle = next;
			this.status = 'ready';
			this.error = null;
			return { kind: 'ready' };
		} catch (err) {
			if (gen !== this.#gen || ac.signal.aborted) return { kind: 'aborted' };
			const message = err instanceof Error ? err.message : String(err);
			this.bundle = null;
			this.status = 'error';
			this.error = message;
			return { kind: 'error', message };
		}
	};

	/** Same-address re-fetch (Continue); pass worklist after Commit so the gate sees the snapshot. */
	reload = (
		worklist?: Worklist | null
	): Promise<TaskFormLoadResult> => {
		const addr = this.#lastAddr;
		if (!addr) {
			return Promise.resolve({
				kind: 'error',
				message: 'no address to reload'
			});
		}
		return this.load(
			addr,
			worklist !== undefined ? worklist : this.#lastWorklist
		);
	};
}

export function createTaskFormMount(
	options: TaskFormMountOptions = {}
): TaskFormMount {
	return new TaskFormMount(options);
}

/** Process-wide slot for Task beforeLoad + page bind (not OrgSession). */
export const taskFormMount = createTaskFormMount();
