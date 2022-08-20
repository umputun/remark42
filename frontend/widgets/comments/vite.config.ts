/// <reference types="vitest" />
/// <reference types="vite/client" />

import * as fs from 'fs'
import * as path from 'path'
import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

export default defineConfig({
	resolve: {
		alias: [{ find: '@tests', replacement: path.resolve(__dirname, 'tests') }],
	},
	plugins: [svelte({ hot: !process.env.VITEST })],
	server: {
		https: {
			key: fs.readFileSync('./certs/localhost-key.pem'),
			cert: fs.readFileSync('./certs/localhost.pem'),
		},
		proxy: {
			'/api': { target: 'https://demo.remark42.com', changeOrigin: true },
			'/auth': { target: 'https://demo.remark42.com', changeOrigin: true },
		},
	},
	test: {
		globals: true,
		environment: 'jsdom',
		setupFiles: ['./tests/setup.ts'],
		include: ['src/**/*.spec.ts'],
	},
})
