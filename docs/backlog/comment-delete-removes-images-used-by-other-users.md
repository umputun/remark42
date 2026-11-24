---
worth: maybe
where: backend/app/store/service/service.go:deleteRemovedPictures
added: 2026-10-29
---
# comment delete removes images another user or another site still shows

When a comment is deleted or a picture is edited out of it, `deleteRemovedPictures` keeps an upload
that another comment on the same page shows, or any other comment of its author on the same site.
It does not look further, and the image store is shared across pages and sites. An upload can
therefore still be removed while another user's comment on a different page, or a comment on another
site of the same instance, shows it: those references are not checked, and that comment then shows a
broken image.

Surfaced by the revmux review of the image id traversal fix (round 1, pre-existing). Options: check
references across all pages and sites sharing the image store before deleting, or stop deleting files on comment delete and leave
orphans to a cleanup pass. Which one is a design call, hence `maybe`.
