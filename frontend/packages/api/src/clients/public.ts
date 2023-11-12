import type { ClientParams, Provider, User } from './index'
import { createFetcher } from '../lib/fetcher'
import { API_BASE } from '../consts'

export type Config = {
	version: string
	auth_providers: Provider[]
	edit_duration: number
	max_comment_size: number
	admins: string[]
	admin_email: string
	low_score: number
	critical_score: number
	positive_score: boolean
	readonly_age: number
	max_image_size: number
	simple_view: boolean
	anon_vote: boolean
	email_notifications: boolean
	telegram_notifications: boolean
	emoji_enabled: boolean
}

export type Comment = {
	/** comment id */
	id: string
	/** parent id */
	pid: string
	/** comment text, after md processing */
	text: string
	/** original comment text */
	orig?: string
	user: User
	locator: {
		/** site id */
		site: string
		/** page url */
		url: string
	}
	score: number
	voted_ips: { Timestamp: string; Value: boolean }[]
	/**
	 * vote delta,
	 * if user hasn't voted delta will be 0,
	 * -1/+1 for downvote/upvote
	 */
	vote: 0 | 1 | -1
	/** comment controversy */
	controversy?: number
	/** pointer to have empty default in json response */
	edit?: {
		time: string
		summary: string
	}
	/** timestamp */
	time: string
	pin?: boolean
	delete?: boolean
	/** page title */
	title?: string
}

export type CommentsTree = {
	comment: Comment
	replies: Comment[]
}

export type CommentPayload = {
	title?: string
	pid?: string
	text: string
}

export type Sort = '-active' | '+active'
export type GetUserCommentsParams = {
	url: string
	sort?: Sort
	limit?: number
	skip?: number
}
export type Vote = -1 | 1

export function createPublicClient({ site, baseUrl }: ClientParams) {
	const fetcher = createFetcher(site, `${baseUrl}${API_BASE}`)

	/** Get server config */
	async function getConfig(): Promise<Config> {
		const config = await fetcher.get<Config>('/config')

		return config
	}

	/** Get current authorized user */
	async function getUser(): Promise<User | null> {
		const user = await fetcher.get<User | null>('/user').catch(() => null)

		return user
	}

	/** Get comments */
	async function getComments<T extends string | GetUserCommentsParams>(
		params: T,
	): Promise<T extends string ? CommentsTree : Comment[]> {
		if (typeof params === 'string') {
			const comments = await fetcher.get<Comment[]>('/comments', { url: params })
			return comments as T extends string ? CommentsTree : Comment[]
		}
		const commentsTree = await fetcher.get<CommentsTree>('/find', { ...params, format: 'tree' })
		return commentsTree as T extends string ? CommentsTree : Comment[]
	}

	/**
	 * Add new comment
	 */
	async function addComment(url: string, payload: CommentPayload): Promise<Comment> {
		const comment = await fetcher.post<Comment>('/comment', {
			payload: { ...payload, locator: { site, url } },
		})
		return comment
	}

	/** Update comment */
	async function updateComment(url: string, id: string, text: string): Promise<Comment> {
		return fetcher.put(`/comment/${id}`, { query: { url }, payload: { text } })
	}

	/** Remove comment on a page */
	async function removeComment(url: string, id: string): Promise<void> {
		await fetcher.put(`/comment/${id}`, { query: { url }, payload: { delete: true } })
	}

	type VotePayload = { url: string; vote: Vote }
	/** Vote for a comment */
	async function vote(url: string, id: string, vote: Vote): Promise<VotePayload> {
		const result = await fetcher.put<VotePayload>(`/vote/${id}`, { query: { url, vote } })
		return result
	}

	return {
		getConfig,
		getUser,
		getComments,
		addComment,
		updateComment,
		removeComment,
		vote,
	}
}
