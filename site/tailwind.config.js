const colors = require('tailwindcss/colors')
const { spacing } = require('tailwindcss/defaultTheme')

module.exports = {
	purge: ['.eleventy.js', 'src/**/*.{njk,md,html,js}'],
	mode: 'jit',
	darkMode: 'class',
	theme: {
		extend: {
			animation: {
				'from-right-36': 'from-right-36 300ms forwards',
				'to-right-36': 'to-right-36 300ms forwards',
			},
			colors: {
				trueGray: colors.trueGray,
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
				padding: spacing[4],
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
						pre: {
							color: theme('colors.gray.500'),
							backgroundColor: theme('colors.gray.100'),
						},
					},
				},
				dark: {
					css: [
						{
							color: theme('colors.gray.300'),
							a: {
								color: theme('colors.gray.200'),
							},
							strong: {
								color: theme('colors.gray.200'),
							},
							'ol > li::before': {
								color: theme('colors.gray.400'),
							},
							'ul > li::before': {
								backgroundColor: theme('colors.gray.600'),
							},
							hr: {
								borderColor: theme('colors.gray.300'),
							},
							blockquote: {
								color: theme('colors.gray.300'),
								borderLeftColor: theme('colors.gray.600'),
							},
							h1: {
								color: theme('colors.gray.200'),
							},
							h2: {
								color: theme('colors.gray.200'),
							},
							h3: {
								color: theme('colors.gray.200'),
							},
							h4: {
								color: theme('colors.gray.200'),
							},
							'figure figcaption': {
								color: theme('colors.gray.400'),
							},
							code: {
								color: theme('colors.gray.200'),
							},
							'a code': {
								color: theme('colors.gray.200'),
							},
							pre: {
								backgroundColor: theme('colors.gray.800'),
							},
							thead: {
								color: theme('colors.gray.200'),
								borderBottomColor: theme('colors.gray.400'),
							},
							'tbody tr': {
								borderBottomColor: theme('colors.gray.600'),
							},
							hr: {
								borderColor: theme('colors.gray.500'),
							},
						},
					],
				},
			}),
			keyframes: {
				'from-right-36': {
					'0%': {
						right: '-9rem',
					},
					'100%': {
						right: '0.5rem',
					},
				},
				'to-right-36': {
					'0%': {
						right: '0.5rem',
					},
					'100%': {
						right: '-9rem',
					},
				},
			},
		},
	},
	variants: {},
	plugins: [require('@tailwindcss/typography')],
}
