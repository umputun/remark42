<script lang="ts">
	import LazyComponent from 'svelte-lazy-component'
	import type { Comment as IComment } from '@remark42/api'
	import {
		addComment,
		deleteComment,
		foldThread,
		hideComments,
		updateComment,
	} from '../../stores/comments'
	import User from '../user.svelte'
	import CommentForm from '../comment-form/comment-form.svelte'
	import Button from '../ui/button.svelte'
	import { t } from '../../lib/i18n'
	import { user } from '../../stores/user'
	import CommentVotes from './comment-votes.svelte'

	export let comment: IComment
	export let isFolded = false

	// `comment.orig` can be passed to an input/textarea because it renders any text safely
	let value = ''
	let isReplying = false
	let isEditing = false
	let isEditable = false

	$: canVote = !$user || $user?.id.startsWith('anonymous_')

	function handleClickReply() {
		isReplying = true
	}
	function handleClickCancelReply() {
		isReplying = false
	}

	function handleClickHide() {
		hideComments(comment.user)
	}
	function handleClickEdit() {
		value = comment.orig || ''
		isEditing = true
	}
	function handleClickCancelEdit() {
		value = ''
		isEditing = false
	}
	function handleClickDelete() {
		deleteComment(comment.id)
	}

	async function handleAddComment() {
		addComment(value, comment.id)
		isReplying = false
		value = ''
	}

	async function handleUpdateComment() {
		await updateComment(comment.id, value)
		isEditing = false
		value = ''
	}

	function handleClickUser() {
		if (!isFolded) {
			return
		}

		foldThread(comment.id, false)
	}
</script>

<article class="comment" class:comment_pinned={comment.pin} class:deleted={comment.delete}>
	<header class="comment-header" data-hidden={comment.hidden}>
		<User
			id={comment.user.id}
			name={comment.user.name}
			picture={comment.user.picture}
			on:click={handleClickUser}
		/>
		<CommentVotes id={comment.id} vote={comment.vote} score={comment.score} disabled={canVote} />
	</header>
	{#if !isFolded}
		{#if !isEditing}
			{#if !comment.delete}
				<div class="comment-content">
					<!-- NEVER RENDER `comment.orig` AS HTML -->
					{@html comment.text}
				</div>
			{:else}
				Deleted
			{/if}
		{:else}
			<CommentForm expanded bind:value on:submit={handleUpdateComment}>
				<Button slot="actions" kind="seamless" on:click={handleClickCancelEdit}>
					{$t('Cancel')}
				</Button>
			</CommentForm>
		{/if}

		<footer class="comment-footer">
			{#if !isEditing}
				<div class="comment-actions">
					{#if isReplying}
						<Button on:click={handleClickCancelReply} kind="seamless">
							{$t('Cancel')}
						</Button>
					{:else}
						<Button on:click={handleClickReply} kind="seamless">
							{$t('Reply')}
						</Button>
					{/if}
					{#if isEditable && !isReplying}
						<Button on:click={handleClickEdit} kind="seamless">{$t('Edit')}</Button>
						<!-- Admins have their own delete button  -->
						{#if !$user?.admin}
							<Button on:click={handleClickDelete} kind="seamless">{$t('Delete')}</Button>
						{/if}
					{/if}
					{#if isEditable}
						<div role="timer" />
					{/if}
					<div class="comments-actions-extra">
						{#if !isReplying}
							<Button on:click={handleClickHide} kind="seamless">{$t('Hide')}</Button>
						{/if}
						{#if $user && !isReplying}
							<LazyComponent
								shouldLoad={$user.admin}
								loader={() => import('./admin-actions.svelte')}
								id={comment.id}
								userId={$user.id}
								userBlocked={$user.block}
								content={comment.text}
								pinned={comment.pin}
							/>
						{/if}
					</div>
				</div>
			{/if}
			{#if isReplying}
				<div class="comment-replyForm">
					<CommentForm expanded bind:value on:submit={handleAddComment} />
				</div>
			{/if}
		</footer>
	{/if}
</article>

<style lang="postcss">
	.comment {
		@apply mb-4;
	}

	.comment_pinned {
		@apply bg-teal-50;
	}

	.comment :global(img) {
		@apply max-w-full;
	}

	.comment-header {
		@apply flex justify-between items-center;
	}

	.comment-content {
		@apply text-base my-2 pl-4;
	}

	.comment-footer {
		@apply pl-4;
	}

	.comment-actions {
		@apply flex gap-2 opacity-50 transition-opacity delay-75 duration-100;
	}

	.comment-actions:hover {
		@apply opacity-100;
	}

	.comments-actions-extra {
		@apply invisible;
	}
	.comment-actions:hover .comments-actions-extra {
		@apply visible;
	}

	.comment-replyForm {
		@apply w-full flex-shrink-0;
	}
</style>
