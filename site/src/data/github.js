const fetch = require('node-fetch')

const DEFAULT_DATA = { latestVersion: '' }
let currentData = null

module.exports = async function getLatestReleaseVersion() {
	if (currentData) {
		return currentData
	}

	const res = await fetch(
		'https://api.github.com/repos/umputun/remark42/releases'
	)

	if (!res.ok) {
		console.error(`[ERROR] Status: ${res.status}: ${res.statusText}`)

		return DEFAULT_DATA
	}

	const data = await res.json()

	currentData = { latestVersion: data[0].tag_name || '' }

	return currentData
}
