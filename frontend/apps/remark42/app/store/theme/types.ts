import { Theme } from 'common/types';

export const THEME_SET = 'THEME/SET';

export interface THEME_SET_ACTION {
  type: typeof THEME_SET;
  theme: Theme;
}

export type THEME_ACTIONS = THEME_SET_ACTION;
