import { navigate, p } from './index.js';
import type { PostCommitTarget } from '../living/worklist-nav.js';

/** Resolve ADR-0006 target to an href for `<a>` / Button `href` navigation. */
export function hrefLivingTarget(target: PostCommitTarget): string {
	if (target.kind === 'home') {
		return p('/');
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

/** Apply ADR-0006 post-Commit / exit / deep-link target. */
export function navigateLivingTarget(target: PostCommitTarget): void {
	if (target.kind === 'home') {
		void navigate('/');
		return;
	}
	if (target.kind === 'overview') {
		void navigate('/:jobId/:jobInstanceId', {
			params: {
				jobId: target.jobId,
				jobInstanceId: target.jobInstanceId
			}
		});
		return;
	}
	void navigate('/:jobId/:jobInstanceId/:taskId', {
		params: {
			jobId: target.jobId,
			jobInstanceId: target.jobInstanceId,
			taskId: target.taskId
		}
	});
}
