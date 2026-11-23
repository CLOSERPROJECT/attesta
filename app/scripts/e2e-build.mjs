/**
 * Production build for Playwright. Retries once when the known Tailwind +
 * SvelteKit postbuild analyse race leaves `build/` incomplete
 * (ERR_MODULE_NOT_FOUND on `.svelte-kit/output/server/index.js`).
 */
import { spawnSync } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const appRoot = path.join(path.dirname(fileURLToPath(import.meta.url)), '..');
const marker = path.join(appRoot, 'build', 'index.html');

function runBuild() {
	return spawnSync('bun', ['run', 'build'], {
		cwd: appRoot,
		stdio: 'inherit',
		env: process.env
	});
}

function ok() {
	return fs.existsSync(marker);
}

let result = runBuild();
if (result.status !== 0 || !ok()) {
	console.warn('[e2e-build] first build incomplete; retrying once…');
	result = runBuild();
}

if (result.status !== 0 || !ok()) {
	console.error('[e2e-build] build failed — missing', marker);
	process.exit(result.status || 1);
}
