import { describe, it, expect } from 'vitest'
import { render } from '@tests/lib'

import App from './App.svelte'

describe('App', () => {
	it('should render', () => {
		expect(() => render(App)).not.toThrow()
	})
})
