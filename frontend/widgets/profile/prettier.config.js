import tailwindcssPlugin from 'prettier-plugin-tailwindcss'
import defaultConfig from '../../prettier.config'

const config = {
  ...defaultConfig,
  tailwindConfig: './tailwind.config.js',
  plugins: [tailwindcssPlugin]
}

export default config
