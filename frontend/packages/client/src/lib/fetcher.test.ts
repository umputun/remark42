import { vi, describe, expect, it } from "vitest";
import { createFetcher } from "./fetcher";
import { JWT_HEADER, SITE_HEADER, XSRF_HEADER } from "../consts";
import * as cookie from "./cookies";

describe.concurrent("Fetcher", () => {
	it("adds base url", async () => {
		vi.spyOn(window, "fetch").mockResolvedValueOnce(new Response(""));
		await createFetcher("remark42", "/api").get("/test");

		expect(window.fetch).toHaveBeenCalledWith("/api/test", expect.any(Object));
	});

	it("sets site as header", async () => {
		vi.spyOn(window, "fetch").mockResolvedValueOnce(new Response(""));
		await createFetcher("remark42", "/api").get("/test");

		expect(window.fetch).toHaveBeenCalledWith(
			expect.any(String),
			expect.objectContaining({
				headers: expect.toHaveHeader(SITE_HEADER, "remark42"),
			})
		);
	});

	it("sets xsrf header from cookies", async () => {
		const token = "xsrf-token";
		vi.spyOn(cookie, "getCookie").mockReturnValueOnce(token);
		vi.spyOn(window, "fetch").mockResolvedValueOnce(new Response(""));

		await createFetcher("remark42", "/api").get("/test");
		expect(window.fetch).toHaveBeenCalledWith(
			expect.any(String),
			expect.objectContaining({
				headers: expect.toHaveHeader(XSRF_HEADER, token),
			})
		);
	});

	it("sets sorted query string", async () => {
		vi.spyOn(window, "fetch").mockResolvedValueOnce(new Response(""));
		await createFetcher("remark42", "/api").get("/query-string", {
			x: 3,
			p: 2,
			a: 1,
		});

		expect(window.fetch).toHaveBeenCalledWith(
			"/api/query-string?a=1&p=2&x=3",
			expect.any(Object)
		);
	});

	it.each(["get", "post", "put", "delete"] as const)(
		"implements %s",
		async (method) => {
			vi.spyOn(window, "fetch").mockResolvedValueOnce(new Response(""));
			await createFetcher("remark42", "/api")[method]("/test");

			expect(window.fetch).toBeCalledWith(
				expect.any(String),
				expect.objectContaining({ method })
			);
		}
	);

	it("sends json", async () => {
		const payload = { name: "test" };
		const jsonPayload = JSON.stringify(payload);
		vi.spyOn(window, "fetch").mockResolvedValueOnce(new Response(jsonPayload));
		const result = await createFetcher("remark42", "/api").post("/test/json", {
			payload,
		});

		expect(result).toEqual(payload);
		expect(window.fetch).toBeCalledWith(
			"/api/test/json",
			expect.objectContaining({
				body: jsonPayload,
				headers: new Headers({ "Content-Type": "application/json" }),
			})
		);
	});

	it("sends text", async () => {
		const payload = "text";
		vi.spyOn(window, "fetch").mockResolvedValueOnce(new Response(payload));
		const result = await createFetcher("remark42", "/api").post("/test/text", {
			payload,
		});

		expect(result).toBe(payload);
		expect(window.fetch).toBeCalledWith(
			"/api/test/text",
			expect.objectContaining({ body: payload })
		);
	});

	it.each([401, 403])(
		"throws unauthorized on respose with status %s",
		async (status) => {
			vi.spyOn(window, "fetch").mockResolvedValueOnce(
				new Response(null, { status })
			);
			const fetcher = createFetcher("remark42", "/api");
			await expect(fetcher.get("/test")).rejects.toThrowError("Unauthorized");
		}
	);

	it.each([300, 400, 500])(
		"throws error on respose with status %s",
		async (status) => {
			vi.spyOn(window, "fetch").mockResolvedValueOnce(
				new Response("Error", { status })
			);
			const fetcher = createFetcher("remark42", "/api");
			await expect(fetcher.get("/test")).rejects.toThrow();
		}
	);

	it("sets active token and then cleans it after unauthorized response", async () => {
		const token = "jwt-token";
		const headersWithToken = new Headers({ [JWT_HEADER]: token });

		vi.spyOn(window, "fetch").mockResolvedValueOnce(
			new Response(undefined, { headers: headersWithToken })
		);

		const fetcher = createFetcher("remark42", "/api");

		expect(fetcher.token).toBe(null); // no token on initial state

		// first call without token
		await fetcher.get("/first-call");
		expect(window.fetch).toBeCalledWith(
			expect.any(String),
			expect.objectContaining({
				headers: expect.not.toHaveHeader(JWT_HEADER),
			})
		);
		expect(fetcher.token).toBe(token); // token saved after first call

		vi.spyOn(window, "fetch").mockResolvedValueOnce(new Response());

		// the second call with token
		await fetcher.get("/second-call");
		expect(window.fetch).toBeCalledWith(
			expect.any(String),
			expect.objectContaining({
				headers: expect.toHaveHeader(JWT_HEADER, token),
			})
		);
		expect(fetcher.token).toBe(token); // token preserved after second call

		vi.spyOn(window, "fetch").mockResolvedValueOnce(
			new Response(undefined, { status: 401 })
		);
		// the third call should be with token
		await expect(fetcher.get("/logout")).rejects.toThrow();
		expect(window.fetch).toBeCalledWith(
			expect.any(String),
			expect.objectContaining({
				headers: expect.toHaveHeader(JWT_HEADER, token),
			})
		);
		expect(fetcher.token).toBeNull(); // token cleaned after unauthorized response

		vi.spyOn(window, "fetch").mockResolvedValueOnce(
			new Response(undefined, { status: 401 })
		);
		// the fourth call should be without token
		await expect(fetcher.get("/user")).rejects.toBe("Unauthorized");
		expect(window.fetch).toBeCalledWith(
			expect.any(String),
			expect.objectContaining({
				headers: expect.not.toHaveHeader(JWT_HEADER),
			})
		);
	});
});
