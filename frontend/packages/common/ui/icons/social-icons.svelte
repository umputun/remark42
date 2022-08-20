<script lang="ts">
	import { icon } from './icon.lib'

	import dev from './assets/dev.svg'
	import facebook from './assets/facebook.svg'
	import github from './assets/github.svg'
	import google from './assets/google.svg'
	import microsoft from './assets/microsoft.svg'
	import patreon from './assets/patreon.svg'
	import telegram from './assets/telegram.svg'
	import twitter from './assets/twitter.svg'
	import yandex from './assets/yandex.svg'

	const IconsMap = {
		dev,
		facebook,
		github,
		google,
		microsoft,
		patreon,
		telegram,
		twitter,
		yandex,
	} as const

	export let name: keyof typeof IconsMap
</script>

<div use:icon={IconsMap[name]} />
