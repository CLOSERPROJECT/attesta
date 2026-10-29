import { defineConfig } from '@playwright/test';

const port = Number(process.env.APP_PORT) || 5180;

export default defineConfig({
	webServer: {
		command: `bun run build && bun run preview -- --port ${port} --strictPort`,
		port,
		reuseExistingServer: !process.env.CI
	},
	use: {
		baseURL: `http://127.0.0.1:${port}`
	},
	testMatch: '**/*.e2e.{ts,js}'
});
