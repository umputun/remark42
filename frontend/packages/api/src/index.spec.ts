import { client } from "index";

describe("module", () => {
	it("should export client", () => {
		expect(client).toBeDefined();
	});
});
