export function capitalizeFirstLetter(str: string): string {
	return `${str.charAt(0).toLocaleUpperCase()}${str.slice(1)}`
}

export function getButtonVariant(num: number) {
	if (num === 2) {
		return 'name'
	}

	if (num === 1) {
		return 'full'
	}

	return 'icon'
}
