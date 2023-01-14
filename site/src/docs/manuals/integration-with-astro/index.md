# Using Remark42 in Astro (w/Svelte Components)

[Astro](https://astro.build/) is an all-in-one web framework for building fast, content-focused websites.

- **Content-focused:** Astro was designed for content-rich websites.
- **Server-first:** Websites run faster when they render HTML on the server.
- **Fast by default:** It should be impossible to build a slow website in Astro.
- **Easy to use:** You don't need to be an expert to build something with Astro.
- **Fully-featured, but flexible:** Over 100+ Astro integrations to choose from.

While Astro supports numerous front-end framework [integrations](https://docs.astro.build/en/guides/integrations-guide/#official-integrations) (e.g. React, Vue, SolidJS, etc.) this integration implements [Svelte](https://docs.astro.build/en/guides/integrations-guide/svelte/) components and Astro's [islands architecture](https://docs.astro.build/en/concepts/islands/) for partial hydration. 



## Svelte Component (Embedded Frame)
### remark42-embed.svelte
```js
<svelte:head>
  <script async lang="javascript">
    var remark_config = {
      host: "https://your.website.here",
      site_id: "yourwebsite",
      components: ["embed"],
      show_rss_subsription: false,
      theme: localStorage.getItem("color-theme") ?? "light",
    };
    !(function (e, n) {
      for (var o = 0; o < e.length; o++) {
        var r = n.createElement("script"),
          c = ".js",
          d = n.head || n.body;
        "noModule" in r ? ((r.type = "module"), (c = ".mjs")) : (r.async = !0),
          (r.defer = !0),
          (r.src = remark_config.host + "/web/" + e[o] + c),
          d.appendChild(r);
      }
    })(remark_config.components || ["embed"], document);
  </script>
</svelte:head>

<div id="remark42" />
```

## Astro Layout (Page)
### blogpageLayout.astro (partial Astro layout)
```js
---
...
import Remark42Embed from "../components/remark42-embed.svelte";

export interface Props {
  frontmatter: Frontmatter;
}

const { frontmatter } = Astro.props;
---
<html lang="en">
    ...
          <article class="markdown">
            <slot />
          </article>
          {frontmatter.comments && <Remark42Embed client:visible />}
    ...
</html>
```
Note the use of the `client:visible` directive which will hydrate the component once the element becomes visible.

## Svelte Component (Counter)
### remark42-counter.svelte
```js
<script>
  export let url;
</script>

<svelte:head>
  <script async lang="javascript">
    var remark_config = {
      host: "https://your.website.here",
      site_id: "yourwebsite",
      components: ["counter"],
      show_rss_subsription: false,
      theme: localStorage.getItem("color-theme") ?? "light",
    };
    !(function (e, n) {
      for (var o = 0; o < e.length; o++) {
        var r = n.createElement("script"),
          c = ".js",
          d = n.head || n.body;
        "noModule" in r ? ((r.type = "module"), (c = ".mjs")) : (r.async = !0),
          (r.defer = !0),
          (r.src = remark_config.host + "/web/" + e[o] + c),
          d.appendChild(r);
      }
    })(remark_config.components || ["embed"], document);
  </script>
</svelte:head>

<span class="remark42__counter" data-url={url.href} />
```

## Astro Component (Card)
### blogCard.astro (partial Astro component)
```js
---
const { post } = Astro.props;

import Remark42Count from "../components/remark42-count.svelte";
...
---
<div>
    ...
    <!-- Read Time & Comment Count -->
    {
      post.frontmatter.comments && (
        <div class="my-2 text-center font-Roboto font-light text-gray-600 dark:text-gray-300">
          {post.frontmatter.minutes} minutes &nbsp;&bull;&nbsp;
          <Remark42Count url={new URL(post.url, Astro.site)} client:idle /> comments
        </div>
      )
    }
    ...
</div>
```
Note the use of the `client:idle` directive which will hydrate the component once the page has completed the initial load.