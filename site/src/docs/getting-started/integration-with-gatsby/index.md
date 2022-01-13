# Using Remark42 in Gatsby projects
## Sample Comments.js React component:

```js

import * as React from "react"

// This function will insert the usual <script> tag of
// Remark42 into the specified DOM location (parentElement)
const insertScript = (id, parentElement) => {
  const script = window.document.createElement("script")
  script.type = "text/javascript"
  script.async = true
  script.id = id
  /* For Gatsby it's important to manually provide the URL
  and make sure it does not contain a trailing slash ("/").
  Because otherwise the comments for paths with/without 
  the trailing slash are stored separately in the BoltDB database.
  When following a Gatsby Link a page is loaded without the trailing slash,
  but when refreshing the page (F5) it is loaded with the trailing slash.
  So essentially every URL can become duplicated in the DB and you may not see
  your comments from the inverse URL at your present URL.
  Making sure url is provided without the trailing slash
  in the remark42 config solves this. */
  let url = window.location.origin + window.location.pathname
  if(url.endsWith("/")) {
    url = url.slice(0, -1)
  }
  // Now the actual config and script-fetching function:
  script.innerHTML = `
    var remark_config = {
      host: "https://remark42.example.com",
      site_id: "example-name",
      url: "${url}",
      theme: "dark",
      components: ["embed"],
    };
    !function(e,n){for(var o=0;o<e.length;o++){var r=n.createElement("script"),c=".js",d=n.head||n.body;"noModule"in r?(r.type="module",c=".mjs"):r.async=!0,r.defer=!0,r.src=remark_config.host+"/web/"+e[o]+c,d.appendChild(r)}}(remark_config.components||["embed"],document);`
  parentElement.appendChild(script)
}

/* This function removes the previously added script from the DOM.
Might be necessary when page transitions happen, to make sure a new instance 
is created, pointing to the current URL. Although this might actually 
not be necessary, please do your own testing.*/
const removeScript = (id, parentElement) => {
  const script = window.document.getElementById(id)
  if (script) {
    parentElement.removeChild(script)
  }
}

// This function will be provided to useEffect and will insert the script
// when the component is mounted and remove it when it unmounts
const manageScript = () => {
  if (!window) {
    return
  }
  const document = window.document
  if (document.getElementById("remark42")) {
    insertScript("comments-script", document.body)
  }
  return () => removeScript("comments-script", document.body)
}

/* Another function for another useEffect - this is the most crucial part for
Gatsby compatibility. It will ensure that each page loads its own appropriate
comments, and that comments will be properly loaded */
const recreateRemark42Instance = () => {
    if (!window) {
      return
    }
    const remark42 = window.REMARK42
    if (remark42) {
      remark42.destroy()
      remark42.createInstance(window.remark_config)
    }
}

// The location prop is {props.location.pathname} from the parent component.
// I.e. invoke the component like this in the parent: <Comments location={props.location.pathname} />
const Comments = ({ location }) => {
  // Insert the two useEffect hooks. Maybe you can combine them into one? Feel free if you want to.
  React.useEffect(manageScript, [location])
  React.useEffect(recreateRemark42Instance, [location])

  return (
    <>
      <h2>Comments</h2>
      { /* This div is the target for actual comments insertion */ }
      <div id="remark42"></div>
    </>
  )
}

export default Comments

```
