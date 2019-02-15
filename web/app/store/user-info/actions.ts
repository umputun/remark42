import api from '@app/common/api';
import { Comment, User } from '@app/common/types';
import { userInfo } from '@app/common/user-info-settings';

import { StoreAction } from '../index';
import { USER_INFO_SET } from './types';

export const getUserComments = (userId: User['id']): StoreAction<Comment[] | null> => (_dispatch, getState) =>
  getState().userComments![userId] || null;

export const getComments = (id: User['id']): StoreAction<Comment[] | null> => (_dispatch, getState) => {
  const comments = getState().userComments![id];
  if (comments) {
    return comments;
  }
  return null;
};

export const fetchInfo = (): StoreAction<Promise<Comment[] | null>> => async dispatch => {
  if (!userInfo.id) {
    return null;
  }
  // TODO: limit
  const info = await api.getUserComments({ userId: userInfo.id, limit: 10 });
  dispatch({
    type: USER_INFO_SET,
    id: userInfo.id,
    comments: info.comments,
  });
  return info.comments;
};
