export function icon(target: HTMLDivElement, src: string) {
	// avoid loading when run tests
	if (import.meta.env.VITEST) {
		return
	}
	fetch(src)
		.then((res) => {
			if (!res.ok) {
				throw new Error(`Failed to load icon ${src}`)
			}

			return res.text()
		})
		.then((html) => {
			target.innerHTML = html
		})
		.catch(() => {
			console.error(`Could not load icon ${src}`)
		})
}
