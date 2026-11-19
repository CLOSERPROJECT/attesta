import { defineConfig } from '@playwright/test';

const port = Number(process.env.APP_PORT) || 5180;
const baseURL = `http://localhost:${port}`;

export default defineConfig({
	webServer: {
		// Static `build/` under /app — not `vite preview` (fragile `.svelte-kit/output`).
		command: `bun run build:e2e && bun run preview:e2e`,
		url: `${baseURL}/app/`,
		reuseExistingServer: !process.env.CI,
		timeout: 180_000
	},
	use: {
		baseURL
	},
	testMatch: '**/*.e2e.{ts,js}'
});
