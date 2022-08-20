<script>
	import { authApi } from '../../lib/api'
	import { user } from '../../stores/user'

	export let fullframe = false
	import Button from '../ui/button.svelte'

	let email = ''
	let username = ''

	function handleClickBack(evt) {
		evt.preventDefault()
		fullframe = false
	}

	function handleSubmit(evt) {
		evt.preventDefault()

		authApi.anonymous(username).then((data) => {
			user.set(data)
		})
	}
</script>

<form on:submit={handleSubmit}>
	{#if fullframe}
		<Button on:click={handleClickBack}>Back</Button>
	{/if}

	<!-- <Email>
  <Anonymous/> -->
	<input bind:value={username} placeholder="Username" />
	<!-- <input bind:value={email} /> -->

	<Button type="submit">Submit</Button>
</form>
