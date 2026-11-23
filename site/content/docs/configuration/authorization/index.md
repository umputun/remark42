---
title: Authorization
---

## Sign-in Providers

Users can sign in through external providers, [email](https://remark42.com/docs/configuration/email/) or [anonymous login](#anonymous). Configure at least one method to allow commenting; OAuth2 is optional when email or anonymous login is enabled.

### Apple

1. Log in [to the developer account](https://developer.apple.com/account).
1. If you don't have an App ID yet, [create one](https://developer.apple.com/account/resources/identifiers/add/bundleId). Later on, you'll need **TeamID**, which is an "App ID Prefix" value.
1. Enable the "Sign in with Apple" capability for your App ID in [the Certificates, Identifiers & Profiles](https://developer.apple.com/account/resources/identifiers/list) section.
1. Create [Service ID](https://developer.apple.com/account/resources/identifiers/list/serviceId) and bind with App ID from the previous step. Apple will display the description field value to end-users on sign-in. You'll need that service **Identifier as a ClientID** later on.
1. Configure "Sign in with Apple" for created Service ID. Add the domain where you will use that auth to "Domains and subdomains", and its callback URL, such as `https://remark42.mysite.com/auth/apple/callback`, to "Return URLs".
1. Register a [New Key](https://developer.apple.com/account/resources/authkeys/list) (**private key**) for the "Sign in with Apple" feature and download it, you'll need to put it to `/srv/var/apple.p8` path inside the container. Also, write down the private **Key ID**.
1. Add your Remark42 domain name and sender email in the Certificates, Identifiers & Profiles >> [More](https://developer.apple.com/account/resources/services/configure) section as a new Email Source.

After completing the previous steps, you can configure the Apple auth provider. You'll need to set the following environment variables:

- `AUTH_APPLE_CID` (**required**) - Client ID (App ID or Services ID)
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
     - **Application privacy policy link** - `/web/privacy.html` of your Remark42 installation, e.g. `https://remark42.mysite.com/web/privacy.html` (please check that it works). The page shipped with Remark42 describes remark42.com rather than your site; to serve your own, add a `privacy.html` alongside the frontend files in the `web-root` / `REMARK_WEB_ROOT` directory (`/srv/web` in the Docker image), which is served in preference to the built-in one. Add it to that directory rather than creating one: once the directory exists it supplies the whole frontend, so a web root holding only a `privacy.html` leaves the widget unreachable
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

1. Register a new application [in the Microsoft Entra admin center](https://learn.microsoft.com/en-us/graph/auth-register-app-v2)
2. Under **"Supported account types"** select **"Accounts in any organizational directory (Any Microsoft Entra directory - Multitenant) and personal Microsoft accounts"**
3. Under **"Authentication/Platform configurations/Web"** enter the correct URL constructed as domain + `/auth/microsoft/callback`, i.e., `https://remark42.mysite.com/auth/microsoft/callback`
4. In **"Overview"** take note of the **Application (client) ID** (`AUTH_MICROSOFT_CID`)
5. Select **"Certificates & secrets"** and click on **"+ New Client Secret"** (`AUTH_MICROSOFT_CSEC`)

By default, Remark42 authenticates against the `common` endpoint, which requires a multi-tenant application. A single-tenant application (`signInAudience: AzureADMyOrg`) is rejected there with [`AADSTS50194`](https://learn.microsoft.com/en-us/entra/identity-platform/reference-error-codes). The account types the application actually accepts are the intersection of the endpoint and the registration, so selecting personal Microsoft accounts as well is what makes them usable; choose `signInAudience: AzureADMultipleOrgs` instead to keep sign-in to work or school accounts only. You can check and change the value under **"Manifest"** for an application that already exists.

To use a different endpoint, set `AUTH_MICROSOFT_TENANT` to your tenant ID or domain name for a single-tenant Entra ID application (`AzureADMyOrg`), to `organizations` for a multi-tenant application accepting work or school accounts only (`AzureADMultipleOrgs`), or to `consumers` for an application limited to personal Microsoft accounts (`PersonalMicrosoftAccount`).

### X (Twitter)

1. Create a new app in the [X Developer Console](https://console.x.com)
2. On the app page, under **"User authentication settings"**, press **"Set up"** and choose **"Read"** in **"App permissions"**
3. In **"Callback URI / redirect URL"** enter your `REMARK_URL` with `/auth/twitter/callback` appended, e.g., `https://remark42.mysite.com/auth/twitter/callback`. Set the required **"Website URL"** to the URL of your site and save the settings
4. Under **"Keys & Tokens"**, in **"OAuth 1.0a keys"**, copy the **Consumer Key** and the **Consumer Key Secret** into `AUTH_TWITTER_CID` and `AUTH_TWITTER_CSEC`. If the console no longer shows them in full, **"Regenerate"** issues a new pair

Remark42 signs users in to X with OAuth 1.0a. The **Client ID** and **Client Secret** under **"OAuth 2.0 keys"** are a different pair and do not work here.

### Yandex

1. Create a new **"OAuth App"**: https://oauth.yandex.com/client/new
2. Fill **"App name"** for your site
3. Under **Platforms** select **"Web services"** and enter **"Callback URI #1"** constructed as domain + `/auth/yandex/callback`, i.e., `https://remark42.mysite.com/auth/yandex/callback`
4. Select **Permissions**. You need the following permissions only from the **"Yandex.Passport API"** section:

- Access to the user avatar
- Access to username, first name and surname, gender

5. Fill out the rest of the fields if needed
6. Take note of the **ID** (`AUTH_YANDEX_CID`) and **Password** (`AUTH_YANDEX_CSEC`)

For more details refer to the [Yandex ID documentation](https://yandex.com/dev/id/doc/en/).

### Patreon

1. Create a new Patreon client https://www.patreon.com/portal/registration/register-clients
2. Fill **App Name**, **Description**
3. In the field **Redirect URIs** enter the correct URI constructed as domain + `/auth/patreon/callback`, i.e., `https://remark42.mysite.com/auth/patreon/callback`
4. Expand client details and note the **Client ID** and **Client Secret**. Those will be used as `AUTH_PATREON_CID` and `AUTH_PATREON_CSEC`

With `SUBSCRIBERS_ONLY` set, only paid Patreon subscribers can comment. Patreon reports whether a user pays only at the sign-in, so the paid status counts for two weeks from the sign-in with Patreon; after that the user signs in with Patreon again to keep commenting, and a user who no longer pays loses access then. A session from before an upgrade to this behaviour, or from before `SUBSCRIBERS_ONLY` was set, has no such period and counts as unpaid until the next sign-in with Patreon.

### Discord

1. Click on **New Application** to create Oauth client https://discord.com/developers/applications
2. After filling **"NAME"**, navigate to **"OAuth2"** option on the left sidebar
3. Under **"Redirects"** enter the correct url constructed as domain + `/auth/discord/callback`. ie `https://remark42.mysite.com/auth/discord/callback`
4. Take note of the **CLIENT ID** and **CLIENT SECRET**, as they are values for `AUTH_DISCORD_CID` and `AUTH_DISCORD_CSEC` respectively

### Custom OAuth2 Provider

You can configure any OAuth2-compatible provider by setting these variables:

- `AUTH_CUSTOM_NAME` - provider name used in auth routes
- `AUTH_CUSTOM_CID` - OAuth client ID
- `AUTH_CUSTOM_CSEC` - OAuth client secret
- `AUTH_CUSTOM_AUTH_URL` - authorization endpoint
- `AUTH_CUSTOM_TOKEN_URL` - token endpoint
- `AUTH_CUSTOM_INFO_URL` - user info endpoint
- `AUTH_CUSTOM_SCOPES` - optional scopes, comma-separated
- `AUTH_CUSTOM_ID_FIELD` - user info field used as unique id (default `sub`); set it when the provider identifies users by another field, such as `id`
- `AUTH_CUSTOM_NAME_FIELD` - optional user info field used as display name (default `name`)
- `AUTH_CUSTOM_PICTURE_FIELD` - optional user info field used as avatar URL (default `picture`)
- `AUTH_CUSTOM_EMAIL_FIELD` - optional user info field holding the email (default `email`); used only to make the user's unique id when the response has no value in the `AUTH_CUSTOM_ID_FIELD` field, and not stored

Callback URL format:

`https://<remark42-url>/auth/<AUTH_CUSTOM_NAME>/callback`

Notes:

- `AUTH_CUSTOM_NAME` is lower-cased, and must then match `^[a-z0-9][a-z0-9_-]*$` and must not be the name of a built-in provider, or Remark42 will fail to start: `email`, `anonymous`, `google`, `github`, `facebook`, `yandex`, `twitter`, `microsoft`, `patreon`, `discord`, `telegram`, `dev`, `apple`.
- If any required custom variable is missing, Remark42 will fail to start.
- Remark42 currently supports only one custom OAuth2 provider at a time.
- A user is identified by the value of the `AUTH_CUSTOM_ID_FIELD` field, so it has to be a value that every user has and that the provider never changes and never gives to another user. When the user info has no value in that field, the user is identified by the email from `AUTH_CUSTOM_EMAIL_FIELD`, and the server log has a warning for each such sign-in. That is safe only with a provider that verifies email addresses and never gives an address to a second account, also after the first one changes its address or is closed: with any other provider, whoever gets another user's address into their own account signs in to Remark42 as that user. A sign-in whose user info has neither value is refused, and the server log names the missing field. The display name and the picture never identify a user. Changing `AUTH_CUSTOM_ID_FIELD` on a running instance gives a new ID to every user whose identifying value changes with it, which includes users identified by their email until then: their earlier comments stay under the old ID, and an admin among them has to be listed again in `ADMIN_SHARED_ID`.
- The field variables name top-level keys of the user info response only; a field nested inside an object cannot be used.
- Remark42 sends a PKCE (S256) challenge with every authorization request; providers without PKCE support ignore it.
- The user's picture is downloaded from the URL in the picture field without the access token, and only from a public address. A picture on a private, loopback or link-local address, or one that needs the token, is not loaded, and the user gets a generated avatar.

### Your Site's Own Accounts

A site with its own user accounts can sign its users in to the widget by issuing Remark42 tokens on its server. The steps assume the default `ADMIN_TYPE=shared`; with `ADMIN_TYPE=rpc`, tokens are signed with the key the admin store supplies instead of `SECRET`.

1. Serve Remark42 under a path of your site, such as `https://example.com/remark42`, as the [manual for running without a subdomain](/docs/manuals/subdomain/) shows, so your server can set the cookies for its own host only. With Remark42 on a subdomain, the cookies need the parent domain, and every host under it, such as a hosted help or status page on a CNAME, receives the token and can act as the user. Do that only when every host under the domain is one you trust.
1. Enable at least one sign-in provider, for example `AUTH_ANON=true`.
1. On your server, issue a JWT signed with HS256 and the value of `SECRET`:

   ```json
   {
     "aud": "<site ID>",
     "iss": "remark42",
     "exp": 1798761600,
     "jti": "<random unique ID>",
     "user": {
       "id": "anonymous_<your user ID>",
       "name": "User Name",
       "picture": "https://example.com/avatar.png"
     }
   }
   ```

   `aud` is the site ID, one of the values of `SITE`. `user.id` is the name of an enabled provider, an underscore and your own ID for the user; a token with any other prefix is refused with "provider is not allowed". When the provider name itself contains an underscore, also add `"auth_provider": {"name": "<provider>"}` at the top level of the token. `picture` is optional. A user under the `anonymous_` prefix has the limits of an [anonymous user](#anonymous).
1. Set two cookies together, with `Path=/` and no `Domain` attribute, or the parent domain when Remark42 is on a subdomain: `JWT` with the token, and `XSRF-TOKEN` with the token's `jti`. Set `JWT` with `HttpOnly` and `Secure`, and serve your site and Remark42 over HTTPS: a script that reads `JWT` can send it in the `X-JWT` header, which has no XSRF check. `XSRF-TOKEN` must not be `HttpOnly`: the widget copies it into the `X-XSRF-TOKEN` header, and a `JWT` cookie without a matching header is refused.

The widget then opens signed in. A call from your server can send the token in the `X-JWT` header instead. Remark42 does not read the `Authorization: Bearer` header, and `remark_config` has no option for a token.

- An expired token is renewed only on the cookie route: Remark42 sets both cookies again with a fresh token on the next request. A token in the `X-JWT` header is refused once it expires.
- Signing out in the widget does not remove cookies your site set for a parent domain, and once Remark42 renews a token it sets its own pair for its host. When the user signs out of your site, call Remark42's `/auth/logout` from the browser with credentials and the `X-XSRF-TOKEN` header, then clear the two cookies you set.
- Whoever holds `SECRET` can issue a token for any user, an admin included. Keep `SECRET` on the server and issue tokens only there.

If your site runs an OAuth2 server, the [Custom OAuth2 Provider](#custom-oauth2-provider) signs users in without self-issued tokens.

### Telegram

1. Contact [@BotFather](https://t.me/botfather) and follow his instructions to create your bot (call it, for example, "My site auth bot")
1. Write down the resulting token as `TELEGRAM_TOKEN` into remark42 config, and also set `AUTH_TELEGRAM` to `true` to enable telegram auth for your users.

### Anonymous

Optionally, anonymous access can be turned on. In this case, an extra `anonymous` provider will allow logins without any social login with any name satisfying two conditions:

- the name is 3 to 64 characters long
- the name contains only letters, ASCII digits, underscores, dots, hyphens and ordinary spaces, with no space at the start or end

Email sign-in accepts the same names.

An anonymous user is the name together with the client address the sign-in came from (see [Trusted proxies and client IP](/docs/configuration/parameters/#trusted-proxies-and-client-ip)). The same name signed in from another address is another user, who cannot edit the comments posted earlier. Anonymous users cannot upload pictures, subscribe to replies or request deletion of their data, and they can vote only when both `ANON_VOTE` and `VOTES_IP` are enabled.
