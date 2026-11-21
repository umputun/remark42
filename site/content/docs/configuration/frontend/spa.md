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
