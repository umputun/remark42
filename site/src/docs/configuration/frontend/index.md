---
title: Frontend Configuration
menuTitle: Frontend
parent: Configuration
order: 100
---

## Comments

It's the main widget that renders a list of comments.

Add this snippet to the bottom of web page:

```html
<script>
	var remark_config = {
		host: 'REMARK_URL', // hostname of Remark42 server, same as REMARK_URL in backend config, e.g. "https://demo.remark42.com"
		site_id: 'YOUR_SITE_ID',
		components: ['embed'], // optional param; which components to load. default to ["embed"]
		// to load all components define components as ['embed', 'last-comments', 'counter']
		// available component are:
		//   - 'embed': basic comments widget
		//   - 'last-comments': last comments widget, see `Last Comments` section below
		//   - 'counter': counter widget, see `Counter` section below
		url: 'PAGE_URL', // optional param; if it isn't defined
		// `window.location.origin + window.location.pathname` will be used
		//
		// Note that if you use query parameters as significant part of URL
		// (the one that actually changes content on page)
		// you will have to configure URL manually to keep query params, as
		// `window.location.origin + window.location.pathname` doesn't contain query params and
		// hash. For example, default URL for `https://example/com/example-post?id=1#hash`
		// would be `https://example/com/example-post`
		//
		// The problem with query params is that they often contain useless params added by
		// various trackers (utm params) and doesn't have defined order, so Remark42 treats differently
		// all this examples:
		// https://example.com/?postid=1&date=2007-02-11
		// https://example.com/?date=2007-02-11&postid=1
		// https://example.com/?date=2007-02-11&postid=1&utm_source=google
		//
		// If you deal with query parameters make sure you pass only significant part of it
		// in well defined order
		max_shown_comments: 10, // optional param; if it isn't defined default value (15) will be used
		theme: 'dark', // optional param; if it isn't defined default value ('light') will be used
		page_title: 'Moving to Remark42', // optional param; if it isn't defined `document.title` will be used
		locale: 'en', // set up locale and language, if it isn't defined default value ('en') will be used
		show_email_subscription: false, // optional param; by default it is `true` and you can see email subscription feature
		// in interface when enable it from backend side
		// if you set this param in `false` you will get notifications email notifications as admin
		// but your users won't have interface for subscription
		simple_view: false, // optional param; overrides the parameter from the backend
		// minimized UI with basic info only
	}
</script>
<script>
	!(function (e, n) {
		for (var o = 0; o < e.length; o++) {
			var r = n.createElement('script'),
				c = '.js',
				d = n.head || n.body
			'noModule' in r ? ((r.type = 'module'), (c = '.mjs')) : (r.async = !0),
				(r.defer = !0),
				(r.src = remark_config.host + '/web/' + e[o] + c),
				d.appendChild(r)
		}
	})(remark_config.components || ['embed'], document)
</script>
```

And then add this node in the place where you want to see Remark42 widget:

```html
<div id="remark42"></div>
```

After that widget will be rendered inside this node.

If you want to set this up on a Single Page App, see [appropriate doc page](site/src/docs/configuration/frontend/index.md).

##### Themes

Right now Remark42 has two themes: light and dark. You can pick one using a configuration object, but there is also a possibility to switch between themes in runtime. For this purpose Remark42 adds to `window` object named `REMARK42`, which contains a function `changeTheme`. Just call this function and pass a name of the theme that you want to turn on:

```js
window.REMARK42.changeTheme('light')
```

##### Locales

Right now Remark42 is translated to English (en), Belarusian (be), Brazilian Portuguese (bp), Bulgarian (bg), Chinese (zh), Finnish (fi), French (fr), German (de), Japanese (ja), Korean (ko), Polish (pl), Russian (ru), Spanish (es), Turkish (tr), Ukrainian (ua) and Vietnamese (vi) languages. You can pick one using [configuration object](#setup-on-your-website).

Do you want to translate Remark42 to other locales? Please see [this documentation](site/src/docs/contributing/translations/index.md) for details.

#### Last comments

It's a widget that renders the list of last comments from your site.

Add this snippet to the bottom of web page, or adjust already present `remark_config` to have `last-comments` in `components` list:

```html
<script>
	var remark_config = {
		host: 'REMARK_URL', // hostname of Remark42 server, same as REMARK_URL in backend config, e.g. "https://demo.remark42.com"
		site_id: 'YOUR_SITE_ID',
		components: ['last-comments'],
	}
</script>
```

And then add this node in the place where you want to see last comments widget:

```html
<div class="remark42__last-comments" data-max="50"></div>
```

`data-max` sets the max amount of comments (default: `15`).

#### Counter

It's a widget that renders several comments for the specified page.

Add this snippet to the bottom of web page, or adjust already present `remark_config` to have `counter` in `components` list:

```html
<script>
	var remark_config = {
		host: 'REMARK_URL', // hostname of Remark42 server, same as REMARK_URL in backend config, e.g. "https://demo.remark42.com"
		site_id: 'YOUR_SITE_ID',
		components: ['counter'],
	}
</script>
```

And then add a node like this in the place where you want to see a number of comments:

```html
<span
	class="remark42__counter"
	data-url="https://domain.com/path/to/article/"
></span>
```

You can use as many nodes like this as you need to. The script will found all of them by the class `remark__counter`, and it will use `data-url` attribute to define the page with comments.

Also script can use `url` property from `remark_config` object, or `window.location.origin + window.location.pathname` if nothing else is defined.

## Widgets

### Counter widget

### Last comments widget

## API for Single-Page Applications

Originally tested on [Nuxt.js](https://nuxtjs.org/), but it should be applicable to all SPAs.

- Add the following JavaScript to your `index.html`, which in this case, it is identical to `<script defer src="$HOST/web/embed.js"></script>`

```js
;(function () {
  var host = // Your remark42 host
  var components = ['embed'] // Your choice of remark42 components

  ;(function(c) {
    for (let i = 0; i < c.length; i++) {
      const d = document
      const s = d.createElement('script')
      s.src = remark_config.host + '/web/' + c[i] + '.js'
      s.defer = true
      ;(d.head || d.body).appendChild(s)
    }
  })(components)
})
```

- Created `remark42Instance` when the `div` containing remark42 has appeared, usually at `mounted` or `componentDidMount` of the SPA lifecycle. Destroy the previous instance first, if necessary.

```ts
  initRemark42() {
    if (window.REMARK42) {
      if (this.remark42Instance) {
        this.remark42Instance.destroy()
      }

      this.remark42Instance = window.REMARK42.createInstance({
        node: this.$refs.remark42 as HTMLElement,
        ...remark42_config  // See <https://github.com/patarapolw/remark42#setup-on-your-website>
      })
    }
  }

  mounted() {
    if (window.REMARK42) {
      this.initRemark42()
    } else {
      window.addEventListener('REMARK42::ready', () => {
        this.initRemark42()
      })
    }
  }
```

- Ensure that this is called every time route changes

```ts
  @Watch('$route.path')
  onRouteChange() {
    this.initRemark42()
  }
```

- And, destroyed before routeLeave

```ts
  beforeRouteLeave() {
    if (this.remark42Instance) {
      this.remark42Instance.destroy()
    }
  }
```
