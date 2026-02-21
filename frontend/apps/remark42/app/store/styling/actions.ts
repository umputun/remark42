import { ThemeStyling } from 'common/theme';

import { StoreAction } from '../';
import { STYLING_SET } from './types';

export const setStyling =
  (styling: ThemeStyling | undefined): StoreAction =>
  (dispatch) =>
    dispatch({
      type: STYLING_SET,
      styling,
    });
