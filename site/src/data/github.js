const fetch = require('node-fetch')

module.exports = async function getLatestReleaseVersion() {
	const res = await fetch(
		'https://api.github.com/repos/umputun/remark42/releases'
	)
	const data = await res.json()

	return { latestVersion: data[0].tag_name }
}
