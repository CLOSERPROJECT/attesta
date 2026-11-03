<script lang="ts">
	import type { Snippet } from 'svelte';

	import ArrowLeft from '@lucide/svelte/icons/arrow-left';

	import attestaMark from '#lib/assets/attesta-mark.svg';
	import { living } from '#lib/living/living-app.svelte.js';
	import {
		livingRouteCoordsFromPath,
		navPlaceKindFromRoute,
		resolveNavPlace
	} from '#lib/living/nav-place.js';
	import { Badge } from '#lib/shadcn/ui/badge/index.js';
	import { Button, buttonVariants } from '#lib/shadcn/ui/button/index.js';
	import { route } from '#lib/router/index.js';
	import { navigateLivingTarget } from '#lib/router/navigate-target.js';

	let { children }: { children: Snippet } = $props();

	// Pathname segments only — sv-router leaks group*/taskId across matches.
	const coords = $derived(livingRouteCoordsFromPath({ pathname: route.pathname }));
	const jobId = $derived(coords.jobId);
	const routeInstance = $derived(coords.jobInstanceId);
	const groupId = $derived(coords.groupId);
	const groupRoute = $derived(coords.groupInstanceId);
	const provisionalJob = $derived(
		!groupId && (routeInstance === '' || routeInstance === 'new')
	);
	const provisionalSpawn = $derived(
		Boolean(groupId) && (groupRoute === '' || groupRoute === 'new')
	);
	const jobInstanceId = $derived(provisionalJob ? '' : routeInstance);

	const placeKind = $derived(navPlaceKindFromRoute({ pathname: route.pathname }));
	const taskId = $derived(placeKind === 'task' ? coords.taskId : '');

	const place = $derived(
		resolveNavPlace({
			kind: placeKind,
			jobId,
			jobInstanceId,
			taskId,
			provisionalJob: placeKind === 'task' ? provisionalJob : false,
			provisionalSpawn: placeKind === 'task' ? provisionalSpawn : false,
			worklist: living.result?.worklist
		})
	);

	const roles = $derived(living.result?.roles ?? []);
</script>

<nav
	class="flex items-center justify-between gap-3 border-b border-zinc-200 px-4 py-3 text-sm"
	data-testid="living-nav"
>
	<div class="flex min-w-0 items-center gap-2">
		{#if place.showBack}
			<Button
				type="button"
				variant="outline"
				size="icon"
				aria-label="Back"
				data-testid="nav-back"
				onclick={() => navigateLivingTarget(place.backTarget)}
			>
				<ArrowLeft class="size-4" />
			</Button>
		{:else}
			<span
				class={buttonVariants({ variant: 'outline', size: 'icon' })}
				data-testid="nav-home-mark"
			>
				<img src={attestaMark} alt="Attesta" class="size-5" />
			</span>
		{/if}

		<div class="min-w-0 text-left" data-testid="nav-place">
			<p class="truncate font-semibold text-zinc-900" data-testid="nav-place-primary">
				{place.primaryLabel}
			</p>
			{#if place.jobTitleLine}
				<p class="truncate text-xs text-zinc-500" data-testid="nav-place-job-title">
					{place.jobTitleLine}
				</p>
			{/if}
		</div>
	</div>

	{#if roles.length > 0}
		<div class="flex shrink-0 flex-wrap justify-end gap-1" data-testid="nav-roles">
			{#each roles as role (role)}
				<Badge variant="outline" data-testid="nav-role-chip">{role}</Badge>
			{/each}
		</div>
	{/if}
</nav>

<main class="p-4">
	{@render children()}
</main>
