/// <reference types="vitest" />
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [sveltekit()],
	server: {
		proxy: {
			'/api': {
				target: 'http://host.docker.internal:3000',
				changeOrigin: true,
				secure: false,
			},
		},
	},
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