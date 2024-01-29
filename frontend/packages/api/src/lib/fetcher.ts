import { JWT_HEADER, XSRF_COOKIE, XSRF_HEADER } from '../consts'
import { getCookie } from './cookies'

export type QueryParams = Record<string, string | number> | undefined
export type Payload = BodyInit | Record<string, unknown> | null

/** JWT token received from server and will be send by each request, if it present */
let token: string | null = null

export function createFetcher(site: string, baseUrl: string) {
	const fetcher = {
		get<T>(uri: string, query?: QueryParams): Promise<T> {
			return request<T>('get', uri, query)
		},
		put<T>(uri: string, params?: { query?: QueryParams; payload?: Payload }): Promise<T> {
			return request<T>('put', uri, params?.query, params?.payload)
		},
		post<T>(uri: string, params?: { query?: QueryParams; payload?: Payload }): Promise<T> {
			return request<T>('post', uri, params?.query, params?.payload)
		},
		delete<T>(uri: string, query?: QueryParams): Promise<T> {
			return request<T>('delete', uri, query)
		},
		get token() {
			return token
		},
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
		query?: QueryParams,
		body?: Payload,
	): Promise<T> {
		const url = `${baseUrl}${uri}${getSearchParams(query)}`
		const headers = new Headers({ 'X-Site-Id': site })
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
		} else if (body !== undefined) {
			params.body = body
		}

		return fetch(url, params).then<T>((res) => {
			if ([401, 403].includes(res.status)) {
				token = null

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

	return fetcher
}

function getSearchParams(query?: QueryParams) {
	if (!query) {
		return ''
	}

	// overrides type of init in URLSearchParams constructor because it's not correct and accepts QueryParams
	const searchParams = new URLSearchParams(
		Object.entries(query).reduce<Record<string, string>>(
			(acc, [k, v]) => Object.assign(acc, { [k]: v.toString() }),
			{},
		),
	)

	searchParams.sort()

	return `?${searchParams.toString()}`
}
