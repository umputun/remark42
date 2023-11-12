import { describe, it, expect, vi } from 'vitest'
import { createAuthClient } from './auth'

describe('Auth Client', () => {
	it('authorizes as anonymouser', async () => {
		const payload = { id: 1 }
		vi.spyOn(window, 'fetch').mockResolvedValueOnce(new Response(JSON.stringify(payload)))
		const client = createAuthClient({ site: 'mysite', baseUrl: '/remark42' })

		await expect(client.anonymous('username')).resolves.toEqual(payload)
		expect(window.fetch).toBeCalledWith(
			'/remark42/auth/anonymous/login?aud=mysite&user=username',
			expect.objectContaining({ method: 'get' }),
		)
	})
	it('authorizes as anonymouse', async () => {
		const payload = { id: 1 }
		vi.spyOn(window, 'fetch').mockResolvedValueOnce(new Response(JSON.stringify(payload)))
		const client = createAuthClient({ site: 'mysite', baseUrl: '/remark42' })

		await expect(client.anonymous('username')).resolves.toEqual(payload)
		expect(window.fetch).toBeCalledWith(
			'/remark42/auth/anonymous/login?aud=mysite&user=username',
			expect.objectContaining({ method: 'get' }),
		)
	})

	it('should authorize with email', async () => {
		const payload = { id: 1 }
		vi.spyOn(window, 'fetch').mockResolvedValueOnce(new Response())
		const client = createAuthClient({ site: 'mysite', baseUrl: '/remark42' })

		const tokenVerification = await client.email('username@example.com', 'username')
		expect(window.fetch).toBeCalledWith(
			'/remark42/auth/email/login?address=username%40example.com&user=username',
			expect.objectContaining({ method: 'get' }),
		)

		vi.spyOn(window, 'fetch').mockResolvedValueOnce(new Response(JSON.stringify(payload)))
		await expect(tokenVerification('verification-token')).resolves.toEqual(payload)
		expect(window.fetch).toBeCalledWith(
			'/remark42/auth/email/login?token=verification-token',
			expect.objectContaining({ method: 'get' }),
		)
	})

	it('authorizes with telegram', async () => {
		const payload = { bot: 'remark42bot', token: 'telegram-token' }
		const user = { id: 1 }

		const client = createAuthClient({ site: 'mysite', baseUrl: '/remark42' })
		vi.spyOn(window, 'fetch').mockResolvedValueOnce(new Response(JSON.stringify(payload)))

		const telegramAuth = await client.telegram()
		expect(telegramAuth).toEqual(expect.objectContaining(payload))
		expect(window.fetch).toBeCalledWith(
			'/remark42/auth/telegram/login',
			expect.objectContaining({ method: 'get' }),
		)

		vi.spyOn(window, 'fetch').mockResolvedValueOnce(new Response(JSON.stringify(user)))

		await expect(telegramAuth.verify()).resolves.toEqual(user)
		expect(window.fetch).toBeCalledWith(
			'/remark42/auth/telegram/login?token=telegram-token',
			expect.objectContaining({ method: 'get' }),
		)
	})

	it('should logout', async () => {
		vi.spyOn(window, 'fetch').mockResolvedValueOnce(new Response())
		const client = createAuthClient({ site: 'mysite', baseUrl: '/remark42' })
		await expect(client.logout()).resolves.toBeUndefined()

		expect(window.fetch).toBeCalledWith(
			'/remark42/auth/logout',
			expect.objectContaining({ method: 'get' }),
		)
	})
})
