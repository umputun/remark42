import type { Theme } from 'common/types';
import * as settings from 'common/settings';
import { resolveTheme } from 'utils/theme';

import type { THEME_SET_ACTION } from './types';
import { THEME_SET } from './types';

// the store holds the concrete theme the widget renders with; an 'auto' setting is resolved to the
// OS preference here, and remark.tsx keeps it updated as the OS theme changes
export function theme(state: Theme = resolveTheme(settings.theme), action: THEME_SET_ACTION): Theme {
  switch (action.type) {
    case THEME_SET: {
      return action.theme;
    }
    default:
      return state;
  }
}
