major features:

- improve design
  - disable inputs for guests
  - hide reply links for guests
  - check mobile ui
  - add icons for social networks 
  - remove grey avatars 'cause we have default img
  - add time format 'X hours ago'
  - use static buttons 'reply', 'pin', etc instead of dynamic
  - maybe we should change layout: vote buttons | avatar | info; info: (top: name, points, time), (center: text), (bottom: controls)
- edit comment
  - `PUT /api/v1/comment/{id}?site=site-id&url=post-url`
- add description of web part to readme

optimizations:

- rewrite fetcher if we need it (do we really need axios?)
- remove dev-tools if we have some
- remove babel-polyfill if we don't need it
  
  
minor features:
  
- add manual sort
  - `GET /api/v1/find?site=site-id&url=post-url&sort=fld&format=tree`
- add comments counter
  - `GET /api/v1/count?site=site-id&url=post-url`


other:

- check todos
