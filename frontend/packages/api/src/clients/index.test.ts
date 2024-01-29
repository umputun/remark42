import { test, expect } from 'vitest'
import { createClient } from '..'

test('create client', () => {
	expect(() => createClient({ site: 'remark42', baseUrl: '' })).not.toThrow()
})
