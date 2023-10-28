import 'jest-fetch-mock';
import type { Theme } from 'common/types';

type RemarkConfig = {
  // Hostname of Remark42 server, same as REMARK_URL in backend config, e.g. "https://demo.remark42.com".
  host?: string;
  // The SITE that you passed to Remark42 instance on start of backend.
  site_id: string;
  // Optional, 'window.location.origin + window.location.pathname' by default.
  // URL to the page with comments, it is used as unique identificator for comments thread
  //
  // Note that if you use query parameters as significant part of URL(the one that actually changes content on page)
  // you will have to configure URL manually to keep query params, as 'window.location.origin + window.location.pathname'
  // doesn't contain query params and hash. For example, default URL for 'https://example/com/example-post?id=1#hash'
  // would be 'https://example/com/example-post'
  url?: string;
  // Optional, '15' by default. Maximum number of comments that is rendered on mobile version.
  max_shown_comments?: number;
  // Optional, '15' by default. Maximum number of comments in the last comments widget.
  max_last_comments?: number;
  // Optional, 'dark' or 'light', 'light' by default. Changes UI theme.
  theme?: Theme;
  // Optional, 'document.title' by default. Title for current comments page.
  page_title?: string;
  // Optional, 'en' by default. Interface localization.
  locale?: string;
  // Optional, 'true' by default. Enables email subscription feature in interface when enable it from backend side,
  // if you set this param in 'false' you will get notifications email notifications as admin but your users
  // won't have interface for subscription
  show_email_subscription?: boolean;
  // Optional, 'true' by default. Enables telegram subscription feature in interface when enable it from backend side,
  // if you set this param in 'false' you will get telegram notifications as admin but your users
  // won't have interface for subscription
  show_telegram_subscription?: boolean;
  // Optional, 'true' by default. Enables RSS subscription feature in interface.
  show_rss_subscription?: boolean;
  // Optional, 'false' by default. Overrides the parameter from the backend minimized UI with basic info only.
  simple_view?: boolean;
  // Optional, 'false' by default. Hides footer with signature and links to Remark42.
  no_footer?: boolean;
  __colors__?: Record<string, string>;
};

declare global {
  interface Window {
    __REDUX_DEVTOOLS_EXTENSION_COMPOSE__?: typeof compose;
    /** only for dev env */
    ReduxStore: typeof store;
    remark_config: RemarkConfig;
    REMARK42: {
      changeTheme?: (theme: Theme) => void;
      destroy?: () => void;
      createInstance: (remark_config: RemarkConfig) =>
        | {
            changeTheme(theme: Theme): void;
            destroy(): void;
          }
        | undefined;
    };
  }
}
