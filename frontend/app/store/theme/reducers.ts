import { Theme } from '@app/common/types';
import { StaticStore } from '@app/common/static_store';

import { THEME_SET_ACTION, THEME_SET } from './types';

export const theme = (state: Theme = StaticStore.query.theme, action: THEME_SET_ACTION): Theme => {
  switch (action.type) {
    case THEME_SET: {
      return action.theme;
    }
    default:
      return state;
  }
};

export default { theme };
