import type { ClientParams, Provider, User } from './index'
import { createFetcher } from '../lib/fetcher'
import { API_BASE } from '../consts'

export interface Config {
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
	telegram_bot_username: string
	emoji_enabled: boolean
}

export interface Comment {
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

export interface CommentsTree {
	comment: Comment
	replies: Comment[]
}

export interface CommentPayload {
	title?: string
	pid?: string
	text: string
}

export type Sort = '-active' | '+active'
export interface GetUserCommentsParams {
	url: string
	sort?: Sort
	limit?: number
	skip?: number
}
export type Vote = -1 | 1

export function createPublicClient({ siteId: site, baseUrl }: ClientParams) {
	const fetcher = createFetcher(site, `${baseUrl}${API_BASE}`)

	/**
	 * Get server config
	 */
	async function getConfig(): Promise<Config> {
		return fetcher.get('/config')
	}

	/**
	 * Get current authorized user
	 */
	async function getUser(): Promise<User | null> {
		return fetcher.get<User | null>('/user').catch(() => null)
	}

	/**
	 * Get comments
	 */
	async function getComments(url: string): Promise<CommentsTree>
	async function getComments(params: GetUserCommentsParams): Promise<Comment[]>
	async function getComments(
		params: string | GetUserCommentsParams
	): Promise<Comment[] | CommentsTree> {
		if (typeof params === 'string') {
			return fetcher.get('/comments', { url: params })
		}

		return fetcher.get<CommentsTree>('/find', { ...params, format: 'tree' })
	}

	/**
	 * Add new comment
	 */
	async function addComment(url: string, payload: CommentPayload): Promise<Comment> {
		const locator = { site, url }
		return fetcher.post('/comment', {}, { ...payload, locator })
	}

	/**
	 * Update comment
	 */
	async function updateComment(url: string, id: string, text: string): Promise<Comment> {
		return fetcher.put(`/comment/${id}`, { url }, { text })
	}

	/**
	 * Remove comment on a page
	 */
	async function removeComment(url: string, id: string): Promise<void> {
		return fetcher.put(`/comment/${id}`, { url }, { delete: true })
	}

	/**
	 * Vote for a comment
	 */
	async function vote(url: string, id: string, vote: Vote): Promise<{ id: string; vote: number }> {
		return fetcher.put<{ id: string; vote: number }>(`/vote/${id}`, { url, vote })
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
