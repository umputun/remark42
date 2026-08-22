import type { Theme } from 'common/types';
import * as settings from 'common/settings';

import type { THEME_SET_ACTION } from './types';
import { THEME_SET } from './types';

export function theme(state: Theme = settings.theme, action: THEME_SET_ACTION): Theme {
  switch (action.type) {
    case THEME_SET: {
      return action.theme;
    }
    default:
      return state;
  }
}
