/* eslint-disable @typescript-eslint/consistent-type-definitions */
import { expect } from 'vitest'

declare module 'vitest' {
	interface Assertion<T> {
		toHaveHeader(received: Headers, expected: Headers): T
	}
	interface AsymmetricMatchersContaining {
		toHaveHeader(name: string, value?: string): void
	}
}

expect.extend({
	toHaveHeader(received?: Headers, expectedHeader?: string, expectedHeaderValue?: string) {
		if (expectedHeader && expectedHeaderValue) {
			if (!received) {
				return {
					message: () => 'no headers is received',
					pass: false,
					expected: { [expectedHeader]: expectedHeaderValue },
					actual: undefined,
				}
			}

			return {
				message: () =>
					`expected to have header "${expectedHeader}${
						expectedHeaderValue ? `: ${expectedHeaderValue}` : ''
					}"`,
				pass: received.get(expectedHeader) === expectedHeaderValue,
				expected: { [expectedHeader]: expectedHeaderValue },
				actual: { [expectedHeader]: received.get(expectedHeader) },
			}
		}
		return {
			message: () => 'expected to have header',
			pass: !expectedHeader || !received || received.has(expectedHeader),
		}
	},
})
