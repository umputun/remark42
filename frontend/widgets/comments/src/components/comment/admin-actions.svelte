<script lang="ts">
	import { copy } from '../../lib/copy'

	import { t } from '../../lib/i18n'

	import Button from '../ui/button.svelte'

	export let id: string
	export let pinned: boolean | undefined
	export let content: string
	export let userId: string
	export let userBlocked = false

	function blockUser() {
		console.log(userId)
	}
	function unblockUser() {
		console.log(userId)
	}
	function deleteComment() {
		console.log(id)
	}

	function pinComment() {}
	function unpinComment() {}
</script>

<Button on:click={() => copy(content)} kind="seamless">{$t('Copy')}</Button>
{#if pinned}
	<Button on:click={pinComment} kind="seamless">{$t('Pin')}</Button>
{:else}
	<Button on:click={unpinComment} kind="seamless">{$t('Unpin')}</Button>
{/if}
{#if userBlocked}
	<Button on:click={unblockUser} kind="seamless">{$t('Unblock')}</Button>
{:else}
	<Button on:click={blockUser} kind="seamless">{$t('Block')}</Button>
{/if}
<Button on:click={deleteComment} kind="seamless">{$t('Delete')}</Button>
