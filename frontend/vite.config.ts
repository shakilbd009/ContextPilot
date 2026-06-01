/// <reference types="vitest" />
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';
import { svelteTesting } from '@testing-library/svelte/vite';

export default defineConfig({
	plugins: [sveltekit(), svelteTesting()],
	server: {
		proxy: {
			'/api': {
				target: 'http://host.docker.internal:3000',
				changeOrigin: true,
				secure: false,
			},
		},
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
