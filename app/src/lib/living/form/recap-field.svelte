<script lang="ts" module>
	import type { FieldCommonProps } from '@sjsf/form';

	declare module '@sjsf/form' {
		interface FoundationalComponents {
			recapField: {};
		}
		interface ComponentProps {
			recapField: FieldCommonProps<string>;
		}
		interface ComponentBindings {
			recapField: 'value';
		}
		interface ActionFields {
			recapField: {};
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
	import type { ChromeTone } from './chrome-tone.js';

	let {
		config,
		value: _value = $bindable(),
		uiOption
	}: ComponentProps['recapField'] = $props();

	const ctx = getFormContext();
	const chromeCtx = getChromeContext();

	const id = $derived(uiOption('chromeId') ?? String(config.path.at(-1) ?? 'chrome'));
	const display = $derived(chromeCtx.get(id));
	const title = $derived(display.title ?? null);
	const rows = $derived(display.rows ?? []);

	const Template = $derived(getComponent(ctx, 'fieldTemplate', config));
	const errors = $derived(getFieldErrors(ctx, config.path));
</script>

{#snippet statusIcon(tone: ChromeTone | null | undefined)}
	{#if tone === 'warning'}
		<svg class="recap__icon" viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
			<path
				fill="currentColor"
				d="M8 1.2 1.6 13.2a.75.75 0 0 0 .66 1.1h11.48a.75.75 0 0 0 .66-1.1L8 1.2Zm0 4.3a.7.7 0 0 1 .7.7v2.8a.7.7 0 1 1-1.4 0V6.2a.7.7 0 0 1 .7-.7Zm0 5.6a.85.85 0 1 1 0 1.7.85.85 0 0 1 0-1.7Z"
			/>
		</svg>
	{:else if tone === 'danger'}
		<svg class="recap__icon" viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
			<path
				fill="currentColor"
				d="M8 1.25a6.75 6.75 0 1 0 0 13.5 6.75 6.75 0 0 0 0-13.5Zm2.4 4.05a.65.65 0 0 1 .92.92L8.92 8l2.4 2.4a.65.65 0 1 1-.92.92L8 8.92l-2.4 2.4a.65.65 0 1 1-.92-.92L7.08 8l-2.4-2.4a.65.65 0 0 1 .92-.92L8 7.08l2.4-2.4Z"
			/>
		</svg>
	{:else if tone === 'green'}
		<svg class="recap__icon" viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
			<path
				fill="currentColor"
				d="M6.4 11.45 3.3 8.35a.7.7 0 0 1 1-1l2.1 2.1 5.3-5.3a.7.7 0 1 1 1 1l-5.8 5.8a.7.7 0 0 1-1 0Z"
			/>
		</svg>
	{/if}
{/snippet}

<Template
	type="template"
	showTitle={false}
	useLabel={false}
	widgetType="textWidget"
	{uiOption}
	value={null}
	{config}
	{errors}
>
	<section class="recap" data-testid="recap-field" data-chrome-id={id}>
		{#if title}
			<h3 class="recap__title">{title}</h3>
		{/if}
		<ul class="recap__rows">
			{#each rows as row (row.label)}
				<li
					class="recap__row"
					data-tone={row.tone ?? undefined}
					class:recap__row--info={row.tone === 'info'}
					class:recap__row--warning={row.tone === 'warning'}
					class:recap__row--danger={row.tone === 'danger'}
					class:recap__row--green={row.tone === 'green'}
				>
					<span class="recap__label">{row.label}</span>
					{#if row.error}
						<span class="recap__error">{row.error}</span>
					{:else}
						<span class="recap__value">
							{@render statusIcon(row.tone)}
							{row.value ?? '…'}
						</span>
					{/if}
				</li>
			{/each}
		</ul>
	</section>
</Template>

<style>
	/* Match shadcn4 field labels (`text-sm` → --text-sm / 0.875rem). */
	.recap {
		margin: 0 0 1rem;
		padding: 0.75rem 0.9rem;
		border: 1px solid oklch(0.88 0.01 250);
		border-radius: 0.5rem;
		background: oklch(0.97 0.005 250);
		font-size: var(--text-sm, 0.875rem);
		line-height: var(--text-sm--line-height, calc(1.25 / 0.875));
	}

	.recap__title {
		margin: 0 0 0.55rem;
		font-size: inherit;
		font-weight: 500;
		color: oklch(0.35 0.02 250);
	}

	.recap__rows {
		margin: 0;
		padding: 0;
		list-style: none;
		display: grid;
		gap: 0.4rem;
	}

	.recap__row {
		display: flex;
		justify-content: space-between;
		align-items: baseline;
		gap: 1rem;
		font-size: inherit;
	}

	.recap__label {
		color: oklch(0.48 0.015 250);
	}

	.recap__value {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		font-weight: 600;
		color: oklch(0.32 0.02 250);
	}

	.recap__icon {
		flex-shrink: 0;
		display: block;
	}

	.recap__error {
		color: oklch(0.52 0.2 25);
		font-size: inherit;
	}

	.recap__row--info .recap__value {
		color: oklch(0.5 0.04 250);
		font-weight: 500;
	}

	.recap__row--warning .recap__value {
		color: oklch(0.58 0.15 75);
	}

	.recap__row--danger .recap__value {
		color: oklch(0.52 0.2 25);
	}

	.recap__row--green .recap__value {
		color: oklch(0.48 0.14 145);
	}
</style>
