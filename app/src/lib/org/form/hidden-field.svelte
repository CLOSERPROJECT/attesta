<script lang="ts" module>
	import type { FieldCommonProps } from '@sjsf/form';

	declare module '@sjsf/form' {
		interface FoundationalComponents {
			hiddenField: {};
		}
		interface ComponentProps {
			hiddenField: FieldCommonProps<string | undefined>;
		}
		interface ComponentBindings {
			hiddenField: 'value';
		}
		interface ActionFields {
			hiddenField: {};
		}
		interface UiOptions {
			/** When true, paint Config title if the bound string is non-empty. */
			presenceMark?: boolean;
		}
	}
</script>

<script lang="ts">
	import type { ComponentProps } from '@sjsf/form';
	import { shouldPaintPresenceMark } from './presence-mark.js';

	let {
		value = $bindable(),
		config,
		uiOption
	}: ComponentProps['hiddenField'] = $props();

	const showMark = $derived(shouldPaintPresenceMark(uiOption('presenceMark'), value));
	const label = $derived((uiOption('title') ?? config.title ?? '').trim());
</script>

{#if showMark}
	<span
		class="presence-mark text-muted-foreground shrink-0 self-center text-sm"
		data-presence-mark
		data-testid="presence-mark"
		aria-label={label || undefined}
		title={value || undefined}
	>
		{label}
	</span>
{:else}
	<!-- Marker so compact-array CSS can :has([data-hidden-field]) collapse the property wrapper. -->
	<span data-hidden-field data-testid="hidden-field" hidden aria-hidden="true"></span>
{/if}

<style>
	.presence-mark {
		white-space: nowrap;
	}
</style>
