import { ThemeStyling } from 'common/theme';

export const STYLING_SET = 'STYLING/SET';

export interface STYLING_SET_ACTION {
  type: typeof STYLING_SET;
  styling: ThemeStyling | undefined;
}

export type STYLING_ACTIONS = STYLING_SET_ACTION;
