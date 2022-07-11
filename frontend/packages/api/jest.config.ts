import type { Config } from "jest";

const config: Config = {
	testEnvironment: "jsdom",
	transform: {
		"^.+\\.ts$": [
			"@swc/jest",
			{
				jsc: {
					parser: {
						syntax: "typescript",
						decorators: false,
					},
					target: "es2021",
				},
			},
		],
	},
	moduleDirectories: ["node_modules", "src"],
	collectCoverageFrom: ["src/**/*.ts"],
};

export default config;
