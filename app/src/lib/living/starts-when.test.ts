import { describe, expect, test } from 'vitest';

import {
	canStartJob,
	isStartsWhenSatisfied,
	referencedStartsWhenTask
} from './starts-when.js';
import { emptyDomain, type OrgConfig, type Submission, type TaskConfig } from './types.js';

const palletWeightTask: TaskConfig = {
	id: 'palletWeight',
	dynamicProperties: { weights: { read: 'leftovers', completeness: 'all' } }
};

const lineLoadSomeTask: TaskConfig = {
	id: 'lineLoad',
	dynamicProperties: { progress: { read: 'leftovers', completeness: 'some' } }
};

const config: OrgConfig = {
	id: 'truck-loop',
	version: '1',
	jobs: {
		intake: {
			repeatable: true,
			label: 'intake',
			tasks: [palletWeightTask]
		},
		line: {
			repeatable: true,
			label: 'line',
			startLabel: 'Nuovo turno',
			startsWhen: {
				task: { jobId: 'intake', taskId: 'palletWeight' },
				quantifier: 'some'
			},
			tasks: [{ id: 'shift' }, lineLoadSomeTask]
		}
	}
};

function sub(jobId: string, taskId: string, jobInstanceId: string): Submission {
	return {
		jobId,
		taskId,
		jobInstanceId,
		timestamp: 't',
		userId: 'u',
		facts: {},
		idsTouched: []
	};
}

describe('isStartsWhenSatisfied', () => {
	test('absent gate allows', () => {
		expect(isStartsWhenSatisfied(null, [], [], palletWeightTask)).toBe(true);
	});

	test('some requires matching Submission on eligible Task', () => {
		const gate = {
			task: { jobId: 'intake', taskId: 'palletWeight' },
			quantifier: 'some'
		};
		expect(isStartsWhenSatisfied(gate, [], [], palletWeightTask)).toBe(false);
		expect(
			isStartsWhenSatisfied(
				gate,
				[sub('intake', 'palletWeight', 'load-1')],
				[{ jobId: 'intake', id: 'load-1', label: 'load-1' }],
				palletWeightTask
			)
		).toBe(true);
	});

	test('completeness some Task never satisfies', () => {
		const gate = {
			task: { jobId: 'line', taskId: 'lineLoad' },
			quantifier: 'some'
		};
		expect(
			isStartsWhenSatisfied(
				gate,
				[sub('line', 'lineLoad', '1st')],
				[{ jobId: 'line', id: '1st', label: 'Turno: 1st' }],
				lineLoadSomeTask
			)
		).toBe(false);
	});
});

describe('canStartJob', () => {
	test('line blocked until intake palletWeight gate', () => {
		const domain = emptyDomain();
		const line = config.jobs.line!;
		expect(canStartJob({ job: line, jobId: 'line', domain, config })).toBe(false);

		domain.submissions = [sub('intake', 'palletWeight', 'NIR-1')];
		domain.jobInstances = [{ jobId: 'intake', id: 'NIR-1', label: 'NIR-1' }];
		expect(canStartJob({ job: line, jobId: 'line', domain, config })).toBe(true);
	});

	test('repeatable stays startable after first instance', () => {
		const domain = emptyDomain();
		domain.submissions = [sub('intake', 'palletWeight', 'NIR-1')];
		domain.jobInstances = [
			{ jobId: 'intake', id: 'NIR-1', label: 'NIR-1' },
			{ jobId: 'line', id: '1st', label: 'Turno: 1st' }
		];
		expect(
			canStartJob({
				job: config.jobs.line!,
				jobId: 'line',
				domain,
				config
			})
		).toBe(true);
	});

	test('referencedStartsWhenTask reads intake palletWeight', () => {
		expect(referencedStartsWhenTask(config, config.jobs.line!.startsWhen)).toBe(
			palletWeightTask
		);
	});
});
