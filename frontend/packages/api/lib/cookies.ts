export function getCookie(name: string): string | undefined {
	const matches = document.cookie.match(
		new RegExp(`(?:^|; )${name.replace(/([.$?*|{}()[\]\\/+^])/g, '\\$1')}=([^;]*)`)
	)

	if (matches === null) {
		return
	}

	return decodeURIComponent(matches[1])
}
