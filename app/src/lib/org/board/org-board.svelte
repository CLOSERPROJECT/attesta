<script lang="ts">
	import { onMount, untrack } from 'svelte';
	import { searchParams } from 'sv-router';

	import * as m from '#lib/paraglide/messages.js';

	import type { CommitOutcome } from '../commit-port.js';
	import { applyCommitFollowThrough, decideCommitFollowThrough } from '../commit-follow-through.js';
	import { orgSession } from '../org-app.svelte.js';
	import { createSseDomainEventsPort } from '../sse-domain-events-port.js';
	import { fetchStreamTitle } from '../stream-meta.js';
	import { normalizeTaskAddress, taskAddressKey, type TaskAddress } from '../task-address.js';
	import { createHttpTimelinePort } from '../timeline/http-timeline-port.js';
	import TimelineStep from '../timeline/timeline-step.svelte';
	import { navigateOrgTarget, orgMyStreamDashboardPath } from '../../router/navigate-target.js';
	import { navigate } from '../../router/index.js';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import ChevronsDownUpIcon from '@lucide/svelte/icons/chevrons-down-up';
	import { Button, buttonVariants } from '#lib/shadcn/ui/button/index.js';
	import { Checkbox } from '#lib/shadcn/ui/checkbox/index.js';
	import { Label } from '#lib/shadcn/ui/label/index.js';
	import { cn } from '#lib/shadcn/utils.js';
	import { setBoardEmbed, type BoardEmbedApi } from './board-context.js';
	import type { OwnDomainChange } from './board-refresh-plan.js';
	import { createBoardSession, type BoardScope } from './board-session.svelte.js';
	import { draftAddressForTask } from '../drafts.js';
	import {
		BoardTimelineModel,
		recapAddressesFromTimeline,
		setBoardTimelineModel
	} from './board-timeline-model.svelte.js';
	import { boardFormCache, type BoardFormEntry } from './board-form-cache.svelte.js';
	import { boardRecapCache } from './board-recap-cache.svelte.js';
	import { BoardNestCollapse, setBoardNestCollapse } from './board-nest-collapse.svelte.js';
	import EmbeddedTaskPanel from './embedded-task-panel.svelte';
	import { stayAftermath } from './commit-stay-recap.js';
	import {
		addressFromBoardTask,
		BOARD_TASK_SEARCH_KEY,
		boardTaskFromAddress,
		encodeBoardTaskParam,
		parseBoardTaskParam,
		type BoardTaskFocus
	} from './board-url.js';
	import { resolveTaskDeepLink } from '../worklist-nav.js';

	import '../timeline/org-timeline.css';

	type Props = {
		scope: BoardScope;
	};

	let { scope }: Props = $props();

	const nestCollapse = new BoardNestCollapse();
	setBoardNestCollapse(nestCollapse);

	function syncFocusToUrl(next: BoardTaskFocus | null) {
		if (next) {
			searchParams.set(BOARD_TASK_SEARCH_KEY, encodeBoardTaskParam(next), {
				replace: true
			});
		} else if (searchParams.get(BOARD_TASK_SEARCH_KEY) != null) {
			searchParams.delete(BOARD_TASK_SEARCH_KEY, undefined, { replace: true });
		}
	}

	const model = new BoardTimelineModel({ syncUrl: syncFocusToUrl });
	setBoardTimelineModel(model);

	const timelinePort = createHttpTimelinePort();

	const session = createBoardSession({
		// One-shot init; `$effect` below keeps scope in sync with route props.
		scope: untrack(() => scope),
		timeline: timelinePort,
		events: createSseDomainEventsPort(),
		commits: orgSession,
		model,
		formCache: boardFormCache,
		recapCache: boardRecapCache,
		navigateToInstance: async ({ stream: s, instance: i, focus }) => {
			const search =
				focus != null ? { [BOARD_TASK_SEARCH_KEY]: encodeBoardTaskParam(focus) } : undefined;
			await navigate('/:stream/i/:instance', {
				params: { stream: s, instance: i },
				search,
				replace: true
			});
		},
		hydrate: (args) => {
			void orgSession.hydrate(
				{ stream: args.stream, instance: args.instance },
				{ keepOnError: args.background === true }
			);
		},
		dropSeededDraft: (addr) => {
			const d = draftAddressForTask(addr);
			orgSession.dropSeededDraft(d.taskPath, d.draftBagInstanceKey);
		}
	});

	const view = $derived(session.view);
	const error = $derived(session.error);
	const loading = $derived(session.loading);
	const stream = $derived(scope.stream);
	const instance = $derived(scope.kind === 'instance' ? scope.instance : '');
	const isBoot = $derived(scope.kind === 'boot');
	let streamTitle = $state('');
	let shellPainted = $state(false);
	/** After a deep-link gate deny, do not auto-focus first actionable. */
	let suppressAutoFocus = $state(false);

	/** Stay Commit → land on once-recap / many-recap (keyed by address). */
	let confirmRecapByKey = $state.raw<Record<string, true>>({});

	const scopeKey = $derived(session.scopeKey);

	const focusFromUrl = $derived.by(() => {
		const raw = searchParams.get(BOARD_TASK_SEARCH_KEY);
		return parseBoardTaskParam(raw == null ? null : String(raw));
	});

	const dashboardHref = $derived(orgMyStreamDashboardPath(stream));
	const cacheEntries = $derived(boardFormCache.entries);

	const focusedAddress = $derived(
		model.selection ? addressFromBoardTask(stream, instance, model.selection) : null
	);

	const orphanFocus = $derived(model.orphanFocus);
	const showAll = $derived(model.showAll);
	const actionableOnlyEmpty = $derived(model.isEmpty);

	function setFocus(addr: TaskAddress) {
		const prev = focusedAddress;
		if (prev && taskAddressKey(prev) !== taskAddressKey(addr)) {
			clearConfirmRecap(prev);
		}
		model.select(addr);
		const el = document.querySelector(`[data-address-key="${CSS.escape(taskAddressKey(addr))}"]`);
		el?.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
	}

	function clearFocus() {
		clearConfirmRecap();
		model.clearSelection();
	}

	function collapseAllNests() {
		nestCollapse.collapseAll();
		model.collapseAllBodies();
	}

	function getEntry(addr: TaskAddress): BoardFormEntry | undefined {
		void cacheEntries;
		return boardFormCache.get(taskAddressKey(addr));
	}

	function confirmRecapFor(addr: TaskAddress): boolean {
		return confirmRecapByKey[taskAddressKey(addr)] === true;
	}

	function markConfirmRecap(addr: TaskAddress) {
		const key = taskAddressKey(addr);
		if (key in confirmRecapByKey) return;
		confirmRecapByKey = { ...confirmRecapByKey, [key]: true };
	}

	function clearConfirmRecap(addr?: TaskAddress) {
		if (!addr) {
			if (Object.keys(confirmRecapByKey).length === 0) return;
			confirmRecapByKey = {};
			return;
		}
		const key = taskAddressKey(addr);
		if (!(key in confirmRecapByKey)) return;
		const next = { ...confirmRecapByKey };
		delete next[key];
		confirmRecapByKey = next;
	}

	function commitChanged(
		addr: TaskAddress,
		outcome: CommitOutcome,
		follow: OwnDomainChange['follow'],
		submitMany = false
	) {
		void session.domainChanged({
			origin: 'own',
			revision: outcome.revision ?? null,
			addr,
			follow,
			submitMany
		});
	}

	function pinAfterCommit(addr: TaskAddress, outcome: CommitOutcome) {
		model.pinJustCompleted(addr, outcome.submissionId);
	}

	function onCommitted(addr: TaskAddress, outcome: CommitOutcome, opts?: { submitMany?: boolean }) {
		const decision = decideCommitFollowThrough(outcome, addr);
		pinAfterCommit(addr, outcome);
		if (decision.kind === 'stay') {
			const aftermath = stayAftermath(Boolean(opts?.submitMany));
			if (aftermath.clearUnderFormLinks) {
				orgSession.clearLinksStay();
			}
			commitChanged(addr, outcome, 'stay', Boolean(opts?.submitMany));
			markConfirmRecap(addr);
			return;
		}

		clearConfirmRecap(addr);
		const target = decision.target;

		if (target.kind === 'task') {
			const nextAddr = normalizeTaskAddress(target);
			if (outcome.nextTaskFormBundle) {
				boardFormCache.putReady(nextAddr, outcome.nextTaskFormBundle);
			}
			if (session.scope.kind === 'boot' && target.instance) {
				commitChanged(addr, outcome, 'leaving');
				void session.adoptInstance(target.instance, {
					focus: boardTaskFromAddress(nextAddr),
					bundleAddr: nextAddr,
					bundle: outcome.nextTaskFormBundle
				});
				return;
			}
			applyCommitFollowThrough(decision, {
				clearLinksStay: () => orgSession.clearLinksStay(),
				navigate: (t) => {
					if (t.kind === 'task') {
						setFocus(normalizeTaskAddress(t));
						commitChanged(addr, outcome, 'advance');
						return;
					}
					commitChanged(addr, outcome, 'leaving');
					navigateOrgTarget(t);
				}
			});
			return;
		}

		if (target.kind === 'home') {
			if (session.scope.kind === 'boot') {
				commitChanged(addr, outcome, 'leaving');
				void session.adoptInstance(target.instance, { focus: null });
				orgSession.clearLinksStay();
				return;
			}
			clearFocus();
			orgSession.clearLinksStay();
			commitChanged(addr, outcome, 'advance');
			return;
		}

		commitChanged(addr, outcome, 'leaving');
		applyCommitFollowThrough(decision, {
			clearLinksStay: () => orgSession.clearLinksStay(),
			navigate: navigateOrgTarget
		});
	}

	const boardApi: BoardEmbedApi = {
		getEntry,
		get focus() {
			return model.selection;
		},
		isFocused(addr) {
			return model.isSelected(addr);
		},
		setFocus,
		isEmbedOpen(addr) {
			return model.isBodyOpen(addr);
		},
		setEmbedOpen(addr, open) {
			model.setBodyOpen(addr, open);
		},
		toggleEmbedOpen(addr) {
			return model.toggleBody(addr);
		},
		confirmRecapFor,
		onCommitted,
		acceptStale: (addr) => session.acceptStale(addr)
	};
	setBoardEmbed(boardApi);

	onMount(() => {
		shellPainted = true;
		void fetchStreamTitle(stream).then((title) => {
			streamTitle = title;
		});
		// Do not dispose the process-wide cache on unmount — Play→instance soft-nav
		// remounts the board and must keep putReady / in-flight preload entries.
	});

	// Load via BoardSession (boot = { stream }; instance = { stream, instance }).
	$effect(() => {
		session.setScope(scope);
		void session.load();
	});

	// One events subscription per stream; instance changes keep it (Domain is per stream).
	$effect(() => {
		void stream;
		return untrack(() => session.connect());
	});

	// URL → selection (opens body only when focus changes — see setFocusFromUrl).
	$effect(() => {
		const fromUrl = focusFromUrl;
		if (fromUrl) {
			model.setFocusFromUrl(fromUrl);
		}
	});

	// Orphan chrome: open once when the orphan address appears (not on every re-run).
	$effect(() => {
		const orphan = model.orphanFocus;
		if (!orphan) return;
		const key = taskAddressKey(orphan);
		untrack(() => {
			if (!model.isBodyOpen(orphan)) model.setBodyOpen(orphan, true);
		});
		void key;
	});

	// After shell paint: preload actionable forms + recaps; auto-focus when sole actionable.
	$effect(() => {
		if (!shellPainted || loading) return;
		void scopeKey;
		void isBoot;
		void model.showAll;
		void model.view;
		void model.selection;

		const addrs: TaskAddress[] = [...model.actionableAddresses];
		if (model.selection) {
			addrs.push(addressFromBoardTask(stream, instance, model.selection));
		}
		// untrack: enqueue reads $state entries; without it, #setEntry re-fires this effect.
		untrack(() => {
			if (addrs.length > 0) {
				boardFormCache.enqueue(addrs);
			}
			// task-recap requires stream instance; boot Play has none.
			if (view && !isBoot) {
				boardRecapCache.enqueue(recapAddressesFromTimeline(view));
			}
		});

		if (!focusFromUrl && !model.selection && !suppressAutoFocus) {
			model.applyLoadPolicy();
		}
	});

	// Deep-link gate once Worklist is available (unauthorized / unknown → clear focus).
	$effect(() => {
		const worklist = orgSession.result?.worklist;
		const addr = focusedAddress;
		if (!worklist || !addr || isBoot) return;
		const gate = resolveTaskDeepLink(
			addr.stream,
			addr.instance,
			addr.jobId,
			addr.jobInstanceId,
			addr.taskId,
			addr.provisionalJob,
			worklist
		);
		if (gate.kind === 'task') return;
		suppressAutoFocus = true;
		clearFocus();
		if (gate.kind !== 'home' || gate.instance !== instance) {
			navigateOrgTarget(gate);
		}
	});
</script>

<div
	class="org-timeline-page process-page"
	data-testid="org-timeline"
	data-org-board="1"
	data-board-scope={scope.kind}
	data-show-all={showAll ? '1' : '0'}
	data-route-key={scopeKey}
>
	<div class="org-timeline-toolbar" data-testid="board-chrome">
		<header class="org-timeline-header flex items-center gap-2">
			<Button
				href={dashboardHref}
				variant="outline"
				size="icon"
				class="shrink-0"
				aria-label={m.nav_back()}
				data-testid="board-chrome-back"
				data-sveltekit-reload
				rel="external"
				target="_self"
			>
				<ArrowLeft class="size-4" />
			</Button>
			<span class="timeline-header-line min-w-0">
				<span class="timeline-header-index" data-testid="board-chrome-stream">
					{streamTitle || stream}
				</span>
				{#if scope.kind === 'instance'}
					<span class="timeline-header-title" data-testid="board-chrome-label">
						{view?.label || instance}
					</span>
				{/if}
			</span>
		</header>
		{#if view}
			<div class="stream-show-all-filter">
				<Label class={cn(buttonVariants({ variant: 'outline', size: 'sm' }), 'cursor-pointer')}>
					<Checkbox
						class="js-show-all-filter border-foreground/35 bg-background shadow-none"
						bind:checked={() => model.showAll, (v) => model.setShowAll(!!v)}
						data-testid="show-all-filter"
					/>
					{m.show_all()}
				</Label>
				<Button
					type="button"
					variant="outline"
					size="sm"
					data-testid="collapse-all"
					onclick={collapseAllNests}
				>
					<ChevronsDownUpIcon data-icon="inline-start" />
					{m.collapse_all()}
				</Button>
			</div>
		{/if}
	</div>

	{#if orgSession.commitError}
		<p class="text-sm text-red-700" data-testid="org-commit-error" role="alert">
			{m.commit_denied()}: {orgSession.commitError}
		</p>
	{/if}

	{#if loading && !view}
		<p class="text-sm text-zinc-500" data-testid="org-timeline-status">
			{m.loading()}
		</p>
	{:else if error && !view}
		<p class="text-sm text-red-700" data-testid="org-timeline-error" role="alert">
			{error}
		</p>
	{:else}
		{#if orphanFocus}
			<section class="mb-6" data-testid="orphan-focus-embed">
				<EmbeddedTaskPanel
					address={orphanFocus}
					entry={getEntry(orphanFocus)}
					focused={true}
					confirmRecap={confirmRecapFor(orphanFocus)}
					open={model.isBodyOpen(orphanFocus)}
					onOpenChange={(next) => model.setBodyOpen(orphanFocus, next)}
					onFocus={() => setFocus(orphanFocus)}
					onCommitted={(outcome) => onCommitted(orphanFocus, outcome)}
					onAcceptStale={() => session.acceptStale(orphanFocus)}
				/>
			</section>
		{/if}

		{#if view}
			<div class="process-stream-block">
				<div class="js-show-all-filter-root stream-show-all-filter-root">
					{#if actionableOnlyEmpty}
						<div
							class="stream-show-all-filter-empty flex flex-col items-center justify-center rounded-lg border border-dashed border-border bg-muted/60 p-4 text-center text-muted-foreground"
						>
							<p class="m-0 text-xs leading-5 font-medium text-muted-foreground">
								No matching tasks
							</p>
							<p class="m-0 text-xs leading-4 text-muted-foreground">
								Turn on Show all to see the full stream timeline.
							</p>
						</div>
					{:else}
						<div class="stream-timeline-list">
							{#each model.visibleSteps as vs (vs.step.stepId)}
								<TimelineStep projected={vs} {stream} {instance} />
							{/each}
						</div>
					{/if}
				</div>
			</div>
		{/if}
	{/if}
</div>
