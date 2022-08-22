import { test } from '@playwright/test'
import { nanoid } from 'nanoid'
import * as path from 'path'

test.describe('Post comment', () => {
	test.beforeEach(async ({ page }) => {
		await page.goto('/web/')
	})

	test('as dev user', async ({ page, browserName }) => {
		const iframe = page.frameLocator('iframe[name]')
		await iframe.locator('text=Sign In').click()
		const [authPage] = await Promise.all([
			page.waitForEvent('popup'),
			iframe.locator("[title='Sign In with Dev']").click(),
		])
		await authPage.locator('text=Authorize').click()
		// triggers tab visibility and enables widget to re-render with auth state
		await page.press('iframe[name]', 'Tab')
		await iframe.locator('textarea').click()
		const message = `Hello world! ${nanoid()}`
		await iframe.locator('textarea').type(message)
		await iframe.locator('text=Send').click()
		// checks if comment was posted
		iframe.locator(`text=${message}`).first()
		await page.reload()
		// checks if saved comment is visible
		iframe.locator(`text=${message}`).first()
	})
})
