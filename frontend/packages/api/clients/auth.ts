import type { ClientParams, User } from './index'
import { createFetcher } from '../lib/fetcher'

export function createAuthClient({ siteId, baseUrl }: ClientParams) {
	const fetcher = createFetcher(siteId, `${baseUrl}/auth`)

	async function anonymous(user: string): Promise<User> {
		return fetcher.get<User>('/anonymous/login', { user, aud: siteId })
	}

	async function email(email: string, username: string): Promise<(token: string) => Promise<User>> {
		const EMAIL_SIGNIN_ENDPOINT = '/email/login'

		await fetcher.get<undefined>(EMAIL_SIGNIN_ENDPOINT, { address: email, user: username })

		return function tokenVerification(token: string): Promise<User> {
			return fetcher.get<User>(EMAIL_SIGNIN_ENDPOINT, { token })
		}
	}

	async function telegram() {
		const TELEGRAM_SIGNIN_ENDPOINT = '/telegram/login'

		const { bot, token } = await fetcher.get<{ bot: string; token: string }>(
			TELEGRAM_SIGNIN_ENDPOINT
		)

		return {
			bot,
			token,
			verify() {
				return fetcher.get(TELEGRAM_SIGNIN_ENDPOINT, { token })
			},
		}
	}

	async function logout(): Promise<void> {
		return fetcher.get<void>('/logout')
	}

	return {
		anonymous,
		email,
		telegram,
		logout,
	}
}
