import { describe, it, expect, vi } from 'vitest'
import { createPublicClient } from './public'

describe('Public Client', () => {
	it('getConfig: should return config', async () => {
		const payload = { x: 1, y: 2 }
		vi.spyOn(window, 'fetch').mockResolvedValueOnce(new Response(JSON.stringify(payload)))
		const client = createPublicClient({ site: 'mysite', baseUrl: '/remark42' })

		await expect(client.getConfig()).resolves.toEqual(payload)
		expect(window.fetch).toBeCalledWith(
			'/remark42/api/v1/config',
			expect.objectContaining({ method: 'get' }),
		)
	})

	it('getComments: should return page comments', async () => {
		const payload = { post: { id: '1' }, node: [{ id: 1 }] }
		vi.spyOn(window, 'fetch').mockResolvedValueOnce(new Response(JSON.stringify(payload)))
		const client = createPublicClient({ site: 'mysite', baseUrl: '/remark42' })

		await expect(client.getComments('/post/1')).resolves.toEqual(payload)
		expect(window.fetch).toBeCalledWith(
			'/remark42/api/v1/comments?url=%2Fpost%2F1',
			expect.objectContaining({ method: 'get' }),
		)
	})

	it.each([
		{ url: '/post/1' },
		{ url: '/post/2', limit: 10 },
		{ url: '/post/3', skip: 10 },
		{ url: '/post/4', limit: 10, skip: 10 },
	] as const)(
		'getComments: should return user comments with limit $limit and skip $skip',
		async (params) => {
			const payload = [{ id: 1 }, { id: 2 }]
			vi.spyOn(window, 'fetch').mockResolvedValueOnce(new Response(JSON.stringify(payload)))
			const client = createPublicClient({ site: 'mysite', baseUrl: '/remark42' })

			const queryParams = new URLSearchParams(params as unknown as Record<string, string>)
			await expect(client.getComments(params)).resolves.toEqual(payload)
			expect(window.fetch).toBeCalledWith(
				`/remark42/api/v1/find?format=tree&${queryParams.toString()}`,
				expect.objectContaining({ method: 'get' }),
			)
		},
	)

	it('addComment: should add comment', async () => {
		const payload = { id: '1', text: 'test' }
		vi.spyOn(window, 'fetch').mockResolvedValueOnce(new Response(JSON.stringify(payload)))
		const client = createPublicClient({ site: 'remark42', baseUrl: '/remark42' })
		const postUrl = '/post/my-first-post'
		await expect(client.addComment(postUrl, { text: 'test' })).resolves.toEqual(payload)
		expect(window.fetch).toBeCalledWith(
			'/remark42/api/v1/comment',
			expect.objectContaining({
				method: 'post',
				body: JSON.stringify({ text: payload.text, locator: { site: 'remark42', url: postUrl } }),
			}),
		)
	})

	it('updates comment', async () => {
		const payload = { id: 1, text: 'test' }
		vi.spyOn(window, 'fetch').mockResolvedValueOnce(new Response(JSON.stringify(payload)))
		const client = createPublicClient({ site: 'mysite', baseUrl: '/remark42' })

		await expect(
			client.updateComment('/post/my-first-post', 'comment-id-1', 'test'),
		).resolves.toEqual(payload)
		expect(window.fetch).toBeCalledWith(
			'/remark42/api/v1/comment/comment-id-1?url=%2Fpost%2Fmy-first-post',
			expect.objectContaining({ method: 'put', body: JSON.stringify({ text: payload.text }) }),
		)
	})

	it('should remove comment', async () => {
		vi.spyOn(window, 'fetch').mockResolvedValueOnce(new Response())
		const client = createPublicClient({ site: 'mysite', baseUrl: '/remark42' })

		await expect(
			client.removeComment('/post/my-first-post', 'comment-id-1'),
		).resolves.toBeUndefined()
		expect(window.fetch).toBeCalledWith(
			'/remark42/api/v1/comment/comment-id-1?url=%2Fpost%2Fmy-first-post',
			expect.objectContaining({ method: 'put' }),
		)
	})

	it.each([
		{ vote: 1, value: 'upvote' },
		{ vote: -1, value: 'downvote' },
	] as const)(`vote: should $value for comment`, async ({ vote }) => {
		const payload = { id: 1, vote: 2 }
		vi.spyOn(window, 'fetch').mockResolvedValueOnce(new Response(JSON.stringify(payload)))
		const client = createPublicClient({ site: 'mysite', baseUrl: '/remark42' })

		await expect(client.vote('/post/my-first-post', 'comment-id-1', vote)).resolves.toEqual(payload)
		expect(window.fetch).toBeCalledWith(
			`/remark42/api/v1/vote/comment-id-1?url=%2Fpost%2Fmy-first-post&vote=${vote}`,
			expect.objectContaining({ method: 'put' }),
		)
	})

	it.each([null, { id: '1', username: 'user' }])('should return user', async (payload) => {
		vi.spyOn(window, 'fetch').mockResolvedValueOnce(new Response(JSON.stringify(payload)))
		const client = createPublicClient({ site: 'mysite', baseUrl: '/remark42' })
		await expect(client.getUser()).resolves.toEqual(payload)
		expect(window.fetch).toBeCalledWith(
			'/remark42/api/v1/user',
			expect.objectContaining({ method: 'get' }),
		)
	})
})
