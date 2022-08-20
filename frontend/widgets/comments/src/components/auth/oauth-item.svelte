<script lang="ts">
	import type { OAuthProvider } from '@remark42/api'
	import SocialIcons from '../ui/icons/social-icons.svelte'

	export let provider: OAuthProvider
	export let location: string

	const BASE_URL = ''
	const siteId = ''
</script>

<a
	on:click
	target="_blank"
	rel="noopener noreferrer"
	href={`${BASE_URL}/auth/${provider}/login?from=${location}&site=${siteId}`}
	data-provider={provider}
>
	<SocialIcons name={provider} />
</a>
