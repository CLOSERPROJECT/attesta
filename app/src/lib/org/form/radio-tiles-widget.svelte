<script lang="ts" module>
	import type { HTMLInputAttributes } from 'svelte/elements';
	import type { SchemaValue } from '@sjsf/form';
	import type { SingleSelectOptions, WidgetCommonProps } from '@sjsf/form/fields/widgets';

	declare module '@sjsf/form' {
		interface FoundationalComponents {
			radioTilesWidget: {};
		}
		interface ComponentProps {
			/** Compatible with selectWidget so `ui:components.selectWidget: radioTilesWidget` typechecks. */
			radioTilesWidget: WidgetCommonProps<SchemaValue> & SingleSelectOptions;
		}
		interface ComponentBindings {
			radioTilesWidget: 'value';
		}
		interface UiOptions {
			radioTiles?: HTMLInputAttributes;
		}
	}
</script>

<script lang="ts">
	import { getFormContext, inputAttributes, type ComponentProps } from '@sjsf/form';
	import { idMapper, singleOption } from '@sjsf/form/options.svelte';

	let {
		handlers,
		config,
		value = $bindable(),
		options,
		mapped = singleOption({
			mapper: () => idMapper(options),
			value: () => value,
			update: (v) => (value = v)
		})
	}: ComponentProps['radioTilesWidget'] = $props();

	const ctx = getFormContext();

	const attributes = $derived(
		inputAttributes(ctx, config, 'radioTiles', handlers, {
			type: 'radio'
		})
	);
</script>

<div class="radio-tiles" data-testid="radio-tiles" role="radiogroup">
	{#each options as option (option.id)}
		{@const optionValue = option.mappedValue ?? option.id}
		{@const selected = mapped.current === optionValue}
		<label
			class={[
				'radio-tiles__tile',
				{
					'radio-tiles__tile--selected': selected,
					'radio-tiles__tile--disabled': option.disabled || attributes.disabled
				}
			]}
		>
			<input
				bind:group={mapped.current}
				value={optionValue}
				{...attributes}
				id={option.id}
				disabled={option.disabled || attributes.disabled}
			/>
			<span class="radio-tiles__label">{option.label}</span>
		</label>
	{/each}
</div>

<style>
	.radio-tiles {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(9rem, 1fr));
		gap: 0.5rem;
	}

	.radio-tiles__tile {
		display: flex;
		flex-direction: row;
		align-items: center;
		gap: 0.5rem;
		padding: 0.75rem;
		min-height: 2.75rem;
		cursor: pointer;
		border: none;
		background: var(--input);
		border-radius: var(--input-radius, 2px);
		box-sizing: border-box;
		margin: 0;
	}

	.radio-tiles__tile--selected {
		border: none;
		background: color-mix(in srgb, var(--primary) 12%, #ffffff);
		color: var(--foreground);
		outline: 2px solid var(--primary);
		outline-offset: 0;
	}

	.radio-tiles__tile--disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.radio-tiles__label {
		line-height: 1.25;
	}
</style>
