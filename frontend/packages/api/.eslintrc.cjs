module.exports = {
	root: true,
	env: {
		browser: true,
		es2021: true,
		node: true,
	},
	extends: [
		'eslint:recommended',
		'plugin:@typescript-eslint/recommended',
		'plugin:@typescript-eslint/recommended-requiring-type-checking',
		'plugin:@typescript-eslint/strict',
		'plugin:prettier/recommended',
	],
	parser: '@typescript-eslint/parser',
	parserOptions: {
		ecmaVersion: 'latest',
		sourceType: 'module',
		project: './tsconfig.eslint.json',
	},
	plugins: ['@typescript-eslint'],
	rules: {
		// `interface` does not allow disctinated types and
		// `type does not work if you extend types of a dependency
		'@typescript-eslint/consistent-type-definitions': 'off',
	},
}
