import { defineConfig } from '@playwright/test';

const port = Number(process.env.APP_PORT) || 5180;

export default defineConfig({
	webServer: {
		command: `bun run build && bun run preview -- --port ${port} --strictPort`,
		port,
		reuseExistingServer: !process.env.CI
	},
	use: {
		// Vite preview binds localhost (often IPv6-only); 127.0.0.1 refuses.
		baseURL: `http://localhost:${port}`
	},
	testMatch: '**/*.e2e.{ts,js}'
});
