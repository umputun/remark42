import api from '@app/common/api';
import { Sorting } from '@app/common/types';
import { COOKIE_SORT_KEY } from '@app/common/constants';
import { setCookie } from '@app/common/cookies';

import { StoreAction } from '../index';
import { setPostInfo } from '../post_info/actions';
import { setComments } from '../comments/actions';
import { SORT_SET, SORT_SET_ACTION } from './types';

export const setSort = (sort: Sorting): StoreAction<Promise<void>> => async dispatch => {
  try {
    setCookie(COOKIE_SORT_KEY, sort, { expires: 60 * 60 * 24 * 365 }); // save sorting for a year
  } catch (e) {
    // can't save; ignore it
  }

  const info = await api.getPostComments(sort);

  const action: SORT_SET_ACTION = {
    type: SORT_SET,
    sort,
  };

  dispatch(action);
  dispatch(setPostInfo(info.info));
  dispatch(setComments(info.comments));
};
