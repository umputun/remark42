---
title: Authorization
---

## OAuth Providers

Authentication is handled by external providers. You should set up OAuth2 for at least one to allow users to comment. It is not mandatory to have all of them, but one should be correctly configured.

### Apple

1. Log in [to the developer account](https://developer.apple.com/account).
1. If you don't have an App ID yet, [create one](https://developer.apple.com/account/resources/identifiers/add/bundleId). Later on, you'll need **TeamID**, which is an "App ID Prefix" value.
1. Enable the "Sign in with Apple" capability for your App ID in [the Certificates, Identifiers & Profiles](https://developer.apple.com/account/resources/identifiers/list) section.
1. Create [Service ID](https://developer.apple.com/account/resources/identifiers/list/serviceId) and bind with App ID from the previous step. Apple will display the description field value to end-users on sign-in. You'll need that service **Identifier as a ClientID** later on.
1. Configure "Sign in with Apple" for created Service ID. Add the domain where you will use that auth to "Domains and subdomains" and its main page URL (like `https://example.com/` to "Return URLs".
1. Register a [New Key](https://developer.apple.com/account/resources/authkeys/list) (**private key**) for the "Sign in with Apple" feature and download it, you'll need to put it to `/srv/var/apple.p8` path inside the container. Also, write down the private **Key ID**.
1. Add your Remark42 domain name and sender email in the Certificates, Identifiers & Profiles >> [More](https://developer.apple.com/account/resources/services/configure) section as a new Email Source.

After completing the previous steps, you can configure the Apple auth provider. You'll need to set the following environment variables:

- `AUTH_APPLE_CID` (**required**) - Client ID
- `AUTH_APPLE_TID` (**required**) - Team ID
- `AUTH_APPLE_KID` (**required**) - Private Key ID
- `AUTH_APPLE_PRIVATE_KEY_FILEPATH` (default `/srv/var/apple.p8`) - Private key file location

### Facebook

1. Open the list of apps on the [Facebook Developers Platform](https://developers.facebook.com/apps)
2. Create a new app with [this manual](https://developers.facebook.com/docs/development/create-an-app/) or use an existing app
3. Open your app and choose **"Facebook Login"** and then **"Web"**
4. Set **"Site URL"** to your domain, e.g., `https://remark42.mysite.com`
5. Under **"Facebook login"**/**"Settings"** fill in "Valid OAuth redirect URIs" with your callback URL constructed as domain plus `/auth/facebook/callback`, e.g. `https://remark42.mysite.com/auth/facebook/callback`
6. Select **"App Review"** and turn the public flag on. This step may ask you to provide a link to your privacy policy
7. Write down the client ID and secret as `AUTH_FACEBOOK_CID` and `AUTH_FACEBOOK_CSEC`

### GitHub

1. Create a new **"OAuth App"**: https://github.com/settings/developers
2. Fill **"Application Name"** and **"Homepage URL"** for your site
3. Under **"Authorization callback URL"** enter the correct URL constructed as domain + `/auth/github/callback`, i.e., `https://remark42.mysite.com/auth/github/callback`
4. Take note of the **Client ID** (as `AUTH_GITHUB_CID`) and **Client Secret** (`AUTH_GITHUB_CSEC`)

### Google

1. Create a new project: https://console.cloud.google.com/projectcreate
2. Choose the new project from the top right project dropdown (only if another project is selected)
3. In the project Dashboard center pane, choose **"APIs & Services"**
4. In the left Nav pane, choose **"Credentials"**
5. In the center pane, choose the **"OAuth consent screen"** tab.

    - Select "**External**" and click "Create"
    - Fill in **"App name"** and select **User support email**
    - Upload a logo, if you want to
    - In the **App Domain** section:
        - **Application home page** - your site URL, e.g., `https://mysite.com`
        - **Application privacy policy link** - `/web/privacy.html` of your Remark42 installation, e.g. `https://remark42.mysite.com/web/privacy.html` (please check that it works)
        - **Terms of service** - leave empty
    - **Authorized domains** - your site domain, e.g., `mysite.com`
    - **Developer contact information** - add your email, and then click **Save and continue**
    - On the **Scopes** tab, just click **Save and continue**
    - On the **Test users**, add your email, then click **Save and continue**
    - Before going to the next step, set the app to "Production" and send it to verification

6. In the center pane, choose the **"Credentials"** tab

    - Open the **"Create credentials"** drop-down
    - Choose **"OAuth client ID"**
    - Choose **"Web application"**
    - Application **Name** is freeform; choose something appropriate, like "Comments on mysite.com"
    - **Authorized JavaScript Origins** should be your domain, e.g., `https://remark42.mysite.com`
    - **Authorized redirect URIs** is the location of OAuth2/callback constructed as domain + `/auth/google/callback`, e.g., `https://remark42.mysite.com/auth/google/callback`
    - Click **"Create"**

7. Take note of the **Client ID** (`AUTH_GOOGLE_CID`) and **Client Secret** (`AUTH_GOOGLE_CSEC`)

_instructions for Google OAuth2 setup borrowed from [oauth2_proxy](https://github.com/bitly/oauth2_proxy)_

### Microsoft

1. Register a new application [using the Azure portal](https://docs.microsoft.com/en-us/graph/auth-register-app-v2)
2. Under **"Authentication/Platform configurations/Web"** enter the correct URL constructed as domain + `/auth/microsoft/callback`, i.e., `https://example.mysite.com/auth/microsoft/callback`
3. In **"Overview"** take note of the **Application (client) ID** (`AUTH_MICROSOFT_CID`)
4. Choose the new project from the top right project dropdown (only if another project is selected)
5. Select **"Certificates & secrets"** and click on **"+ New Client Secret"** (`AUTH_MICROSOFT_CSEC`)

### Twitter

> **Important**: Twitter developer accounts created after _November 15th 2021_ need "Elevated access" to use the Standard v1.1 API routes required to work properly. Apply for this access from within the Twitter developer portal.

1. Create a new Twitter application https://developer.twitter.com/en/apps
2. Fill **App name**, **Description** and **URL** of your site
3. In the field **Callback URLs** enter the correct URL of your callback handler, e.g. domain + `/auth/twitter/callback`
4. Under **Key and tokens** take note of the **Consumer API Key** and **Consumer API Secret key**. Those will be used as `AUTH_TWITTER_CID` and `AUTH_TWITTER_CSEC`

### Yandex

1. Create a new **"OAuth App"**: https://oauth.yandex.com/client/new
2. Fill **"App name"** for your site
3. Under **Platforms** select **"Web services"** and enter **"Callback URI #1"** constructed as domain + `/auth/yandex/callback`, i.e., `https://remark42.mysite.com/auth/yandex/callback`
4. Select **Permissions**. You need the following permissions only from the **"Yandex.Passport API"** section:

- Access to the user avatar
- Access to username, first name and surname, gender

5. Fill out the rest of the fields if needed
6. Take note of the **ID** (`AUTH_YANDEX_CID`) and **Password** (`AUTH_YANDEX_CSEC`)

For more details refer to [Yandex OAuth](https://yandex.com/dev/oauth/doc/dg/concepts/about.html) and [Yandex.Passport](https://yandex.com/dev/passport/doc/dg/index.html) API documentation.

### Patreon

1. Create a new Patreon client https://www.patreon.com/portal/registration/register-clients
2. Fill **App Name**, **Description**
3. In the field **Redirect URIs** enter the correct URI constructed as domain + `/auth/patreon/callback`, i.e., `https://example.mysite.com/auth/patreon/callback`
4. Expand client details and note the **Client ID** and **Client Secret**. Those will be used as `AUTH_PATREON_CID` and `AUTH_PATREON_CSEC`

### Telegram

1. Contact [@BotFather](https://t.me/botfather) and follow his instructions to create your bot (call it, for example, "My site auth bot")
1. Write down the resulting token as `TELEGRAM_TOKEN` into remark42 config, and also set `AUTH_TELEGRAM` to `true` to enable telegram auth for your users.

### Anonymous

Optionally, anonymous access can be turned on. In this case, an extra `anonymous` provider will allow logins without any social login with any name satisfying two conditions:

- the name should be at least three characters long
- the name has to start from the letter and contains letters, numbers, underscores and spaces only**
