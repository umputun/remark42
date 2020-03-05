import { Sorting } from '@app/common/types';
import { COOKIE_SORT_KEY } from '@app/common/constants';
import { setCookie } from '@app/common/cookies';

import { StoreAction } from '../index';
import { fetchComments } from '../comments/actions';
import { SORT_SET, SORT_SET_ACTION } from './types';

function setSortCookie(sort: Sorting) {
  try {
    setCookie(COOKIE_SORT_KEY, sort, { expires: 60 * 60 * 24 * 365, path: '/' }); // save sorting for a year
  } catch {
    // can't save; ignore it
  }
}

function setSort(sort: Sorting): SORT_SET_ACTION {
  return {
    type: SORT_SET,
    sort,
  };
}

export const changeSort = (sort: Sorting): StoreAction<Promise<void>> => async (dispatch, getState) => {
  const { sort: originalSort } = getState();

  try {
    setSortCookie(sort);
    dispatch(setSort(sort));
    await dispatch(fetchComments());
  } catch {
    // restore sort in case of error, probably network error
    setSortCookie(originalSort);
    dispatch(setSort(originalSort));
  }
};
