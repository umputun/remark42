import * as settings from 'common/settings';
import { ThemeStyling } from 'common/theme';

import { STYLING_SET, STYLING_SET_ACTION } from './types';

export function styling(state: ThemeStyling = settings.styling || {}, action: STYLING_SET_ACTION): ThemeStyling {
  switch (action.type) {
    case STYLING_SET: {
      return action.styling || {};
    }
    default:
      return state;
  }
}
