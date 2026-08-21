function toggleTheme() {
	const root = document.documentElement
	const theme = root.classList.contains('dark') ? 'light' : 'dark'

	root.classList.toggle('dark')
	localStorage.setItem('theme', theme)

	if (window.REMARK42) {
		window.REMARK42.changeTheme(theme)
	}
}
