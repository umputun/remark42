const { format } = require('date-fns')
const htmlmin = require('html-minifier')
const syntaxHighlightPlugin = require('@11ty/eleventy-plugin-syntaxhighlight')

function noteContainer() {
	const { utils } = require('markdown-it')()
	const elementRegexp = /^note\s+(.*)$/

	return {
		validate(params) {
			return params.trim().match(elementRegexp)
		},

		render(tokens, idx) {
			const { info, nesting } = tokens[idx]
			const matches = info.trim().match(elementRegexp)

			if (nesting === 1) {
				const icon = utils.escapeHtml(matches[1])

				return `<aside class="relative pr-4 pl-12 py-1 bg-gray-50 dark:bg-gray-800"><span class="absolute left-4 top-6 text-xl">${icon}</span>`
			}

			return `</aside>`
		},
	}
}

function getMarkdownLib() {
	const markdownIt = require('markdown-it')
	const markdownItAnchor = require('markdown-it-anchor')
	const markdownItContainer = require('markdown-it-container')

	return markdownIt({
		html: true,
		breaks: true,
		linkify: true,
	})
		.use(markdownItAnchor, {
			permalink: true,
			permalinkClass: '',
			permalinkSymbol: '',
		})
		.use(markdownItContainer, 'note', noteContainer())
}

module.exports = function (eleventyConfig) {
	// TODO: create version with commit sha and current version of Remark42
	eleventyConfig.addShortcode('version', () => `${Date.now()}`)
	eleventyConfig.setUseGitIgnore(false)
	eleventyConfig.addWatchTarget('./.tmp/style.css')
	eleventyConfig.addPassthroughCopy({ './.tmp/style.css': './style.css' })
	eleventyConfig.addPassthroughCopy({ './public': './' })
	eleventyConfig.addPassthroughCopy('./src/**/*.{gif,jpg,png,svg}')

	eleventyConfig.addCollection('pages', (collection) =>
		collection.getFilteredByGlob('pages/*.md')
	)

	eleventyConfig.addFilter('humanizeDate', (date) =>
		format(new Date(date), 'LLL dd, yyyy')
	)

	eleventyConfig.addFilter('robotizeDate', (date) =>
		format(new Date(date), 'yyyy-MM-dd')
	)

	eleventyConfig.addFilter(
		'debug',
		(content = {}) => `<pre>${JSON.stringify(content, null, 2)}</pre>`
	)

	// Minify HTML output
	eleventyConfig.addTransform('htmlmin', function (content, outputPath) {
		if (!outputPath.endsWith('.html')) {
			return content
		}

		return htmlmin.minify(content, {
			removeComments: true,
			collapseWhitespace: true,
		})
	})

	eleventyConfig.setLibrary('md', getMarkdownLib())
	eleventyConfig.addPlugin(syntaxHighlightPlugin)

	eleventyConfig.addCollection('docs', (collection) =>
		collection.getFilteredByGlob('src/docs/**/*.md')
	)

	return {
		dir: {
			input: 'src',
			output: 'build',
			data: 'data',
			layouts: 'layouts',
			includes: 'includes',
		},
	}
}
