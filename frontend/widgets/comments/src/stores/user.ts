import type { User } from '@remark42/api'
import { get, writable } from 'svelte/store'
import { authApi, publicApi } from '../lib/api'

const timeout = 1000 * 60 * 5 // 5 minutes

const lastUpdate = writable(0)
export const isLoading = writable(false)
export const user = writable<undefined | null | User>(undefined, revalidateUser)

user.subscribe(() => {
	lastUpdate.set(Date.now())
})

function revalidateUser() {
	if (get(isLoading) || get(user) !== undefined || get(lastUpdate) > Date.now() - timeout) {
		return
	}

	isLoading.set(true)
	publicApi
		.getUser()
		.then((data) => user.set(data))
		.finally(() => isLoading.set(false))
}

export async function logout() {
	return authApi.logout().then(() => {
		user.set(undefined)
	})
}

if (import.meta.hot) {
	user.subscribe((s) => console.log('@store/user', s))
}
