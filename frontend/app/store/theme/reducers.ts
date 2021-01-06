import { Theme } from 'common/types';
import { StaticStore } from 'common/static-store';

import { THEME_SET_ACTION, THEME_SET } from './types';

export function theme(state: Theme = StaticStore.query.theme, action: THEME_SET_ACTION): Theme {
  switch (action.type) {
    case THEME_SET: {
      return action.theme;
    }
    default:
      return state;
  }
}
