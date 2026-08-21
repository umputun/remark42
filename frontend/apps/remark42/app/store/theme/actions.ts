import type { Theme } from 'common/types';

import type { StoreAction } from '../';
import { THEME_SET } from './types';

export const setTheme =
  (theme: Theme): StoreAction =>
  (dispatch) =>
    dispatch({
      type: THEME_SET,
      theme,
    });
