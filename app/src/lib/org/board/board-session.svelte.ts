import type { CommitOutcome } from '../commit-port.js';
import type { TaskFormBundle } from '../form/task-form-bundle.js';
import { taskAddressKey, type TaskAddress } from '../task-address.js';
import type { TimelineLoadQuery, TimelinePort } from '../timeline/timeline-port.js';
import type { TimelineView } from '../timeline/types.js';
import type { DomainChange } from './board-refresh-plan.js';
import { planBoardRefresh } from './board-refresh-plan.js';
import type { BoardFormCache } from './board-form-cache.svelte.js';
import type { BoardRecapCache } from './board-recap-cache.svelte.js';
import type { BoardTimelineModel } from './board-timeline-model.svelte.js';
import type { BoardTaskFocus } from './board-url.js';

/** Route-edge scope: boot (Play / not-instanced) or minted stream instance. */
export type BoardScope =
	| { kind: 'boot'; stream: string }
	| { kind: 'instance'; stream: string; instance: string };

export type AdoptInstanceOptions = {
	focus?: BoardTaskFocus | null;
	bundleAddr?: TaskAddress;
	bundle?: CommitOutcome['nextTaskFormBundle'] | TaskFormBundle | null;
};

export type NavigateToInstance = (args: {
	stream: string;
	instance: string;
	focus?: BoardTaskFocus | null;
}) => Promise<void>;

export type BoardSessionOptions = {
	scope: BoardScope;
	timeline: TimelinePort;
	model: BoardTimelineModel;
	formCache: BoardFormCache;
	recapCache: BoardRecapCache;
	navigateToInstance: NavigateToInstance;
	hydrate?: (args: { stream: string; instance: string }) => void;
};

export type BoardSession = {
	readonly scope: BoardScope;
	readonly scopeKey: string;
	view: TimelineView | null;
	loading: boolean;
	error: string;
	load(): Promise<void>;
	reload(): Promise<void>;
	/**
	 * The Domain changed (own Commit today; remote writes later). Refreshes the
	 * projections planBoardRefresh selects. Resolves when the refresh finished.
	 */
	domainChanged(change: DomainChange): Promise<void>;
	adoptInstance(instance: string, opts?: AdoptInstanceOptions): Promise<void>;
	/** Sync session when route props change (without soft-nav). */
	setScope(scope: BoardScope): void;
};

export function boardScopeKey(scope: BoardScope): string {
	if (scope.kind === 'boot') return `${scope.stream}||boot`;
	return `${scope.stream}|${scope.instance}|instance`;
}

export function timelineQueryForScope(scope: BoardScope): TimelineLoadQuery {
	if (scope.kind === 'boot') return { stream: scope.stream };
	return { stream: scope.stream, instance: scope.instance };
}

function streamOf(scope: BoardScope): string {
	return scope.stream;
}

function instanceOf(scope: BoardScope): string {
	return scope.kind === 'instance' ? scope.instance : '';
}

/**
 * Owns scope key, timeline load/reload, Domain-change projection refresh
 * (domainChanged), cache beginScope, and Play→instance soft-nav.
 * OrgBoard paints from session.view / model; GUI must not invent play via /streams.
 */
export function createBoardSession(opts: BoardSessionOptions): BoardSession {
	let scope = $state.raw(opts.scope);
	let view = $state.raw<TimelineView | null>(null);
	let loading = $state(false);
	let error = $state('');
	let gen = 0;

	const { timeline, model, formCache, recapCache, navigateToInstance, hydrate } =
		opts;

	function applyScopeToModel(next: BoardScope) {
		model.setScope(streamOf(next), instanceOf(next));
	}

	applyScopeToModel(opts.scope);

	async function fetchView(
		query: TimelineLoadQuery,
		mode: 'load' | 'reload'
	): Promise<void> {
		const myGen = ++gen;
		if (mode === 'load') {
			loading = true;
			error = '';
			const key = boardScopeKey(scope);
			formCache.beginScope(key);
			recapCache.beginScope(key);
		}
		applyScopeToModel(scope);

		try {
			const next = await timeline.load(query);
			if (myGen !== gen) return;
			view = next;
			model.setView(next);
			error = '';
			if (scope.kind === 'instance') {
				hydrate?.({ stream: scope.stream, instance: scope.instance });
			}
		} catch (e: unknown) {
			if (myGen !== gen) return;
			if (mode === 'load') {
				view = null;
				model.setView(null);
				error = e instanceof Error ? e.message : String(e);
			}
			/* reload: keep prior view */
		} finally {
			if (myGen === gen && mode === 'load') {
				loading = false;
			}
		}
	}

	return {
		get scope() {
			return scope;
		},
		get scopeKey() {
			return boardScopeKey(scope);
		},
		get view() {
			return view;
		},
		get loading() {
			return loading;
		},
		get error() {
			return error;
		},
		setScope(next: BoardScope) {
			scope = next;
			applyScopeToModel(next);
		},
		async load() {
			await fetchView(timelineQueryForScope(scope), 'load');
		},
		async reload() {
			await fetchView(timelineQueryForScope(scope), 'reload');
		},
		async domainChanged(change: DomainChange) {
			const plan = planBoardRefresh([change], {
				scopeKind: scope.kind,
				hasView: view !== null
			});
			if (plan.remountForms) {
				formCache.refreshAllAfterDomainCommit();
			}
			for (const addr of plan.recapRefresh) {
				recapCache.invalidate(taskAddressKey(addr));
				recapCache.enqueue([addr]);
			}
			if (plan.timeline) {
				await fetchView(timelineQueryForScope(scope), 'reload');
			} else if (plan.worklistOnly && scope.kind === 'instance') {
				hydrate?.({ stream: scope.stream, instance: scope.instance });
			}
		},
		async adoptInstance(instance: string, adoptOpts: AdoptInstanceOptions = {}) {
			if (adoptOpts.bundle && adoptOpts.bundleAddr) {
				formCache.putReady(adoptOpts.bundleAddr, adoptOpts.bundle);
			}
			const stream = streamOf(scope);
			scope = { kind: 'instance', stream, instance };
			applyScopeToModel(scope);
			await navigateToInstance({
				stream,
				instance,
				focus: adoptOpts.focus
			});
		}
	};
}
