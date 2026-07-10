import { test, expect, type Page } from '@playwright/test'

// the parent page sets color-scheme on the iframe element from the theme param. if the iframe
// document does not carry the same color-scheme before its bundle runs, the canvas is painted
// opaque white instead of staying transparent. block the bundle to freeze the document in that
// pre-script state and assert the inline head script has already applied the scheme.
test.describe('Iframe color scheme', () => {
	test.beforeEach(async ({ page }) => {
		await page.route(/remark\.m?js$/, (route) => route.abort())
	})

	const cases = [
		{ name: 'dark theme', query: '?site_id=remark&theme=dark', expected: 'dark' },
		{ name: 'light theme', query: '?site_id=remark&theme=light', expected: 'light' },
		{ name: 'no theme falls back to light', query: '?site_id=remark', expected: 'light' },
	]

	for (const { name, query, expected } of cases) {
		test(name, async ({ page }) => {
			await page.goto(`/web/iframe.html${query}`)

			const inline = await page.evaluate(() => document.documentElement.style.colorScheme)
			expect(inline).toBe(expected)

			const computed = await page.evaluate(() => getComputedStyle(document.documentElement).colorScheme)
			expect(computed).toBe(expected)
		})
	}
})

// browsers paint a default surface for an iframe before its document is parsed, and that surface
// is opaque when the element carries a color-scheme the document does not have yet. WebKit shows
// it as a white flash on dark host pages. the parent keeps the iframe hidden until the document
// reports itself inited, so the surface is never presented.
test.describe('Iframe reveal', () => {
	// REVEAL_TIMEOUT in app/utils/create-iframe.ts. the fallback timer starts when the
	// iframe is created, during page load, so any assertion with a deadline at or past
	// this value can be satisfied by the fallback alone and says nothing about the
	// message path. bound the message-path assertions well under it.
	const REVEAL_TIMEOUT = 5000
	const MESSAGE_REVEAL_BUDGET = 1500

	const visibility = (page: Page) =>
		page.evaluate(() => {
			const iframe = document.querySelector<HTMLIFrameElement>('#remark42 iframe')
			return iframe ? iframe.style.visibility : 'no-iframe'
		})

	test('stays hidden until the document reports inited', async ({ page }) => {
		await page.route(/\/web\/iframe\.html/, (route) => route.abort())
		const start = Date.now()
		await page.goto('/web/')
		await page.waitForSelector('#remark42 iframe', { state: 'attached' })

		expect(await visibility(page)).toBe('hidden')
		// a slow run could have let the fallback fire, which would make the assertion
		// above pass or fail for the wrong reason. fail loudly instead of flaking.
		expect(Date.now() - start).toBeLessThan(REVEAL_TIMEOUT)
	})

	// must reveal from the inited message, not the fallback: a broken message listener would
	// leave the widget invisible for 5s on every load. the fallback timer starts when the
	// iframe is created, partway through goto(), so bounding only the poll leaves the
	// navigation window unmeasured. time the whole thing.
	test('is revealed by the inited message, well before the fallback', async ({ page }) => {
		const start = Date.now()
		await page.goto('/web/')

		await expect.poll(() => visibility(page), { timeout: MESSAGE_REVEAL_BUDGET }).toBe('visible')
		expect(Date.now() - start).toBeLessThan(REVEAL_TIMEOUT)
		await expect(page.locator('#remark42 iframe')).toBeVisible()
	})

	// the aborted document never reports its height, so the iframe box stays empty and
	// toBeVisible() would fail on geometry. assert the property the fallback actually sets.
	test('is revealed by the timeout when inited never arrives', async ({ page }) => {
		await page.route(/\/web\/iframe\.html/, (route) => route.abort())
		await page.goto('/web/')
		await page.waitForSelector('#remark42 iframe', { state: 'attached' })

		await expect.poll(() => visibility(page), { timeout: REVEAL_TIMEOUT * 2 }).toBe('visible')
	})
})
