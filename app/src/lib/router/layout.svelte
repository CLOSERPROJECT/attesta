<script lang="ts">
	import type { Snippet } from 'svelte';
	import { onMount } from 'svelte';

	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import Scan from '@lucide/svelte/icons/scan';

	import attestaMark from '#lib/assets/attesta-mark.svg';
	import { living } from '#lib/living/living-app.svelte.js';
	import LivingToast from '#lib/living/LivingToast.svelte';
	import ScanDialog from '#lib/living/ScanDialog.svelte';
	import { createHttpQrResolvePort } from '#lib/living/http-qr-resolve-port.js';
	import {
		livingRouteCoordsFromPath,
		navPlaceKindFromRoute,
		resolveNavPlace
	} from '#lib/living/nav-place.js';
	import { takeColdEntityIdRaw } from '#lib/living/qr-resolve-apply.js';
	import { runQrScan } from '#lib/living/qr-scan-flow.js';
	import { livingToast } from '#lib/living/toast.svelte.js';
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

	const qrPort = createHttpQrResolvePort();
	let scanOpen = $state(false);
	let scanBusy = $state(false);

	async function handleRaw(raw: string) {
		if (scanBusy) return;
		scanBusy = true;
		try {
			await runQrScan(
				{
					resolve: qrPort.resolve,
					drafts: living,
					navigate: navigateLivingTarget,
					toast: (message) => livingToast.show(message)
				},
				raw
			);
		} finally {
			scanBusy = false;
		}
	}

	onMount(() => {
		const { raw, strippedHref } = takeColdEntityIdRaw(window.location.href);
		if (!raw) return;
		window.history.replaceState(window.history.state, '', strippedHref);
		void handleRaw(raw);
	});
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
			{#if place.jobTitleLine}
				<p class="truncate text-xs text-zinc-500" data-testid="nav-place-job-title">
					{place.jobTitleLine}
				</p>
			{/if}
			<p class="truncate font-semibold text-zinc-900" data-testid="nav-place-primary">
				{place.primaryLabel}
			</p>
		</div>
	</div>

	<div class="flex shrink-0 items-center gap-2">
		{#if roles.length > 0}
			<div class="hidden flex-wrap justify-end gap-1 md:flex" data-testid="nav-roles">
				{#each roles as role (role)}
					<Badge variant="outline" data-testid="nav-role-chip">{role}</Badge>
				{/each}
			</div>
		{/if}
		<Button
			type="button"
			variant="outline"
			size="sm"
			aria-label="Scan"
			data-testid="nav-scan"
			disabled={scanBusy}
			onclick={() => {
				scanOpen = true;
			}}
		>
			<Scan class="size-4" />
			Scan
		</Button>
	</div>
</nav>

<main class="p-4">
	{@render children()}
</main>

<ScanDialog
	open={scanOpen}
	busy={scanBusy}
	onOpenChange={(next) => {
		scanOpen = next;
	}}
	onScan={handleRaw}
/>
<LivingToast />
