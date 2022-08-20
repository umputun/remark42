<script lang="ts">
	import { icon } from './icon.lib'

	// icons
	import bell from './assets/bell.svg'
	import signout from './assets/signout.svg'
	import chevron from './assets/chevron.svg'
	import rss from './assets/rss.svg'
	import verified from './assets/verified.svg'
	import gear from './assets/gear.svg'
	import cross from './assets/cross.svg'

	const IconsMap = {
		// icons
		chevron,
		cross,
		bell,
		gear,
		signout,
		rss,
		paid: null,
		verified,
	} as const

	export let name: keyof typeof IconsMap
	let className: string = 'w-4 h-4'
	export { className as class }
</script>

<div class="icon {className}" use:icon={IconsMap[name]} />

<style lang="postcss">
	.icon > :global(svg) {
		@apply h-full w-full;
	}
</style>
