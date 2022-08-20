import { beforeEach, describe, expect } from 'vitest'
import { mockEndpoint } from '../test-utils'
import { BlockTTL, createAdminClient } from '../../clients/admin'

interface Context {
	client: ReturnType<typeof createAdminClient>
}

describe<Context>('Admin Client', (adminClient) => {
	beforeEach<Context>((ctx) => {
		ctx.client = createAdminClient({ siteId: 'mysite', baseUrl: '/remark42' })
	})

	adminClient('should return list of blocked users', async ({ client }) => {
		const data = [{ id: 1 }, { id: 2 }]

		mockEndpoint('/remark42/api/v1/blocked', { body: data })
		await expect(client.getBlockedUsers()).resolves.toEqual(data)
	})

	const ttlCases: [BlockTTL, string][] = [
		['permanently', '0'],
		['1440m', '1440m'],
		['43200m', '43200m'],
	]

	ttlCases.forEach(([ttl, expected]) => {
		adminClient(`should block user with ttl: ${ttl}`, async ({ client }) => {
			const data = { block: true, site_id: 'remark42', user_id: '1' }
			const ref = mockEndpoint('/remark42/api/v1/user/1', { method: 'put', body: data })

			await expect(client.blockUser('1', ttl)).resolves.toEqual(data)
			expect(ref.req.url.searchParams.get('ttl')).toBe(expected)
		})
	})

	adminClient('should unblock user', async ({ client }) => {
		const data = { block: false, site_id: 'remark42', user_id: '1' }
		const ref = mockEndpoint('/remark42/api/v1/user/1', { method: 'put', body: data })

		await expect(client.unblockUser('1')).resolves.toEqual(data)
		expect(ref.req.url.searchParams.get('block')).toBe('0')
	})

	adminClient('should mark user as verified', async ({ client }) => {
		const ref = mockEndpoint('/remark42/api/v1/verify/1', { method: 'put' })

		await client.verifyUser('1')
		expect(ref.req.url.searchParams.get('verified')).toBe('1')
	})

	adminClient('should mark user as unverified', async ({ client }) => {
		const ref = mockEndpoint('/remark42/api/v1/verify/1', { method: 'put' })

		await client.unverifyUser('1')
		expect(ref.req.url.searchParams.get('verified')).toBe('0')
	})

	adminClient('should approve removing request', async ({ client }) => {
		const ref = mockEndpoint('/remark42/api/v1/deleteme')

		await client.approveRemovingRequest('token')
		expect(ref.req.url.searchParams.get('token')).toBe('token')
	})

	adminClient('should pin comment', async ({ client }) => {
		const ref = mockEndpoint('/remark42/api/v1/pin/1', { method: 'put' })

		await client.pinComment('1')
		expect(ref.req.url.searchParams.get('pinned')).toBe('1')
	})

	adminClient('should unpin comment', async ({ client }) => {
		const ref = mockEndpoint('/remark42/api/v1/pin/1', { method: 'put' })

		await client.unpinComment('1')
		expect(ref.req.url.searchParams.get('pinned')).toBe('0')
	})

	adminClient('should remove comment', async ({ client }) => {
		const ref = mockEndpoint('/remark42/api/v1/comment/1', { method: 'delete' })
		const url = '/post/1'

		await client.removeComment(url, '1')
		expect(ref.req.url.searchParams.get('url')).toBe(url)
	})

	adminClient('should enable commenting on a page', async ({ client }) => {
		const ref = mockEndpoint('/remark42/api/v1/readonly', { method: 'put' })
		const url = '/post/1'

		await client.enableCommenting(url)
		expect(ref.req.url.searchParams.get('ro')).toBe('1')
		expect(ref.req.url.searchParams.get('url')).toBe(url)
	})

	adminClient('should disable commenting on a page', async ({ client }) => {
		const ref = mockEndpoint('/remark42/api/v1/readonly', { method: 'put' })
		const url = '/post/1'

		await client.disableCommenting('/post/1')
		expect(ref.req.url.searchParams.get('ro')).toBe('0')
		expect(ref.req.url.searchParams.get('url')).toBe(url)
	})
})
