import type { Config as ServerConfig, Provider, OAuthProvider, FormProvider } from '@remark42/api'
import { writable } from 'svelte/store'
import { publicApi } from '../lib/api'

export type Config = {
	simpleView: boolean
	emojiSuggestions: boolean
	providers: {
		oauth: OAuthProvider[]
		form: FormProvider[]
	}
}

const isLoading = writable(false)

export const config = writable<Config>({
	simpleView: false,
	emojiSuggestions: true,
	providers: {
		oauth: [],
		form: [],
	},
})

export function formatProviders(providers: Provider[]): {
	oauth: OAuthProvider[]
	form: FormProvider[]
} {
	const oauth: OAuthProvider[] = []
	const form: FormProvider[] = []

	providers.forEach((p) => {
		;['email', 'anonymous'].includes(p)
			? form.push(p as FormProvider)
			: oauth.push(p as OAuthProvider)
	})

	return { oauth, form }
}

function formatConfig(config: ServerConfig): Config {
	return {
		providers: formatProviders(config.auth_providers),
		simpleView: config.simple_view,
		emojiSuggestions: config.emoji_enabled,
	}
}

export function fetchConfig() {
	isLoading.set(true)
	publicApi
		.getConfig()
		.then((data) => config.set(formatConfig(data)))
		.finally(() => isLoading.set(false))
}

if (import.meta.hot) {
	config.subscribe((s) => console.log('@store/config', s))
}
