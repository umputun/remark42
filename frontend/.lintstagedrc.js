module.exports = {
	"./**/*.{ts,tsx,js,jsx}": [
		"cd apps/remark42 && pnpm lint-staged:lint:scripts",
		"cd apps/remark42 && pnpm lint-staged:format",
	],
	"./**/*.css": [
		"cd apps/remark42 && pnpm lint-staged:lint:styles",
		"cd apps/remark42 && pnpm lint-staged:format",
	],
	"./templates/**.html": [
		"cd apps/remark42 && pnpm lint-staged:lint:styles",
		"cd apps/remark42 && pnpm lint-staged:format",
	],
};
