# Remark42 site

Sources for [remark42.com](https://remark42.com), built with [Hugo](https://gohugo.io).

## Requirements

Hugo, the plain build; nothing here needs the extended one. The version the published image builds with is `ARG HUGO_VERSION` in `Dockerfile`, and any release at or above it works locally. Nothing updates that pin automatically: Dependabot's docker ecosystem reads `FROM` references, and the Hugo version is a bare string in a download URL, so bumping it is a manual edit. `brew install hugo`, or see the [installation guide](https://gohugo.io/installation/). Nothing else: no Node, no package manager.

## Development

```shell
hugo server
```

Serves the site on <http://localhost:1313> and rebuilds on change.

Alternatively, without installing Hugo:

```shell
docker compose up
```

## Build

```shell
hugo --minify --cleanDestinationDir --destination build
```

Writes the static site to `build/`, which is what the Docker image serves. `--cleanDestinationDir` matters on a rebuild: without it a page you deleted, and the fingerprinted stylesheets of earlier builds, stay behind.

## Layout

| Path                       | Contents                                                              |
| -------------------------- | --------------------------------------------------------------------- |
| `content/`                 | Pages as markdown. `content/docs/` is the documentation tree          |
| `layouts/`                 | Templates. `partials/`, `shortcodes/` and `_markup/` render hooks     |
| `assets/`                  | `styles.css` and the scripts, fingerprinted at build time             |
| `static/`                  | Files copied to the site root as-is: favicons, manifest, `robots.txt` |
| `data/nav.json`            | The documentation sidebar                                             |
| `data/anchor_aliases.json` | Previous anchors of headings, kept as link targets                    |
| `data/meta.json`           | Title and description of each page for search and link previews       |
| `hugo.toml`                | Site configuration                                                    |

## Writing docs

A page is a markdown file with a `title` in its front matter. A directory becomes a section when it holds `_index.md`, and a page that carries its own images is a directory with `index.md` and the images beside it.

Adding a page to the sidebar means adding an entry to `data/nav.json`; the paths there are relative to `/docs`.

The title and description a page shows in search results and link previews are set in `data/meta.json`, keyed by the page's URL path, such as `/docs/manuals/nginx/`. Without an entry a page gets "<its title> | Remark42" as the title, and as the description its front-matter `metaDescription` or `description`, or else the start of its text.

`data/anchor_aliases.json` keeps a heading's previous anchors working after its anchor changes. For each page it maps a heading's current anchor to its previous anchor, or to a list of them, and the heading renders each one as an empty target. A renamed heading listed there needs its entry re-keyed under the new anchor, with the old anchor added to its list; otherwise every previous anchor of that heading stops resolving.

Callouts use the `note` shortcode, which takes the emoji to show:

```markdown
{{< note "💡" >}}
Anything markdown here.
{{< /note >}}
```

Two documentation pages are symlinked into the repository as `backend/README.md` and `frontend/apps/remark42/README.md`, so moving or renaming `content/docs/contributing/backend/index.md` or `content/docs/contributing/frontend/index.md` means repointing those symlinks. GitHub renders those two as plain markdown, where a shortcode shows as raw text, so their callouts are blockquotes instead of the `note` shortcode.

## Styling

`assets/styles.css` is hand-written, with the palette and light/dark values as custom properties at the top. The dark theme is applied by a `dark` class on `<html>`, set before first paint by `assets/inline.js` and toggled by `assets/script.js`.

Code highlighting is Hugo's built-in Chroma. The rules at the end of the stylesheet come from `hugo gen chromastyles --style=github` and `--style=github-dark`.

## Dates

The "Updated" line on a documentation page and the `lastmod` of every sitemap entry show the date of the last commit that changed the page. The image build has no git history, so `scripts/page-dates.sh` writes `data/lastmod.json` before it, with the commit date of every tracked file under `content/`. The file is not committed; `layouts/partials/page-date.html` reads it.

The workflow `ci-site.yml` runs the script on a full clone before it builds the image, and builds with `REQUIRE_PAGE_DATES=true`, which fails the build on a page without a date. The script refuses a shallow clone, where the cut history can give a page a wrong date.

A local build shows the published dates once `scripts/page-dates.sh` has run. The file stays in the checkout and takes precedence, so run the script again after pulling changes to the content. Without the file, `HUGO_ENABLEGITINFO=true hugo` reads git itself; its date can differ from the published one for a page whose last change arrived through a merge commit. With neither, the build succeeds and the pages carry no date.
