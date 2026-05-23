const mq = window.matchMedia('(prefers-color-scheme: dark)')
const theme = localStorage.getItem('theme')

if ((theme && theme === 'dark') || (!theme && mq.matches)) {
	document.documentElement.classList.add('dark')
}

// fetch the latest release version client-side so the badge tracks releases
// without needing a site rebuild. GitHub serves this with Cache-Control:
// public, max-age=60 so the per-visitor cost is bounded. fallback stays
// empty on any failure — graceful degradation, no broken UI.
fetch('https://api.github.com/repos/umputun/remark42/releases/latest')
	.then((r) => (r.ok ? r.json() : null))
	.then((d) => {
		if (!d || !d.tag_name) return
		document.querySelectorAll('[data-remark42-version]').forEach((el) => {
			el.textContent = d.tag_name
		})
	})
	.catch(() => {})
