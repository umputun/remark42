<script>
	export let shouldLoad
	export let loader
</script>

{#if shouldLoad}
	{#await loader().then((m) => m.default) then Component}
		<Component {...$$restProps}>
			<slot />
			<slot name="always" />
		</Component>
	{/await}
{:else}
	<slot />
	<slot name="always" />
{/if}
