import { describe, expect, it, vi } from 'vitest'
import { BlockTTL, createAdminClient } from './admin'

describe('Admin Client', () => {
	it('returns list of blocked users', async () => {
		const payload = [{ id: 1 }, { id: 2 }]
		vi.spyOn(window, 'fetch').mockResolvedValueOnce(new Response(JSON.stringify(payload)))

		const client = createAdminClient({ site: 'mysite', baseUrl: '/remark42' })
		await expect(client.getBlockedUsers()).resolves.toEqual(payload)
	})

	it.each<{ ttl: BlockTTL; expected: string }>([
		{ ttl: 'permanently', expected: '0' },
		{ ttl: '1440m', expected: '1440m' },
		{ ttl: '43200m', expected: '43200m' },
	])('blocks user with ttl $ttl', async ({ ttl, expected }) => {
		vi.spyOn(window, 'fetch').mockResolvedValueOnce(new Response())
		const client = createAdminClient({ site: 'mysite', baseUrl: '/remark42' })

		await client.blockUser('1', ttl)
		expect(window.fetch).toBeCalledWith(
			`/remark42/api/v1/user/1?block=1&ttl=${expected}`,
			expect.objectContaining({ method: 'put' }),
		)
	})

	it('unblocks user', async () => {
		vi.spyOn(window, 'fetch').mockResolvedValueOnce(new Response())
		const client = createAdminClient({ site: 'mysite', baseUrl: '/remark42' })

		await client.unblockUser('1')
		expect(window.fetch).toBeCalledWith(
			'/remark42/api/v1/user/1?block=0',
			expect.objectContaining({ method: 'put' }),
		)
	})

	it('marks user as verified', async () => {
		vi.spyOn(window, 'fetch').mockResolvedValueOnce(new Response())
		const client = createAdminClient({ site: 'mysite', baseUrl: '/remark42' })

		await client.verifyUser('1')
		expect(window.fetch).toBeCalledWith(
			'/remark42/api/v1/verify/1?verified=1',
			expect.objectContaining({ method: 'put' }),
		)
	})

	it('marks user as unverified', async () => {
		vi.spyOn(window, 'fetch').mockResolvedValueOnce(new Response())
		const client = createAdminClient({ site: 'mysite', baseUrl: '/remark42' })

		await client.unverifyUser('1')
		expect(window.fetch).toBeCalledWith(
			'/remark42/api/v1/verify/1?verified=0',
			expect.objectContaining({ method: 'put' }),
		)
	})

	it('should approve removing request', async () => {
		vi.spyOn(window, 'fetch').mockResolvedValueOnce(new Response())
		const client = createAdminClient({ site: 'mysite', baseUrl: '/remark42' })

		await client.approveRemovingRequest('token')
		expect(window.fetch).toBeCalledWith(
			'/remark42/api/v1/deleteme?token=token',
			expect.objectContaining({ method: 'get' }),
		)
	})

	it('pins comment', async () => {
		vi.spyOn(window, 'fetch').mockResolvedValueOnce(new Response())
		const client = createAdminClient({ site: 'mysite', baseUrl: '/remark42' })

		await client.pinComment('1')
		expect(window.fetch).toBeCalledWith(
			'/remark42/api/v1/pin/1?pinned=1',
			expect.objectContaining({ method: 'put' }),
		)
	})

	it('unpins comment', async () => {
		vi.spyOn(window, 'fetch').mockResolvedValueOnce(new Response())
		const client = createAdminClient({ site: 'mysite', baseUrl: '/remark42' })

		await client.unpinComment('1')
		expect(window.fetch).toBeCalledWith(
			'/remark42/api/v1/pin/1?pinned=0',
			expect.objectContaining({ method: 'put' }),
		)
	})

	it('removes comment', async () => {
		vi.spyOn(window, 'fetch').mockResolvedValueOnce(new Response())
		const client = createAdminClient({ site: 'mysite', baseUrl: '/remark42' })

		const url = '/post/1'
		await client.removeComment(url, '1')
		expect(window.fetch).toBeCalledWith(
			'/remark42/api/v1/comment/1?url=%2Fpost%2F1',
			expect.objectContaining({ method: 'delete' }),
		)
	})

	it('enables commenting on a page', async () => {
		vi.spyOn(window, 'fetch').mockResolvedValueOnce(new Response())
		const client = createAdminClient({ site: 'mysite', baseUrl: '/remark42' })

		const url = '/post/1'
		await client.enableCommenting(url)
		expect(window.fetch).toBeCalledWith(
			'/remark42/api/v1/readonly?ro=1&url=%2Fpost%2F1',
			expect.objectContaining({ method: 'put' }),
		)
	})

	it('disables commenting on a page', async () => {
		vi.spyOn(window, 'fetch').mockResolvedValueOnce(new Response())
		const client = createAdminClient({ site: 'mysite', baseUrl: '/remark42' })

		const url = '/post/1'

		await client.disableCommenting(url)
		expect(window.fetch).toBeCalledWith(
			'/remark42/api/v1/readonly?ro=0&url=%2Fpost%2F1',
			expect.objectContaining({ method: 'put' }),
		)
	})
})
