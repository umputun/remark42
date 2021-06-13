function toggleTheme() {
	const root = document.documentElement

	root.classList.toggle('dark')

	if (window.REMARK42) {
		const isDark = root.classList.contains('dark')

		window.REMARK42.changeTheme(isDark ? 'dark' : 'light')
	}
}
