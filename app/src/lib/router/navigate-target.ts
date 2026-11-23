import { navigate, p } from './index.js';
import {
	BOARD_TASK_SEARCH_KEY,
	encodeBoardTaskParam
} from '../org/board/board-url.js';
import type { PostCommitTarget } from '../org/worklist-nav.js';

/** Org catalog key for /my stream dashboard (matches server orgCatalogKey). */
export function orgMyStreamDashboardPath(stream: string): string {
	return `/my/streams/org-${stream}/`;
}

/** SPA instance home (timeline board). Prefer this over legacy /my Org instance URLs. */
export function orgSpaInstanceHomePath(stream: string, instance: string): string {
	return p('/:stream/i/:instance', {
		params: { stream, instance }
	});
}

/** @deprecated Use orgSpaInstanceHomePath — kept for call sites that still name /my. */
export function orgMyInstanceTimelinePath(stream: string, instance: string): string {
	return orgSpaInstanceHomePath(stream, instance);
}

/**
 * Board soft-focus href: `/{stream}/i/{instance}?task=jobId/jobInstanceId/taskId`.
 * Legacy path-shaped Task URLs rewrite here so the board stays mounted.
 */
export function orgSpaBoardTaskPath(target: {
	stream: string;
	instance: string;
	jobId: string;
	jobInstanceId: string;
	taskId: string;
}): string {
	const base = orgSpaInstanceHomePath(target.stream, target.instance);
	const task = encodeBoardTaskParam({
		jobId: target.jobId,
		jobInstanceId: target.jobInstanceId,
		taskId: target.taskId
	});
	return `${base}?${BOARD_TASK_SEARCH_KEY}=${encodeURIComponent(task)}`;
}

/** Resolve ADR-0006 / ADR-0014 target to an href for `<a>` / Button `href` navigation. */
export function hrefOrgTarget(target: PostCommitTarget): string {
	if (target.kind === 'my') {
		return '/my';
	}
	if (target.kind === 'myStream') {
		return orgMyStreamDashboardPath(target.stream);
	}
	if (target.kind === 'home') {
		return orgSpaInstanceHomePath(target.stream, target.instance);
	}
	return orgSpaBoardTaskPath(target);
}

/** Programmatic navigate; return value is for `throw` inside sv-router hooks. */
export function navigateOrgTargetAsync(target: PostCommitTarget): Promise<unknown> {
	if (target.kind === 'my' || target.kind === 'myStream') {
		const href = hrefOrgTarget(target);
		if (typeof window !== 'undefined') {
			window.location.assign(href);
		}
		return Promise.resolve();
	}
	if (target.kind === 'home') {
		return navigate('/:stream/i/:instance', {
			params: {
				stream: target.stream,
				instance: target.instance
			}
		});
	}
	return navigate('/:stream/i/:instance', {
		params: {
			stream: target.stream,
			instance: target.instance
		},
		search: {
			[BOARD_TASK_SEARCH_KEY]: encodeBoardTaskParam({
				jobId: target.jobId,
				jobInstanceId: target.jobInstanceId,
				taskId: target.taskId
			})
		}
	});
}

/** Apply ADR-0006 post-Commit / exit / deep-link target. */
export function navigateOrgTarget(target: PostCommitTarget): void {
	void navigateOrgTargetAsync(target);
}
