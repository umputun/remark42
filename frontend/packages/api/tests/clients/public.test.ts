import { beforeEach, describe, expect } from 'vitest'
import { mockEndpoint } from '../test-utils'
import { createPublicClient, GetUserCommentsParams, Vote } from '../../clients/public'

interface Context {
	client: ReturnType<typeof createPublicClient>
}

describe<Context>('Public Client', (publicClient) => {
	beforeEach<Context>((ctx) => {
		ctx.client = createPublicClient({ siteId: 'mysite', baseUrl: '/remark42' })
	})

	publicClient('getConfig: should return config', async ({ client }) => {
		const data = { x: 1, y: 2 }

		mockEndpoint('/remark42/api/v1/config', { body: data })
		await expect(client.getConfig()).resolves.toEqual(data)
	})

	publicClient('getComments: should return page comments', async ({ client }) => {
		const data = { post: { id: '1' }, node: [{ id: 1 }] }
		const ref = mockEndpoint('/remark42/api/v1/comments', { body: data })

		await expect(client.getComments('/post/1')).resolves.toEqual(data)
		expect(ref.req.url.searchParams.get('url')).toBe('/post/1')
	})

	const commentRequestCases: GetUserCommentsParams[] = [
		{ url: '' },
		{ url: '' },
		{ url: '', limit: 10 },
		{ url: '', skip: 10 },
		{ url: '', skip: 10, limit: 0 },
	]

	commentRequestCases.forEach((params) => {
		publicClient(
			`getComments: should return user comments with params: ${JSON.stringify(params)}`,
			async ({ client }) => {
				const data = [{ id: 1 }, { id: 2 }]
				const ref = mockEndpoint('/remark42/api/v1/find', { body: data })

				await expect(client.getComments(params)).resolves.toEqual(data)
				expect(ref.req.url.searchParams.get('limit')).toBe(
					params.limit === undefined ? null : `${params.limit}`
				)
				expect(ref.req.url.searchParams.get('skip')).toBe(
					params.skip === undefined ? null : `${params.skip}`
				)
			}
		)
	})

	publicClient('addComment: should add comment', async ({ client }) => {
		const data = { id: '1', text: 'test' }
		const ref = mockEndpoint('/remark42/api/v1/comment', { method: 'post', body: data })

		await expect(client.addComment('/post/my-first-post', { text: 'test' })).resolves.toEqual(data)
		await expect(ref.req.json()).resolves.toEqual({
			text: data.text,
			locator: {
				site: 'mysite',
				url: '/post/my-first-post',
			},
		})
	})

	publicClient('updateComment: should update comment', async ({ client }) => {
		const data = { id: 1, body: 'test' }
		const ref = mockEndpoint('/remark42/api/v1/comment/1', { method: 'put', body: data })

		await expect(client.updateComment('/post/my-first-post', '1', 'test')).resolves.toEqual(data)
		await expect(ref.req.json()).resolves.toEqual({ text: 'test' })
		expect(ref.req.url.searchParams.get('url')).toBe('/post/my-first-post')
	})

	publicClient('should remove comment', async ({ client }) => {
		const ref = mockEndpoint('/remark42/api/v1/comment/1', { method: 'put' })

		await expect(client.removeComment('/post/my-first-post', '1')).resolves.toBe('')
		expect(ref.req.url.searchParams.get('url')).toBe('/post/my-first-post')
	})

	const voteRequestCases: { vote: Vote; value: string }[] = [
		{ vote: 1, value: 'upvote' },
		{ vote: -1, value: 'downvote' },
	]
	voteRequestCases.forEach(({ vote, value }) => {
		publicClient(`vote: should ${value} for comment`, async ({ client }) => {
			const data = { id: 1, vote: 2 }
			const ref = mockEndpoint('/remark42/api/v1/vote/1', { method: 'put', body: data })

			await expect(client.vote('/post/my-first-post', '1', vote)).resolves.toEqual(data)
			expect(ref.req.url.searchParams.get('url')).toBe('/post/my-first-post')
			expect(ref.req.url.searchParams.get('vote')).toBe(`${vote}`)
		})
	})

	const userCases = [null, { id: '1', username: 'user' }]
	userCases.forEach((user) => {
		publicClient('should return user', async ({ client }) => {
			mockEndpoint('/remark42/api/v1/user', { body: user })
			await expect(client.getUser()).resolves.toEqual(user)
		})
	})
})
