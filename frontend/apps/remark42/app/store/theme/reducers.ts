import { Theme } from 'common/types';
import * as settings from 'common/settings';

import { THEME_SET_ACTION, THEME_SET } from './types';

export function theme(state: Theme = settings.theme, action: THEME_SET_ACTION): Theme {
  switch (action.type) {
    case THEME_SET: {
      return action.theme;
    }
    default:
      return state;
  }
}
