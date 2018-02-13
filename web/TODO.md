critical features:

- improve iframe
  - add default attrs
  - auto height
- get settings from outside
  - get path from url or by params
  - get siteid by params
- load css by js ?


optimizations:

- rewrite fetcher if we need it (do we really need axios?)
- remove mimic and other dev-tools
- remove babel if we don't need it


major features:

- edit comment
  - `PUT /api/v1/comment/{id}?site=site-id&url=post-url`
- add description of web part to readme
  
  
minor features:
  
- add manual sort
  - `GET /api/v1/find?site=site-id&url=post-url&sort=fld&format=tree`
- add manual format
  - `GET /api/v1/find?site=site-id&url=post-url&sort=fld&format=tree`
- add comments counter
  - `GET /api/v1/count?site=site-id&url=post-url`


other:

- check todos
