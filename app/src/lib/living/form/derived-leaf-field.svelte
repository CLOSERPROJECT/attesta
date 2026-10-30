<script lang="ts">
	import {
		getComponent,
		getFieldErrors,
		getFormContext,
		type ComponentProps
	} from '@sjsf/form';
	import { tryGetConfigFormContext } from './config-form-context.js';
	import { configPathKey } from './derived-keys.js';

	let {
		config,
		value = $bindable(),
		uiOption
	}: ComponentProps['derivedLeafField'] = $props();

	const ctx = getFormContext();
	const Template = $derived(getComponent(ctx, 'fieldTemplate', config));
	const errors = $derived(getFieldErrors(ctx, config.path));

	const pathKey = $derived(configPathKey(config.path));
	const formCtx = $derived(tryGetConfigFormContext());

	const overridable = $derived(formCtx?.isOverridable(pathKey) ?? false);
	const overridden = $derived(formCtx?.isOverridden(pathKey) ?? false);
	const formula = $derived(formCtx ? formCtx.computed(pathKey) : value);

	function displayFormula(): string {
		if (formula === undefined || formula === null) return '—';
		return String(formula);
	}

	function asLeafValue(next: unknown): number | string | undefined {
		if (typeof next === 'number' || typeof next === 'string') return next;
		if (next === undefined || next === null) return undefined;
		const n = Number(next);
		return Number.isFinite(n) ? n : undefined;
	}

	function startOverride(): void {
		if (!formCtx || !overridable) return;
		const next = asLeafValue(formula);
		formCtx.writePath(pathKey, next);
	}

	function onOverrideInput(raw: string): void {
		const trimmed = raw.trim();
		const next = trimmed === '' ? undefined : Number(trimmed);
		formCtx?.writePath(pathKey, next);
	}

	function restore(): void {
		formCtx?.restore(pathKey);
	}
</script>

<Template
	type="template"
	showTitle={false}
	useLabel={false}
	widgetType="numberWidget"
	{uiOption}
	value={undefined}
	{config}
	{errors}
>
	<div class="derived-leaf" data-testid="derived-leaf" data-path={pathKey}>
		{#if overridden}
			<input
				type="number"
				data-testid="derived-override"
				value={value === undefined || value === null ? '' : String(value)}
				oninput={(e) => onOverrideInput(e.currentTarget.value)}
			/>
			{#if overridable}
				<button type="button" data-testid="derived-restore" onclick={restore}>
					restore
				</button>
			{/if}
		{:else}
			<span data-testid="derived-formula">{displayFormula()}</span>
			{#if overridable}
				<button type="button" data-testid="derived-override-start" onclick={startOverride}>
					override
				</button>
			{/if}
		{/if}
	</div>
</Template>

<style>
	.derived-leaf {
		display: flex;
		align-items: center;
		gap: 0.35rem;
	}
	.derived-leaf input[type='number'] {
		width: 100%;
		max-width: 6rem;
		border: 1px solid #d4d4d8;
		border-radius: 0.25rem;
		padding: 0.25rem 0.5rem;
	}
	.derived-leaf button {
		font-size: 0.75rem;
		text-decoration: underline;
		background: none;
		border: none;
		padding: 0;
		cursor: pointer;
		color: #52525b;
	}
</style>
