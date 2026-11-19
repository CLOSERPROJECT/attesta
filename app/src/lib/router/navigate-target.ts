import { navigate, p } from './index.js';
import type { PostCommitTarget } from '../org/worklist-nav.js';

/** Org catalog key for /my stream dashboard (matches server orgCatalogKey). */
export function orgMyStreamDashboardPath(stream: string): string {
	return `/my/streams/org-${stream}/`;
}

/** Org catalog instance timeline — Commit `home` and instance route land here. */
export function orgMyInstanceTimelinePath(stream: string, instance: string): string {
	return `/my/streams/org-${stream}/instance/${instance}`;
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
		return orgMyInstanceTimelinePath(target.stream, target.instance);
	}
	return p('/:stream/i/:instance/:jobId/:jobInstanceId/:taskId', {
		params: {
			stream: target.stream,
			instance: target.instance,
			jobId: target.jobId,
			jobInstanceId: target.jobInstanceId,
			taskId: target.taskId
		}
	});
}

/** Programmatic navigate; return value is for `throw` inside sv-router hooks. */
export function navigateOrgTargetAsync(target: PostCommitTarget): Promise<unknown> {
	if (target.kind === 'my' || target.kind === 'myStream' || target.kind === 'home') {
		const href = hrefOrgTarget(target);
		if (typeof window !== 'undefined') {
			window.location.assign(href);
		}
		return Promise.resolve();
	}
	return navigate('/:stream/i/:instance/:jobId/:jobInstanceId/:taskId', {
		params: {
			stream: target.stream,
			instance: target.instance,
			jobId: target.jobId,
			jobInstanceId: target.jobInstanceId,
			taskId: target.taskId
		}
	});
}

/** Apply ADR-0006 post-Commit / exit / deep-link target. */
export function navigateOrgTarget(target: PostCommitTarget): void {
	void navigateOrgTargetAsync(target);
}
