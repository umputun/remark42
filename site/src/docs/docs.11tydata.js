module.exports = {
	eleventyComputed: {
		eleventyNavigation: {
			key: (data) => data.key || data.menuTitle || data.title,
			title: (data) => data.menuTitle || data.title,
			parent: (data) => data.parent,
			order: (data) => data.order,
		},
	},
}
