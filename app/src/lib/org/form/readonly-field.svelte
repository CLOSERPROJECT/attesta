<script lang="ts" module>
	import type { FieldCommonProps } from '@sjsf/form';

	declare module '@sjsf/form' {
		interface FoundationalComponents {
			readonlyField: {};
		}
		interface ComponentProps {
			readonlyField: FieldCommonProps<string>;
		}
		interface ComponentBindings {
			readonlyField: 'value';
		}
		interface ActionFields {
			readonlyField: {};
		}
		interface UiOptions {
			chromeId?: string;
			/** When true, omit paint if chrome display value is null (errors still show). */
			hideIfNull?: boolean;
			placeholder?: string;
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
	import { Input } from '@sjsf/shadcn4-theme/new-york';
	import { getChromeContext } from './chrome-context.js';

	let {
		config,
		value: _value = $bindable(),
		uiOption
	}: ComponentProps['readonlyField'] = $props();

	const ctx = getFormContext();
	const chromeCtx = getChromeContext();

	const id = $derived(uiOption('chromeId') ?? String(config.path.at(-1) ?? 'chrome'));
	const display = $derived(chromeCtx.get(id));
	const hideEmpty = $derived(
		uiOption('hideIfNull') === true && display.value == null && !display.error
	);
	const placeholder = $derived((uiOption('placeholder') as string | undefined) ?? '');

	const Template = $derived(getComponent(ctx, 'fieldTemplate', config));
	const errors = $derived(getFieldErrors(ctx, config.path));
</script>

{#if !hideEmpty}
	<Template
		type="template"
		showTitle={true}
		useLabel={true}
		widgetType="textWidget"
		{uiOption}
		value={display.value}
		{config}
		{errors}
	>
		<div class="readonly-field computed-field" data-testid="readonly-field" data-chrome-id={id}>
			<Input
				readonly
				tabindex={-1}
				value={display.value ?? ''}
				{placeholder}
				class="readonly-field__input"
			/>
		</div>
		{#if display.error}
			<p class="readonly-field__error">{display.error}</p>
		{/if}
	</Template>
{/if}

<style>
	.readonly-field__error {
		margin: 0.25rem 0 0.75rem;
		color: oklch(0.55 0.18 25);
		font-size: 0.85rem;
	}
</style>
