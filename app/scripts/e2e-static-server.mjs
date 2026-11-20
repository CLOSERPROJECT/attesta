/**
 * Serve adapter-static `build/` under paths.base `/app` for Playwright.
 * Avoids `vite preview`, which reads live from `.svelte-kit/output` and can
 * die mid-suite (preview_build_missing / ENOENT → ERR_CONNECTION_REFUSED).
 */
import http from 'node:http';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import sirv from 'sirv';

const appRoot = path.join(path.dirname(fileURLToPath(import.meta.url)), '..');
const buildDir = path.join(appRoot, 'build');
const port = Number(process.env.APP_PORT) || 5180;
const host = process.env.APP_HOST || 'localhost';
const base = '/app';

if (!fs.existsSync(path.join(buildDir, 'index.html'))) {
	console.error(`Missing ${buildDir}/index.html — run bun run build first`);
	process.exit(1);
}

const serve = sirv(buildDir, {
	etag: true,
	maxAge: 0,
	dev: true
});

const server = http.createServer((req, res) => {
	const raw = req.url ?? '/';
	const q = raw.indexOf('?');
	const pathname = q === -1 ? raw : raw.slice(0, q);
	const search = q === -1 ? '' : raw.slice(q);

	if (pathname === base) {
		res.writeHead(307, { Location: `${base}/${search}` });
		res.end();
		return;
	}

	if (pathname === '/' || pathname === '') {
		res.writeHead(302, { Location: `${base}/` });
		res.end();
		return;
	}

	if (!pathname.startsWith(`${base}/`)) {
		res.statusCode = 404;
		res.end('Not found');
		return;
	}

	const stripped = pathname.slice(base.length) || '/';
	req.url = stripped + search;

	serve(req, res, () => {
		req.url = '/index.html';
		serve(req, res, () => {
			res.statusCode = 404;
			res.end('Not found');
		});
	});
});

server.listen(port, host, () => {
	console.log(`e2e static: http://${host}:${port}${base}/`);
});
