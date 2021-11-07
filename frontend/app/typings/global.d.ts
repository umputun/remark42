import 'jest-fetch-mock';
import type { Theme } from 'common/types';

type RemarkConfig = {
  host: string;
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
};

declare global {
  interface Window {
    remark_config: RemarkConfig;
    REMARK42: {
      changeTheme?: (theme: Theme) => void;
      destroy?: () => void;
      createInstance: (
        remark_config: RemarkConfig
      ) =>
        | {
            changeTheme(theme: Theme): void;
            destroy(): void;
          }
        | undefined;
    };
  }

  namespace NodeJS {
    interface Global {
      Headers: typeof Headers;
      localStorage: typeof Storage;
    }
  }
}

/**
 * Variable responsive for dynamic setting public path for
 * assets. Dynamic imports with relative url will be resolved over this path.
 *
 * https://webpack.js.org/guides/public-path/#on-the-fly
 */
declare let __webpack_public_path__: string;
