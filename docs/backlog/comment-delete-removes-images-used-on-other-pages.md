---
worth: maybe
where: backend/app/store/service/service.go:Delete
added: 2026-10-29
---
# comment delete removes images still used on other pages

`DataStore.Delete` removes a comment's uploaded images unless another comment on the same page
references them, because the reuse check runs `Engine.Find` with the deleted comment's own locator.
The image store is shared across pages and sites, so an image URL copied into a comment on page B is
deleted when an admin deletes that comment, and the original comment on page A shows a broken image.

Surfaced by the revmux review of the image id traversal fix (round 1, pre-existing). Options: check
references across all pages and sites sharing the image store before deleting, or stop deleting files on comment delete and leave
orphans to a cleanup pass. Which one is a design call, hence `maybe`.
