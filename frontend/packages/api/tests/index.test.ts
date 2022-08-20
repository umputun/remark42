import { test, expect } from 'vitest'
import { createClient } from '..'

test('create client', () => {
	expect(() => createClient({ siteId: 'site', baseUrl: '' })).not.toThrow()
})
