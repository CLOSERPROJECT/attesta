<script lang="ts">
	import Check from '@lucide/svelte/icons/check';
	import { buttonAttributes, getFormContext, type ComponentProps } from '@sjsf/form';
	import { getThemeContext } from '@sjsf/shadcn4-theme';

	const { children, config }: ComponentProps['submitButton'] = $props();

	const ctx = getFormContext();
	const themeCtx = getThemeContext();
	const { Button } = $derived(themeCtx.components);
</script>

<Button {...buttonAttributes(ctx, config, 'shadcn4SubmitButton', 'submit', {})}>
	<Check data-icon="inline-start" aria-hidden="true" />
	{@render children()}
</Button>
