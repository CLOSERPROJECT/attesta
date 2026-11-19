import { describe, expect, test } from 'vitest';

import feragameConfigYaml from '$org-fixtures/feragame-config.yaml?raw';
import { loadFeragameConfig, taskConfig } from './form/load-feragame-config.js';

function taskRoles(jobId: string, taskId: string): string[] {
	const config = loadFeragameConfig(feragameConfigYaml);
	const task = taskConfig(config.jobs[jobId], taskId);
	expect(task, `${jobId}/${taskId}`).toBeTruthy();
	const roles = task!.roles;
	expect(Array.isArray(roles), `${jobId}/${taskId}.roles`).toBe(true);
	return roles as string[];
}

describe('feragame plant fixture ($org-fixtures)', () => {
	const config = loadFeragameConfig(feragameConfigYaml);

	test('exactly two Jobs: intake, lot_processing', () => {
		expect(Object.keys(config.jobs).sort()).toEqual(['intake', 'lot_processing']);
		expect(config.jobs).not.toHaveProperty('line');
		expect(config.jobs).not.toHaveProperty('workingDay');
		expect(config.jobs).not.toHaveProperty('passport');
	});

	test('lot_processing Task inventory matches LOCKED', () => {
		expect(config.jobs.lot_processing.tasks.map((t) => t.id)).toEqual([
			'startLot',
			'weigh',
			'markProgress',
			'containerReg',
			'closeLot',
			'lock'
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
		expect(taskRoles('lot_processing', 'lock')).toEqual(['administration']);
	});

	test('intake and lot_processing/lock readyWhen fold sibling gates into Domain', () => {
		const register = taskConfig(config.jobs.intake, 'register')!;
		const weightCheck = taskConfig(config.jobs.intake, 'weightCheck')!;
		const lock = taskConfig(config.jobs.lot_processing, 'lock')!;
		expect(String(register.readyWhen)).toContain('truckArrive');
		expect(String(register.readyWhen)).toContain('weightCheck');
		expect(String(weightCheck.readyWhen)).toContain('truckArrive');
		expect(String(weightCheck.readyWhen)).toContain('lot');
		expect(String(lock.readyWhen)).not.toContain('select');
		expect(String(lock.readyWhen)).toContain('fullyProcessed');
		expect(String(lock.readyWhen)).toContain('locked');
		expect(lock.title).toBe('Verifica dati e DPP');
	});
});
