const { format } = require('date-fns')
const htmlmin = require('html-minifier')
const syntaxHighlightPlugin = require('@11ty/eleventy-plugin-syntaxhighlight')
const navigationPlugin = require('@11ty/eleventy-navigation')

function getMarkdownLib() {
	const markdownIt = require('markdown-it')
	const markdownItAnchor = require('markdown-it-anchor')

	return markdownIt({
		html: true,
		breaks: true,
		linkify: true,
	}).use(markdownItAnchor, {
		permalink: true,
		permalinkClass: '',
		permalinkSymbol: '',
	})
}

module.exports = function (eleventyConfig) {
	// TODO: create version with commit sha and current version of Remark42
	eleventyConfig.addShortcode('version', () => `${Date.now()}`)
	eleventyConfig.setUseGitIgnore(false)
	eleventyConfig.addWatchTarget('./.tmp/style.css')
	eleventyConfig.addPassthroughCopy({ './.tmp/style.css': './style.css' })
	eleventyConfig.addPassthroughCopy({ './public': './' })

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
		(content) => `<pre>${JSON.stringify(content, null, 2)}</pre>`
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
	eleventyConfig.addPlugin(navigationPlugin)

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
