<script lang="ts" module>
	import type { FieldCommonProps } from '@sjsf/form';
	import type { SchemaValue } from '@sjsf/form/core';

	declare module '@sjsf/form' {
		interface FoundationalComponents {
			compactArrayItemField: {};
		}
		interface ComponentProps {
			compactArrayItemField: FieldCommonProps<SchemaValue> & {
				index: number;
			};
		}
		interface ComponentBindings {
			compactArrayItemField: 'value';
		}
		interface ActionFields {
			compactArrayItemField: {};
		}
	}
</script>

<script lang="ts">
	import {
		getComponent,
		getFieldComponent,
		getFieldErrors,
		getFormContext,
		Text,
		type ComponentProps,
		type UiOptions
	} from '@sjsf/form';
	import { getArrayContext } from '@sjsf/form/fields/array/context.svelte';

	let {
		index,
		value = $bindable(),
		config,
		uiOption,
		translate
	}: ComponentProps['compactArrayItemField'] = $props();

	const ctx = getFormContext();
	const arrayCtx = getArrayContext();

	const Template = $derived(getComponent(ctx, 'arrayItemTemplate', config));
	const Field = $derived(getFieldComponent(ctx, config));
	const Button = $derived(getComponent(ctx, 'button', config));

	const canRemove = $derived(arrayCtx.canRemove(index));
	const errors = $derived(getFieldErrors(ctx, config.path));

	/** Force no title/label on the item field tree (paint also sets hideTitle). */
	function childUiOption<O extends keyof UiOptions>(opt: O): UiOptions[O] | undefined {
		if (opt === 'hideTitle') return true as UiOptions[O];
		if (opt === 'useLabel') return false as UiOptions[O];
		return uiOption(opt);
	}
</script>

{#snippet buttons()}
	{#if canRemove}
		<Button
			{errors}
			{config}
			disabled={false}
			type="array-item-remove"
			onclick={() => {
				arrayCtx.removeItem(index);
			}}
		>
			<Text {config} id="remove-array-item" {translate} />
		</Button>
	{/if}
{/snippet}

<div
	class="compact-array-item [&_button]:size-9 [&_button]:p-0 [&_button_svg]:size-4"
	data-testid="compact-array-item"
>
	<Template
		type="template"
		{index}
		{value}
		{config}
		{errors}
		buttons={canRemove ? buttons : undefined}
		{uiOption}
	>
		<Field
			type="field"
			bind:value={value as undefined}
			{config}
			uiOption={childUiOption}
			{translate}
		/>
	</Template>
</div>

<style>
	.compact-array-item {
		width: 100%;
	}

	.compact-array-item :global([data-slot='field']) {
		gap: 0;
	}

	/* Row: weight column grows; checkbox column hugs control size. */
	.compact-array-item :global([data-layout='object-properties'] > *:first-child) {
		flex: 1 1 0;
		min-width: 0;
	}

	.compact-array-item :global([data-layout='object-properties'] > *:last-child) {
		flex: 0 0 auto;
		flex-shrink: 0;
		width: auto;
		min-width: 2.25rem;
		align-self: stretch;
		display: flex;
		align-items: center;
	}

	.compact-array-item :global([data-layout='object-properties'] > *:last-child [data-slot='field']) {
		width: auto;
		align-items: center;
		justify-content: center;
	}
</style>
