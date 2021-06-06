const typographyPlugin = require('@tailwindcss/typography')

module.exports = {
	purge: ['src/**/*.{njk,md,html,js}'],
	mode: 'jit',
	future: {
		purgeLayersByDefault: true,
	},
	darkMode: 'class',
	theme: {
		extend: {
			container: {
				center: true,
				screens: {
					sm: '100%',
					md: '860px',
					lg: '940px',
					xl: false,
				},
			},
		},
	},
	variants: {},
	plugins: [typographyPlugin],
}
