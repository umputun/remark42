import type { Vote, Comment, Sort, User } from '@remark42/api'
import { get, writable } from 'svelte/store'
import { publicApi } from '../lib/api'

export declare type Thread = {
	comment: Comment & { hidden?: boolean }
	replies?: Thread[]
	isFolded?: boolean
}
export type CommentsStore = Thread[]

function getPersistedSort(): Sort {
	return (localStorage.getItem('sort') as Sort) || '-active'
}

function persistSort(s: Sort) {
	localStorage.setItem('sort', s)
}
const isLoading = writable(false)
export const sort = writable<Sort>(getPersistedSort())
const threads: Map<string, Thread> = new Map()
export const comments = writable<CommentsStore>([], () => {
	revalidateComments(get(sort))
})

sort.subscribe((s) => {
	persistSort(s)
	revalidateComments(s)
})

function readPersistedHiddenUsers(): Map<string, User> {
	const hu = localStorage.getItem('hiddenUsers')

	try {
		return hu ? new Map(Object.entries(JSON.parse(hu))) : new Map()
	} catch (err) {
		return new Map()
	}
}

function persistHiddenUsers(users: Map<string, User>) {
	localStorage.setItem('hiddenUsers', JSON.stringify(users))
}

function revalidateComments(sort: Sort) {
	if (get(isLoading)) {
		return
	}

	isLoading.set(true)
	publicApi.getComments({ url: 'https://remark42.com/demo/', sort }).then((data: any) => {
		formatThreads(data.comments)
		foldThreads()
		filterUserComments([...readPersistedHiddenUsers().keys()])
		comments.set(data.comments)
		isLoading.set(false)
	})
}

function formatThreads(comments: Thread[]) {
	function walk(cc: Thread[]) {
		// debugger
		cc?.forEach((thread) => {
			threads.set(thread.comment.id, thread)

			if (thread.replies) {
				walk(thread.replies)
			}
		})
	}

	walk(comments)
}

export async function addComment(text: string, pid?: string) {
	const comment = await publicApi.addComment('https://remark42.com/demo/', {
		pid,
		text,
	})
	const thread = { comment }

	comments.update((c) => {
		threads.set(comment.id, thread)

		if (pid) {
			const parent = threads.get(pid)

			if (parent) {
				const parentReplies = parent.replies ?? []

				parent.replies = [...parentReplies, thread]
			}
		} else {
			c.unshift(thread)
		}

		return c
	})
}

function filterUserComments(userId: string): void
function filterUserComments(userIds: string[]): void
function filterUserComments(userId: string | string[]): void {
	for (let thread of threads.values()) {
		thread.comment.hidden =
			(typeof userId === 'string' && thread.comment.user.id === userId) ||
			userId.includes(thread.comment.user.id)
	}
}

function getPersistedFoldedThreads(): string[] {
	const data = localStorage.getItem('foldedThreads')
	let foldedThreads: string[] = []

	try {
		foldedThreads = data ? JSON.parse(data) : []
	} catch (err) {}

	return foldedThreads
}

function persistFoldeThreads() {
	const foldedThreads = Array.from(threads.values()).filter((t) => t.isFolded)
	localStorage.setItem('foldedThreads', JSON.stringify(foldedThreads.map((t) => t.comment.id)))
}

function foldThreads() {
	const foldedThreads = getPersistedFoldedThreads()

	foldedThreads.forEach((id) => {
		const thread = threads.get(id)

		if (!thread) {
			return
		}

		thread.isFolded = true
	})
}

export function hideComments(user: User) {
	const hiddenUsers = readPersistedHiddenUsers()
	hiddenUsers.set(user.id, user)
	filterUserComments(user.id)
	persistHiddenUsers(hiddenUsers)
	comments.update((c) => c)
}

export async function voteForComment(id: string, inc: Vote) {
	const { vote } = await publicApi.vote('https://remark42.com/demo/', id, inc)

	comments.update((c) => {
		const thread = threads.get(id)

		if (!thread) {
			return c
		}

		thread.comment.vote = vote as Vote
		thread.comment.score = thread.comment.score + inc

		return c
	})
}
export async function updateComment(id: string, text: string) {
	const comment = await publicApi.updateComment('https://remark42.com/demo/', id, text)

	comments.update((c) => {
		const thread = threads.get(comment.id)

		if (thread) {
			thread.comment = comment
		}

		persistFoldeThreads()

		return c
	})
}

export async function deleteComment(id: string) {
	const comment: Comment = await publicApi.removeComment('https://remark42.com/demo/', id)

	comments.update((c) => {
		const thread = threads.get(comment.id)

		if (thread) {
			thread.comment = comment
		}

		return c
	})
}

export function foldThread(commentId: string, fold: boolean) {
	const thread = threads.get(commentId)

	if (thread) {
		thread.isFolded = fold
	}

	persistFoldeThreads()
	comments.update((c) => c)
}
