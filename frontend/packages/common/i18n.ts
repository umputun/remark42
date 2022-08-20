import { derived, writable } from 'svelte/store'

export const locale = writable('en')
export const dictionary = writable<Record<string, string>>({})

export async function loadLocale(locale: string) {
	console.log('locale', locale)
}

type Transformer = (m: string) => string

export const t = derived(dictionary, (d) => {
	return (msg: string, transformer: Transformer = (m) => m) => {
		return transformer(d[msg] ?? msg)
	}
})
