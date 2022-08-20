<script lang="ts">
	import { fade } from 'svelte/transition'
	import Avatar from './ui/avatar.svelte'

	export let id: string | undefined
	export let name: string | undefined
	export let picture: string | undefined
	export let fadein = false
</script>

<div data-id={id} class="user" in:fade={{ duration: fadein ? 100 : 0 }} on:click>
	{#if name && picture}
		<Avatar src={picture} username={name} size={20} />
		<span>{name}</span>
		<slot name="actions" />
	{/if}
</div>

<style lang="postcss">
	.user {
		@apply h-6 flex items-center gap-2;
	}
</style>
