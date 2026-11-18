import type { FormState } from '@sjsf/form';
import { createConfigForm, type CreateConfigFormResult } from './create-config-form.svelte.js';
import type { ChromeContext } from './chrome-context.js';
import {
	startLiveTaskChrome,
	type LiveTaskChromeHandle,
	type LiveTaskFormPorts
} from './live-task-chrome.svelte.js';
import type { TaskFormPlan } from './task-form-plan.js';

export type { LiveTaskFormPorts } from './live-task-chrome.svelte.js';

export type LiveTaskFormOptions = {
	/** Bun tests: skip sjsf createForm (chrome API still runs). */
	deferForm?: boolean;
};

/**
 * Live substep apply: ConfigForm when derived present + chrome (readonly / dataReport) settlement.
 * Override policy stays on ConfigForm — not reimplemented here.
 * Plan/schema swaps use reactive getters (sjsf); prefer updatePlan over dispose+recreate.
 */
export class LiveTaskForm {
	#plan = $state.raw<TaskFormPlan>(null!);
	#form: CreateConfigFormResult<Record<string, unknown>> | undefined;
	#chrome: LiveTaskChromeHandle;
	#ports: LiveTaskFormPorts;
	#options: LiveTaskFormOptions;
	#formInit: Promise<void> | undefined;

	constructor(
		plan: TaskFormPlan,
		ports: LiveTaskFormPorts,
		options: LiveTaskFormOptions = {}
	) {
		this.#plan = plan;
		this.#ports = ports;
		this.#options = options;
		this.#chrome = startLiveTaskChrome(() => this.#plan, ports);

		if (plan.surface === 'live' && !options.deferForm) {
			this.#mountForm();
		}
	}

	get plan(): TaskFormPlan {
		return this.#plan;
	}

	get form(): CreateConfigFormResult<Record<string, unknown>> | undefined {
		return this.#form;
	}

	whenFormReady = (): Promise<void> => this.#formInit ?? Promise.resolve();

	/**
	 * Swap plan/schemas/chrome in place. Recreates the sjsf form only when the
	 * derived vs non-derived createConfigForm branch must change.
	 */
	updatePlan = (next: TaskFormPlan): void => {
		const prevHadDerived = this.#plan.derived !== undefined;
		const nextHasDerived = next.derived !== undefined;
		this.#plan = next;

		if (!this.#form) {
			if (next.surface === 'live' && !this.#options.deferForm) {
				this.#mountForm();
			}
			return;
		}

		if (prevHadDerived !== nextHasDerived) {
			this.#disposeFormOnly();
			if (next.surface === 'live' && !this.#options.deferForm) {
				this.#mountForm();
			}
			return;
		}

		if (this.#form.configForm && next.derived !== undefined) {
			this.#form.configForm.replaceDerived(next.derived);
		}
	};

	#mountForm = (): void => {
		const ports = this.#ports;
		const form = createConfigForm(
			() => this.#plan.schemas,
			ports.getValue,
			ports.setValue,
			{
				...(this.#plan.derived !== undefined ? { derived: this.#plan.derived } : {}),
				getDomain: ports.getDomain,
				getCatalogs: ports.getCatalogs,
				onSubmit: ports.onSubmit
					? (value) => {
							void ports.onSubmit?.(
								this.#plan.sealSubmission(value as Record<string, unknown>)
							);
						}
					: undefined,
				onSubmitError: ports.onSubmitError
			}
		);
		(form as FormState<Record<string, unknown>> & { liveTaskForm?: LiveTaskForm }).liveTaskForm =
			this;
		this.#form = form;
	};

	#disposeFormOnly = (): void => {
		this.#form?.disposeLocalDraftSync?.();
		this.#form = undefined;
	};

	chromeContext = (): ChromeContext => ({
		get: (id) => this.chromeDisplay(id)
	});

	chromeDisplay = (id: string) => this.#chrome.chromeDisplay(id);

	whenSettled = async (): Promise<void> => {
		await this.#formInit;
		await this.#chrome.whenSettled();
		await this.form?.configForm?.whenSettled();
	};

	dispose = (): void => {
		this.#chrome.dispose();
		this.#disposeFormOnly();
	};
}
