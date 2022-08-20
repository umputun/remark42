<script lang="ts">
	import { onMount, onDestroy } from 'svelte'

	import Button from './button.svelte'

	export let buttonLabel = 'Dropdown'
	let show: boolean = false
	let root: HTMLDivElement
	let clickInside = false
	let disableClosing = false

	function toggleVisibility() {
		show = !show
	}

	function handleClickInside() {
		clickInside = true
	}

	function handleClickOutside() {
		// save current state
		const isClickInside = clickInside
		// reset state
		clickInside = false
		// if click was inside the dropdown, do nothing
		if (disableClosing || isClickInside) {
			return
		}
		// hide dropdown
		show = false
	}

	onMount(() => {
		root.addEventListener('click', handleClickInside, { capture: true })
		document.addEventListener('click', handleClickOutside)
	})

	onDestroy(() => {
		root.removeEventListener('click', handleClickInside)
		document.removeEventListener('click', handleClickOutside)
	})
</script>

<div class="dropdown" bind:this={root}>
	<slot name="button" onClick={toggleVisibility}>
		<Button on:click={toggleVisibility}>{buttonLabel}</Button>
	</slot>
	{#if show}
		<div class="dropdown-content">
			<slot />
		</div>
	{/if}
</div>

<style lang="postcss">
	.dropdown {
		@apply relative leading-[0];
	}
	.dropdown-content {
		position: absolute;
		top: 100%;
		border: 1px solid #ccc;
		padding: 10px;
		background: #fff;
	}
</style>
