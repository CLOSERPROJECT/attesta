import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { paraglideVitePlugin } from '@inlang/paraglide-js';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vitest/config';
import { playwright } from '@vitest/browser-playwright';
import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';

const livingFixturesDir = path.resolve(
	path.dirname(fileURLToPath(import.meta.url)),
	'../server/internal/living/fixtures'
);

export default defineConfig({
	plugins: [
		tailwindcss(),
		sveltekit({
			compilerOptions: {
				// Force runes mode for the project, except for libraries. Can be removed in svelte 6.
				runes: ({ filename }) =>
					filename.split(/[/\\]/).includes('node_modules') ? undefined : true
			},
			adapter: adapter({
				pages: 'build',
				assets: 'build',
				fallback: 'index.html',
				strict: true
			}),
			paths: {
				base: '/app'
			}
		}),

		paraglideVitePlugin({
			project: './project.inlang',
			outdir: './src/lib/paraglide',
			emitTsDeclarations: true
		})
	],
	resolve: {
		alias: {
			// Single Feragame plant fixture (Go //go:embed); SPA tests import via ?raw.
			'$living-fixtures': livingFixturesDir
		}
	},
	server: {
		host: true,
		port: Number(process.env.APP_PORT) || 5180,
		strictPort: true,
		hmr: {
			host: process.env.VITE_HMR_HOST || 'localhost'
		},
		fs: {
			allow: [livingFixturesDir]
		}
	},
	preview: {
		host: true,
		port: Number(process.env.APP_PORT) || 5180,
		strictPort: true
	},
	test: {
		expect: { requireAssertions: true },
		projects: [
			{
				extends: './vite.config.ts',
				test: {
					name: 'client',
					browser: {
						enabled: true,
						provider: playwright(),
						instances: [{ browser: 'chromium', headless: true }]
					},
					include: ['src/**/*.svelte.{test,spec}.{js,ts}'],
					exclude: ['src/lib/server/**']
				}
			},

			{
				extends: './vite.config.ts',
				test: {
					name: 'server',
					environment: 'node',
					include: ['src/**/*.{test,spec}.{js,ts}'],
					exclude: ['src/**/*.svelte.{test,spec}.{js,ts}']
				}
			}
		]
	}
});
