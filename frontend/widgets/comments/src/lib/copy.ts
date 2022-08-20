export async function copy(content: string): Promise<void> {
	// Use new API for modern browsers
	if ('clipboard' in window.navigator) {
		const blob = new Blob([content], { type: 'text/plain' })
		return navigator.clipboard.write([new ClipboardItem({ [blob.type]: blob })])
	}

	// We use `div` instead of `input` or `textarea` because we want to copy styles
	const container = document.createElement('div')
	const previouslyFocusedElement = document.activeElement as HTMLElement

	container.innerHTML = content

	Object.assign(container.style, {
		contain: 'strict',
		position: 'absolute',
		left: '-9999px',
		fontSize: '12pt', // Prevent zooming on iOS
	})

	document.body.appendChild(container)

	const selection = window.getSelection()
	// save original selection
	const originalRange = selection && selection.rangeCount > 0 ? selection.getRangeAt(0) : null
	const range = document.createRange()

	range.selectNodeContents(container)

	if (selection) {
		selection.removeAllRanges()
		selection.addRange(range)
	}

	let success = false
	try {
		success = document.execCommand('copy')
	} catch (err) {}

	if (selection) {
		selection.removeAllRanges()
	}

	document.body.removeChild(container)

	// Put the selection back in case had it before
	if (originalRange && selection) {
		selection.addRange(originalRange)
	}

	// Get the focus back on the previously focused element, if any
	if (previouslyFocusedElement) {
		previouslyFocusedElement.focus()
	}

	return success ? Promise.resolve() : Promise.reject()
}
