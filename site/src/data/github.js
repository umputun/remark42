const DEFAULT_DATA = { latestVersion: '' }
let currentData = null

module.exports = async function getLatestReleaseVersion() {
	if (currentData) {
		return currentData
	}
	try {
		const fetch = await import('node-fetch')
		const res = await fetch(
			'https://api.github.com/repos/umputun/remark42/releases'
		)

		if (!res.ok) {
			throw new Error(`[ERROR] Status: ${res.status}: ${res.statusText}`)
		}

		const data = await res.json()

		currentData = { latestVersion: data[0].tag_name || '' }

		return currentData
	} catch (e) {
		console.error(e.message)
		return DEFAULT_DATA
	}
}
