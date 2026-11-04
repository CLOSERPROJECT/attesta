<script lang="ts">
	import { onMount } from 'svelte';
	import { getLocale, locales, setLocale } from '#lib/paraglide/runtime.js';
	import './layout.css';
	import favicon from '#lib/assets/favicon.svg';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();

	// Go static bypasses paraglideMiddleware; sync lang after client locale resolve.
	onMount(() => {
		document.documentElement.lang = getLocale();
	});
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

{@render children()}

<div style="display:none">
	{#each locales as locale (locale)}
		<button type="button" onclick={() => setLocale(locale)}>{locale}</button>
	{/each}
</div>
