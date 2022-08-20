<script lang="ts">
	import { t } from '../../lib/i18n'
	import { voteForComment } from '../../stores/comments'
	import Icon from '../ui/icons/icon.svelte'

	export let id: string
	export let vote: 0 | -1 | 1
	export let score: number
	export let disabled: boolean | undefined = undefined

	// let errorMessage: string | undefined

	$: isUpvoted = vote === 1
	$: isDownvoted = vote === -1
	$: value = loadingState?.score ?? score
	// isPositiveScore = StaticStore.config.positive_score && value > -1;
	$: isPositiveScore = false
	$: downvoteIsDisabled = loadingState !== null || isDownvoted || isPositiveScore

	let loadingState: { vote: number; score: number } | null = null

	async function handleClick(evt: MouseEvent) {
		const { value } = (evt.currentTarget as HTMLButtonElement).dataset
		const increment = Number(value) as -1 | 1

		loadingState = { vote: vote + increment, score: score + increment }

		try {
			await voteForComment(id, increment)
			// errorMessage = undefined
			setTimeout(() => {
				loadingState = null
			}, 200)
		} catch (err) {
			// setErrorMessage(extractErrorMessageFromResponse(err, intl))
			// errorMessage = err instanceof Error ? err.message : t('Something went wrong')
			loadingState = null
		}
	}

	// export const messages = defineMessages({
	//   score: {
	//     id: 'vote.score',
	//     defaultMessage: 'Votes score',
	//   },
	//   upvote: {
	//     id: 'vote.upvote',
	//     defaultMessage: 'Vote up',
	//   },
	//   downvote: {
	//     id: 'vote.downvote',
	//     defaultMessage: 'Vote down',
	//   },
	//   controversy: {
	//     id: 'vote.controversy',
	//     defaultMessage: 'Controversy: {value}',
	//   },
	// }); -->
</script>

<span class="votes" class:votes-disabled={disabled}>
	{#if !disabled}
		<button
			class="vote-button vote-button_downvote"
			class:vote-button_downvote-active={isDownvoted}
			on:click={handleClick}
			data-value={-1}
			title={$t('Vote down')}
			disabled={downvoteIsDisabled}
		>
			<Icon name="chevron" />
		</button>
	{/if}
	<div
		class="score"
		class:positive-score={score > 0}
		class:negative-score={score < 0}
		title={$t('Votes score')}
	>
		{value}
	</div>
	{#if !disabled}
		<button
			class="vote-button vote-button_upvote"
			class:vote-button_upvote-active={isUpvoted}
			on:click={handleClick}
			data-value={1}
			title={$t('Vote up')}
			disabled={loadingState !== null || isUpvoted}
		>
			<Icon name="angle" />
		</button>
	{/if}
</span>

<style lang="postcss">
	.votes {
		@apply flex items-center font-bold select-none;
	}

	.votes-disabled {
		@apply justify-center;
	}

	.vote-button {
		@apply flex items-center p-1 h-6 w-6 opacity-40;
		transition: opacity 0.15s color 0.15s;
	}

	.votes:hover .vote-button {
		@apply opacity-100;
	}

	.vote-button_upvote :global(svg) {
		@apply rotate-180;
	}

	.vote-button_upvote:hover,
	.vote-button_upvote-active {
		@apply opacity-100;
	}

	.vote-button_downvote:hover,
	.vote-button_downvote-active {
		@apply opacity-100;
	}

	.score {
		@apply mx-1 p-1 font-bold rounded min-w-[2rem] text-center;
	}

	.negative-score {
		@apply bg-red-500/10 dark:text-red-600;
	}

	.positive-score {
		@apply bg-green-500/10 dark:text-green-600;
	}

	/* .errorMessage {
		white-space: nowrap;
		font-weight: 500;
	} */
</style>
