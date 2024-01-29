import { createAdminClient } from './admin'
import { createAuthClient } from './auth'
import { createPublicClient } from './public'

export type User = {
	id: string
	name: string
	/** url to avatar */
	picture: string
	admin: boolean
	block: boolean
	verified: boolean
	/** subscription to email notification */
	email_subscription?: boolean
	/** users with Patreon auth can have paid status */
	paid_sub?: boolean
}

export type OAuthProvider =
	| 'apple'
	| 'facebook'
	| 'twitter'
	| 'google'
	| 'yandex'
	| 'github'
	| 'microsoft'
	| 'patreon'
	| 'telegram'
	| 'dev'
export type FormProvider = 'email' | 'anonymous'
export type Provider = OAuthProvider | FormProvider

export type ClientParams = {
	site: string
	baseUrl: string
}

export type Client = {
	admin: ReturnType<typeof createAdminClient>
	auth: ReturnType<typeof createAuthClient>
	public: ReturnType<typeof createPublicClient>
}

let client: Client | undefined

export function createClient(params: ClientParams): Client {
	if (client === undefined) {
		client = {
			auth: createAuthClient(params),
			admin: createAdminClient(params),
			public: createPublicClient(params),
		}
	}

	return client
}
