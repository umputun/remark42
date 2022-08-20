<script lang="ts">
	export let as: keyof HTMLElementTagNameMap = 'button'
	export let disabled = false
	export let kind: 'primary' | 'secondary' | 'seamless' = 'primary'
	export let type = 'button'
	export let title: string | undefined = undefined
</script>

<svelte:element this={as} class="button {kind}" on:click {type} {disabled} {title}>
	<slot />
</svelte:element>

<style lang="postcss">
	.button {
		@apply py-1 px-3 rounded;
	}

	.seamless {
		@apply p-0 text-inherit dark:text-slate-200;
	}

	.primary {
		@apply bg-teal-700 text-white text-sm;
	}
	.secondary {
		@apply bg-teal-200 dark:text-white dark:bg-slate-100/10 text-sm hover:dark:bg-slate-100/20;
	}
</style>
