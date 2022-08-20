import svelte from 'rollup-plugin-svelte'
import resolve from 'rollup-plugin-node-resolve'
import pkg from './package.json'

export default {
	input: './index.svelte',
	output: [
		{ file: pkg.module, format: 'es' },
		{ file: pkg.main, format: 'cjs', name: pkg.name },
	],
	plugins: [svelte(), resolve()],
}
