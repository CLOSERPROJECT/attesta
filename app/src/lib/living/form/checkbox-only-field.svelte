<script lang="ts">
	import {
		getComponent,
		getFieldErrors,
		getFormContext,
		type ComponentProps
	} from '@sjsf/form';

	let {
		config,
		value = $bindable(),
		uiOption
	}: ComponentProps['checkboxOnlyField'] = $props();

	const ctx = getFormContext();
	const Template = $derived(getComponent(ctx, 'fieldTemplate', config));
	const errors = $derived(getFieldErrors(ctx, config.path));
	const label = $derived(config.title || 'Fonti doppie');
</script>

<Template
	type="template"
	showTitle={false}
	useLabel={false}
	widgetType="checkboxWidget"
	{uiOption}
	value={undefined}
	{config}
	{errors}
>
	<!-- size-9 hit target (input-like chrome); default checkbox inside; no text label. -->
	<label
		class="checkbox-only-hit size-9 shrink-0 rounded-md border border-input shadow-xs"
		data-slot="input"
		title={label}
	>
		<input
			type="checkbox"
			class="checkbox-only"
			data-testid="checkbox-only"
			aria-label={label}
			checked={value === true}
			onchange={(e) => {
				value = e.currentTarget.checked;
			}}
		/>
	</label>
</Template>

<style>
	.checkbox-only-hit {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		box-sizing: border-box;
		min-width: 2.25rem;
		min-height: 2.25rem;
		cursor: pointer;
		margin: 0;
	}

	.checkbox-only {
		margin: 0;
		cursor: pointer;
	}
</style>
