<script lang="ts">
	export let id: string
	export let value = ''
	export let placeholder = ''
	export let disabled = false
	export let autoresizable = true

	function resize({ target }: { target: HTMLTextAreaElement }) {
		target.style.height = ''
		target.style.height = `${target.scrollHeight}px`
	}

	function autoresize(target: HTMLTextAreaElement, autoresizable: boolean) {
		if (!autoresizable) {
			target.style.removeProperty('height')
			target.style.removeProperty('hidden')
			return
		}

		resize({ target })
		target.style.overflow = 'hidden'
		// @ts-ignore
		target.addEventListener('input', resize)

		return {
			destroy() {
				// @ts-ignore
				target.removeEventListener('input', resize)
			},
		}
	}
</script>

<textarea
	on:input
	on:focus
	on:blur
	bind:value
	use:autoresize={autoresizable}
	{id}
	{disabled}
	{placeholder}
	class="textarea"
	class:autoresizable
/>

<style lang="postcss" global>
	.textarea {
		@apply block py-1 h-6 leading-4 w-full resize-none bg-transparent text-base;
	}
	.autoresizable {
		@apply min-h-[60px] h-auto;
	}
</style>
