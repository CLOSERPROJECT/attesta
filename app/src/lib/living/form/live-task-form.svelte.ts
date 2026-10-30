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
 * Live substep apply: ConfigForm when derived present + chrome (valuePrint / recap / dataReport) settlement.
 * Override policy stays on ConfigForm — not reimplemented here.
 */
export class LiveTaskForm {
	readonly plan: TaskFormPlan;
	#form: CreateConfigFormResult<Record<string, unknown>> | undefined;
	readonly #chrome: LiveTaskChromeHandle;
	#formInit: Promise<void> | undefined;

	constructor(
		plan: TaskFormPlan,
		ports: LiveTaskFormPorts,
		options: LiveTaskFormOptions = {}
	) {
		this.plan = plan;
		this.#chrome = startLiveTaskChrome(plan, ports);

		if (plan.surface === 'live' && !options.deferForm) {
			this.#mountForm(plan, ports);
		}
	}

	get form(): CreateConfigFormResult<Record<string, unknown>> | undefined {
		return this.#form;
	}

	whenFormReady = (): Promise<void> => this.#formInit ?? Promise.resolve();

	#mountForm = (plan: TaskFormPlan, ports: LiveTaskFormPorts): void => {
		const form = createConfigForm(plan.schemas, ports.getValue, ports.setValue, {
			...(plan.derived !== undefined ? { derived: plan.derived } : {}),
			getDomain: ports.getDomain,
			onSubmit: ports.onSubmit
				? (value) => {
						void ports.onSubmit?.(plan.sealSubmission(value as Record<string, unknown>));
					}
				: undefined,
			onSubmitError: ports.onSubmitError
		});
		(form as FormState<Record<string, unknown>> & { liveTaskForm?: LiveTaskForm }).liveTaskForm =
			this;
		this.#form = form;
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
		this.#form?.disposeLocalDraftSync?.();
		this.#form = undefined;
	};
}
