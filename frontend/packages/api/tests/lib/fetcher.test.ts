import { beforeEach, describe, expect } from 'vitest'
import { mockEndpoint } from '../test-utils'
import { JWT_HEADER, XSRF_COOKIE, XSRF_HEADER } from '../../consts'
import { Client, createFetcher } from '../../lib/fetcher'

interface Context {
	client: Client
}

describe<Context>('Fetcher', (fetcher) => {
	beforeEach<Context>((ctx) => {
		ctx.client = createFetcher('remark42', '')
	})

	fetcher('get', async ({ client }) => {
		const ref = mockEndpoint('/test')

		await client.get('/test')
		expect(ref.req.method).toBe('GET')
	})

	fetcher('post', async ({ client }) => {
		const ref = mockEndpoint('/test', { method: 'post' })

		await client.post('/test')
		expect(ref.req.method).toBe('POST')
	})

	fetcher('put', async ({ client }) => {
		const ref = mockEndpoint('/test', { method: 'put' })

		await client.put('/test')
		expect(ref.req.method).toBe('PUT')
	})

	fetcher('delete', async ({ client }) => {
		const ref = mockEndpoint('/test', { method: 'delete' })

		await client.delete('/test')
		expect(ref.req.method).toBe('DELETE')
	})
	fetcher('should send json', async ({ client }) => {
		const data = { name: 'test' }
		const ref = mockEndpoint('/test', { method: 'post', body: data })

		await expect(client.post('/test', {}, data)).resolves.toStrictEqual(data)
		await expect(ref.req.json()).resolves.toStrictEqual(data)
		expect(ref.req.headers.get('Content-Type')).toBe('application/json')
	})

	fetcher('should send text', async ({ client }) => {
		const data = 'text'
		const ref = mockEndpoint('/test', { method: 'post', body: data })

		await expect(client.post('/test', {}, data)).resolves.toBe(data)
		await expect(ref.req.text()).resolves.toBe(data)
		expect(ref.req.headers.get('Content-Type')).toMatch('text/plain')
	})

	fetcher('should send query', async ({ client }) => {
		const ref = mockEndpoint('/test')

		await expect(client.get('/test', { x: 1, p: 2 })).resolves.toBe('')
		expect(ref.req.url.searchParams.get('x')).toBe('1')
		expect(ref.req.url.searchParams.get('p')).toBe('2')
	})

	fetcher('should sort query params', async ({ client }) => {
		const ref = mockEndpoint('/test')

		await expect(client.get('/test', { x: 1, p: 2 })).resolves.toBe('')
		expect(ref.req.url.search).toBe('?p=2&site=remark42&x=1')
	})

	fetcher(
		'should set active token and then clean it on unauthorized response',
		async ({ client }) => {
			let ref = mockEndpoint('/user', { headers: { [JWT_HEADER]: 'token' } })

			// token should be saved
			await client.get('/user')
			// the first call should be without token
			expect(ref.req.headers.get(JWT_HEADER)).toBe(null)
			// the second call should be with token
			await client.get('/user')
			// check if the second call was with token
			expect(ref.req.headers.get(JWT_HEADER)).toBe('token')

			// unauthorized response should clean token
			ref = mockEndpoint('/user', { status: 401 })
			// the third call should be with token but token should be cleaned after it
			await expect(client.get('/user')).rejects.toBe('Unauthorized')
			// the fourth call should be without token
			await expect(client.get('/user')).rejects.toBe('Unauthorized')
			// check if the fourth call was with token
			expect(ref.req.headers.get(JWT_HEADER)).toBe(null)
		}
	)

	fetcher('should add XSRF header if we have it in cookies', async ({ client }) => {
		const ref = mockEndpoint('/user')

		Object.defineProperty(document, 'cookie', {
			writable: true,
			value: `${XSRF_COOKIE}=token`,
		})

		await client.get('/user')
		expect(ref.req.headers.get(XSRF_HEADER)).toBe('token')
	})

	fetcher('should throw error on api response with status code 400', async ({ client }) => {
		mockEndpoint('/user', { status: 400 })

		await expect(client.get('/user')).rejects.toBe('')
	})
})
