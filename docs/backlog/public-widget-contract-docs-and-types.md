---
worth: yes
where: site/content/docs/configuration/frontend/_index.md, frontend/apps/remark42/app/typings/global.d.ts
added: 2026-08-21
---
# The public embed contract has no reference page and no shippable type declaration

Discussion #1714 asked for a description of `window.REMARK42` and TypeScript types, in December 2023.
The contract was written out in a reply on 2026-08-20, but a discussion comment is not documentation,
and the two halves of the request are still open.

## The docs half

The pieces exist and are scattered. `configuration/frontend/_index.md:81` documents `changeTheme` only,
in a section about theming. `configuration/frontend/spa.md` shows `createInstance`, `destroy` and the
`REMARK42::ready` event, but as a single-page-app recipe rather than as a reference. The Astro and
Gatsby manuals repeat fragments again. There is no page that states the surface once.

Three behaviours are documented nowhere and will bite anyone building against it, all in `app/embed.ts`:

- `createInstance` throws rather than returning an error in three cases: no element with id `remark42`,
  `window.remark_config` undefined, or `site_id` unset
- those checks read the **global** `window.remark_config`, while the rest of the function reads the
  object passed as the argument, so passing a config does not remove the need for a valid global
- `createInstance` reuses a direct-child iframe carrying `data-remark42-iframe` rather than creating one,
  and the config passed to that second call is ignored, so calling it again without a `destroy` silently
  does nothing. The reuse also leaves the first instance's `message`, `hashchange` and `click` listeners
  installed, so each call adds another set

Also undocumented: the embed script creates an instance *before* dispatching `REMARK42::ready`, so the
event means the global is safe to touch rather than that nothing is mounted; and `__colors__` reaches
the iframe through `window.name` (`templates/iframe.ejs:42-44`) rather than the URL, so it is read once
at boot and is not a runtime API.

## The types half, and the open decision

`app/typings/global.d.ts` is internal: it imports `jest-fetch-mock` and `common/types`, so it cannot be
shipped as-is. A public declaration would be a standalone file with `Theme`, `RemarkConfig` and the
`window.REMARK42` surface and no internal imports. That part is easy.

**How it reaches a consumer is not decided.** `tsc` picks up types from `node_modules`, and npm is the
direction we have just moved away from: #1715 declined packaging the widgets and #2172 removed
`@remark42/api`. Options, in the order I would rank them:

1. Publish the file on the site as a copy-paste block, with no package. Zero infrastructure, works
   today, and consumers paste it into their own `global.d.ts`. This is what the Astro and Gatsby
   manuals need, since both currently declare `REMARK42: any` and `remark_config: any`
   (`integration-with-astro/index.md:150`, `integration-with-gatsby/index.md:109`), which is our own
   documentation admitting the gap.
2. A types-only npm package. Note this does not contradict #1715: that was declined because shipping
   *widget code* through npm puts the widget in the host origin and breaks OAuth. A package containing
   no runtime code has none of that problem. It does add a publish step and a version to keep in sync.

Option 1 clears the reported need. Option 2 only pays off if we want `npm i -D` ergonomics, which is a
separate call.

Whatever is chosen, fixing the two integration manuals to stop using `any` is the visible win, and
#1714 should get the pull request linked when it lands, as promised there.
