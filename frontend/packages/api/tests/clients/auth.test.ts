import { beforeEach, describe, expect } from 'vitest'
import { mockEndpoint } from '../test-utils'
import { createAuthClient } from '../../clients/auth'

interface Content {
	client: ReturnType<typeof createAuthClient>
}

describe<Content>('Auth Client', (authClient) => {
	beforeEach<Content>((ctx) => {
		ctx.client = createAuthClient({ siteId: 'mysite', baseUrl: '/remark42' })
	})

	authClient('should authorize as anonymouse', async ({ client }) => {
		const data = { id: 1 }
		const ref = mockEndpoint('/remark42/auth/anonymous/login', { body: data })

		await expect(client.anonymous('username')).resolves.toEqual(data)
		expect(ref.req.url.searchParams.get('aud')).toBe('mysite')
		expect(ref.req.url.searchParams.get('user')).toBe('username')
	})

	authClient('should authorize with email', async ({ client }) => {
		const data = { id: 1 }
		const ref = mockEndpoint('/remark42/auth/email/login', { body: data })
		const tokenVerification = await client.email('username@example.com', 'username')

		expect(ref.req.url.searchParams.get('address')).toBe('username@example.com')
		expect(ref.req.url.searchParams.get('user')).toBe('username')

		await expect(tokenVerification('token')).resolves.toEqual(data)
		expect(ref.req.url.searchParams.get('token')).toBe('token')
	})

	authClient('should authorize with telegram', async ({ client }) => {
		const data = {
			bot: 'remark42bot',
			token: 'token',
		}
		const user = { id: 1 }

		mockEndpoint('/remark42/auth/telegram/login', { body: data })

		const telegramAuth = await client.telegram()

		expect(telegramAuth.bot).toBe(data.bot)
		expect(telegramAuth.token).toBe(data.token)

		const ref = mockEndpoint('/remark42/auth/telegram/login', { body: user })

		await expect(telegramAuth.verify()).resolves.toEqual(user)
		expect(ref.req.url.searchParams.get('token')).toBe('token')
	})

	authClient('should logout', async ({ client }) => {
		mockEndpoint('/remark42/auth/logout')
		await expect(client.logout()).resolves.toBe('')
	})
})
