import { JWT_HEADER, XSRF_COOKIE, XSRF_HEADER } from '../consts'
import { getCookie } from '../lib/cookies'

export type QueryParams = Record<string, string | number | string[] | number[] | undefined>
export type Payload = BodyInit | Record<string, unknown> | null
export type BodylessMethod = <T>(url: string, query?: QueryParams) => Promise<T>
export type BodyMethod = <T>(url: string, query?: QueryParams, body?: Payload) => Promise<T>
export interface Client {
	get: BodylessMethod
	put: BodyMethod
	post: BodyMethod
	delete: BodylessMethod
}

/** JWT token received from server and will be send by each request, if it present */
let token: string | undefined

export const createFetcher = (site: string, baseUrl: string): Client => {
	const client = {
		get: <T>(uri: string, query?: QueryParams): Promise<T> => request<T>('get', uri, query),
		put: <T>(uri: string, query?: QueryParams, body?: Payload): Promise<T> =>
			request<T>('put', uri, query, body),
		post: <T>(uri: string, query?: QueryParams, body?: Payload): Promise<T> =>
			request<T>('post', uri, query, body),
		delete: <T>(uri: string, query?: QueryParams, body?: Payload): Promise<T> =>
			request<T>('delete', uri, query, body),
	}

	/**
	 * Fetcher is abstraction on top of fetch
	 *
	 * @method - a string to set http method
	 * @uri – uri to API endpoint
	 * @query - collection of query params. They will be concatenated to URL. `siteId` will be added automatically.
	 * @body - data for sending to the server. If you pass object it will be stringified. If you pass form data it will be sent as is. Content type headers will be added automatically.
	 */
	async function request<T>(
		method: string,
		uri: string,
		query: QueryParams = {},
		body?: Payload
	): Promise<T> {
		const searchParams = new URLSearchParams({ site, ...query })
		searchParams.sort()
		const url = `${baseUrl}${uri}?${searchParams.toString()}`
		const headers = new Headers()
		const params: RequestInit = { method, headers }

		// Save token in memory and pass it into headers in case if storing cookies is disabled
		if (token) {
			headers.set(JWT_HEADER, token)
		}

		// An HTTP header cannot be empty.
		// Although some webservers allow this (nginx, Apache), others answer 400 Bad Request (lighttpd).
		const xsrfToken = getCookie(XSRF_COOKIE)
		if (typeof xsrfToken === 'string') {
			headers.set(XSRF_HEADER, xsrfToken)
		}

		if (typeof body === 'object' && body !== null && !(body instanceof FormData)) {
			headers.set('Content-Type', 'application/json')
			params.body = JSON.stringify(body)
		} else {
			params.body = body
		}

		return fetch(url, params).then<T>((res) => {
			if ([401, 403].includes(res.status)) {
				token = undefined

				return Promise.reject('Unauthorized')
			}

			token = res.headers.get(JWT_HEADER) ?? token

			return res
				.text()
				.catch(Object)
				.then((data: string) => {
					if (res.status < 200 || res.status > 299) {
						return Promise.reject(data)
					}
					try {
						return JSON.parse(data) as T
					} catch (e) {
						return data as unknown as T
					}
				})
		})
	}

	return client
}
