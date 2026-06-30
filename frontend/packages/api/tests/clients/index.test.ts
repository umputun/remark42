import { describe, expect, it } from 'vitest'
import { createClient } from '../..'

describe('Client', () => {
	it('should create a client', () => {
		const params = { siteId: 'mysite', baseUrl: 'http://localhost/remark42' }
		const client = createClient(params)

		expect(client).toBeDefined()
		expect(client.admin).toBeDefined()
		expect(client.auth).toBeDefined()
		expect(client.public).toBeDefined()
	})
})
