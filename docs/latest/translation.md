---
title: Translation
---

## How to add new language translation to Remark42

Translation files are stored in [/frontend/app/locales](https://github.com/umputun/remark42/tree/master/frontend/app/locales)
directory with `.json` extension and content like following:

```json
{
  "anonymousLoginForm.length-limit": "Username must be at least 3 characters long",
  "anonymousLoginForm.log-in": "Log in",
  "anonymousLoginForm.symbol-limit": "Username must start from the letter and contain only latin letters, numbers, underscores, and spaces",
<...>
}
```

### How to add a new translation

We truly appreciate people spending time contributing their translations to remark42. Please go through the steps
below in order to have your translation start being available to all remark42 users and included in the next release.

1.  create a fork of [umputun/remark42](https://github.com/umputun/remark42) repo, and if you already have one please
    pull the latest changes from the upstream master branch. It could be done like that:
    ```shell
    git remote add upstream https://github.com/umputun/remark42.git
    git fetch upstream
    git rebase upstream/master
    git push
    ```
1.  add a new locale with [two-letter code](https://en.wikipedia.org/wiki/List_of_ISO_639-1_codes)
    of the language you want to make the translation into to list in
    [frontend/tasks/supportedLocales.json](https://github.com/umputun/remark42/blob/master/frontend/tasks/supportedLocales.json)
1.  run `npm install` in `frontend` folder
1.  run `npm run translation:extract` in `frontend` folder
1.  run `npm run translation:generate` in `frontend` folder
1.  translate all values in the newly created json file in
    [frontend/app/locales/](https://github.com/umputun/remark42/blob/master/frontend/app/locales/)
1.  commit all changes above in your fork
1.  test your changes in the interface:

    1.  uncomment `locale: "ru"` line in [frontend/index.ejs](https://github.com/umputun/remark42/blob/master/frontend/index.ejs#L133)
        and replace `ru` with your translation language code
    1.  [run remark42 in Docker](https://github.com/umputun/remark42#development) by issuing following commands
        from the root directory of your remark42 fork:

            ```shell
            docker-compose -f compose-dev-frontend.yml build
            docker-compose -f compose-dev-frontend.yml up
            ```

    1.  open [http://127.0.0.1:8080/web](http://127.0.0.1:8080/web), log in, make a comment, make a reply to a comment,
        and make sure your translation looks as you expect it to look
    1.  make a screenshot from [http://127.0.0.1:8080](http://127.0.0.1:8080) with your translation in place

1.  after all previous steps are done, create a [Pull Request](https://github.com/umputun/remark42/pulls) to umputun/remark42
    repo with your changes, attaching a screenshot or two from your local test instance to it
