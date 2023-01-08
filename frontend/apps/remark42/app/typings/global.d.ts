import 'jest-fetch-mock';
import type { Theme } from 'common/types';

type RemarkConfig = {
  host?: string;
  site_id: string;
  url?: string;
  max_shown_comments?: number;
  theme?: Theme;
  page_title?: string;
  locale?: string;
  show_email_subscription?: boolean;
  max_last_comments?: number;
  __colors__?: Record<string, string>;
  simple_view?: boolean;
  no_footer?: boolean;
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
