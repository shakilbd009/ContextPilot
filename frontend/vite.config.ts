/// <reference types="vitest" />
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';
import { svelteTesting } from '@testing-library/svelte/vite';

export default defineConfig({
	plugins: [sveltekit(), svelteTesting()],
	server: {
		// No Vite proxy. The SvelteKit dev server handles every /api/*
		// request via its own +server.ts handlers — those handlers proxy
		// to the Go backend (via $env/dynamic/private SERVER_URL) when the
		// data lives there (e.g. /api/meetings → /api/v1/meetings). Having
		// Vite also proxy /api to host.docker.internal:3000 silently
		// hijacks the SvelteKit handlers, sending requests to the Go
		// backend which only knows /api/v1/*, and they 404 / time out.
		// This is the root cause of every browser-originated
		// fetch('/api/upcoming', ...) / fetch('/api/meetings', ...)
		// failure in the E2E suite: the SvelteKit handler never runs.
		// See commit ops(t_b48e503b).
	},
	// @ts-expect-error — test block is recognized by vitest (via the triple-slash reference above) but not by Vite's UserConfigExport type
	test: {
		include: ['src/**/*.{test,spec}.{js,ts}'],
		environment: 'jsdom',
		globals: true,
		setupFiles: ['./src/test-setup.ts'],
		coverage: {
			provider: 'v8',
			include: ['src/lib/**'],
			exclude: ['node_modules', 'src/test-setup.ts'],
		},
		resolve: {
			conditions: ['browser', 'import', 'module', 'default'],
		},
	},
});
