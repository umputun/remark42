import { rest, RestRequest } from 'msw'
import { setupServer } from 'msw/node'

export const server = setupServer()

interface RequestRef {
	req: RestRequest
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
		rest[method](url, (req, res, ctx) => {
			const transformers = [ctx.status(status), ctx.json(body)]

			if (headers) {
				transformers.push(ctx.set(headers))
			}

			result.req = req
			return res(...transformers)
		})
	)

	return result
}
