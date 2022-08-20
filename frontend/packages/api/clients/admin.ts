import type { ClientParams, User } from './index'
import { API_BASE } from '../consts'
import { createFetcher } from '../lib/fetcher'

export type BlockTTL = 'permanently' | '43200m' | '10080m' | '1440m'
export interface BlockUserResponse {
	block: boolean
	site_id: string
	user_id: string
}

export function createAdminClient({ siteId, baseUrl }: ClientParams) {
	const fetcher = createFetcher(siteId, `${baseUrl}${API_BASE}`)

	async function toggleUserVerification(id: string, verified: 0 | 1): Promise<void> {
		return fetcher.put(`/verify/${id}`, { verified })
	}

	async function toggleCommentPin(id: string, pinned: 0 | 1): Promise<void> {
		return fetcher.put(`/pin/${id}`, { pinned })
	}

	async function toggleCommenting(url: string, ro: 0 | 1): Promise<void> {
		return fetcher.put('/readonly', { url, ro })
	}

	async function toggleUserBlock(id: string, ttl?: BlockTTL): Promise<BlockUserResponse> {
		const params = ttl
			? {
					block: 1,
					ttl: ttl === 'permanently' ? 0 : ttl,
			  }
			: { block: 0 }

		return fetcher.put<BlockUserResponse>(`/user/${id}`, params)
	}

	/**
	 * Request list of blocked users
	 */
	async function getBlockedUsers(): Promise<User[]> {
		return fetcher.get<User[]>('/blocked')
	}

	/**
	 * Block user from commenting
	 * @param id user ID
	 * @param ttl block duration
	 */
	async function blockUser(id: string, ttl: BlockTTL): Promise<BlockUserResponse> {
		return toggleUserBlock(id, ttl)
	}

	/**
	 * Unblock user from commenting
	 * @param id user ID
	 */
	async function unblockUser(id: string): Promise<BlockUserResponse> {
		return toggleUserBlock(id)
	}

	/**
	 * Mark user as verified
	 * @param id user ID
	 */
	async function verifyUser(id: string): Promise<void> {
		return toggleUserVerification(id, 1)
	}
	/**
	 * Mark user as unverified
	 * @param id user ID
	 */
	async function unverifyUser(id: string): Promise<void> {
		return toggleUserVerification(id, 0)
	}
	/**
	 * Approve request to remove user data
	 * @param token token from email
	 */
	async function approveRemovingRequest(token: string): Promise<void> {
		return fetcher.get('/deleteme', { token })
	}

	/**
	 * Mark comment as pinned
	 * @param id comment ID
	 */
	async function pinComment(id: string): Promise<void> {
		return toggleCommentPin(id, 1)
	}
	/**
	 * Mark comment as unpinned
	 * @param id comment ID
	 */
	async function unpinComment(id: string): Promise<void> {
		return toggleCommentPin(id, 0)
	}
	/**
	 * Remove comment
	 * @param url page URL
	 * @param id comment ID
	 */
	async function removeComment(url: string, id: string): Promise<void> {
		return fetcher.delete(`/comment/${id}`, { url })
	}
	/**
	 * Enable commenting on a page
	 * @param url page URL
	 */
	async function enableCommenting(url: string) {
		return toggleCommenting(url, 1)
	}
	/**
	 * Disable commenting on a page
	 * @param url page URL
	 */
	async function disableCommenting(url: string) {
		return toggleCommenting(url, 0)
	}

	return {
		getBlockedUsers,
		blockUser,
		unblockUser,
		verifyUser,
		unverifyUser,
		approveRemovingRequest,
		pinComment,
		unpinComment,
		removeComment,
		enableCommenting,
		disableCommenting,
	}
}
