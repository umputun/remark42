---
title: Frontend Configuration
aliases:
  - /docs/configuration/
---

## Configuration

- **`host`**`: string` – URL of the Remark42 server including the scheme, same as `REMARK_URL` in backend config, e.g. "https://demo.remark42.com". The [initialization script](#basic-configuration) needs it to build the script addresses. A page that loads the widget with its own `<script type="module" src="https://remark42.example.com/web/embed.mjs"></script>` can leave it out: the widget then uses the server's `REMARK_URL`
- **`site_id`**`: string` (required) – the `SITE` that you passed to Remark42 instance on start of backend.
- **`url`**`: string` (optional, `window.location.origin + window.location.pathname` by default) – url to the page with comments, it is used as unique identifier for comments thread
  Note that if you use query parameters as significant part of URL (the one that actually changes content on page) you will have to configure URL manually to keep query params, as `window.location.origin + window.location.pathname` doesn't contain query params and hash. For example, default URL for `https://example.com/example-post?id=1#hash` would be `https://example.com/example-post`. Everything from `#` on is dropped, also from a `url` you set, so pages that differ only after `#` share one thread
- **`components`**`: ['embed' | 'last-comments' | 'counter']` (optional, `['embed']` by default) – an array of widgets that should be rendered on a page. You may use more than one widget on a page.
  Available components are:
  - `'embed'` – basic comments widget
  - `'last-comments'` – last comments widget, see [Last Comments](#last-comments-widget) section below
  - `'counter'` – counter widget, see [Counter](#counter-widget) section below
- **`node`**`: HTMLElement` (optional, the element with the id `remark42` by default) – the DOM element to mount the comments widget into. Mostly useful for [single-page applications](/docs/configuration/frontend/spa/), where the element is available as a component reference. Pass it in the configuration object given to `REMARK42.createInstance`
- **`max_shown_comments`**`: number` (optional, `10` by default) – initial number of top-level comments shown on mobile
- **`comments_page_size`**`: number` (optional, `100` by default) – how many comments the widget loads at a time. A top-level comment always comes with all its replies, so a page can hold more; "Show more" at the bottom loads the next page, and a link to a comment on a page not loaded yet loads the rest of the thread. `0` loads the whole thread at once
- **`max_last_comments`**`: number` (optional, `15` by default) – maximum number of comments in the last comments widget
- **`default_sort`**`: string` (optional, `'-active'` by default) – comment order for a reader who has not picked one in the sort picker: `'-score'` (best), `'+score'` (worst), `'-time'` (newest), `'+time'` (oldest), `'-active'` (recently updated), `'+active'` (least recently updated), `'-controversy'` (most controversial) or `'+controversy'` (least controversial). A reader's own choice in the sort picker is remembered and wins over this setting
- **`theme`**`: 'light' | 'dark' | 'auto'` (optional, `'light'` by default) – changes UI theme, `'auto'` follows the visitor's system preference, see [Themes](#themes)
- **`page_title`**`: string` (optional, `document.title` by default) – title for current comments page
- **`locale`**`: enum` (optional, `'en'` by default) – interface localization, [check possible localizations](#locales)
- **`show_email_subscription`**`: boolean` (optional, `true` by default) – shows the email subscription control when email notifications for users are enabled on the backend; `false` hides it from readers and does not affect admin notifications
- **`show_telegram_subscription`**`: boolean` (optional, `true` by default) – shows Telegram subscription controls when Telegram user notifications are enabled on the backend
- **`show_rss_subscription`**`: boolean` (optional, `true` by default) – enables RSS subscription feature in interface
- **`simple_view`**`: boolean` (optional, `false` by default) – simplifies the comment form, hiding the formatting toolbar and the preview button; `false` does not disable `SIMPLE_VIEW` enabled on the backend
- **`no_footer`**`: boolean` (optional, `false` by default) – hides footer with signature and links to Remark42
- **`custom_properties`**`: object` (optional) – CSS custom properties for the widget, see [Matching your site's font](#matching-your-sites-font)

Configuration example:

```html
<script>
	var remark_config = {
		host: 'https://remark42.example.com',
		site_id: 'my_site',
		components: ['embed', 'last-comments'],
		max_shown_comments: 100,
		theme: 'dark',
		page_title: 'My custom title for a page',
		locale: 'es',
		show_email_subscription: false,
		simple_view: true,
		no_footer: false
	}
</script>
```

## Basic configuration

Place configuration on a page of your site.
Add following **initialization** script after it.

<!-- prettier-ignore-start -->
```html
<script>!function(e,n){for(var o=0;o<e.length;o++){var r=n.createElement("script"),d=n.head||n.body;r.type="module",r.async=!0,r.defer=!0,r.src=remark_config.host+"/web/"+e[o]+".mjs",d.appendChild(r)}}(remark_config.components||["embed"],document);</script>
```
<!-- prettier-ignore-end -->

Snippets written for older versions that load `embed.js`, `counter.js` or `last-comments.js` keep working: the server answers those names with the `.mjs` files.

## Comments

It's the main widget that renders a list of comments with ability of commenting.
Add following snippet in the place where you want to see Remark42 widget. The comments widget will be rendered in that place.

```html
<div id="remark42"></div>
```

{{< note "💡" >}}
**Note:** The configuration has to come before the initialization script, which reads it. The script and the `remark42` element can be in any order: the widget waits until the page has loaded.
{{< /note >}}

You can place any placeholder content inside the `remark42` div — it will be automatically removed once the comments widget has loaded. This is useful for showing a loading indicator or message while the widget initialises:

```html
<div id="remark42">Comments loading...</div>
```

If you want to set this up on a Single Page App, see the [appropriate doc page](https://remark42.com/docs/configuration/frontend/spa/).

### Themes

Remark42 has two themes: light and dark. You can pick one using a configuration object, but there is also a possibility to switch between themes in runtime. For this purpose, Remark42 adds an object named `REMARK42` to `window`, which contains a function `changeTheme`. Just call this function and pass a name of the theme that you want to turn on:

```js
window.REMARK42.changeTheme("light")
```

Set `theme` to `'auto'` to follow the visitor's system preference instead: the widget opens in dark when their system asks for a dark color scheme (the `prefers-color-scheme` media query) and in light otherwise, and switches as soon as that preference changes, without a reload. This suits a site whose own styles follow the same media query. `changeTheme("auto")` turns this on at runtime, and `changeTheme("light")` or `changeTheme("dark")` turns it off again.

`changeTheme` can be called as soon as `window.REMARK42` exists, even before the comments have finished loading: the widget opens in the last theme it was given. Set `theme` in the configuration object to the theme your page is in when it loads, so the widget does not have to switch after it appears.

### Matching your site's font

To match the widget's font to your site's, set `remark_config.custom_properties['--font-family']`
to the font your site uses. You don't need to modify or rebuild Remark42 itself to do this:

```js
var remark_config = {
	host: "REMARK_URL",
	site_id: "YOUR_SITE_ID",
	custom_properties: {
		"--font-family": "Georgia, serif",
	},
}
```

The widget reads this once, when it loads. Changing it after the widget is already showing has no
effect until the page is reloaded.

**Note:** `custom_properties` technically accepts any CSS custom property the widget's stylesheets
define (see [`custom-properties.css`](https://github.com/umputun/remark42/blob/master/frontend/apps/remark42/app/styles/custom-properties.css)), not only `--font-family`. Only `--font-family` is supported,
though — `--font-size`, for example, isn't wired to anything: most components hardcode their own
font-size instead of reading a variable, so setting it has no visible effect. Even a property that
does work today isn't guaranteed to keep working, or keep meaning the same thing, in a future
version. Treat anything beyond `--font-family` as unsupported and best-effort, not a documented API.


### Locales

Right now Remark42 is translated to English (en), Russian (ru), German (de), Finnish (fi), Spanish (es), Chinese (zh), Turkish (tr), Bulgarian (bg), Ukrainian (ua), Polish (pl), Vietnamese (vi), Belarusian (be), French (fr), Japanese (ja), Korean (ko), Brazilian Portuguese (bp), Italian (it), Arabic (ar), Traditional Chinese (zh-tw), Thai (th), Czech (cs), Persian (fa), Macedonian (mk), Romanian (ro), Swedish (sv), Hebrew (he), Dutch (nl), Icelandic (is) and Hungarian (hu). You can pick one with `locale` in the [configuration object](#configuration).

Do you want to translate Remark42 to other locales? Please see [this documentation](https://remark42.com/docs/contributing/translations/) for details.

### What the widget keeps in the reader's browser

The widget uses the browser's `localStorage` to save the sort order, collapsed threads, hidden users, the email address used for a successful email sign-in, and the text of comments that were typed and not sent.

Unsent text is saved per page for a new comment and per comment for a reply or an edit. The draft of a new comment returns after a reload, and the draft of a reply returns when the reply form is opened again; an edit form shows the published text. Drafts are not tied to an account: signing out clears the saved email address and leaves the drafts, so whoever opens the same page in the same browser profile finds them. A reader on a shared computer should empty the comment forms they used before leaving.

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

{{< note "💡" >}}
**Note:** `components` lists every widget the page uses, for example `["embed", "last-comments"]`.
{{< /note >}}

And then add this node in the place where you want to see last comments widget:

```html
<div class="remark42__last-comments" data-max="50"></div>
```

`data-max` sets the maximum number of comments for this widget, overriding `max_last_comments` in `remark_config`. If neither is set, the default is `15`.

### Counter widget

This widget displays the number of comments for the specified page.
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

{{< note "💡" >}}
**Note:** `components` lists every widget the page uses, for example `["embed", "counter"]`.
{{< /note >}}

And then add a node like this in the place where you want to see a number of comments:

```html
<span
	class="remark42__counter"
	data-url="https://domain.com/path/to/article/"
></span>
```

You can use as many nodes like this as you need to. The script will find all of them by the class `remark42__counter`, and it will use the `data-url` attribute to define the page with comments.

Also, the script can use `url` property from `remark_config` object or `window.location.origin + window.location.pathname` if nothing else is defined.
