<script lang="ts">
	import { t } from '../lib/i18n'
</script>

<div>
	{@html $t('Powered by <a>Remark42</a>', (m) =>
		m.replace('<a>', '<a href="https://remark42.com" target="_blank">')
	)}
</div>
