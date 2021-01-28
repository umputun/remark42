import { getUserComments } from 'common/api';
import { Comment } from 'common/types';
import { userInfo } from 'common/user-info-settings';

import { StoreAction } from '../index';
import { USER_INFO_SET } from './types';

export const fetchInfo = (): StoreAction<Promise<Comment[] | null>> => async (dispatch) => {
  if (!userInfo.id) {
    return null;
  }
  // TODO: limit
  const info = await getUserComments(userInfo.id, 10);
  dispatch({
    type: USER_INFO_SET,
    id: userInfo.id,
    comments: info.comments,
  });
  return info.comments;
};
