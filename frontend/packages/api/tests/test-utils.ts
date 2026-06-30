import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'

export const server = setupServer()

interface CapturedRequest {
	url: URL
	method: string
	headers: Headers
	json: () => Promise<unknown>
	text: () => Promise<string>
}

interface RequestRef {
	req: CapturedRequest
}

export function mockEndpoint(
	url: string,
	params: {
		method?: 'get' | 'put' | 'post' | 'delete'
		body?: number | string | null | Record<string, unknown> | unknown[]
		status?: number
		headers?: Record<string, string | string[]>
	} = {}
): RequestRef {
	const { body, method = 'get', status = 200, headers } = params
	const result = { req: {} } as RequestRef

	server.use(
		http[method](url, ({ request }) => {
			const captured = request.clone()
			result.req = {
				url: new URL(request.url),
				method: request.method,
				headers: request.headers,
				json: () => captured.clone().json(),
				text: () => captured.clone().text(),
			}

			const responseHeaders = new Headers()
			if (headers) {
				for (const [key, value] of Object.entries(headers)) {
					responseHeaders.set(key, Array.isArray(value) ? value.join(', ') : value)
				}
			}

			return body === undefined
				? new HttpResponse(null, { status, headers: responseHeaders })
				: HttpResponse.json(body, { status, headers: responseHeaders })
		})
	)

	return result
}
