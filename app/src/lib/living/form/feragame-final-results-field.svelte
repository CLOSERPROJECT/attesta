<script lang="ts" module>
	import type { FieldCommonProps } from '@sjsf/form';

	declare module '@sjsf/form' {
		interface FoundationalComponents {
			feragameFinalResultsField: {};
		}
		interface ComponentProps {
			feragameFinalResultsField: FieldCommonProps<string>;
		}
		interface ComponentBindings {
			feragameFinalResultsField: 'value';
		}
		interface ActionFields {
			feragameFinalResultsField: {};
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
	}: ComponentProps['feragameFinalResultsField'] = $props();

	const ctx = getFormContext();
	const chromeCtx = getChromeContext();

	const id = $derived(uiOption('chromeId') ?? String(config.path.at(-1) ?? 'chrome'));
	const display = $derived(chromeCtx.get(id));
	const title = $derived(display.title ?? null);
	const chromeError = $derived(display.error);
	const sections = $derived(display.sections ?? []);

	const Template = $derived(getComponent(ctx, 'fieldTemplate', config));
	const errors = $derived(getFieldErrors(ctx, config.path));
</script>

{#snippet statusIcon(tone: ChromeTone | null | undefined)}
	{#if tone === 'warning'}
		<svg class="final-results__icon" viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
			<path
				fill="currentColor"
				d="M8 1.2 1.6 13.2a.75.75 0 0 0 .66 1.1h11.48a.75.75 0 0 0 .66-1.1L8 1.2Zm0 4.3a.7.7 0 0 1 .7.7v2.8a.7.7 0 1 1-1.4 0V6.2a.7.7 0 0 1 .7-.7Zm0 5.6a.85.85 0 1 1 0 1.7.85.85 0 0 1 0-1.7Z"
			/>
		</svg>
	{:else if tone === 'danger'}
		<svg class="final-results__icon" viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
			<path
				fill="currentColor"
				d="M8 1.25a6.75 6.75 0 1 0 0 13.5 6.75 6.75 0 0 0 0-13.5Zm2.4 4.05a.65.65 0 0 1 .92.92L8.92 8l2.4 2.4a.65.65 0 1 1-.92.92L8 8.92l-2.4 2.4a.65.65 0 1 1-.92-.92L7.08 8l-2.4-2.4a.65.65 0 0 1 .92-.92L8 7.08l2.4-2.4Z"
			/>
		</svg>
	{:else if tone === 'green'}
		<svg class="final-results__icon" viewBox="0 0 16 16" width="14" height="14" aria-hidden="true">
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
	<section
		class="final-results"
		data-testid="feragame-final-results-field"
		data-chrome-id={id}
	>
		{#if title}
			<h3 class="final-results__title">{title}</h3>
		{/if}
		{#if chromeError}
			<p class="final-results__error" data-testid="final-results-error">{chromeError}</p>
		{:else if sections.length === 0}
			<p class="final-results__empty">Nessun carico primario</p>
		{:else}
			{#each sections as section (section.loadId)}
				<article class="final-results__section" data-load-id={section.loadId}>
					<h4 class="final-results__load">{section.loadId}</h4>
					{#if section.secondaryLoadIds && section.secondaryLoadIds.length > 0}
						<p class="final-results__secondary">
							anche: {section.secondaryLoadIds.join(', ')}
						</p>
					{/if}
					{#if section.rows && section.rows.length > 0}
						<ul class="final-results__rows">
							{#each section.rows as row (row.label)}
								<li
									class="final-results__row"
									data-tone={row.tone ?? undefined}
									class:final-results__row--info={row.tone === 'info'}
									class:final-results__row--warning={row.tone === 'warning'}
									class:final-results__row--danger={row.tone === 'danger'}
									class:final-results__row--green={row.tone === 'green'}
								>
									<span class="final-results__label">{row.label}</span>
									{#if row.error}
										<span class="final-results__error">{row.error}</span>
									{:else}
										<span class="final-results__value">
											{@render statusIcon(row.tone)}
											{row.value ?? '…'}
										</span>
									{/if}
								</li>
							{/each}
						</ul>
					{/if}
				</article>
			{/each}
		{/if}
	</section>
</Template>

<style>
	.final-results {
		margin: 0 0 1rem;
		padding: 0.75rem 0.9rem;
		border: 1px solid oklch(0.88 0.01 250);
		border-radius: 0.5rem;
		background: oklch(0.97 0.005 250);
		font-size: var(--text-sm, 0.875rem);
		line-height: var(--text-sm--line-height, calc(1.25 / 0.875));
	}

	.final-results__title {
		margin: 0 0 0.55rem;
		font-size: inherit;
		font-weight: 500;
		color: oklch(0.35 0.02 250);
	}

	.final-results__empty {
		margin: 0;
		color: oklch(0.5 0.015 250);
		font-weight: 500;
	}

	.final-results__section {
		margin: 0 0 0.75rem;
	}

	.final-results__section:last-child {
		margin-bottom: 0;
	}

	.final-results__load {
		margin: 0 0 0.25rem;
		font-size: inherit;
		font-weight: 600;
		color: oklch(0.32 0.02 250);
	}

	.final-results__secondary {
		margin: 0 0 0.4rem;
		color: oklch(0.55 0.01 250);
		font-size: inherit;
		font-weight: 400;
	}

	.final-results__rows {
		margin: 0;
		padding: 0;
		list-style: none;
		display: grid;
		gap: 0.4rem;
	}

	.final-results__row {
		display: flex;
		justify-content: space-between;
		align-items: baseline;
		gap: 1rem;
		font-size: inherit;
	}

	.final-results__label {
		color: oklch(0.48 0.015 250);
	}

	.final-results__value {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
		font-weight: 600;
		color: oklch(0.32 0.02 250);
	}

	.final-results__icon {
		flex-shrink: 0;
		display: block;
	}

	.final-results__error {
		color: oklch(0.52 0.2 25);
		font-size: inherit;
	}

	.final-results__row--info .final-results__value {
		color: oklch(0.5 0.04 250);
		font-weight: 500;
	}

	.final-results__row--warning .final-results__value {
		color: oklch(0.58 0.15 75);
	}

	.final-results__row--danger .final-results__value {
		color: oklch(0.52 0.2 25);
	}

	.final-results__row--green .final-results__value {
		color: oklch(0.48 0.14 145);
	}
</style>
