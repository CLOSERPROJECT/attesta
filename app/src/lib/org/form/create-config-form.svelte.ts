import {
	createForm,
	type FailureValidationResult,
	type FormState,
	type Schema,
	type UiSchemaRoot,
	type ValidatorFactoryOptions
} from '@sjsf/form';
import { createFormIdBuilder } from '@sjsf/form/id-builders/modern';
import { createFormMerger } from '@sjsf/form/mergers/modern';
import { fromRecord } from '@sjsf/form/lib/resolver';
import { resolver } from '@sjsf/form/resolvers/compat';
import { icons } from '@sjsf/lucide-icons';
import * as m from '#lib/paraglide/messages.js';
import { getLocale } from '#lib/paraglide/runtime.js';
import {
	ConfigForm,
	type ConfigFormDerivedEntry
} from './config-form.svelte.js';
import { syncDeepLocalToDraft } from './deep-local-draft-sync.svelte.js';
import { theme } from './form-theme.js';
import { createAjvFormValidator } from './ajv-form-validator.js';
import { getSjsfTranslation } from './sjsf-translation.js';
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
	getCatalogs?: () => unknown;
	/**
	 * First sync tick — non-dirty bag populate (schema defaults / remount).
	 * Required so mount does not dirty via `setValue`.
	 */
	seedValue: (next: T) => void;
};

/**
 * Live-form value lives in deep `$state` so sjsf in-place mutations (array Add /
 * objectCtx.set) re-render. `syncDeepLocalToDraft` is the only Draft bridge —
 * first tick via `seedValue` (non-dirty), later edits via `setValue` (dirty).
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

/**
 * sjsf createForm reads `schema` / `uiSchema` reactively — pass a getter (or
 * object with getters) so plan/projection swaps update in place. Do not recreate
 * the form or `{#key}` remount (x0k anti-pattern: recreating form).
 */
export function createConfigForm<T extends Record<string, unknown>>(
	schemas: ConfigFormSchemas | (() => ConfigFormSchemas),
	getValue: () => T,
	setValue: (next: T) => void,
	options: CreateConfigFormOptions<T>
): CreateConfigFormResult<T> {
	const getSchemas = typeof schemas === 'function' ? schemas : () => schemas;
	// Cache resolved schema by raw identity so getters stay stable across reads.
	let cachedRaw: Schema | undefined;
	let cachedResolved: Schema | undefined;
	const readSchema = (): Schema => {
		const raw = getSchemas().schema;
		if (raw !== cachedRaw) {
			cachedRaw = raw;
			// Resolve `$today` so sjsf re-merge/reset never injects the sentinel.
			cachedResolved = resolveSchemaDateDefaultSentinels(raw);
		}
		return cachedResolved!;
	};
	const readUiSchema = (): UiSchemaRoot => options.uiSchema ?? getSchemas().uiSchema;
	// createForm skips merge when `value` is bound — seed defaults here (NIR +
	// resolved date). Merger preserves operator / partial drafts in formData.
	const seededValue = mergeSchemaDefaultsIntoFormData(readSchema(), getValue());
	const locale = getLocale();
	const translation = getSjsfTranslation(locale);
	const extraUiOptions = fromRecord({
		shadcn4ComboboxEmptyText: m.form_combobox_empty({}, { locale }),
		shadcn4DateRangePickerPlaceholder: m.form_date_range_placeholder({}, { locale })
	});
	const validator = (factoryOptions?: ValidatorFactoryOptions) =>
		createAjvFormValidator<T>(factoryOptions, locale);

	if (options.derived !== undefined) {
		const engine = new ConfigForm<T>({
			schema: readSchema(),
			uiSchema: readUiSchema(),
			initialValue: seededValue,
			derived: options.derived,
			getDomain: options.getDomain,
			getCatalogs: options.getCatalogs
		});
		const [, setBound] = engine.valueBinding();
		const disposeLocalDraftSync = syncDeepLocalToDraft(
			() => engine.value as Record<string, unknown>,
			{
				seedDraft: (next) => options.seedValue(next as T),
				setDraft: (next) => setValue(next as T)
			}
		);
		const form = createForm<T>({
			theme,
			get schema() {
				return readSchema();
			},
			get uiSchema() {
				return readUiSchema();
			},
			resolver,
			translation,
			extraUiOptions,
			merger: createFormMerger,
			validator,
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
		{
			seedDraft: (next) => options.seedValue(next as T),
			setDraft: (next) => setValue(next as T)
		}
	);
	const form = createForm<T>({
		theme,
		get schema() {
			return readSchema();
		},
		get uiSchema() {
			return readUiSchema();
		},
		resolver,
		translation,
		extraUiOptions,
		merger: createFormMerger,
		validator,
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
