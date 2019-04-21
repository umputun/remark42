import api from '@app/common/api';
import { User, BlockedUser, AuthProvider, BlockTTL } from '@app/common/types';
import { ttlToTime } from '@app/utils/ttl-to-time';

import { StoreAction } from '../index';
import { USER_BAN, USER_SET, USER_UNBAN, BLOCKED_VISIBLE_SET, USER_BANLIST_SET } from './types';
import { setComments, unsetCommentMode } from '../comments/actions';
import { setUserVerified as uSetUserVerified } from '../comments/utils';

export const fetchUser = (): StoreAction<Promise<User | null>> => async dispatch => {
  const user = await api.getUser();
  dispatch({
    type: USER_SET,
    user,
  });
  return user;
};

export const logIn = (provider: AuthProvider): StoreAction<Promise<User | null>> => async dispatch => {
  const user = await api.logIn(provider);
  dispatch({
    type: USER_SET,
    user,
  });
  return user;
};

export const logout = (): StoreAction<Promise<void>> => async dispatch => {
  await api.logOut();
  dispatch(unsetCommentMode());
  dispatch({
    type: USER_SET,
    user: null,
  });
};

export const fetchBlockedUsers = (): StoreAction<Promise<BlockedUser[]>> => async dispatch => {
  const list = await api.getBlocked();
  dispatch({
    type: USER_BANLIST_SET,
    list,
  });
  return list;
};

export const blockUser = (
  id: User['id'],
  name: string,
  ttl: BlockTTL
): StoreAction<Promise<void>> => async dispatch => {
  await api.blockUser(id, ttl);
  dispatch({
    type: USER_BAN,
    user: {
      id,
      name,
      time: ttlToTime(ttl),
    },
  });
};

export const unblockUser = (id: User['id']): StoreAction<Promise<void>> => async dispatch => {
  await api.unblockUser(id);
  dispatch({
    type: USER_UNBAN,
    id,
  });
};

export const setVirifiedStatus = (id: User['id'], status: boolean): StoreAction<Promise<void>> => async (
  dispatch,
  getState
) => {
  if (status) {
    await api.setVerifyStatus(id);
  } else {
    await api.removeVerifyStatus(id);
  }
  const comments = getState().comments;
  dispatch(setComments(uSetUserVerified(comments, id, status)));
};

export const setBlockedVisibleState = (state: boolean): StoreAction<boolean> => dispatch => {
  dispatch({
    type: BLOCKED_VISIBLE_SET,
    state,
  });
  return state;
};
