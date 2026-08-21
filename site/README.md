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

| Path                  | Contents                                                             |
| --------------------- | -------------------------------------------------------------------- |
| `content/`            | Pages as markdown. `content/docs/` is the documentation tree          |
| `layouts/`            | Templates. `partials/`, `shortcodes/` and `_markup/` render hooks     |
| `assets/`             | `styles.css` and the scripts, fingerprinted at build time             |
| `static/`             | Files copied to the site root as-is: favicons, manifest, `robots.txt` |
| `data/nav.json`       | The documentation sidebar                                             |
| `hugo.toml`           | Site configuration                                                    |

## Writing docs

A page is a markdown file with a `title` in its front matter. A directory becomes a section when it holds `_index.md`, and a page that carries its own images is a directory with `index.md` and the images beside it.

Adding a page to the sidebar means adding an entry to `data/nav.json`; the paths there are relative to `/docs`.

Callouts use the `note` shortcode, which takes the emoji to show:

```markdown
{{< note "💡" >}}
Anything markdown here.
{{< /note >}}
```

Two documentation pages are symlinked into the repository as `backend/README.md` and `frontend/apps/remark42/README.md`, so moving or renaming `content/docs/contributing/backend/index.md` or `content/docs/contributing/frontend/index.md` means repointing those symlinks.

## Styling

`assets/styles.css` is hand-written, with the palette and light/dark values as custom properties at the top. The dark theme is applied by a `dark` class on `<html>`, set before first paint by `assets/inline.js` and toggled by `assets/script.js`.

Code highlighting is Hugo's built-in Chroma. The rules at the end of the stylesheet come from `hugo gen chromastyles --style=github` and `--style=github-dark`.

## Dates

The "Updated" line on a documentation page comes from `.Lastmod`. `hugo.toml` resolves it through `[frontmatter] lastmod`, which tries git, then a `lastmod` in the page's front matter, then the file's modification time; without that chain it would resolve to `.Date` and every page would read `Jan 01, 0001`. A deployed page therefore shows the build date, since a checkout does not preserve modification times.

`HUGO_ENABLEGITINFO=true hugo` gives real per-page commit dates. It is off by default because the image build context is `site/` alone, where there is no `.git` for Hugo to read and it fails rather than falling back.
