import { Theme } from '@app/common/types';
import { CommentsConfig } from '@app/common/config-types';

declare global {
  interface Window {
    remark_config: CommentsConfig;
    REMARK42: {
      changeTheme?: (theme: Theme) => void;
      destroy?: () => void;
      createInstance: (
        remark_config: CommentsConfig
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
      fetch: typeof fetch;
    }
  }
}
