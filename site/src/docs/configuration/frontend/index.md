---
title: Frontend Configuration
---

## Configuration

- **`host`**`: string` (required) – hostname of Remark42 server, same as REMARK_URL in backend config, e.g. "https://demo.remark42.com"
- **`site_id`**`: string` (optional, `remark` by default) – the `SITE` that you passed to Remark42 instance on start of backend.
- **`url`**`: string` (optional, `window.location.origin + window.location.pathname` by default) – url to the page with comments, it is used as unique identificator for comments thread
  Note that if you use query parameters as significant part of URL (the one that actually changes content on page) you will have to configure URL manually to keep query params, as `window.location.origin + window.location.pathname` doesn't contain query params and hash. For example, default URL for `https://example/com/example-post?id=1#hash` would be `https://example/com/example-post`
- **`components`**`: ['embed' | 'last-comments' | 'counter']` (optional, `['embed']` by default) – an array of widgets that should be rendered on a page. You may use more than one widget on a page.
  Available components are:
  - `'embed'` – basic comments widget
  - `'last-comments'` – last comments widget, see [Last Comments](#last-comments-widget) section below
  - `'counter'` – counter widget, see [Counter](#counter-widget) section below
- **`max_shown_comments`**`: number` (optional, `15` by default) – maximum number of comments that is rendered on mobile version
- **`max_last_comments`**`: number` (optional, `15` by default) – maximum number of comments in the last comments widget
- **`theme`**`: 'light' | 'dark'` (optional, `'light'` by default) – changes UI theme
- **`page_title`**`: string` (optional, `document.title` by default) – title for current comments page
- **`locale`**`: enum` (optional, `'en'` by default) – interface localization, [check possible localizations](#locales)
- **`show_email_subscription`**`: boolean` (optional, `true` by default) – enables email subscription feature in interface when enable it from backend side, if you set this param in `false` you will get notifications email notifications as admin but your users won't have interface for subscription
- **`show_rss_subscription`**`: boolean` (optional, `true` by default) – enables RSS subscription feature in interface
- **`simple_view`**`: boolean` (optional, `false` by default) – overrides the parameter from the backend minimized UI with basic info only
- **`no_footer`**`: boolean` (optional, `false` by default) – hides footer with signature and links to Remark42
- **`__font__`**`: string` (optional, `system-ui` by default) - a string containing the fonts being used, delimited by commas for alternative fonts if the first one doesn't exist on the system.
- **`__colors__`**`: ['--primary-color:': 'r, g, b', '--primary-text-color:' 'r, g, b', .., '--shadow.color:' 'r, g, b, a']` (optional) - An array containing different rgb values for changing the interface colors, and an rgba value for changing the shadows. Changing these settings will only work using the light theme. Colors not included in the array will have their default values. Note that interface elements might change their default color group in the future, so any changes made here are likely to break with future updates.

Example with all of the params:

```html
<script>
	var remark_config = {
		host: 'https://remark42.example.com',
		url: '/blog/my-first-blog-post/',
		site_id: 'my_site',
		components: ['embed', 'last-comments'],
		max_shown_comments: 100,
		max_last_comments: 100,
		theme: 'light',
		page_title: 'My custom title for a page',
		locale: 'es',
		show_email_subscription: false,
		show_telegram_subscription: false,
		show_rss_subscription: false,
		simple_view: true,
		no_footer: false,
		__font__: 'Comic Sans MS, Comic Sans, Chalkboard SE, Comic Neue, sans-serif',
		__colors__: {
		  '--primary-background-color': '0, 0, 255',
			'--secondary-background-color': '255, 0, 0',
			'--primary-color': '255, 255, 0',
			'--primary-brighter-color': '255, 255, 0',
			'--primary-darker-color': '0, 0, 0',
			'--primary-text-color': '255, 255, 255',
			'--secondary-text-color': '255, 255, 0',
			'--inverse-text-color': '255, 0, 0',
			'--lighter-text-color': '255, 255, 255',
			'--line-color': '0, 0, 0',
			'--line-brighter-color': '0, 0, 0',
			'--deactivated-color': '255, 0, 0',
			'--highlight-color': '255, 0, 255',
			'--selection-color': '255, 255, 0',
			'--downvote-color': '255, 0, 0',
			'--upvote-color': '0, 255, 0',
			'--verified-color': '0, 0, 255',
			'--white-color': '255, 255, 255',
			'--error-color': '255, 0, 0',
			'--error-background': '255, 255, 0',
			'--shadow-color': '0, 0, 0, 0.5'
		}
	}
</script>
```

## Basic configuration

Place configuration on a page of your site.
Add following **initialization** script after it.

<!-- prettier-ignore-start -->
```html
<script>!function(e,n){for(var o=0;o<e.length;o++){var r=n.createElement("script"),c=".js",d=n.head||n.body;"noModule"in r?(r.type="module",c=".mjs"):r.async=!0,r.defer=!0,r.src=remark_config.host+"/web/"+e[o]+c,d.appendChild(r)}}(remark_config.components||["embed"],document);</script>
```
<!-- prettier-ignore-end -->

## Comments

It's the main widget that renders a list of comments with ability of commenting.
Add following snippet in the place where you want to see Remark42 widget. The comments widget will be rendered in that place.

```html
<div id="remark42"></div>
```

::: note 💡
**Note:** The initialization script should be placed after the code mentioned above.
:::

You can place any placeholder content inside the `remark42` div — it will be automatically removed once the comments widget has loaded. This is useful for showing a loading indicator or message while the widget initialises:

```html
<div id="remark42">Comments loading...</div>
```

If you want to set this up on a Single Page App, see the [appropriate doc page](https://remark42.com/docs/configuration/frontend/spa/).

#### Themes

Remark42 has two themes: light and dark. You can pick one using a configuration object, but there is also a possibility to switch between themes in runtime. For this purpose, Remark42 adds to the `window` object named `REMARK42`, which contains a function `changeTheme`. Just call this function and pass a name of the theme that you want to turn on:

```js
window.REMARK42.changeTheme("light")
```

#### Locales

Right now Remark42 is translated to English (en), Belarusian (be), Brazilian Portuguese (bp), Bulgarian (bg), Chinese (zh), Finnish (fi), French (fr), German (de), Japanese (ja), Korean (ko), Polish (pl), Russian (ru), Spanish (es), Turkish (tr), Ukrainian (ua), Italian (it) and Vietnamese (vi) languages. You can pick one using a [configuration object](https://remark42.com/docs/getting-started/installation/#setup-on-your-website).

Do you want to translate Remark42 to other locales? Please see [this documentation](https://remark42.com/docs/contributing/translations/) for details.

## Widgets

### Last comments widget

It's a widget that renders the list of last comments from your site.

Add this snippet to the bottom of web page, or adjust already present `remark_config` to have `last-comments` in `components` list:

```html
<script>
	var remark_config = {
		host: "REMARK_URL",
		site_id: "YOUR_SITE_ID",
		components: ["last-comments"],
	}
</script>
```

::: note 💡
**Note:** If you want to render not only last comments widget you need to add all of the names of widget that you want to initialize.
:::

And then add this node in the place where you want to see last comments widget:

```html
<div class="remark42__last-comments" data-max="50"></div>
```

`data-max` sets the max amount of comments (default: `15`).

### Counter widget

It's a widget that renders several comments for the specified page.
Add this snippet to the bottom of web page, or adjust already present `remark_config` to have `counter` in `components` list:

```html
<script>
	var remark_config = {
		host: "REMARK_URL",
		site_id: "YOUR_SITE_ID",
		components: ["counter"],
	}
</script>
```

::: note 💡
**Note:** If you want to render not only comments widget you need to add all of the names of widget that you want to initialize.
:::

And then add a node like this in the place where you want to see a number of comments:

```html
<span
	class="remark42__counter"
	data-url="https://domain.com/path/to/article/"
></span>
```

You can use as many nodes like this as you need to. The script will find all of them by the class `remark__counter`, and it will use the `data-url` attribute to define the page with comments.

Also, the script can use `url` property from `remark_config` object or `window.location.origin + window.location.pathname` if nothing else is defined.
