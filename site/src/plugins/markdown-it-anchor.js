function renderPermalink(Token, nextToken) {
	const content = [
		Object.assign(new Token('html_button', 'button', 1), {
			attrs: [
				['aria-label', 'Copy permalink'],
				['class', 'mr-2'],
				['data-permalink', ''],
				['type', 'button'],
			],
		}),
		Object.assign(new Token('html_block', '', 0), {
			content: this.permalinkSymbol,
		}),
		new Token('html_button', 'button', -1),
	];

	if (this.permalinkBefore) {
		nextToken.children.unshift(...content);
	} else {
		nextToken.children.push(...content);
	}
}

function slugify(str) {
	return encodeURIComponent(String(str)
		.trim().toLowerCase().replace(/\s+/g, '-'));
}

function uniqueSlug(slugs, slug) {
	if (slugs[slug]) {
		let uniq = `${slug}-1`;
		let i = 2;

		while (Object.prototype.hasOwnProperty.call(slugs, uniq)) {
			uniq = `${slug}-${i++}`;
		}

		slugs[uniq] = true;

		return uniq;
	}

	slugs[slug] = true;

	return slug;
}

function isLevelSelected(level, tag) {
	return level < parseInt(tag.substr(1), 10);
}

function markdownItAnchor(md, options) {
	markdownItAnchor.defaults = {
		...markdownItAnchor.defaults,
		...options,
	};

	const slugs = {};

	md.core.ruler.push('anchor', (state) => {
		state.tokens.forEach((token) => {
			if (token.type !== 'heading_open') return;
			if (!isLevelSelected(this.defaults.level, token.tag)) return;

			const nextToken = state.tokens[state.tokens.indexOf(token) + 1];

			if (!nextToken) return;

			const title = nextToken.children
				.filter(({ type }) => type === 'text' || type === 'code_inline')
				.reduce((acc, cur) => acc + cur.content, '');
			const id = token.attrGet('id');
			const slug = uniqueSlug(slugs, id || this.defaults.slugify(title));

			token.attrPush(['id', slug]);

			if (this.defaults.permalink) {
				this.defaults.renderPermalink(state.Token, nextToken);
			}

			if (this.defaults.callback) {
				this.defaults.callback(token, { slug, title });
			}
		});
	});
}

markdownItAnchor.defaults = {
	renderPermalink,
	slugify,
	level: 1,
	permalink: true,
	permalinkBefore: true,
	permalinkSymbol: '#',
};

module.exports = markdownItAnchor;
