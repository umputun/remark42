import { defineConfig } from 'vitest/config'

export default defineConfig({
	test: {
		environment: 'jsdom',
		environmentOptions: {
			jsdom: {
				url: 'http://localhost',
			},
		},
		setupFiles: ['./tests/setup.ts'],
		include: ['tests/**/*.test.ts'],
	},
})
