<script lang="ts">
	import type { CommentsStore } from '../stores/comments'
	import { foldThread } from '../stores/comments'
	import Comment from './comment/comment.svelte'

	export let level = 0
	export let comments: CommentsStore
</script>

{#each comments as { comment, replies, isFolded }}
	{#if !comment.hidden}
		<div class="thread" class:thread_folded={isFolded}>
			<Comment {comment} {isFolded} />
			{#if level < 7}
				<button
					class="thread-fold"
					type="button"
					arial-hidden=""
					on:click={() => foldThread(comment.id, !isFolded)}
				/>
			{/if}
			{#if !isFolded}
				{#if replies?.length}
					{#if level < 7}
						<div class="replies">
							<svelte:self comments={replies} level={level + 1} />
						</div>
					{:else}
						<svelte:self comments={replies} level={level + 1} />
					{/if}
				{/if}
			{/if}
		</div>
	{/if}
{/each}

<style lang="postcss">
	.thread {
		@apply relative;
	}
	.thread-fold {
		@apply block absolute left-0 top-[32px] h-[calc(100%_-_32px)] w-0 py-0 px-2 m-0 cursor-pointer  border-l border-dotted border-slate-200 dark:border-white/40;
		@apply hover:border-slate-400 hover:dark:border-white;
	}
	.replies {
		margin-left: 10px;
	}
</style>
