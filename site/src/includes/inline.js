const mq = window.matchMedia('(prefers-color-scheme: dark)')
const theme = localStorage.getItem('theme')

if ((theme && theme === 'dark') || (!theme && mq.matches)) {
	document.documentElement.classList.add('dark')
}

// fetch the latest release version client-side so the badge tracks releases
// without needing a site rebuild. GitHub serves this with Cache-Control:
// public, max-age=60 so the per-visitor cost is bounded. placeholder is
// `hidden` in the template so a failed/blocked fetch leaves no stray gap.
fetch('https://api.github.com/repos/umputun/remark42/releases/latest')
	.then((r) => (r.ok ? r.json() : Promise.reject(new Error('HTTP ' + r.status))))
	.then((d) => {
		if (!d || !d.tag_name) return
		// script is loaded synchronously in <head>, so a cache hit can resolve
		// before <body> is parsed and [data-remark42-version] exists. defer the
		// DOM update until the document is ready.
		const apply = () => {
			document.querySelectorAll('[data-remark42-version]').forEach((el) => {
				el.textContent = d.tag_name
				el.hidden = false
			})
		}
		if (document.readyState === 'loading') {
			document.addEventListener('DOMContentLoaded', apply)
		} else {
			apply()
		}
	})
	.catch((err) => console.warn('remark42-site: latest version fetch failed', err))
