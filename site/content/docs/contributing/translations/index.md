---
title: Interface Translations
---

## Add a New Language to Remark42

Translation files are stored in [/frontend/apps/remark42/app/locales](https://github.com/umputun/remark42/tree/master/frontend/apps/remark42/app/locales)
directory with `.json` extension and content like following:

```json
{
  "anonymousLoginForm.length-limit": "Username must be at least 3 characters long",
  "anonymousLoginForm.log-in": "Log in",
  "anonymousLoginForm.symbol-limit": "Username must start from the letter and contain only Latin letters, numbers, underscores, and spaces",
  <...>
}
```

{{< note "🚨" >}}
Translations support `{name}` placeholders and paired tags such as `<a>text</a>`. ICU plural,
select and typed-argument syntax is not supported: a message using one either falls back to
English or shows the raw syntax on the page, depending on the message. Apostrophe quoting is not
supported either, so `''` stays as two apostrophes.

You may drop a tag from the English string if your language reads better without the link. If
you keep it, it has to stay paired and keep the same name, and every placeholder has to be one
the English string already uses. A tag that is broken, stray, nested or unknown, or a placeholder
the widget does not supply, makes that whole message fall back to English.

CI checks every value's tags and placeholders against the English string it translates, and
renders the two messages that carry a link. It cannot tell that an ICU form is unsupported,
since that is ordinary text to it, so open your translation in the interface before sending
it.
{{< /note >}}

### Add a new translation

We truly appreciate people spending time contributing their translations to remark42. Please go through the steps
below to have your translation available to all remark42 users and included in the next release.

1.  Create a fork of [umputun/remark42](https://github.com/umputun/remark42) repo, and if you already have one, please pull the latest changes from the upstream master branch. It could be done like that:

    ```shell
    git remote add upstream https://github.com/umputun/remark42.git
    git fetch upstream
    git rebase upstream/master
    git push
    ```

1.  Add a new locale with a [two-letter code](https://en.wikipedia.org/wiki/List_of_ISO_639-1_codes) of the language you want to do the translation into to list in [frontend/apps/remark42/tasks/supportedLocales.json](https://github.com/umputun/remark42/blob/master/frontend/apps/remark42/tasks/supportedLocales.json)
1.  Run `pnpm i` in the `frontend/apps/remark42` folder
1.  Run `pnpm translation:extract` in the `frontend/apps/remark42` folder
1.  Run `pnpm translation:generate` in the `frontend/apps/remark42` folder
1.  Translate all values in the newly created JSON file in
    [frontend/apps/remark42/app/locales/](https://github.com/umputun/remark42/tree/master/frontend/apps/remark42/app/locales)
1.  Commit all changes above in your fork
1.  Test your changes in the interface:

    1.  Uncomment `locale: "ru"` line in [frontend/apps/remark42/templates/demo.ejs](https://github.com/umputun/remark42/blob/master/frontend/apps/remark42/templates/demo.ejs) and replace `ru` with your translation language code
    2.  [Run remark42 in Docker](https://github.com/umputun/remark42#development) by issuing the following commands from the root directory of your remark42 fork:
        `shell docker compose -f compose-dev-frontend.yml build docker compose -f compose-dev-frontend.yml up `

    3.  open <http://127.0.0.1:8080/web/>, log in, make a comment, make a reply to a comment, and make sure your translation looks as you expect it to look
    4.  make a screenshot from <http://127.0.0.1:8080> with your translation in place

1.  after all previous steps are done, create a [Pull Request](https://github.com/umputun/remark42/pulls) to umputun/remark42 repo with your changes, attaching a screenshot or two from your local test instance to it
