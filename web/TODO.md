major features:

- improve design
  - check mobile ui
  - check styles for md text
  - add time format 'X hours ago'
  - try to move voting buttons from the top line to the left side
  - improve thread design
- edit comment
  - `PUT /api/v1/comment/{id}?site=site-id&url=post-url`
- add description of web part to readme
- show comments of blocked users in the list of them


optimizations:

- rewrite fetcher if we need it (do we really need axios?)
- remove dev-tools if we have some
- remove babel-polyfill if we don't need it
  
  
minor features:
  
- maybe add manual sort
  - `GET /api/v1/find?site=site-id&url=post-url&sort=fld&format=tree`
- show list of user comments for admin?


other:

- check todos
