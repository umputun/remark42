import { test, expect } from "@playwright/test";
import { nanoid } from "nanoid";

test.describe("Post comment", () => {
	test.beforeEach(async ({ page }) => {
		await page.goto("/web/");
	});

	test("as dev user", async ({ page, browserName }) => {
		await page.frameLocator("iframe[name]").locator("text=Sign In").click();
		const [authPage] = await Promise.all([
			page.waitForEvent("popup"),
			page
				.frameLocator("iframe[name]")
				.locator("[title='Sign In with Dev']")
				.click(),
		]);
		await authPage.locator("text=Authorize").click();

		// firefox doesn't see iframe after auth
		if (browserName === "firefox") {
			await page.reload();
		} else {
			await page.press("iframe[name]", "Tab");
		}
		const message = `Hello world, ${browserName}! ${nanoid()}`;
		await page
			.frameLocator("iframe[name]")
			.getByPlaceholder("Your comment here")
			.fill(message);
		await page
			.frameLocator("iframe[name]")
			.getByRole("button", { name: "Send" })
			.click();
		await expect(
			page.frameLocator("iframe[name]").getByText(message)
		).toBeVisible();
		await page.reload();
		// checks if comment is saved and visible after reload
		await page
			.frameLocator("iframe[name]")
			.locator("body")
			.screenshot({
				path: `./screenshots/${browserName}.png`,
			});
		await expect(
			page.frameLocator("iframe[name]").getByText(message)
		).toBeVisible();
	});
});
