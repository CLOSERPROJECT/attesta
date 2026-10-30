<script lang="ts" module>
	import type { FieldCommonProps } from '@sjsf/form';

	declare module '@sjsf/form' {
		interface FoundationalComponents {
			valuePrintField: {};
		}
		interface ComponentProps {
			valuePrintField: FieldCommonProps<string>;
		}
		interface ComponentBindings {
			valuePrintField: 'value';
		}
		interface ActionFields {
			valuePrintField: {};
		}
		interface UiOptions {
			/** Chrome map id from substep YAML. */
			chromeId?: string;
		}
	}
</script>

<script lang="ts">
	import {
		getComponent,
		getFieldErrors,
		getFormContext,
		type ComponentProps
	} from '@sjsf/form';
	import { getChromeContext } from './chrome-context.js';

	let {
		config,
		value: _value = $bindable(),
		uiOption
	}: ComponentProps['valuePrintField'] = $props();

	const ctx = getFormContext();
	const chromeCtx = getChromeContext();

	const id = $derived(uiOption('chromeId') ?? String(config.path.at(-1) ?? 'chrome'));
	const display = $derived(chromeCtx.get(id));

	const Template = $derived(getComponent(ctx, 'fieldTemplate', config));
	const errors = $derived(getFieldErrors(ctx, config.path));
</script>

<Template
	type="template"
	showTitle={false}
	useLabel={false}
	widgetType="textWidget"
	{uiOption}
	value={display.value}
	{config}
	{errors}
>
	<div class="value-print" data-testid="value-print" data-chrome-id={id}>
		{#if display.error}
			<p class="value-print__error">{display.error}</p>
		{:else}
			<p class="value-print__value">{display.value ?? '…'}</p>
		{/if}
	</div>
</Template>

<style>
	.value-print__value {
		margin: 0.25rem 0 0.75rem;
		font-size: 1.05rem;
		font-weight: 600;
	}

	.value-print__error {
		margin: 0.25rem 0 0.75rem;
		color: oklch(0.55 0.18 25);
		font-size: 0.85rem;
	}
</style>
