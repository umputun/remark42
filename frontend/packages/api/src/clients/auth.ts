import type { ClientParams, User } from './index'
import { createFetcher } from '../lib/fetcher'

export function createAuthClient({ site, baseUrl }: ClientParams) {
	const fetcher = createFetcher(site, `${baseUrl}/auth`)

	async function anonymous(username: string): Promise<User> {
		const user = await fetcher.get<User>('/anonymous/login', { user: username, aud: site })

		return user
	}

	async function email(email: string, username: string): Promise<(token: string) => Promise<User>> {
		const EMAIL_SIGNIN_ENDPOINT = '/email/login'

		await fetcher.get<undefined>(EMAIL_SIGNIN_ENDPOINT, { address: email, user: username })

		return async function tokenVerification(token: string) {
			const user = await fetcher.get<User>(EMAIL_SIGNIN_ENDPOINT, { token })

			return user
		}
	}

	async function telegram() {
		const TELEGRAM_SIGNIN_ENDPOINT = '/telegram/login'

		const { bot, token } = await fetcher.get<{ bot: string; token: string }>(
			TELEGRAM_SIGNIN_ENDPOINT,
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
		await fetcher.get('/logout')
	}

	return {
		anonymous,
		email,
		telegram,
		logout,
	}
}
