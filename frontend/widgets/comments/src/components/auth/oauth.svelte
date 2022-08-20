<script lang="ts">
	import OauthItem from './oauth-item.svelte'
	import Telegram from './telegram.svelte'
	import { config } from '../../stores/config'

	export let fullframe = false

	function handleOauthClick(evt: MouseEvent) {
		evt.preventDefault()

		if (evt.currentTarget instanceof HTMLAnchorElement && evt.currentTarget.dataset.provider) {
			const { provider } = evt.currentTarget.dataset

			if (provider === 'telegram') {
				fullframe = true
			}
		}
	}

	function handleClickBack() {
		fullframe = false
	}

	const location = encodeURIComponent(
		`${window.location.origin}${window.location.pathname}?selfClose`
	)
</script>

{#if !fullframe}
	<ul>
		{#each $config.providers.oauth as provider}
			<li>
				<OauthItem on:click={handleOauthClick} {provider} {location} />
			</li>
		{/each}
	</ul>
{:else}
	<Telegram on:click={handleClickBack} />
{/if}
