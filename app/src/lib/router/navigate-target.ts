import { navigate, p } from './index.js';
import type { PostCommitTarget } from '../org/worklist-nav.js';

/** Resolve ADR-0006 target to an href for `<a>` / Button `href` navigation. */
export function hrefOrgTarget(target: PostCommitTarget): string {
	if (target.kind === 'home') {
		return p('/');
	}
	if (target.kind === 'archive') {
		return p('/archive');
	}
	if (target.kind === 'overview') {
		return p('/:jobId/:jobInstanceId', {
			params: {
				jobId: target.jobId,
				jobInstanceId: target.jobInstanceId
			}
		});
	}
	return p('/:jobId/:jobInstanceId/:taskId', {
		params: {
			jobId: target.jobId,
			jobInstanceId: target.jobInstanceId,
			taskId: target.taskId
		}
	});
}

/** Programmatic navigate; return value is for `throw` inside sv-router hooks. */
export function navigateOrgTargetAsync(target: PostCommitTarget): Promise<unknown> {
	if (target.kind === 'home') {
		return navigate('/');
	}
	if (target.kind === 'archive') {
		return navigate('/archive');
	}
	if (target.kind === 'overview') {
		return navigate('/:jobId/:jobInstanceId', {
			params: {
				jobId: target.jobId,
				jobInstanceId: target.jobInstanceId
			}
		});
	}
	return navigate('/:jobId/:jobInstanceId/:taskId', {
		params: {
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
