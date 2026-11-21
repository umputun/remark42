---
title: API for Single-Page Application
---

Add the following JavaScript to your `index.html`, which in this case, it is identical to `<script defer type="module" src="$HOST/web/embed.mjs"></script>`

```js
;(function () {
  var host = 'https://demo.remark42.com' // Your remark42 host
  var components = ['embed'] // Your choice of remark42 components

  for (var i = 0; i < components.length; i++) {
    var d = document
    var s = d.createElement('script')
    s.src = host + '/web/' + components[i] + '.mjs'
    s.type = 'module'
    s.defer = true
    ;(d.head || d.body).appendChild(s)
  }
})()
```

Created `remark42Instance` when the `div` containing remark42 has appeared, usually at `mounted` or `componentDidMount` of the SPA lifecycle. Destroy the previous instance first, if necessary.

The widget is mounted into the element passed as `node`. Without it, the element with the id `remark42` is used, so `node` can be left out when your page has one.

```ts
  initRemark42() {
    if (window.REMARK42) {
      if (this.remark42Instance) {
        this.remark42Instance.destroy()
      }

      this.remark42Instance = window.REMARK42.createInstance({
        node: this.$refs.remark42 as HTMLElement,
        ...remark42_config  // See <https://remark42.com/docs/configuration/frontend/>
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

## Things to know about `createInstance`

- `createInstance` returns `{ changeTheme, destroy }` and throws an `Error` instead of returning a failure, in three cases:
  - there is no element to mount into: neither `node` nor an element with the id `remark42`
  - `window.remark_config` is not defined
  - `site_id` is not set in `window.remark_config`
- `node`, when given, must be a DOM element, otherwise `createInstance` throws as well.
- The checks above and the server address (`host`) are read from the **global** `window.remark_config`, while the other options come from the object passed to `createInstance`. Passing a configuration does not remove the need to define a valid global one with at least `host` and `site_id`.
