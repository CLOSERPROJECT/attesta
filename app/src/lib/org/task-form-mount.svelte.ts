import type { TaskFormBundle } from './form/task-form-bundle.js';
import { createHttpTaskFormPort } from './http-task-form-port.js';
import {
	taskAddressKey,
	wireJobInstanceId,
	type TaskAddress
} from './task-address.js';
import type { TaskFormPort } from './task-form-port.js';
import type { Worklist } from './types.js';
import {
	resolveTaskDeepLink,
	type TaskDeepLinkTarget
} from './worklist-nav.js';

export type TaskFormAddress = TaskAddress;

export type TaskFormMountStatus = 'idle' | 'loading' | 'ready' | 'denied' | 'error';

export type TaskFormLoadResult =
	| { kind: 'ready' }
	| { kind: 'error'; message: string }
	| { kind: 'aborted' };

export type TaskFormGateResult =
	| { kind: 'allow' }
	| { kind: 'denied'; redirect: Exclude<TaskDeepLinkTarget, { kind: 'task' }> };

export type TaskFormMountOptions = {
	taskForm?: TaskFormPort;
};

export const taskFormAddressKey = taskAddressKey;

/**
 * Address-owned Task form slot. Fetch via TaskFormPort; deep-link gate via
 * applyGate when Worklist is available. Does not hydrate or touch OrgSession.
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

	constructor(options: TaskFormMountOptions = {}) {
		this.taskFormPort = options.taskForm ?? createHttpTaskFormPort();
	}

	#bindAddress(addr: TaskFormAddress): void {
		this.#lastAddr = addr;
		this.addressKey = taskFormAddressKey(addr);
	}

	/**
	 * Deep-link gate for the bound address (or `addr` if provided).
	 * Allow = no-op; deny = abort in-flight fetch, denied + redirect.
	 * Worklist updates must call this only — never load/reload.
	 */
	applyGate = (
		worklist: Worklist | null | undefined,
		addr?: TaskFormAddress
	): TaskFormGateResult => {
		if (addr) this.#bindAddress(addr);
		const current = this.#lastAddr;
		if (!current) return { kind: 'allow' };

		const gate = resolveTaskDeepLink(
			current.stream,
			current.instance,
			current.jobId,
			current.jobInstanceId,
			current.taskId,
			current.provisionalJob,
			worklist
		);
		if (gate.kind === 'task') return { kind: 'allow' };

		this.#abort?.abort();
		this.#gen += 1;
		this.bundle = null;
		this.status = 'denied';
		this.redirect = gate;
		this.error = null;
		return { kind: 'denied', redirect: gate };
	};

	/** Address → TaskFormPort.fetch only. Does not gate. */
	load = async (addr: TaskFormAddress): Promise<TaskFormLoadResult> => {
		this.#bindAddress(addr);

		const gen = ++this.#gen;
		this.#abort?.abort();
		const ac = new AbortController();
		this.#abort = ac;

		this.redirect = null;
		this.error = null;
		this.status = 'loading';
		this.bundle = null;

		try {
			const next = await this.taskFormPort.fetch({
				stream: addr.stream,
				instance: addr.instance || undefined,
				jobId: addr.jobId,
				taskId: addr.taskId,
				jobInstanceId: wireJobInstanceId(addr)
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

	/** Same-address re-fetch (Continue). Pass post-Commit worklist to applyGate, not here. */
	reload = (): Promise<TaskFormLoadResult> => {
		const addr = this.#lastAddr;
		if (!addr) {
			return Promise.resolve({
				kind: 'error',
				message: 'no address to reload'
			});
		}
		return this.load(addr);
	};
}

export function createTaskFormMount(
	options: TaskFormMountOptions = {}
): TaskFormMount {
	return new TaskFormMount(options);
}

/** Process-wide slot for Task beforeLoad + page bind (not OrgSession). */
export const taskFormMount = createTaskFormMount();
