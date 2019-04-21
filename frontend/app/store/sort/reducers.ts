import { Sorting } from '@app/common/types';
import { COOKIE_SORT_KEY, DEFAULT_SORT } from '@app/common/constants';
import { getCookie } from '@app/common/cookies';

import { SORT_SET, SORT_SET_ACTION } from './types';

const getDefaultSort = (): Sorting => {
  try {
    return (getCookie(COOKIE_SORT_KEY) as Sorting) || DEFAULT_SORT;
  } catch (e) {
    return DEFAULT_SORT;
  }
};

export const sort = (state: Sorting = getDefaultSort(), action: SORT_SET_ACTION): Sorting => {
  switch (action.type) {
    case SORT_SET: {
      return action.sort;
    }
    default:
      return state;
  }
};

export default { sort };
