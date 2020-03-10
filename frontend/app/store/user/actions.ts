import * as api from '@app/common/api';
import { User, BlockedUser, AuthProvider, BlockTTL } from '@app/common/types';
import { ttlToTime } from '@app/utils/ttl-to-time';

import { StoreAction } from '../index';
import {
  USER_BAN,
  USER_SET,
  USER_UNBAN,
  USER_BANLIST_SET,
  USER_HIDELIST_SET,
  USER_HIDE,
  USER_UNHIDE,
  USER_SUBSCRIPTION_SET,
  USER_SET_ACTION,
} from './types';
import { unsetCommentMode, fetchComments } from '../comments/actions';
import { IS_STORAGE_AVAILABLE, LS_HIDDEN_USERS_KEY } from '@app/common/constants';
import { getItem } from '@app/common/local-storage';
import { updateProvider } from '../provider/actions';
import { COMMENTS_PATCH } from '../comments/types';

function setUser(user: User | null = null) {
  return {
    type: USER_SET,
    user,
  } as USER_SET_ACTION;
}

export const fetchUser = (): StoreAction<Promise<User | null>> => async dispatch => {
  const user = await api.getUser();
  dispatch(setUser(user));
  return user;
};

export const logIn = (provider: AuthProvider): StoreAction<Promise<User | null>> => async dispatch => {
  const user = await api.logIn(provider);

  dispatch(updateProvider({ name: provider.name }));
  dispatch(setUser(user));
  dispatch(fetchComments());

  return user;
};

export const logout = (): StoreAction<Promise<void>> => async dispatch => {
  await api.logOut();
  dispatch(unsetCommentMode());
  dispatch(setUser());
};

export const fetchBlockedUsers = (): StoreAction<Promise<BlockedUser[]>> => async dispatch => {
  const list = (await api.getBlocked()) || [];
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

export const unblockUser = (id: User['id']): StoreAction<Promise<void>> => async (dispatch, getState) => {
  await api.unblockUser(id);
  dispatch({
    type: USER_UNBAN,
    id,
  });
  const comments = Object.values(getState().comments).filter(c => c.user.id === id);

  if (!comments.length) return;
  const user = comments[0].user;

  dispatch({
    type: COMMENTS_PATCH,
    ids: comments.map(c => c.id),
    patch: { user: { ...user, block: false } },
  });
};

export const fetchHiddenUsers = (): StoreAction<void> => dispatch => {
  if (!IS_STORAGE_AVAILABLE) return;

  const hiddenUsers = JSON.parse(getItem(LS_HIDDEN_USERS_KEY) || '{}');
  dispatch({ type: USER_HIDELIST_SET, payload: hiddenUsers });
};

export const hideUser = (user: User): StoreAction<void> => (dispatch, getState) => {
  if (IS_STORAGE_AVAILABLE) {
    const hiddenUsers = JSON.parse(getItem(LS_HIDDEN_USERS_KEY) || '{}');
    hiddenUsers[user.id] = user;
    localStorage.setItem(LS_HIDDEN_USERS_KEY, JSON.stringify(hiddenUsers));
  }
  dispatch({ type: USER_HIDE, user });

  dispatch({
    type: COMMENTS_PATCH,
    ids: Object.values(getState().comments)
      .filter(c => c.user.id === user.id)
      .map(c => c.id),
    patch: { hidden: true },
  });
};

export const unhideUser = (userId: string): StoreAction<void> => (dispatch, _getState) => {
  if (IS_STORAGE_AVAILABLE) {
    const hiddenUsers = JSON.parse(getItem(LS_HIDDEN_USERS_KEY) || '{}');
    if (Object.prototype.hasOwnProperty.call(hiddenUsers, userId)) {
      delete hiddenUsers[userId];
    }
    localStorage.setItem(LS_HIDDEN_USERS_KEY, JSON.stringify(hiddenUsers));
  }

  dispatch({ type: USER_UNHIDE, id: userId });

  // no need for comments patch as comments will be refetched after action
};

export const setVerifiedStatus = (id: User['id'], status: boolean): StoreAction<Promise<void>> => async (
  dispatch,
  getState
) => {
  if (status) {
    await api.setVerifiedStatus(id);
  } else {
    await api.removeVerifiedStatus(id);
  }
  const comments = Object.values(getState().comments).filter(c => c.user.id === id);
  if (!comments.length) return;
  const user = comments[0].user;

  dispatch({
    type: COMMENTS_PATCH,
    ids: comments.map(c => c.id),
    patch: { user: { ...user, verified: status } },
  });
};

export const setUserSubscribed = (isSubscribed: boolean) => ({
  type: USER_SUBSCRIPTION_SET,
  payload: isSubscribed,
});
