---
title: Interface Translations
---

## Add a New Language to Remark42

Translation files are stored in [/frontend/app/locales](https://github.com/umputun/remark42/tree/master/frontend/app/locales)
directory with `.json` extension and content like following:

```json
{
  "anonymousLoginForm.length-limit": "Username must be at least 3 characters long",
  "anonymousLoginForm.log-in": "Log in",
  "anonymousLoginForm.symbol-limit": "Username must start from the letter and contain only Latin letters, numbers, underscores, and spaces",
  <...>
}
```

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

1.  Add a new locale with a [two-letter code](https://en.wikipedia.org/wiki/List_of_ISO_639-1_codes) of the language you want to do the translation into to list in [frontend/tasks/supportedLocales.json](https://github.com/umputun/remark42/blob/master/frontend/tasks/supportedLocales.json)
1.  Run `npm install` in the `frontend` folder
1.  Run `npm run translation:extract` in the `frontend` folder
1.  Run `npm run translation:generate` in the `frontend` folder
1.  Translate all values in the newly created JSON file in
    [frontend/app/locales/](https://github.com/umputun/remark42/tree/master/frontend/app/locales)
1.  Commit all changes above in your fork
1.  Test your changes in the interface:

    1.  Uncomment `locale: "ru"` line in [frontend/templates/demo.ejs](https://github.com/umputun/remark42/blob/master/frontend/templates/demo.ejs) and replace `ru` with your translation language code
    2.  [Run remark42 in Docker](https://github.com/umputun/remark42#development) by issuing the following commands from the root directory of your remark42 fork:
        `shell docker-compose -f compose-dev-frontend.yml build docker-compose -f compose-dev-frontend.yml up `

    3.  open [http://127.0.0.1:8080/web](http://127.0.0.1:8080/web), log in, make a comment, make a reply to a comment, and make sure your translation looks as you expect it to look
    4.  make a screenshot from [http://127.0.0.1:8080](http://127.0.0.1:8080) with your translation in place

1.  after all previous steps are done, create a [Pull Request](https://github.com/umputun/remark42/pulls) to umputun/remark42 repo with your changes, attaching a screenshot or two from your local test instance to it
