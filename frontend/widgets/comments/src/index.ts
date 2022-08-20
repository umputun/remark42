import App from './App.svelte'

let app: App

export function initCommentsWidget(target: HTMLElement = document.body) {
	if (!app) {
		app = new App({ target })
	}

	return app
}

export function destroyCommentsWidget() {
	app.$destroy()
}
