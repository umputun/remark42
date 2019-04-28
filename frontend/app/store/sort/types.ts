import { Sorting } from '@app/common/types';

export const SORT_SET = 'SORT/SET';

export interface SORT_SET_ACTION {
  type: typeof SORT_SET;
  sort: Sorting;
}

export type SORT_ACTIONS = SORT_SET_ACTION;
