import {
	createForm,
	type FailureValidationResult,
	type FormState,
	type Schema,
	type UiSchemaRoot
} from '@sjsf/form';
import { createFormIdBuilder } from '@sjsf/form/id-builders/modern';
import { createFormMerger } from '@sjsf/form/mergers/modern';
import { resolver } from '@sjsf/form/resolvers/compat';
import { icons } from '@sjsf/lucide-icons';
import {
	ConfigForm,
	type ConfigFormDerivedEntry
} from './config-form.svelte.js';
import { syncDeepLocalToDraft } from './deep-local-draft-sync.svelte.js';
import { theme } from './form-theme.js';
import { translation } from './form-translation-it.js';
import { createAjvFormValidator } from './ajv-form-validator.js';
import {
	mergeSchemaDefaultsIntoFormData,
	resolveSchemaDateDefaultSentinels
} from './schema-defaults.js';

export type { ConfigForm, ConfigFormDerivedEntry, ConfigFormOptions } from './config-form.svelte.js';

export type ConfigFormSchemas = {
	schema: Schema;
	uiSchema: UiSchemaRoot;
};

export type CreateConfigFormOptions<T extends Record<string, unknown> = Record<string, unknown>> = {
	uiSchema?: UiSchemaRoot;
	onSubmit?: (value: T) => void | Promise<void>;
	/** Ajv rejected submit — createForm skips onSubmit; surface via banner. */
	onSubmitError?: (result: FailureValidationResult) => void;
	/** When set, wraps sjsf `createForm` with ConfigForm derived + override policy. */
	derived?: Record<string, ConfigFormDerivedEntry>;
	/** See ConfigFormOptions.getDomain — `$domain` for derived exprs. */
	getDomain?: () => unknown;
};

/**
 * Live-form value lives in deep `$state` so sjsf in-place mutations (array Add /
 * objectCtx.set) re-render. `syncDeepLocalToDraft` is the only Draft bridge —
 * deep in-place mutates and setter whole-value assign both go through the effect.
 * DraftSession stays `$state.raw` + whole-object setValue for structuredClone persist.
 * Optional `derived` uses ConfigForm as the value owner (ADR-0016 override set).
 * derivedLeaf operator edits call ConfigForm.writePath / restore only — not sjsf
 * bindable assign — so override marking stays on one path.
 */
/** FormState plus optional ConfigForm when `options.derived` was passed. */
export type CreateConfigFormResult<T extends Record<string, unknown>> = FormState<T> & {
	configForm?: ConfigForm<T>;
	liveTaskForm?: import('./live-task-form.svelte.js').LiveTaskForm;
	/** Stops syncDeepLocalToDraft `$effect.root` (call from LiveTaskForm.dispose). */
	disposeLocalDraftSync?: () => void;
};

export function createConfigForm<T extends Record<string, unknown>>(
	schemas: ConfigFormSchemas,
	getValue: () => T,
	setValue: (next: T) => void,
	options: CreateConfigFormOptions<T> = {}
): CreateConfigFormResult<T> {
	const uiSchema = options.uiSchema ?? schemas.uiSchema;
	// Resolve `$today` once so sjsf re-merge/reset never injects the sentinel.
	// createForm skips merge when `value` is bound — seed defaults here (NIR +
	// resolved date). Merger preserves operator / partial drafts in formData.
	const schema = resolveSchemaDateDefaultSentinels(schemas.schema);
	const seededValue = mergeSchemaDefaultsIntoFormData(schema, getValue());

	if (options.derived !== undefined) {
		const engine = new ConfigForm<T>({
			schema,
			uiSchema,
			initialValue: seededValue,
			derived: options.derived,
			getDomain: options.getDomain
		});
		const [, setBound] = engine.valueBinding();
		const disposeLocalDraftSync = syncDeepLocalToDraft(
			() => engine.value as Record<string, unknown>,
			(next) => setValue(next as T)
		);
		const form = createForm<T>({
			theme,
			schema,
			uiSchema,
			resolver,
			translation,
			merger: createFormMerger,
			validator: createAjvFormValidator,
			idBuilder: createFormIdBuilder,
			icons,
			fieldsValidationDebounceMs: 0,
			value: [() => engine.value, (next) => setBound(next)],
			onSubmit: (value) => {
				void options.onSubmit?.(value);
			},
			onSubmitError: (result) => {
				options.onSubmitError?.(result);
			}
		}) as CreateConfigFormResult<T>;
		form.configForm = engine;
		form.disposeLocalDraftSync = disposeLocalDraftSync;
		return form;
	}

	let local = $state($state.snapshot(seededValue) as T);
	const disposeLocalDraftSync = syncDeepLocalToDraft(
		() => local as Record<string, unknown>,
		(next) => setValue(next as T)
	);
	const form = createForm<T>({
		theme,
		schema,
		uiSchema,
		resolver,
		translation,
		merger: createFormMerger,
		validator: createAjvFormValidator,
		idBuilder: createFormIdBuilder,
		icons,
		fieldsValidationDebounceMs: 0,
		value: [
			() => local,
			(next) => {
				local = next;
			}
		],
		onSubmit: (value) => {
			void options.onSubmit?.(value);
		},
		onSubmitError: (result) => {
			options.onSubmitError?.(result);
		}
	}) as CreateConfigFormResult<T>;
	form.disposeLocalDraftSync = disposeLocalDraftSync;
	return form;
}
