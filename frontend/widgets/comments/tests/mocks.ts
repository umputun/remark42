import { vi } from 'vitest'

vi.mock('@remark42/api/clients/auth', () => ({
	createAuthClient() {
		return {
			anonymous: vi.fn(),
		}
	},
}))

vi.mock('@remark42/api/clients/public', () => ({
	createPublicClient() {
		return {
			getConfig: vi.fn().mockResolvedValue({
				auth_providers: [],
			}),
			getUser: vi.fn().mockResolvedValue(null),
			getComments: vi.fn().mockResolvedValue([]),
		}
	},
}))
