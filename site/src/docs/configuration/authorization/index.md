---
title: Authorization
---

## OAuth Providers

Authentication handled by external providers. You should setup OAuth2 for all (or some) of them to allow users to make comments. It is not mandatory to have all of them, but at least one should be correctly configured.

### Facebook

1. Open the list of apps on [Facebook Developers Platform](https://developers.facebook.com/apps)
2. Create a new app with [this manual](https://developers.facebook.com/docs/development/create-an-app/) or use existing app
3. Open your app and choose **"Facebook Login"** and then **"Web"**
4. Set **"Site URL"** to your domain, e.g.: `https://remark42.mysite.com`
5. Under **"Facebook login" / "Settings"** fill "Valid OAuth redirect URIs" with your callback URL constructed as domain plus `/auth/facebook/callback`, e.g `https://remark42.mysite.com/auth/facebook/callback`
6. Select **"App Review"** and turn public flag on. This step may ask you to provide a link to your privacy policy

### GitHub

1. Create a new **"OAuth App"**: https://github.com/settings/developers
2. Fill **"Application Name"** and **"Homepage URL"** for your site
3. Under **"Authorization callback URL"** enter the correct URL constructed as domain + `/auth/github/callback`, i.e. `https://remark42.mysite.com/auth/github/callback`
4. Take note of the **Client ID** and **Client Secret**

### Google

1. Create a new project: https://console.cloud.google.com/projectcreate
2. Choose the new project from the top right project dropdown (only if another project is selected)
3. In the project Dashboard center pane, choose **"API Manager"**
4. In the left Nav pane, choose **"Credentials"**
5. In the center pane, choose **"OAuth consent screen"** tab. Fill in **"Product name shown to users"** and hit save
6. In the center pane, choose **"Credentials"** tab

- Open the **"New credentials"** drop down
- Choose **"OAuth client ID"**
- Choose **"Web application"**
- Application name is freeform, choose something appropriate
- Authorized origins is your domain, e.g.: `https://remark42.mysite.com`
- Authorized redirect URIs is the location of OAuth2/callback constructed as domain + `/auth/google/callback`, e.g.: `https://remark42.mysite.com/auth/google/callback`
- Choose **"Create"**

7. Take note of the **Client ID** and **Client Secret**

_instructions for Google OAuth2 setup borrowed from [oauth2_proxy](https://github.com/bitly/oauth2_proxy)_

### Microsoft

1. Register a new application [using the Azure portal](https://docs.microsoft.com/en-us/graph/auth-register-app-v2)
2. Under **"Authentication/Platform configurations/Web"** enter the correct URL constructed as domain + `/auth/microsoft/callback`, i.e. `https://example.mysite.com/auth/microsoft/callback`
3. In **"Overview"** take note of the **Application (client) ID**
4. Choose the new project from the top right project dropdown (only if another project is selected)
5. Select **"Certificates & secrets"** and click on **"+ New Client Secret"**

### Twitter

1. Create a new Twitter application https://developer.twitter.com/en/apps
2. Fill **App name**, **Description** and **URL** of your site
3. In the field **Callback URLs** enter the correct URL of your callback handler, e.g. domain + `/auth/twitter/callback`
4. Under **Key and tokens** take note of the **Consumer API Key** and **Consumer API Secret key**. Those will be used as `AUTH_TWITTER_CID` and `AUTH_TWITTER_CSEC`

### Yandex

1. Create a new **"OAuth App"**: https://oauth.yandex.com/client/new
2. Fill **"App name"** for your site
3. Under **Platforms** select **"Web services"** and enter **"Callback URI #1"** constructed as domain + `/auth/yandex/callback`, i.e. `https://remark42.mysite.com/auth/yandex/callback`
4. Select **Permissions**. You need the following permissions only from the **"Yandex.Passport API"** section:

- Access to the user avatar
- Access to username, first name and surname, gender

5. Fill out the rest of the fields if needed
6. Take note of the **ID** and **Password**

For more details refer to [Yandex OAuth](https://yandex.com/dev/oauth/doc/dg/concepts/about.html) and [Yandex.Passport](https://yandex.com/dev/passport/doc/dg/index.html) API documentation.

### Patreon Auth provider
1. Create a new Patreon client https://www.patreon.com/portal/registration/register-clients
2. Fill **App Name**, **Description**
3. In the field **Redirect URIs** enter the correct URI constructed as domain + `/auth/patreon/callback`, i.e. `https://example.mysite.com/auth/patreon/callback`
4. Expand client details, take a note of the **Client ID** and **Client Secret**. Those will be used as `AUTH_PATREON_CID` and `AUTH_PATREON_CSEC`

## Anonymous Auth Provider

Optionally, anonymous access can be turned on. In this case, an extra `anonymous` provider will allow logins without any social login with any name satisfying 2 conditions:

- name should be at least 3 characters long
- name has to start from the letter and contains letters, numbers, underscores and spaces only

## Telegram Auth Provider

1. Contact [@BotFather](https://t.me/botfather) and follow his instructions to create your own bot (call it, for example, "My site auth bot")
1. Write down resulting token as `TELEGRAM_TOKEN` into remark42 config, and also set `AUTH_TELEGRAM` to `true` to enable telegram auth for your users.
