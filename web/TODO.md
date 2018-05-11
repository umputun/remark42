major features:

- edit comment
  - `PUT /api/v1/comment/{id}?site=site-id&url=post-url`
- show comments of blocked users in the list of them


optimizations:

- rewrite fetcher if we need it (do we really need axios?)
- remove dev-tools if we have some
- remove babel-polyfill if we don't need it
  
  
minor features:
  
- show list of user comments for admin?


other:

- check todos
