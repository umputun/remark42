---
title: API for Single-Page Application
---

Load the widget script once, in your `index.html`, after a global `remark_config` with `site_id`. The script reads the global while it loads, taking `host` from it (without `host` it uses the server's `REMARK_URL`), and `createInstance` needs `remark_config.site_id`, so the global has to be there even when each instance gets its own configuration:

```html
<script>
  var remark_config = {
    host: 'https://demo.remark42.com', // your Remark42 host
    site_id: 'remark', // your site ID
  }
</script>
<script defer type="module" src="https://demo.remark42.com/web/embed.mjs"></script>
```

The examples below are written as Vue class components (`vue-property-decorator`); the same calls work in any framework.

Create the instance when the element with `id="remark42"` has appeared, usually in `mounted` or `componentDidMount` of the SPA lifecycle: `createInstance` always puts the widget into that element and throws if the page has none. Always destroy the previous instance first: a second `createInstance` reuses the widget it finds in the page, so the previous page's comments stay until `destroy()` is called. Set `url` to the current route on every call: `createInstance` writes the resolved `url` into the object it is given, the global `remark_config` included when the script creates the first instance, so a copied or reused configuration keeps the first page's address.

`createInstance` returns an object with `destroy()`, which removes the widget, and `changeTheme(theme)`, which works as `window.REMARK42.changeTheme` does. It throws when the page has no `remark42` element, or when the global `remark_config` or its `site_id` is missing.

The widget is mounted into the element passed as `node`. Without it, the element with the id `remark42` is used, so `node` can be left out when your page has one.

```ts
  initRemark42() {
    if (window.REMARK42) {
      if (this.remark42Instance) {
        this.remark42Instance.destroy()
      }

      this.remark42Instance = window.REMARK42.createInstance({
        ...window.remark_config, // see https://remark42.com/docs/configuration/frontend/
        url: window.location.origin + window.location.pathname,
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

The script dispatches `REMARK42::ready` when it initialises, even on a page without the `remark42` element, so it can be loaded on a route without comments and `REMARK42.createInstance` called when a route with comments opens. On such a page its first attempt still throws `Remark42: Can't find root node.`, which shows in the console and is expected.

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

## Things to know about `createInstance`

- `createInstance` returns `{ changeTheme, destroy }` and throws an `Error` instead of returning a failure, in three cases:
  - there is no element to mount into: neither `node` nor an element with the id `remark42`
  - `window.remark_config` is not defined
  - `site_id` is not set in `window.remark_config`
- `node`, when given, must be a DOM element, otherwise `createInstance` throws as well.
- The checks above and the server address (`host`) are read from the **global** `window.remark_config`, while the other options come from the object passed to `createInstance`. Passing a configuration does not remove the need to define a valid global one with at least `host` and `site_id`.
