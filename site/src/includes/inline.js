const mq = window.matchMedia('(prefers-color-scheme: dark)')

if (mq.matches) {
	document.documentElement.classList.add('dark')
}
