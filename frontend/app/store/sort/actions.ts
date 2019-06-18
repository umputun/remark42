import { Sorting } from '@app/common/types';
import { COOKIE_SORT_KEY } from '@app/common/constants';
import { setCookie } from '@app/common/cookies';

import { StoreAction } from '../index';
import { fetchComments } from '../comments/actions';
import { SORT_SET, SORT_SET_ACTION } from './types';

function setSortCookie(sort: Sorting) {
  try {
    setCookie(COOKIE_SORT_KEY, sort, { expires: 60 * 60 * 24 * 365 }); // save sorting for a year
  } catch {
    // can't save; ignore it
  }
}

export const setSort = (sort: Sorting): StoreAction<Promise<void>> => async (dispatch, getState) => {
  const originalSort = getState().sort;
  setSortCookie(sort);

  try {
    const action: SORT_SET_ACTION = {
      type: SORT_SET,
      sort,
    };

    await dispatch(action);
    await dispatch(fetchComments(sort));
  } catch {
    // restore sort in case of error, probably network error

    const action: SORT_SET_ACTION = {
      type: SORT_SET,
      sort: originalSort,
    };

    setSortCookie(originalSort);
    await dispatch(action);
  }
};
