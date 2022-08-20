import { createClient } from '../'
import { test, expect } from 'vitest'

test('create client', () => {
	expect(() => createClient({ siteId: 'site', baseUrl: '' })).not.toThrow()
})
