const mq = window.matchMedia('(prefers-color-scheme: dark)')
const theme = localStorage.getItem('theme')

if ((theme && theme === 'dark') || (!theme && mq.matches)) {
	document.documentElement.classList.add('dark')
}

// fetch the latest release version client-side so the badge tracks releases
// without needing a site rebuild. cached in sessionStorage for up to 1h so
// repeated navigations within a tab don't re-hit the GitHub API on every page
// load. placeholder is `hidden` in the template so a failed/blocked fetch
// leaves no stray gap.
const versionCacheKey = 'remark42-latest-version'
const versionCacheTTL = 60 * 60 * 1000

// script is loaded synchronously in <head>, so a cache-hit fetch can resolve
// before <body> is parsed and [data-remark42-version] exists. defer the DOM
// update until the document is ready.
function applyVersion(tag) {
	const update = () => {
		document.querySelectorAll('[data-remark42-version]').forEach((el) => {
			el.textContent = tag
			el.hidden = false
		})
	}
	if (document.readyState === 'loading') {
		document.addEventListener('DOMContentLoaded', update)
	} else {
		update()
	}
}

function readVersionCache() {
	try {
		const raw = sessionStorage.getItem(versionCacheKey)
		if (!raw) return null
		const { tag, expires } = JSON.parse(raw)
		if (!tag || typeof expires !== 'number' || Date.now() > expires) return null
		return tag
	} catch (e) {
		return null
	}
}

function writeVersionCache(tag) {
	try {
		sessionStorage.setItem(versionCacheKey, JSON.stringify({ tag, expires: Date.now() + versionCacheTTL }))
	} catch (e) {
		// sessionStorage unavailable or full — fall through, fetch will run next load
	}
}

const cachedVersion = readVersionCache()
if (cachedVersion) {
	applyVersion(cachedVersion)
} else {
	fetch('https://api.github.com/repos/umputun/remark42/releases/latest')
		.then((r) => (r.ok ? r.json() : Promise.reject(new Error('HTTP ' + r.status))))
		.then((d) => {
			if (!d || !d.tag_name) return
			writeVersionCache(d.tag_name)
			applyVersion(d.tag_name)
		})
		.catch((err) => console.warn('remark42-site: latest version fetch failed', err))
}
