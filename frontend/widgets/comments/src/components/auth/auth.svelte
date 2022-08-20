<script lang="ts">
	import { config } from '../../stores/config'
	import Dropdown from '../ui/dropdown.svelte'
	import Form from './form.svelte'
	import Oauth from './oauth.svelte'

	// async function handleAnonymousAuth(evt: Event) {
	// 	evt.preventDefault()

	// 	const user = await authApi.anonymous(username)

	// 	mutateUser(user)
	// }

	let fullframeForm = false
	let fullframeOauth = false

	$: showOauth = $config?.providers.oauth.length && !fullframeForm
	$: showForm = $config?.providers.form.length && !fullframeOauth
</script>

<Dropdown buttonLabel="Sign Up">
	{#if showOauth}
		<Oauth bind:fullframe={fullframeOauth} />
	{/if}
	{#if showForm}
		<Form bind:fullframe={fullframeOauth} />
	{/if}
</Dropdown>
