module.exports = {
	purge: ['src/**/*.{njk,md,html,js}'],
	mode: 'jit',
	darkMode: 'class',
	theme: {
		extend: {
			colors: {
				brand: {
					50: '#edfdfb',
					100: '#e0fbf8',
					200: '#aef4ee',
					300: '#4be7dc',
					400: '#1ccac1',
					500: '#16a29f',
					600: '#157f7f',
					700: '#126263',
					800: '#125254',
					900: '#134b4e',
				},
			},
			container: {
				center: true,
				screens: {
					sm: '100%',
					md: '860px',
					lg: '940px',
					xl: false,
				},
			},
			typography: (theme) => ({
				DEFAULT: {
					css: {
						'h1, h2': { color: theme('colors.brand.900') },
						'h3,h4,h5,h6': { color: theme('colors.gray.700') },
					},
				},
			}),
		},
	},
	variants: {},
	plugins: [require('@tailwindcss/typography')],
}
