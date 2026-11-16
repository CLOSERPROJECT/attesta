import { describe, expect, test } from 'vitest';

import feragameConfigYaml from '$living-fixtures/feragame-config.yaml?raw';
import { loadFeragameConfig, taskConfig } from './form/load-feragame-config.js';

function taskRoles(jobId: string, taskId: string): string[] {
	const config = loadFeragameConfig(feragameConfigYaml);
	const task = taskConfig(config.jobs[jobId], taskId);
	expect(task, `${jobId}/${taskId}`).toBeTruthy();
	const roles = task!.roles;
	expect(Array.isArray(roles), `${jobId}/${taskId}.roles`).toBe(true);
	return roles as string[];
}

describe('feragame plant fixture ($living-fixtures)', () => {
	const config = loadFeragameConfig(feragameConfigYaml);

	test('exactly three Jobs: intake, lot_processing, passport', () => {
		expect(Object.keys(config.jobs).sort()).toEqual([
			'intake',
			'lot_processing',
			'passport'
		]);
		expect(config.jobs).not.toHaveProperty('line');
		expect(config.jobs).not.toHaveProperty('workingDay');
	});

	test('lot_processing Task inventory matches LOCKED', () => {
		expect(config.jobs.lot_processing.tasks.map((t) => t.id)).toEqual([
			'startLot',
			'weigh',
			'markProgress',
			'containerReg',
			'closeLot'
		]);
		expect(config.jobs.lot_processing.tasks.map((t) => t.id)).not.toContain('openTurno');
		expect(config.jobs.lot_processing.tasks.map((t) => t.id)).not.toContain('closeTurno');
		expect(config.jobs.lot_processing.instanceId).toBe('$.lotId');
		expect(config.jobs.lot_processing.startLabel).toBe('Apri lotto');
	});

	test('role gates match LOCKED', () => {
		expect(taskRoles('intake', 'truckArrive')).toEqual(['administration']);
		expect(taskRoles('intake', 'register')).toEqual(['administration']);
		expect(taskRoles('intake', 'weightCheck')).toEqual(['administration']);

		expect(taskRoles('lot_processing', 'startLot')).toEqual(['logistic']);
		expect(taskRoles('lot_processing', 'weigh')).toEqual(['line']);
		expect(taskRoles('lot_processing', 'markProgress')).toEqual(['line']);
		expect(taskRoles('lot_processing', 'containerReg')).toEqual(['logistic']);
		expect(taskRoles('lot_processing', 'closeLot')).toEqual(['logistic']);

		expect(taskRoles('passport', 'select')).toEqual(['administration']);
		expect(taskRoles('passport', 'lock')).toEqual(['administration']);
	});
});
