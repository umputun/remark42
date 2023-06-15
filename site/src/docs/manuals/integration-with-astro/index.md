# Using Remark42 in Astro

[Astro](https://astro.build/) is an all-in-one web framework for building fast, content-focused websites.

- **Content-focused:** Astro was designed for content-rich websites.
- **Server-first:** Websites run faster when they render HTML on the server.
- **Fast by default:** It should be impossible to build a slow website in Astro.
- **Easy to use:** You don't need to be an expert to build something with Astro.
- **Fully-featured, but flexible:** Over 100+ Astro integrations to choose from.

While Astro supports numerous front-end framework [integrations](https://docs.astro.build/en/guides/integrations-guide/#official-integrations) (e.g. React, Vue, SolidJS, etc.)


## w/ Svelte Components
### Svelte Component (Embedded Frame)
#### remark42-embed.svelte
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

### Astro Layout (Page)
#### blogpageLayout.astro (partial Astro layout)
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

### Svelte Component (Counter)
#### remark42-counter.svelte
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

### Astro Component (Card)
#### blogCard.astro (partial Astro component)
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

## w/ React/Preact Components

This example demonstrates how to integrate the Remark42 commenting system using a React/Preact component. The component manages the insertion and removal of the Remark42 script in the DOM, provides the ability to toggle between light and dark themes, and includes areas for embedding comments and displaying a comment count. This makes it easy to incorporate a commenting system into a React/Preact application.

#### Comment.tsx
```ts
import {useEffect, useState} from 'preact/hooks'

declare global {
  // Declare the global types for REMARK42 and remark_config so they can be used in this module.
  interface Window {
    REMARK42: any
    remark_config: any
  }
}

// Function to insert the Remark42 script into the DOM.
const insertScript = (id: string, parentElement: HTMLElement) => {
  const script = window.document.createElement('script')
  script.type = 'text/javascript'
  script.async = true
  script.id = id

  // Get the current URL, and remove trailing slash if it exists.
  let url = window.location.origin + window.location.pathname
  if (url.endsWith('/')) {
    url = url.slice(0, -1)
  }

  // Get the host from the environment variables.
  const host = import.meta.env.PUBLIC_REMARK_URL

  // Set the inner HTML of the script to load Remark42.
  script.innerHTML = `
    var remark_config = {
      host: "${host}",
      site_id: "remark",
      url: "${url}",
      theme: "dark",
      components: ["counter", "embed"],
    };
    !function(e,n){for(var o=0;o<e.length;o++){var r=n.createElement("script"),c=".js",d=n.head||n.body;"noModule"in r?(r.type="module",c=".mjs"):r.async=!0,r.defer=!0,r.src=remark_config.host+"/web/"+e[o]+c,d.appendChild(r)}}(remark_config.components||["embed"],document);`

  // Append the script to the parent element.
  parentElement.appendChild(script)
}

// Function to remove the Remark42 script from the DOM.
const removeScript = (id: string, parentElement: HTMLElement) => {
  const script = window.document.getElementById(id)
  if (script) {
    parentElement.removeChild(script)
  }
}

// Function to manage the insertion and removal of the Remark42 script.
const manageScript = () => {
  if (!window) {
    return () => {}
  }
  const {document} = window
  if (document.getElementById('remark42')) {
    insertScript('comments-script', document.body)
  }

  // Return a cleanup function to remove the script when the component unmounts.
  return () => removeScript('comments-script', document.body)
}

export function Comments() {
  // State for tracking the current theme.
  const [theme, setTheme] = useState('dark')

  // Use effect to manage the Remark42 script.
  useEffect(manageScript, [])

  // Use effect to update the theme when it changes.
  useEffect(() => {
    if (window.REMARK42) {
      window.REMARK42.changeTheme(theme)
    }
  }, [theme])

  // Function to toggle the theme.
  const toggleTheme = () => {
    setTheme((prevTheme) => (prevTheme === 'dark' ? 'light' : 'dark'))
  }

  return (
    <>
      <h2>Comments</h2>
      {/* Button to change the theme. */}
      <button onClick={toggleTheme}>Change Theme</button>
      {/* The span where Remark42 will embed the comment count. */}
      <p>
        There are <span className="remark42__counter"></span> comments.
      </p>
      {/* The div where Remark42 will embed the comments. */}
      <div id="remark42"></div>
    </>
  )
}
```
#### Here how to use it in your Astro component.
```js
---
import { Comments } from "@components/Comment";
---

<Comments client:visible />
```
