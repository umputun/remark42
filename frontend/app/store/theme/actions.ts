import { Theme } from 'common/types';

import { StoreAction } from '../';
import { THEME_SET } from './types';

export const setTheme = (theme: Theme): StoreAction<void> => (dispatch) =>
  dispatch({
    type: THEME_SET,
    theme,
  });
