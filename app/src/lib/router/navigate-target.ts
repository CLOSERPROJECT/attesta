import { navigate } from './index.js';
import type { PostCommitTarget } from '../living/worklist-overview.js';

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
