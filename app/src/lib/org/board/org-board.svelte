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
	import {
		BoardTimelineModel,
		setBoardTimelineModel
	} from './board-timeline-model.svelte.js';
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

	let view = $state.raw<TimelineView | null>(null);
	let error = $state('');
	let loading = $state(true);
	let streamTitle = $state('');
	let shellPainted = $state(false);
	/** After a deep-link gate deny, do not auto-focus first actionable. */
	let suppressAutoFocus = $state(false);

	/** Stay UI keyed by address. */
	let stayedByKey = $state.raw<Record<string, CommitOutcome>>({});

	const scopeKey = $derived(`${stream}|${instance}|${play ? 'play' : 'home'}`);

	const focusFromUrl = $derived.by(() => {
		const raw = searchParams.get(BOARD_TASK_SEARCH_KEY);
		return parseBoardTaskParam(raw == null ? null : String(raw));
	});

	const dashboardHref = $derived(orgMyStreamDashboardPath(stream));
	const titleText = $derived(streamTitle || stream);
	const cacheEntries = $derived(boardFormCache.entries);

	const focusedAddress = $derived(
		model.selection
			? addressFromBoardTask(stream, instance, model.selection)
			: null
	);

	const orphanFocus = $derived(model.orphanFocus);
	const playAddress = $derived(model.playAddress);
	const showAll = $derived(model.showAll);
	const actionableOnlyEmpty = $derived(model.isEmpty);

	$effect(() => {
		model.setScope(stream, instance);
		model.setPlay(play);
	});

	$effect(() => {
		model.setView(view);
	});

	function setFocus(addr: TaskAddress) {
		model.select(addr);
		const el = document.querySelector(
			`[data-address-key="${CSS.escape(taskAddressKey(addr))}"]`
		);
		el?.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
	}

	function clearFocus() {
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
					model.setPlayAddress(null);
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
					model.setPlayAddress(addr);
					boardFormCache.enqueue([addr]);
					untrack(() => {
						model.select(addr);
					});
				} else {
					model.setPlayAddress(null);
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
				model.setPlayAddress(null);
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

	// URL → selection (deep link opens body); Play keeps start selected.
	$effect(() => {
		const fromUrl = focusFromUrl;
		if (fromUrl) {
			model.setFocusFromUrl(fromUrl);
			return;
		}
		if (play && model.playAddress) {
			model.select(model.playAddress);
		}
	});

	// Orphan chrome is the task itself — open that embed.
	$effect(() => {
		const orphan = model.orphanFocus;
		if (orphan) model.setBodyOpen(orphan, true);
	});

	// After shell paint: preload actionable (+ play start already enqueued).
	$effect(() => {
		if (!shellPainted || loading) return;
		void scopeKey;
		void model.showAll;
		void model.view;
		void model.playAddress;
		void model.selection;

		const addrs: TaskAddress[] = [];
		if (model.playAddress) addrs.push(model.playAddress);
		addrs.push(...model.actionableAddresses);
		if (model.selection) {
			addrs.push(addressFromBoardTask(stream, instance, model.selection));
		}
		if (addrs.length === 0) return;
		boardFormCache.enqueue(addrs);

		// Prefer URL focus; else Play / sole actionable (multi → stay on bare board).
		if (!focusFromUrl && !model.selection && !suppressAutoFocus) {
			model.applyLoadPolicy();
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
							bind:checked={
								() => model.showAll,
								(v) => model.setShowAll(!!v)
							}
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
					focused={focusMatchesAddress(model.selection, playAddress)}
					play={true}
					stayedOutcome={stayedFor(playAddress)}
					open={model.isBodyOpen(playAddress)}
					onOpenChange={(next) => model.setBodyOpen(playAddress!, next)}
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
					open={model.isBodyOpen(orphanFocus)}
					onOpenChange={(next) => model.setBodyOpen(orphanFocus, next)}
					onFocus={() => setFocus(orphanFocus)}
					onCommitted={(outcome) => onCommitted(orphanFocus, outcome)}
					onContinue={() => onContinue(orphanFocus)}
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
							<p class="m-0 text-xs font-medium leading-5 text-muted-foreground">No matching tasks</p>
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
