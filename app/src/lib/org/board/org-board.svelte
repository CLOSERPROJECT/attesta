<script lang="ts">
	import { onMount, untrack } from 'svelte';
	import { searchParams } from 'sv-router';

	import * as m from '#lib/paraglide/messages.js';

	import type { CommitOutcome } from '../commit-port.js';
	import {
		applyCommitFollowThrough,
		applyContinueFollowThrough,
		decideCommitFollowThrough,
		decideContinueFollowThrough
	} from '../commit-follow-through.js';
	import { orgSession } from '../org-app.svelte.js';
	import { applySeedQueryParams } from '../seed-query.js';
	import { fetchStreamTitle } from '../stream-meta.js';
	import {
		normalizeTaskAddress,
		taskAddressKey,
		type TaskAddress
	} from '../task-address.js';
	import { createHttpTimelinePort } from '../timeline/http-timeline-port.js';
	import TimelineStep from '../timeline/timeline-step.svelte';
	import type { TimelineView } from '../timeline/types.js';
	import {
		navigateOrgTarget,
		orgMyStreamDashboardPath
	} from '../../router/navigate-target.js';
	import { navigate } from '../../router/index.js';
	import ChevronsDownUpIcon from '@lucide/svelte/icons/chevrons-down-up';
	import { Button, buttonVariants } from '#lib/shadcn/ui/button/index.js';
	import { Checkbox } from '#lib/shadcn/ui/checkbox/index.js';
	import { Label } from '#lib/shadcn/ui/label/index.js';
	import { cn } from '#lib/shadcn/utils.js';
	import { setBoardEmbed, type BoardEmbedApi } from './board-context.js';
	import { boardEmbedOpen } from './board-embed-open.svelte.js';
	import {
		boardFormCache,
		type BoardFormEntry
	} from './board-form-cache.svelte.js';
	import {
		BoardNestCollapse,
		setBoardNestCollapse
	} from './board-nest-collapse.svelte.js';
	import EmbeddedTaskPanel from './embedded-task-panel.svelte';
	import {
		addressFromBoardTask,
		BOARD_TASK_SEARCH_KEY,
		boardTaskFromAddress,
		encodeBoardTaskParam,
		focusMatchesAddress,
		parseBoardTaskParam,
		type BoardTaskFocus
	} from './board-url.js';
	import {
		actionableAddressesFromTimeline,
		addressVisibleOnTimeline,
		firstActionableAddress
	} from './timeline-actionable.js';
	import { resolveTaskDeepLink } from '../worklist-nav.js';

	import '../timeline/org-timeline.css';

	type StreamListItem = {
		key: string;
		title: string;
		play?: { jobId: string; taskId: string } | null;
	};

	type Props = {
		stream: string;
		/** Empty on Play (`/{stream}/new`). */
		instance?: string;
		play?: boolean;
	};

	let { stream, instance = '', play = false }: Props = $props();

	const nestCollapse = new BoardNestCollapse();
	setBoardNestCollapse(nestCollapse);

	const timelinePort = createHttpTimelinePort();

	let view = $state.raw<TimelineView | null>(null);
	let error = $state('');
	let loading = $state(true);
	let streamTitle = $state('');
	let showAll = $state(false);
	let shellPainted = $state(false);
	/** After a deep-link gate deny, do not auto-focus first actionable. */
	let suppressAutoFocus = $state(false);

	let playAddress = $state.raw<TaskAddress | null>(null);

	/** Stay UI keyed by address. */
	let stayedByKey = $state.raw<Record<string, CommitOutcome>>({});

	const scopeKey = $derived(`${stream}|${instance}|${play ? 'play' : 'home'}`);

	const focusFromUrl = $derived.by(() => {
		const raw = searchParams.get(BOARD_TASK_SEARCH_KEY);
		return parseBoardTaskParam(raw == null ? null : String(raw));
	});

	let focus = $state.raw<BoardTaskFocus | null>(null);

	const hasActionable = $derived(
		!!view?.steps.some(
			(step) =>
				(step.substeps ?? []).some((s) => s.orgActionable) ||
				(step.rows ?? []).some((row) => row.substeps.some((s) => s.orgActionable))
		)
	);
	const actionableOnlyEmpty = $derived(!showAll && !hasActionable && !play);
	const filterRootClass = $derived(
		[
			'js-show-all-filter-root',
			'stream-show-all-filter-root',
			!showAll ? 'is-actionable-only' : '',
			actionableOnlyEmpty ? 'is-actionable-only-empty' : ''
		]
			.filter(Boolean)
			.join(' ')
	);

	const dashboardHref = $derived(orgMyStreamDashboardPath(stream));
	const titleText = $derived(streamTitle || stream);
	const cacheEntries = $derived(boardFormCache.entries);

	const focusedAddress = $derived(
		focus ? addressFromBoardTask(stream, instance, focus) : null
	);

	/** Focused Task with no timeline card yet (e.g. provisional Can start). */
	const orphanFocus = $derived.by(() => {
		const addr = focusedAddress;
		if (!addr || play) return null;
		if (playAddress && taskAddressKey(playAddress) === taskAddressKey(addr)) {
			return null;
		}
		if (view && addressVisibleOnTimeline(view, addr)) return null;
		return addr;
	});

	function syncFocusToUrl(next: BoardTaskFocus | null) {
		if (next) {
			searchParams.set(BOARD_TASK_SEARCH_KEY, encodeBoardTaskParam(next), {
				replace: true
			});
		} else if (searchParams.get(BOARD_TASK_SEARCH_KEY) != null) {
			searchParams.delete(BOARD_TASK_SEARCH_KEY, undefined, { replace: true });
		}
	}

	function setFocus(addr: TaskAddress) {
		const next = boardTaskFromAddress(addr);
		focus = next;
		syncFocusToUrl(next);
		boardEmbedOpen.setOpen(addr, true);
		const el = document.querySelector(
			`[data-address-key="${CSS.escape(taskAddressKey(addr))}"]`
		);
		el?.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
	}

	function isEmbedOpen(addr: TaskAddress): boolean {
		void boardEmbedOpen.closed;
		return boardEmbedOpen.isOpen(addr);
	}

	function setEmbedOpen(addr: TaskAddress, open: boolean): void {
		boardEmbedOpen.setOpen(addr, open);
	}

	function toggleEmbedOpen(addr: TaskAddress): boolean {
		return boardEmbedOpen.toggle(addr);
	}

	function clearFocus() {
		focus = null;
		syncFocusToUrl(null);
	}

	function collapseAllNests() {
		nestCollapse.collapseAll();
		if (view) {
			boardEmbedOpen.closeAll(actionableAddressesFromTimeline(view, { showAll: true }));
		}
		if (playAddress) boardEmbedOpen.setOpen(playAddress, false);
		if (orphanFocus) boardEmbedOpen.setOpen(orphanFocus, false);
	}

	function getEntry(addr: TaskAddress): BoardFormEntry | undefined {
		void cacheEntries;
		return boardFormCache.get(taskAddressKey(addr));
	}

	function stayedFor(addr: TaskAddress): CommitOutcome | null {
		return stayedByKey[taskAddressKey(addr)] ?? null;
	}

	function clearStay(addr?: TaskAddress) {
		if (!addr) {
			stayedByKey = {};
			return;
		}
		const key = taskAddressKey(addr);
		if (!(key in stayedByKey)) return;
		const next = { ...stayedByKey };
		delete next[key];
		stayedByKey = next;
	}

	async function softNavToInstance(
		nextInstance: string,
		opts: { focus?: BoardTaskFocus | null; bundleAddr?: TaskAddress; bundle?: CommitOutcome['nextTaskFormBundle'] } = {}
	) {
		if (opts.bundle && opts.bundleAddr) {
			boardFormCache.putReady(opts.bundleAddr, opts.bundle);
		}
		const search =
			opts.focus != null
				? { [BOARD_TASK_SEARCH_KEY]: encodeBoardTaskParam(opts.focus) }
				: undefined;
		await navigate('/:stream/i/:instance', {
			params: { stream, instance: nextInstance },
			search,
			replace: true
		});
	}

	function onCommitted(addr: TaskAddress, outcome: CommitOutcome) {
		const decision = decideCommitFollowThrough(outcome, addr);
		if (decision.kind === 'stay') {
			stayedByKey = { ...stayedByKey, [taskAddressKey(addr)]: outcome };
			return;
		}

		clearStay(addr);
		const target = decision.target;

		if (target.kind === 'task') {
			const nextAddr = normalizeTaskAddress(target);
			if (outcome.nextTaskFormBundle) {
				boardFormCache.putReady(nextAddr, outcome.nextTaskFormBundle);
			}
			if (play && !instance && target.instance) {
				void softNavToInstance(target.instance, {
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
						void orgSession.hydrate({
							stream: t.stream,
							instance: t.instance
						});
						if (view) {
							void reloadTimeline();
						}
						return;
					}
					navigateOrgTarget(t);
				}
			});
			return;
		}

		if (target.kind === 'home') {
			if (play && !instance) {
				void softNavToInstance(target.instance, { focus: null });
				orgSession.clearLinksStay();
				return;
			}
			clearFocus();
			orgSession.clearLinksStay();
			void reloadTimeline();
			void orgSession.hydrate({ stream: target.stream, instance: target.instance });
			return;
		}

		applyCommitFollowThrough(decision, {
			clearLinksStay: () => orgSession.clearLinksStay(),
			navigate: navigateOrgTarget
		});
	}

	async function onContinue(addr: TaskAddress) {
		const outcome = stayedFor(addr);
		if (!outcome) return;
		const decision = decideContinueFollowThrough(outcome, addr);
		clearStay(addr);
		await applyContinueFollowThrough(decision, {
			clearLinksStay: () => orgSession.clearLinksStay(),
			navigate: (t) => {
				if (t.kind === 'task') {
					const nextAddr = normalizeTaskAddress(t);
					if (outcome.nextTaskFormBundle) {
						boardFormCache.putReady(nextAddr, outcome.nextTaskFormBundle);
					}
					setFocus(nextAddr);
					void reloadTimeline();
					return;
				}
				if (t.kind === 'home') {
					clearFocus();
					void reloadTimeline();
					return;
				}
				navigateOrgTarget(t);
			},
			reload: async () => {
				boardFormCache.invalidate(taskAddressKey(addr));
				boardFormCache.enqueue([addr]);
			}
		});
	}

	async function reloadTimeline() {
		if (!instance && !play) return;
		if (!instance) return;
		try {
			view = await timelinePort.load({ stream, instance });
		} catch {
			/* keep prior view */
		}
	}

	const boardApi: BoardEmbedApi = {
		getEntry,
		get focus() {
			return focus;
		},
		isFocused(addr) {
			return focusMatchesAddress(focus, addr);
		},
		setFocus,
		isEmbedOpen,
		setEmbedOpen,
		toggleEmbedOpen,
		stayedFor,
		onCommitted,
		onContinue,
		get play() {
			return play;
		}
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

	// Load timeline (instance) or play start form.
	$effect(() => {
		const s = stream;
		const i = instance;
		const isPlay = play;
		let cancelled = false;
		loading = true;
		error = '';
		boardFormCache.beginScope(`${s}|${i}|${isPlay ? 'play' : 'home'}`);

		void (async () => {
			try {
				if (isPlay) {
					view = null;
					playAddress = null;
					const res = await fetch('/api/org/streams', {
						credentials: 'include',
						headers: { Accept: 'application/json' },
						cache: 'no-store'
					});
					if (!res.ok) throw new Error(await res.text());
					const items = (await res.json()) as StreamListItem[];
					const row = items.find((item) => item.key === s);
					const anchor = row?.play;
					if (!anchor?.jobId || !anchor.taskId) {
						throw new Error('play start not found');
					}
					if (cancelled) return;
					streamTitle = row?.title?.trim() || s;
					const stripped = applySeedQueryParams(window.location.href, orgSession, {
						jobId: anchor.jobId,
						taskId: anchor.taskId,
						jobInstanceId: ''
					});
					if (stripped) {
						window.history.replaceState(window.history.state, '', stripped);
					}
					const addr = normalizeTaskAddress({
						stream: s,
						instance: '',
						jobId: anchor.jobId,
						jobInstanceId: '',
						taskId: anchor.taskId
					});
					playAddress = addr;
					boardFormCache.enqueue([addr]);
					untrack(() => {
						focus = boardTaskFromAddress(addr);
						syncFocusToUrl(focus);
					});
				} else {
					playAddress = null;
					const next = await timelinePort.load({ stream: s, instance: i });
					if (cancelled) return;
					view = next;
					if (i) {
						void orgSession.hydrate({ stream: s, instance: i });
					}
				}
			} catch (e: unknown) {
				if (cancelled) return;
				view = null;
				playAddress = null;
				error =
					e instanceof Error
						? e.message
						: m.org_timeline_load_failed({ status: 0 });
			} finally {
				if (!cancelled) loading = false;
			}
		})();

		return () => {
			cancelled = true;
		};
	});

	// URL → focus (deep link / soft focus).
	$effect(() => {
		const fromUrl = focusFromUrl;
		if (fromUrl) {
			focus = fromUrl;
			return;
		}
		if (play && playAddress) {
			focus = boardTaskFromAddress(playAddress);
		}
	});

	// After shell paint: preload actionable (+ play start already enqueued).
	$effect(() => {
		if (!shellPainted || loading) return;
		void scopeKey;
		void showAll;
		void view;
		void playAddress;

		const addrs: TaskAddress[] = [];
		if (playAddress) addrs.push(playAddress);
		if (view) {
			addrs.push(...actionableAddressesFromTimeline(view, { showAll }));
		}
		if (focus) {
			addrs.push(addressFromBoardTask(stream, instance, focus));
		}
		if (addrs.length === 0) return;
		boardFormCache.enqueue(addrs);

		// Prefer URL focus; else first actionable / play (unless gate just denied).
		if (!focusFromUrl && !focus && !suppressAutoFocus) {
			const first = playAddress ?? (view ? firstActionableAddress(view) : null);
			if (first) {
				focus = boardTaskFromAddress(first);
				syncFocusToUrl(focus);
			}
		}
	});

	// Deep-link gate once Worklist is available (unauthorized / unknown → clear focus).
	$effect(() => {
		const worklist = orgSession.result?.worklist;
		const addr = focusedAddress;
		if (!worklist || !addr || play) return;
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
	data-testid={play ? 'play-route' : 'org-timeline'}
	data-org-board="1"
	data-show-all={showAll ? '1' : '0'}
	data-route-key={playAddress ? taskAddressKey(playAddress) : scopeKey}
>
	{#if play}
		<div class="org-timeline-toolbar" data-testid="play-chrome">
			<header class="org-timeline-header space-y-1">
				<p class="org-timeline-header-label" data-testid="play-chrome-stream">{titleText}</p>
				<a
					href={dashboardHref}
					class="inline-block text-sm text-zinc-600 underline-offset-2 hover:underline"
					data-testid="play-chrome-dashboard"
					data-sveltekit-reload
					rel="external"
					target="_self"
				>
					{m.back_to_stream_dashboard()}
				</a>
			</header>
		</div>
	{:else}
		<div class="org-timeline-toolbar" data-testid="instance-chrome">
			<header class="org-timeline-header">
				<span class="timeline-header-line">
					<span class="timeline-header-index" data-testid="instance-chrome-stream">
						{streamTitle || stream}
					</span>
					<span class="timeline-header-title" data-testid="instance-chrome-label">
						{view?.label || instance}
					</span>
				</span>
			</header>
			{#if view}
				<div class="stream-show-all-filter">
					<Label
						class={cn(buttonVariants({ variant: 'outline', size: 'sm' }), 'cursor-pointer')}
					>
						<Checkbox
							class="js-show-all-filter border-foreground/35 bg-background shadow-none"
							bind:checked={showAll}
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
	{/if}

	{#if orgSession.commitError}
		<p class="text-sm text-red-700" data-testid="org-commit-error" role="alert">
			{m.commit_denied()}: {orgSession.commitError}
		</p>
	{/if}

	{#if loading && !view && !playAddress}
		<p
			class="text-sm text-zinc-500"
			data-testid={play ? 'play-load-status' : 'org-timeline-status'}
		>
			{m.loading()}
		</p>
	{:else if error && !view && !playAddress}
		<p
			class="text-sm text-red-700"
			data-testid={play ? 'play-load-error' : 'org-timeline-error'}
			role="alert"
		>
			{error}
		</p>
	{:else}
		{#if play && playAddress}
			<section class="mb-6" data-testid="play-embed">
				<EmbeddedTaskPanel
					address={playAddress}
					entry={getEntry(playAddress)}
					focused={focusMatchesAddress(focus, playAddress)}
					play={true}
					stayedOutcome={stayedFor(playAddress)}
					open={isEmbedOpen(playAddress)}
					onOpenChange={(next) => setEmbedOpen(playAddress!, next)}
					onFocus={() => setFocus(playAddress!)}
					onCommitted={(outcome) => onCommitted(playAddress!, outcome)}
					onContinue={() => onContinue(playAddress!)}
				/>
			</section>
		{/if}

		{#if orphanFocus}
			<section class="mb-6" data-testid="orphan-focus-embed">
				<EmbeddedTaskPanel
					address={orphanFocus}
					entry={getEntry(orphanFocus)}
					focused={true}
					stayedOutcome={stayedFor(orphanFocus)}
					open={isEmbedOpen(orphanFocus)}
					onOpenChange={(next) => setEmbedOpen(orphanFocus, next)}
					onFocus={() => setFocus(orphanFocus)}
					onCommitted={(outcome) => onCommitted(orphanFocus, outcome)}
					onContinue={() => onContinue(orphanFocus)}
				/>
			</section>
		{/if}

		{#if view}
			<div class="process-stream-block">
				<div class={filterRootClass}>
					<div class="empty-state stream-show-all-filter-empty" hidden={!actionableOnlyEmpty}>
						<p class="empty-state-title">No matching tasks</p>
						<p class="empty-state-hint">Turn on Show all to see the full stream timeline.</p>
					</div>
					<div class="stream-timeline-list">
						{#each view.steps as step (step.stepId)}
							<TimelineStep {step} {stream} {instance} />
						{/each}
					</div>
				</div>
			</div>
		{/if}
	{/if}
</div>
