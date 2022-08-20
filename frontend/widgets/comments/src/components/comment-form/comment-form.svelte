<script lang="ts">
	import { t } from '../../lib/i18n'
	import Textarea from '../../components/ui/textarea.svelte'
	import LazyComponent from 'svelte-lazy-component'
	import Button from '../ui/button.svelte'
	import { config } from '../../stores/config'
	import '@github/text-expander-element'

	export let as: string = 'form'
	export let expanded = false
	export let value = ''
	export let textareaId = 'comment'
	export let hideSubmitButton = false

	let focused = false
	let { simpleView, emojiSuggestions } = $config

	function handleTextareaFocus() {
		focused = true
		expanded = true
	}

	function handleTextareaBlur() {
		focused = false
	}
</script>

<svelte:element
	this={as}
	on:submit|preventDefault
	class="form"
	class:form-expanded={expanded}
	class:form-focused={focused}
>
	<div class="textarea-container" class:textarea-container-autoresizable={expanded}>
		<LazyComponent shouldLoad={emojiSuggestions} loader={() => import('./text-expander.svelte')}>
			<Textarea
				slot="always"
				id={textareaId}
				placeholder={$t('Write a comment...')}
				bind:value
				on:focus={handleTextareaFocus}
				on:blur={handleTextareaBlur}
				autoresizable={expanded}
			/>
		</LazyComponent>
	</div>
	<footer class="form-footer flex-shrink-0">
		<LazyComponent
			loader={() => import('./markdown-toolbar.svelte')}
			shouldLoad={expanded && !simpleView}
			{textareaId}
		/>
		<div class="ml-auto flex-shrink-0">
			<slot name="actions" />
			{#if !hideSubmitButton}
				{#if expanded}
					<Button type="button" kind="secondary">{$t('Preview')}</Button>
				{/if}
				<Button type="submit">{$t('Submit')}</Button>
			{/if}
		</div>
	</footer>
</svelte:element>

<style lang="postcss" global>
	.form {
		@apply flex items-center p-2 my-2 border border-slate-900/10 dark:border-white/40 rounded;
	}
	.form-expanded {
		@apply flex-col items-stretch;
	}
	.form-focused {
		@apply ring ring-teal-600/20 border border-teal-400;
	}
	.textarea-container {
		@apply h-6 w-full;
	}
	.textarea-container-autoresizable {
		@apply h-auto;
	}

	.form-footer {
		@apply flex items-center;
	}
</style>
