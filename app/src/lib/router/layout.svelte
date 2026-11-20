<script lang="ts">
	import type { Snippet } from 'svelte';
	import { onMount } from 'svelte';

	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import Scan from '@lucide/svelte/icons/scan';

	import attestaMark from '#lib/assets/attesta-mark.svg';
	import * as m from '#lib/paraglide/messages.js';
	import { orgSession } from '#lib/org/org-app.svelte.js';
	import OrgToast from '#lib/org/org-toast.svelte';
	import ScanDialog from '#lib/org/scan-dialog.svelte';
	import { createHttpQrResolvePort } from '#lib/org/http-qr-resolve-port.js';
	import {
		orgRouteCoordsFromPath,
		navPlaceKindFromRoute,
		resolveNavPlace
	} from '#lib/org/nav-place.js';
	import { normalizeTaskAddress } from '#lib/org/task-address.js';
	import { runQrScan } from '#lib/org/qr-scan-flow.js';
	import { orgToast } from '#lib/org/toast.svelte.js';
	import { Badge } from '#lib/shadcn/ui/badge/index.js';
	import { Button, buttonVariants } from '#lib/shadcn/ui/button/index.js';
	import { navigateOrgTarget } from '#lib/router/navigate-target.js';
	import { route } from '#lib/router/index.js';

	let { children }: { children: Snippet } = $props();

	const coords = $derived(orgRouteCoordsFromPath({ pathname: route.pathname }));
	const placeKind = $derived(navPlaceKindFromRoute({ pathname: route.pathname }));
	const addr = $derived(
		normalizeTaskAddress({
			stream: coords.stream,
			instance: coords.instance,
			jobId: coords.jobId,
			jobInstanceId: coords.jobInstanceId,
			taskId: coords.taskId
		})
	);

	const place = $derived(
		resolveNavPlace({
			kind: placeKind,
			stream: coords.stream,
			instance: coords.instance,
			jobId: addr.jobId,
			jobInstanceId: addr.jobInstanceId,
			taskId: placeKind === 'task' ? addr.taskId : '',
			provisionalJob: placeKind === 'task' ? addr.provisionalJob : placeKind === 'play',
			worklist: orgSession.result?.worklist
		})
	);

	const roles = $derived(orgSession.result?.roles ?? []);

	const qrPort = createHttpQrResolvePort();
	let scanOpen = $state(false);
	let scanBusy = $state(false);

	async function handleRaw(raw: string) {
		if (scanBusy || !coords.stream) return;
		scanBusy = true;
		try {
			await runQrScan(
				{
					resolve: qrPort.resolve,
					stream: coords.stream,
					instance: coords.instance || undefined,
					drafts: orgSession,
					navigate: navigateOrgTarget,
					toast: (message) => orgToast.show(message)
				},
				raw
			);
		} finally {
			scanBusy = false;
		}
	}

	onMount(() => {
		if (
			coords.stream &&
			coords.instance &&
			!orgSession.result &&
			!orgSession.loading
		) {
			void orgSession.hydrate({ stream: coords.stream, instance: coords.instance });
		}
	});

	$effect(() => {
		const { stream, instance } = coords;
		if (!stream || !instance) return;
		if (placeKind !== 'home' && placeKind !== 'task') return;
		orgSession.setScope({ stream, instance });
	});
</script>

<nav
	class="flex items-center justify-between gap-3 border-b border-zinc-200 px-4 py-3 text-sm"
	data-testid="org-nav"
>
	<div class="flex min-w-0 items-center gap-2">
		{#if place.showBack}
			<Button
				type="button"
				variant="outline"
				size="icon"
				aria-label={m.nav_back()}
				data-testid="nav-back"
				onclick={() => navigateOrgTarget(place.backTarget)}
			>
				<ArrowLeft class="size-4" />
			</Button>
		{:else}
			<span
				class={buttonVariants({ variant: 'outline', size: 'icon' })}
				data-testid="nav-home-mark"
			>
				<img src={attestaMark} alt={m.brand_attesta()} class="size-5" />
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
		{#if coords.stream}
			<Button
				type="button"
				variant="outline"
				size="sm"
				aria-label={m.nav_scan()}
				data-testid="nav-scan"
				disabled={scanBusy}
				onclick={() => {
					scanOpen = true;
				}}
			>
				<Scan class="size-4" />
				{m.nav_scan()}
			</Button>
		{/if}
	</div>
</nav>

<main>
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
<OrgToast />
