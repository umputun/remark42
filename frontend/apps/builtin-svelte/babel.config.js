module.exports = (api) => ({
	presets: [
		[
			"@babel/preset-env",
			{
				useBuiltIns: "usage",
				corejs: 3,
				bugfixes: true,
				loose: true,
				...(api.caller((c) => c?.target === "browserslist:modern")
					? { targets: { esmodules: true } }
					: {}),
			},
		],
	],
});
